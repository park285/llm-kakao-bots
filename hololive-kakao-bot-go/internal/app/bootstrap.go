package app

import (
	"context"
	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
)

// coreInfrastructure 는 공통 인프라 의존성을 담는다.
type coreInfrastructure struct {
	deps         *bot.Dependencies
	ytStack      *YouTubeStack
	cleanupCache func()
	cleanupDB    func()
}

// initCoreInfrastructure 는 공통 인프라를 초기화한다.
func initCoreInfrastructure(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*coreInfrastructure, error) {
	valkeyMQConfig := ProvideValkeyMQConfig(cfg)
	irisClient := ProvideIrisClient(ctx, valkeyMQConfig, logger)
	messageStack := ProvideMessageStack(cfg)

	valkeyConfig := ProvideValkeyConfig(cfg)
	cacheResources, cleanupCache, err := ProvideCacheResources(valkeyConfig, logger)
	if err != nil {
		return nil, err
	}
	cacheService := ProvideCacheService(cacheResources)

	postgresConfig := ProvidePostgresConfig(cfg)
	databaseResources, cleanupDB, err := ProvideDatabaseResources(postgresConfig, logger)
	if err != nil {
		cleanupCache()
		return nil, err
	}
	postgresService := ProvidePostgresService(databaseResources)

	memberRepository := ProvideMemberRepository(postgresService, logger)
	memberCache, err := ProvideMemberCache(ctx, memberRepository, cacheService, logger)
	if err != nil {
		cleanupDB()
		cleanupCache()
		return nil, err
	}

	holodexAPIKeys := ProvideHolodexAPIKeys(cfg)
	memberServiceAdapter := ProvideMemberServiceAdapter(memberCache)
	scraperService := ProvideScraperService(cacheService, memberServiceAdapter, logger)
	holodexService, err := ProvideHolodexService(holodexAPIKeys, cacheService, scraperService, logger)
	if err != nil {
		cleanupDB()
		cleanupCache()
		return nil, err
	}

	profileService, err := ProvideProfileService(ctx, cacheService, memberServiceAdapter, logger)
	if err != nil {
		cleanupDB()
		cleanupCache()
		return nil, err
	}

	alarmService := ProvideAlarmService(cfg, cacheService, holodexService, logger)
	memberDataProvider := ProvideMembersData(memberServiceAdapter)
	memberMatcher := ProvideMemberMatcher(ctx, memberDataProvider, cacheService, holodexService, logger)
	youTubeStatsRepository := ProvideYouTubeStatsRepository(postgresService, logger)
	youTubeStack := ProvideYouTubeStack(ctx, cfg, cacheService, memberServiceAdapter, youTubeStatsRepository, logger)
	activityLogger := ProvideActivityLogger(cfg, logger)
	settingsService := ProvideSettingsService(logger)

	aclService, err := ProvideACLService(ctx, cfg, postgresService, cacheService, logger)
	if err != nil {
		cleanupDB()
		cleanupCache()
		return nil, err
	}

	deps := ProvideBotDependencies(cfg, logger, irisClient, messageStack, cacheService, postgresService, memberRepository, memberCache, holodexService, profileService, alarmService, memberMatcher, memberDataProvider, youTubeStack, activityLogger, settingsService, aclService)

	return &coreInfrastructure{
		deps:         deps,
		ytStack:      youTubeStack,
		cleanupCache: cleanupCache,
		cleanupDB:    cleanupDB,
	}, nil
}

// InitializeBotDependencies - 봇 의존성을 초기화한다.
func InitializeBotDependencies(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*bot.Dependencies, func(), error) {
	infra, err := initCoreInfrastructure(ctx, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		infra.cleanupDB()
		infra.cleanupCache()
	}

	return infra.deps, cleanup, nil
}

// InitializeBotRuntime - cmd/bot 런타임 (Bot + MQ + Admin API 구성요소)
func InitializeBotRuntime(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*BotRuntime, func(), error) {
	infra, err := initCoreInfrastructure(ctx, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	runtime, err := buildBotRuntime(ctx, cfg, logger, infra)
	if err != nil {
		infra.cleanupDB()
		infra.cleanupCache()
		return nil, nil, err
	}

	cleanup := func() {
		infra.cleanupDB()
		infra.cleanupCache()
	}

	return runtime, cleanup, nil
}

// buildBotRuntime 는 런타임 구성요소를 조립한다.
func buildBotRuntime(ctx context.Context, cfg *config.Config, logger *slog.Logger, infra *coreInfrastructure) (*BotRuntime, error) {
	deps := infra.deps

	botBot, err := ProvideBot(deps)
	if err != nil {
		return nil, err
	}

	valkeyMQConfig := ProvideValkeyMQConfig(cfg)
	valkeyMQConsumer, err := ProvideValkeyMQConsumer(ctx, valkeyMQConfig, logger, botBot, deps.Cache)
	if err != nil {
		return nil, err
	}

	youTubeScheduler := ProvideYouTubeScheduler(deps)
	youTubeService := ProvideYouTubeService(infra.ytStack)
	valkeySessionStore := ProvideSessionStore(deps.Cache, logger)
	loginRateLimiter := ProvideLoginRateLimiter()
	securityConfig := ProvideSecurityConfig(cfg)
	adminCredentials := ProvideAdminCredentials(cfg)

	adminHandler := ProvideAdminHandler(deps.MemberRepo, deps.MemberCache, deps.Cache, deps.Alarm, deps.Holodex, youTubeService, deps.Activity, deps.Settings, deps.ACL, cfg, valkeySessionStore, loginRateLimiter, securityConfig, adminCredentials, logger)

	adminAllowedCIDRs, err := ProvideAdminAllowedCIDRs(cfg)
	if err != nil {
		return nil, err
	}

	adminRouter, err := ProvideAdminRouter(ctx, logger, adminHandler, valkeySessionStore, securityConfig, adminAllowedCIDRs)
	if err != nil {
		return nil, err
	}

	adminAddr := ProvideAdminAddr(cfg)
	adminServer := ProvideAdminServer(adminAddr, adminRouter)

	return &BotRuntime{
		Config:            cfg,
		Logger:            logger,
		Bot:               botBot,
		MQConsumer:        valkeyMQConsumer,
		Scheduler:         youTubeScheduler,
		AdminHandler:      adminHandler,
		Sessions:          valkeySessionStore,
		SecurityConfig:    securityConfig,
		AdminAllowedCIDRs: adminAllowedCIDRs,
		AdminRouter:       adminRouter,
		AdminAddr:         adminAddr,
		AdminServer:       adminServer,
	}, nil
}

// InitializeWarmMemberCache - cmd/tools/warm_member_cache 전용
func InitializeWarmMemberCache(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*member.Cache, func(), error) {
	postgresConfig := ProvidePostgresConfig(cfg)
	databaseResources, cleanupDB, err := ProvideDatabaseResources(postgresConfig, logger)
	if err != nil {
		return nil, nil, err
	}
	postgresService := ProvidePostgresService(databaseResources)
	memberRepository := ProvideMemberRepository(postgresService, logger)

	valkeyConfig := ProvideValkeyConfig(cfg)
	cacheResources, cleanupCache, err := ProvideCacheResources(valkeyConfig, logger)
	if err != nil {
		cleanupDB()
		return nil, nil, err
	}
	cacheService := ProvideCacheService(cacheResources)

	memberCache, err := ProvideMemberCache(ctx, memberRepository, cacheService, logger)
	if err != nil {
		cleanupCache()
		cleanupDB()
		return nil, nil, err
	}

	cleanup := func() {
		cleanupCache()
		cleanupDB()
	}

	return memberCache, cleanup, nil
}

// InitializeDBIntegrationRuntime - cmd/test_db_integration 전용
func InitializeDBIntegrationRuntime(ctx context.Context, pgCfg config.PostgresConfig, logger *slog.Logger) (*DBIntegrationRuntime, func(), error) {
	databaseResources, cleanupDB, err := ProvideDatabaseResources(pgCfg, logger)
	if err != nil {
		return nil, nil, err
	}
	postgresService := ProvidePostgresService(databaseResources)
	memberRepository := ProvideMemberRepository(postgresService, logger)

	memberCache, err := ProvideMemberCacheWithoutValkey(ctx, memberRepository, logger)
	if err != nil {
		cleanupDB()
		return nil, nil, err
	}

	memberServiceAdapter := ProvideMemberServiceAdapter(memberCache)

	runtime := &DBIntegrationRuntime{
		Logger:        logger,
		Repository:    memberRepository,
		Cache:         memberCache,
		MemberAdapter: memberServiceAdapter,
	}

	return runtime, cleanupDB, nil
}

// InitializeFetchProfilesRuntime - cmd/tools/fetch_profiles 전용
func InitializeFetchProfilesRuntime(_ context.Context) (*FetchProfilesRuntime, func(), error) {
	logger, cleanupLogger, err := ProvideFetchProfilesLogger()
	if err != nil {
		return nil, nil, err
	}

	httpClient := ProvideFetchProfilesHTTPClient()

	runtime := &FetchProfilesRuntime{
		Logger:     logger,
		HTTPClient: httpClient,
	}

	return runtime, cleanupLogger, nil
}
