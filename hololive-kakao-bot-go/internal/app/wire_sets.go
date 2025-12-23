//go:build wireinject

package app

import "github.com/google/wire"

// ----------------------------------------------------------------------------
// Wire Provider 세트
// ----------------------------------------------------------------------------

// InfrastructureSet - 핵심 인프라 서비스
var InfrastructureSet = wire.NewSet(
	ProvideValkeyConfig,
	ProvidePostgresConfig,
	ProvideCacheResources,
	ProvideCacheService,
	ProvideDatabaseResources,
	ProvidePostgresService,
)

// MQSet - 메시지 큐 서비스
var MQSet = wire.NewSet(
	ProvideValkeyMQConfig,
	ProvideIrisClient,
)

// MemberSet - 멤버 관련 서비스
var MemberSet = wire.NewSet(
	ProvideMemberRepository,
	ProvideMemberCache,
	ProvideMemberServiceAdapter,
	ProvideMembersData,
)

// HolodexSet - Holodex 관련 서비스
var HolodexSet = wire.NewSet(
	ProvideHolodexAPIKeys,
	ProvideScraperService,
	ProvideHolodexService,
)

// MatcherSet - 매칭 및 프로필 서비스
var MatcherSet = wire.NewSet(
	ProvideProfileService,
	ProvideMemberMatcher,
)

// NotificationSet - 알림 서비스
var NotificationSet = wire.NewSet(
	ProvideAlarmService,
)

// YouTubeSet - YouTube 서비스
var YouTubeSet = wire.NewSet(
	ProvideYouTubeStatsRepository,
	ProvideYouTubeStack,
)

// ApplicationSet - 일반 애플리케이션 서비스
var ApplicationSet = wire.NewSet(
	ProvideActivityLogger,
	ProvideSettingsService,
	ProvideACLService,
	ProvideMessageStack,
)

// BotSet - 최종 봇 의존성 조립
var BotSet = wire.NewSet(
	ProvideBotDependencies,
)

// AppSet - 전체 Provider 세트
var AppSet = wire.NewSet(
	InfrastructureSet,
	MQSet,
	MemberSet,
	HolodexSet,
	MatcherSet,
	NotificationSet,
	YouTubeSet,
	ApplicationSet,
	BotSet,
)

// RuntimeSet - cmd/bot 런타임 조립 (Bot + MQ + Admin API 구성요소)
var RuntimeSet = wire.NewSet(
	AppSet,
	ProvideBot,
	ProvideValkeyMQConsumer,
	ProvideSessionStore,
	ProvideLoginRateLimiter,
	ProvideSecurityConfig,
	ProvideAdminCredentials,
	ProvideAdminHandler,
	ProvideWatchdogProxyHandler,
	ProvideAdminAllowedCIDRs,
	ProvideAdminServer,
	ProvideAdminRouter,
	ProvideAdminAddr,
	ProvideYouTubeService,
	ProvideYouTubeScheduler,
	wire.Struct(
		new(BotRuntime),
		"Config",
		"Logger",
		"Bot",
		"MQConsumer",
		"Scheduler",
		"AdminHandler",
		"Sessions",
		"SecurityConfig",
		"AdminAllowedCIDRs",
		"AdminRouter",
		"AdminAddr",
		"AdminServer",
	),
)

// WarmMemberCacheSet - cmd/tools/warm_member_cache 전용 (불필요한 외부 의존성 제외)
var WarmMemberCacheSet = wire.NewSet(
	InfrastructureSet,
	ProvideMemberRepository,
	ProvideMemberCache,
)

// DBIntegrationSet - cmd/test_db_integration 전용 (Postgres + Cache)
var DBIntegrationSet = wire.NewSet(
	ProvideDatabaseResources,
	ProvidePostgresService,
	ProvideMemberRepository,
	ProvideMemberCacheWithoutValkey,
	ProvideMemberServiceAdapter,
	wire.Struct(
		new(DBIntegrationRuntime),
		"Logger",
		"Repository",
		"Cache",
		"MemberAdapter",
	),
)

// FetchProfilesSet - cmd/tools/fetch_profiles 전용
var FetchProfilesSet = wire.NewSet(
	ProvideFetchProfilesLogger,
	ProvideFetchProfilesHTTPClient,
	wire.Struct(
		new(FetchProfilesRuntime),
		"Logger",
		"HTTPClient",
	),
)
