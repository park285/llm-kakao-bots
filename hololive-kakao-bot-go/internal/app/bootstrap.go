package app

import (
	"context"
	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
)

// coreInfrastructure 는 공통 인프라 의존성을 담는다.
type coreInfrastructure struct {
	deps         *bot.Dependencies
	ytStack      *YouTubeStack
	photoSync    *holodex.PhotoSyncService // 프로필 이미지 동기화 서비스
	cleanupCache func()
	cleanupDB    func()
}

// infraResources 는 캐시/DB 리소스를 담는다.
type infraResources struct {
	cacheService    *cache.Service
	postgresService *database.PostgresService
	memberRepo      *member.Repository
	memberCache     *member.Cache
	cleanupCache    func()
	cleanupDB       func()
}

// initInfraResources 는 캐시/DB 리소스를 초기화한다.
func initInfraResources(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*infraResources, error) {
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

	return &infraResources{
		cacheService:    cacheService,
		postgresService: postgresService,
		memberRepo:      memberRepository,
		memberCache:     memberCache,
		cleanupCache:    cleanupCache,
		cleanupDB:       cleanupDB,
	}, nil
}

// initCoreInfrastructure 는 공통 인프라를 초기화한다.
func initCoreInfrastructure(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*coreInfrastructure, error) {
	valkeyMQConfig := ProvideValkeyMQConfig(cfg)
	irisClient := ProvideIrisClient(ctx, valkeyMQConfig, logger)
	messageStack := ProvideMessageStack(cfg)

	infra, err := initInfraResources(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	cacheService := infra.cacheService
	postgresService := infra.postgresService

	holodexAPIKeys := ProvideHolodexAPIKeys(cfg)
	memberServiceAdapter := ProvideMemberServiceAdapter(infra.memberCache)
	scraperService := ProvideScraperService(cacheService, memberServiceAdapter, logger)
	holodexService, err := ProvideHolodexService(holodexAPIKeys, cacheService, scraperService, logger)
	if err != nil {
		infra.cleanupDB()
		infra.cleanupCache()
		return nil, err
	}

	profileService, err := ProvideProfileService(ctx, cacheService, memberServiceAdapter, logger)
	if err != nil {
		infra.cleanupDB()
		infra.cleanupCache()
		return nil, err
	}

	alarmRepository := ProvideAlarmRepository(postgresService, logger)
	alarmService := ProvideAlarmService(cfg, cacheService, holodexService, alarmRepository, logger)

	// 앱 시작 시 알람 캐시 워밍 (DB에서 Valkey로 일괄 로드)
	if warnErr := alarmService.WarmCacheFromDB(ctx); warnErr != nil {
		logger.Warn("Failed to warm alarm cache from DB", "error", warnErr)
	}

	memberDataProvider := ProvideMembersData(memberServiceAdapter)
	memberMatcher := ProvideMemberMatcher(ctx, memberDataProvider, cacheService, holodexService, logger)
	youTubeStatsRepository := ProvideYouTubeStatsRepository(postgresService, logger)
	youTubeStack := ProvideYouTubeStack(ctx, cfg, cacheService, holodexService, memberServiceAdapter, youTubeStatsRepository, alarmService, irisClient, logger)
	activityLogger := ProvideActivityLogger(cfg, logger)
	settingsService := ProvideSettingsService(logger)

	aclService, err := ProvideACLService(ctx, cfg, postgresService, cacheService, logger)
	if err != nil {
		infra.cleanupDB()
		infra.cleanupCache()
		return nil, err
	}

	deps := ProvideBotDependencies(cfg, logger, irisClient, messageStack, cacheService, postgresService, infra.memberRepo, infra.memberCache, holodexService, profileService, alarmService, memberMatcher, memberDataProvider, youTubeStack, activityLogger, settingsService, aclService)

	// 프로필 이미지 동기화 서비스 생성 (7일 주기)
	photoSyncService := holodex.NewPhotoSyncService(holodexService, infra.memberRepo, logger)

	return &coreInfrastructure{
		deps:         deps,
		ytStack:      youTubeStack,
		photoSync:    photoSyncService,
		cleanupCache: infra.cleanupCache,
		cleanupDB:    infra.cleanupDB,
	}, nil
}

// InitializeBotDependencies - 봇 의존성을 초기화합니다.
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
	systemCollector := ProvideSystemCollector(cfg)

	apiHandler := ProvideAPIHandler(deps.MemberRepo, deps.MemberCache, deps.Cache, deps.Profiles, deps.Alarm, deps.Holodex, youTubeService, infra.ytStack.StatsRepo, deps.Activity, deps.Settings, deps.ACL, systemCollector, logger)

	authService, err := ProvideAuthService(ctx, deps.Postgres, deps.Cache, logger)
	if err != nil {
		return nil, err
	}
	authHandler := ProvideAuthHandler(authService, logger)

	adminRouter, err := ProvideAPIRouter(ctx, cfg, logger, apiHandler, authHandler)
	if err != nil {
		return nil, err
	}

	adminAddr := ProvideAPIAddr(cfg)
	adminServer := ProvideAPIServer(adminAddr, adminRouter)

	return &BotRuntime{
		Config:      cfg,
		Logger:      logger,
		Bot:         botBot,
		MQConsumer:  valkeyMQConsumer,
		Scheduler:   youTubeScheduler,
		PhotoSync:   infra.photoSync, // 프로필 이미지 동기화 서비스
		APIHandler:  apiHandler,
		AdminRouter: adminRouter,
		AdminAddr:   adminAddr,
		AdminServer: adminServer,
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
