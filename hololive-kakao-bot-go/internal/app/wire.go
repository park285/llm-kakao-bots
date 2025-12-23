//go:build wireinject

package app

import (
	"context"

	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
)

//go:generate go run github.com/google/wire/cmd/wire@v0.7.0

// InitializeBotDependencies - Wire가 의존성 그래프를 분석하여 생성 코드 생성
// wire gen 명령으로 wire_gen.go 파일이 자동 생성됨
func InitializeBotDependencies(
	ctx context.Context,
	cfg *config.Config,
	logger *zap.Logger,
) (*bot.Dependencies, func(), error) {
	wire.Build(AppSet)
	return nil, nil, nil
}

// InitializeBotRuntime - cmd/bot 런타임 (Bot + MQ + Admin API 구성요소)
func InitializeBotRuntime(
	ctx context.Context,
	cfg *config.Config,
	logger *zap.Logger,
) (*BotRuntime, func(), error) {
	wire.Build(RuntimeSet)
	return nil, nil, nil
}

// InitializeWarmMemberCache - cmd/tools/warm_member_cache 전용
func InitializeWarmMemberCache(
	ctx context.Context,
	cfg *config.Config,
	logger *zap.Logger,
) (*member.Cache, func(), error) {
	wire.Build(WarmMemberCacheSet)
	return nil, nil, nil
}

// InitializeDBIntegrationRuntime - cmd/test_db_integration 전용
func InitializeDBIntegrationRuntime(
	ctx context.Context,
	pgCfg config.PostgresConfig,
	logger *zap.Logger,
) (*DBIntegrationRuntime, func(), error) {
	wire.Build(DBIntegrationSet)
	return nil, nil, nil
}

// InitializeFetchProfilesRuntime - cmd/tools/fetch_profiles 전용
func InitializeFetchProfilesRuntime(
	ctx context.Context,
) (*FetchProfilesRuntime, func(), error) {
	wire.Build(FetchProfilesSet)
	return nil, nil, nil
}
