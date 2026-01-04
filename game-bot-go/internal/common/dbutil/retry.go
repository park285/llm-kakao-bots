package dbutil

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// RetryConfig: DB 연결 재시도 설정
type RetryConfig struct {
	MaxAttempts int           // 최대 시도 횟수 (기본: 5)
	BaseDelay   time.Duration // 초기 대기 시간 (기본: 2초)
	MaxDelay    time.Duration // 최대 대기 시간 (기본: 30초)
}

// DefaultRetryConfig: 기본 재시도 설정
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   2 * time.Second,
		MaxDelay:    30 * time.Second,
	}
}

// OpenFunc: DB 연결을 시도하는 함수 타입
type OpenFunc func(ctx context.Context) (*gorm.DB, *sql.DB, error)

// OpenWithRetry: exponential backoff로 DB 연결을 재시도합니다.
// 스키마 마이그레이션이 완료되기 전 앱이 시작되는 Race Condition 방어용.
func OpenWithRetry(
	ctx context.Context,
	openFn OpenFunc,
	cfg RetryConfig,
	logger *slog.Logger,
) (*gorm.DB, *sql.DB, error) {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 2 * time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		db, sqlDB, err := openFn(ctx)
		if err == nil {
			if attempt > 0 && logger != nil {
				logger.Info("db_connect_success_after_retry",
					slog.Int("attempts", attempt+1),
				)
			}
			return db, sqlDB, nil
		}

		lastErr = err

		// 마지막 시도면 재시도하지 않음
		if attempt >= cfg.MaxAttempts-1 {
			break
		}

		// Exponential backoff: 2s, 4s, 8s, 16s, ... (최대 MaxDelay)
		delay := cfg.BaseDelay * time.Duration(1<<uint(attempt))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		if logger != nil {
			logger.Warn("db_connect_retry",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", cfg.MaxAttempts),
				slog.Duration("delay", delay),
				slog.Any("error", err),
			)
		}

		// context 취소 확인 후 대기
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("db connect cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return nil, nil, fmt.Errorf("db connect failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
