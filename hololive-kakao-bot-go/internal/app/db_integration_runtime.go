package app

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
)

// DBIntegrationRuntime 는 타입이다.
type DBIntegrationRuntime struct {
	Logger           *zap.Logger
	Repository *member.Repository
	Cache      *member.Cache
	MemberAdapter    *member.ServiceAdapter

	cleanup func()
}

// Close 는 동작을 수행한다.
func (r *DBIntegrationRuntime) Close() {
	if r != nil && r.cleanup != nil {
		r.cleanup()
	}
}

// BuildDBIntegrationRuntime 는 동작을 수행한다.
func BuildDBIntegrationRuntime(
	ctx context.Context,
	pgCfg config.PostgresConfig,
	logger *zap.Logger,
) (*DBIntegrationRuntime, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger must not be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	runtime, cleanup, err := InitializeDBIntegrationRuntime(ctx, pgCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("DB integration 런타임 초기화 실패: %w", err)
	}
	runtime.cleanup = cleanup

	return runtime, nil
}
