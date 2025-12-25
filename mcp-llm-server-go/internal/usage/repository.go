package usage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

// Repository 는 usage DB 접근을 담당한다.
type Repository struct {
	cfg    *config.Config
	logger *slog.Logger
	mu     sync.Mutex
	db     *gorm.DB
	sqlDB  *sql.DB
}

// NewRepository 는 usage 저장소를 생성한다.
func NewRepository(cfg *config.Config, logger *slog.Logger) *Repository {
	return &Repository{
		cfg:    cfg,
		logger: logger,
	}
}

// RecordUsage 는 지정한 날짜(또는 오늘)의 토큰 사용량을 누적 저장한다.
func (r *Repository) RecordUsage(
	ctx context.Context,
	inputTokens int64,
	outputTokens int64,
	reasoningTokens int64,
	requestCount int64,
	usageDate time.Time,
) error {
	if requestCount <= 0 && inputTokens <= 0 && outputTokens <= 0 {
		return nil
	}

	db, err := r.getDB(ctx)
	if err != nil {
		return err
	}

	targetDate := usageDate
	if targetDate.IsZero() {
		targetDate = todayDate()
	}

	row := TokenUsage{
		UsageDate:       targetDate,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		ReasoningTokens: reasoningTokens,
		RequestCount:    requestCount,
		Version:         0,
	}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "usage_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"input_tokens":     gorm.Expr("token_usage.input_tokens + EXCLUDED.input_tokens"),
			"output_tokens":    gorm.Expr("token_usage.output_tokens + EXCLUDED.output_tokens"),
			"reasoning_tokens": gorm.Expr("token_usage.reasoning_tokens + EXCLUDED.reasoning_tokens"),
			"request_count":    gorm.Expr("token_usage.request_count + EXCLUDED.request_count"),
			"version":          gorm.Expr("token_usage.version + 1"),
		}),
	}).Create(&row).Error
}

// GetDailyUsage 는 특정 날짜(또는 오늘)의 사용량을 조회한다.
func (r *Repository) GetDailyUsage(ctx context.Context, usageDate time.Time) (*DailyUsage, error) {
	db, err := r.getDB(ctx)
	if err != nil {
		return nil, err
	}

	targetDate := usageDate
	if targetDate.IsZero() {
		targetDate = todayDate()
	}

	var row TokenUsage
	result := db.WithContext(ctx).Where("usage_date = ?", targetDate).First(&row)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &DailyUsage{
		UsageDate:       row.UsageDate,
		InputTokens:     row.InputTokens,
		OutputTokens:    row.OutputTokens,
		ReasoningTokens: row.ReasoningTokens,
		RequestCount:    row.RequestCount,
	}, nil
}

// GetRecentUsage 는 최근 N일 사용량을 조회한다.
func (r *Repository) GetRecentUsage(ctx context.Context, days int) ([]DailyUsage, error) {
	db, err := r.getDB(ctx)
	if err != nil {
		return nil, err
	}
	if days <= 0 {
		days = 7
	}

	var rows []TokenUsage
	if err := db.WithContext(ctx).Order("usage_date desc").Limit(days).Find(&rows).Error; err != nil {
		return nil, err
	}

	usages := make([]DailyUsage, 0, len(rows))
	for _, row := range rows {
		usages = append(usages, DailyUsage{
			UsageDate:       row.UsageDate,
			InputTokens:     row.InputTokens,
			OutputTokens:    row.OutputTokens,
			ReasoningTokens: row.ReasoningTokens,
			RequestCount:    row.RequestCount,
		})
	}
	return usages, nil
}

// GetTotalUsage 는 최근 N일 합계를 조회한다.
func (r *Repository) GetTotalUsage(ctx context.Context, days int) (DailyUsage, error) {
	db, err := r.getDB(ctx)
	if err != nil {
		return DailyUsage{}, err
	}
	if days <= 0 {
		days = 30
	}

	type aggregate struct {
		InputTokens     int64
		OutputTokens    int64
		ReasoningTokens int64
		RequestCount    int64
	}

	var result aggregate
	if err := db.WithContext(ctx).Raw(`
			SELECT
				COALESCE(SUM(input_tokens), 0) as input_tokens,
				COALESCE(SUM(output_tokens), 0) as output_tokens,
				COALESCE(SUM(reasoning_tokens), 0) as reasoning_tokens,
				COALESCE(SUM(request_count), 0) as request_count
			FROM token_usage
			WHERE usage_date >= CURRENT_DATE - (?::int)`, days).Scan(&result).Error; err != nil {
		return DailyUsage{}, err
	}

	return DailyUsage{
		UsageDate:       todayDate(),
		InputTokens:     result.InputTokens,
		OutputTokens:    result.OutputTokens,
		ReasoningTokens: result.ReasoningTokens,
		RequestCount:    result.RequestCount,
	}, nil
}

// Close 는 DB 연결을 닫는다.
func (r *Repository) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sqlDB == nil {
		return
	}
	_ = r.sqlDB.Close()
	r.sqlDB = nil
	r.db = nil
}

func (r *Repository) getDB(ctx context.Context) (*gorm.DB, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db != nil {
		return r.db, nil
	}
	if r.cfg == nil {
		return nil, errors.New("database config is nil")
	}

	hostUsed := r.cfg.Database.Host
	dsn := r.cfg.Database.DSN()
	gormCfg := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}
	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil && shouldFallbackToLocalhost(err, r.cfg.Database.Host) {
		fallback := r.cfg.Database
		fallback.Host = "127.0.0.1"
		fallbackDSN := fallback.DSN()
		db, err = gorm.Open(postgres.Open(fallbackDSN), gormCfg)
		if err == nil {
			hostUsed = fallback.Host
			if r.logger != nil {
				r.logger.Warn(
					"usage_db_host_fallback",
					"configured_host", r.cfg.Database.Host,
					"effective_host", hostUsed,
				)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("open usage db: %w", err)
	}

	if schemaErr := ensureUsageSchema(db); schemaErr != nil {
		return nil, fmt.Errorf("prepare usage db: %w", schemaErr)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get usage db handle: %w", err)
	}

	sqlDB.SetMaxIdleConns(r.cfg.Database.MinPool)
	sqlDB.SetMaxOpenConns(r.cfg.Database.MaxPool)
	if r.cfg.Database.ConnMaxLifetimeMinutes > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(r.cfg.Database.ConnMaxLifetimeMinutes) * time.Minute)
	}
	if r.cfg.Database.ConnMaxIdleTimeMinutes > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(r.cfg.Database.ConnMaxIdleTimeMinutes) * time.Minute)
	}

	if r.logger != nil {
		r.logger.Info("usage_db_connected", "host", hostUsed, "name", r.cfg.Database.Name)
	}

	r.db = db
	r.sqlDB = sqlDB
	return db, nil
}

func ensureUsageSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}

	if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS token_usage (
				id BIGSERIAL PRIMARY KEY,
				usage_date DATE NOT NULL,
				input_tokens BIGINT NOT NULL DEFAULT 0,
				output_tokens BIGINT NOT NULL DEFAULT 0,
				reasoning_tokens BIGINT NOT NULL DEFAULT 0,
				request_count BIGINT NOT NULL DEFAULT 0,
				version BIGINT NOT NULL DEFAULT 0
			)
		`).Error; err != nil {
		return fmt.Errorf("create token_usage table: %w", err)
	}

	if err := db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_token_usage_usage_date
			ON token_usage (usage_date)
		`).Error; err != nil {
		return fmt.Errorf("create token_usage usage_date unique index: %w", err)
	}

	return nil
}

func todayDate() time.Time {
	now := time.Now().In(time.Local)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func shouldFallbackToLocalhost(err error, host string) bool {
	if err == nil {
		return false
	}
	if host == "" || host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
		return false
	}
	if !strings.EqualFold(host, "postgres") {
		return false
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return strings.EqualFold(dnsErr.Name, host)
	}

	lower := strings.ToLower(err.Error())
	hostLower := strings.ToLower(host)
	if strings.Contains(lower, "lookup "+hostLower) && strings.Contains(lower, "no such host") {
		return true
	}
	return strings.Contains(lower, "no such host") && strings.Contains(lower, hostLower)
}
