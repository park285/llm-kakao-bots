package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq" // PostgreSQL 드라이버 등록
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
)

// PostgresService: PostgreSQL 데이터베이스 연결 및 GORM 인스턴스를 관리하는 서비스
type PostgresService struct {
	db     *sql.DB
	gormDB *gorm.DB
	logger *slog.Logger
}

// PostgresConfig: PostgreSQL 접속 정보(Host, Port, User, Password, Database)를 담는 설정 구조체
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// NewPostgresService: 주어진 설정을 사용하여 PostgreSQL 연결을 수립하고 서비스를 초기화한다.
// 연결 풀 설정 및 초기 헬스 체크(Ping)를 수행하며, GORM 인스턴스도 함께 초기화한다.
func NewPostgresService(cfg PostgresConfig, logger *slog.Logger) (*PostgresService, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	db.SetMaxOpenConns(constants.DatabaseConfig.MaxOpenConns)
	db.SetMaxIdleConns(constants.DatabaseConfig.MaxIdleConns)
	db.SetConnMaxLifetime(constants.DatabaseConfig.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout.DatabasePing)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	logger.Info("PostgreSQL connected",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("database", cfg.Database),
	)

	// Initialize GORM with existing connection
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	return &PostgresService{
		db:     db,
		gormDB: gormDB,
		logger: logger,
	}, nil
}

// GetDB: 기본 sql.DB 인스턴스를 반환한다. (GORM이 아닌 raw SQL 사용 시 활용)
func (ps *PostgresService) GetDB() *sql.DB {
	return ps.db
}

// GetGormDB: GORM DB 인스턴스를 반환한다. (ORM 기반 DB 조작 시 활용)
func (ps *PostgresService) GetGormDB() *gorm.DB {
	return ps.gormDB
}

// Close: 데이터베이스 연결을 안전하게 종료한다.
func (ps *PostgresService) Close() error {
	if ps.db != nil {
		if err := ps.db.Close(); err != nil {
			return fmt.Errorf("failed to close postgres: %w", err)
		}
	}
	return nil
}

// Ping: 데이터베이스 연결 상태를 확인한다. (헬스 체크용)
func (ps *PostgresService) Ping(ctx context.Context) error {
	if err := ps.db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}
	return nil
}
