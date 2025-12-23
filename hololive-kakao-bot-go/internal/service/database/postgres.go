package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL 드라이버 등록
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
)

// PostgresService 는 타입이다.
type PostgresService struct {
	db     *sql.DB
	gormDB *gorm.DB
	logger *zap.Logger
}

// PostgresConfig 는 타입이다.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// NewPostgresService 는 동작을 수행한다.
func NewPostgresService(cfg PostgresConfig, logger *zap.Logger) (*PostgresService, error) {
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
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
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

// GetDB 는 동작을 수행한다.
func (ps *PostgresService) GetDB() *sql.DB {
	return ps.db
}

// GetGormDB 는 동작을 수행한다.
func (ps *PostgresService) GetGormDB() *gorm.DB {
	return ps.gormDB
}

// Close 는 동작을 수행한다.
func (ps *PostgresService) Close() error {
	if ps.db != nil {
		if err := ps.db.Close(); err != nil {
			return fmt.Errorf("failed to close postgres: %w", err)
		}
	}
	return nil
}

// Ping 는 동작을 수행한다.
func (ps *PostgresService) Ping(ctx context.Context) error {
	if err := ps.db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}
	return nil
}
