package app

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/iris"
	"github.com/kapu/hololive-kakao-bot-go/internal/mq"
	"github.com/kapu/hololive-kakao-bot-go/internal/platform/bootstrap"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/matcher"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// ----------------------------------------------------------------------------
// 인프라스트럭처 Provider
// ----------------------------------------------------------------------------

// ProvideValkeyConfig - 설정에서 Valkey 캐시 설정 추출
func ProvideValkeyConfig(cfg *config.Config) config.ValkeyConfig {
	return cfg.Valkey
}

// ProvidePostgresConfig - 설정에서 PostgreSQL 설정 추출
func ProvidePostgresConfig(cfg *config.Config) config.PostgresConfig {
	return cfg.Postgres
}

// ProvideCacheResources - 캐시 리소스 생성 (정리 함수 포함)
func ProvideCacheResources(cfg config.ValkeyConfig, logger *slog.Logger) (*bootstrap.CacheResources, func(), error) {
	resources, err := bootstrap.NewCacheResources(cfg, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("캐시 리소스 생성 실패: %w", err)
	}
	return resources, resources.Close, nil
}

// ProvideCacheService - 캐시 리소스에서 서비스 추출
func ProvideCacheService(resources *bootstrap.CacheResources) *cache.Service {
	return resources.Service
}

// ProvideDatabaseResources - 데이터베이스 리소스 생성 (정리 함수 포함)
func ProvideDatabaseResources(cfg config.PostgresConfig, logger *slog.Logger) (*bootstrap.DatabaseResources, func(), error) {
	resources, err := bootstrap.NewDatabaseResources(cfg, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("데이터베이스 리소스 생성 실패: %w", err)
	}
	return resources, resources.Close, nil
}

// ProvidePostgresService - 데이터베이스 리소스에서 서비스 추출
func ProvidePostgresService(resources *bootstrap.DatabaseResources) *database.PostgresService {
	return resources.Service
}

// ----------------------------------------------------------------------------
// 메시지 큐 Provider
// ----------------------------------------------------------------------------

// ProvideValkeyMQConfig - 설정에서 MQ 설정 생성
func ProvideValkeyMQConfig(cfg *config.Config) mq.ValkeyMQConfig {
	return mq.ValkeyMQConfig{
		Host:          cfg.ValkeyMQ.Host,
		Port:          cfg.ValkeyMQ.Port,
		Password:      cfg.ValkeyMQ.Password,
		StreamKey:     cfg.ValkeyMQ.StreamKey,
		ConsumerGroup: cfg.ValkeyMQ.ConsumerGroup,
		ConsumerName:  cfg.ValkeyMQ.ConsumerName,
	}
}

// ProvideIrisClient - Iris MQ 클라이언트 생성
func ProvideIrisClient(mqCfg mq.ValkeyMQConfig, logger *slog.Logger) iris.Client {
	return mq.NewValkeyMQClient(mqCfg, logger)
}

// ----------------------------------------------------------------------------
// 멤버 서비스 Provider
// ----------------------------------------------------------------------------

// ProvideMemberRepository - 멤버 저장소 생성
func ProvideMemberRepository(postgres *database.PostgresService, logger *slog.Logger) *member.Repository {
	return member.NewMemberRepository(postgres, logger)
}

func buildMemberCache(
	ctx context.Context,
	repo *member.Repository,
	cacheSvc *cache.Service,
	logger *slog.Logger,
) (*member.Cache, error) {
	memberCache, err := member.NewMemberCache(ctx, repo, cacheSvc, logger, member.CacheConfig{
		WarmUp:    true,
		ValkeyTTL: constants.MemberCacheDefaults.ValkeyTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("멤버 캐시 생성 실패: %w", err)
	}
	return memberCache, nil
}

// ProvideMemberCache - 멤버 캐시 생성 (초기 워밍업 포함)
func ProvideMemberCache(
	ctx context.Context,
	repo *member.Repository,
	cacheSvc *cache.Service,
	logger *slog.Logger,
) (*member.Cache, error) {
	memberCache, err := buildMemberCache(ctx, repo, cacheSvc, logger)
	if err != nil {
		return nil, err
	}

	if cacheSvc == nil {
		logger.Warn("Cache service is nil; member database init skipped")
		return memberCache, nil
	}

	// Valkey member database 초기화 (이름 -> 채널ID 맵)
	// NOTE: 기존에는 wire_gen.go에 로직이 섞여 있었으나, 생성 코드 순수성을 위해 provider로 이동
	members, err := repo.GetAllMembers(ctx)
	if err != nil {
		logger.Warn("Failed to load members for member database init", slog.Any("error", err))
		members = []*domain.Member{}
	}

	memberMap := make(map[string]string, len(members))
	for _, m := range members {
		if m != nil && m.ChannelID != "" {
			memberMap[m.Name] = m.ChannelID
		}
	}

	if err := cacheSvc.InitializeMemberDatabase(ctx, memberMap); err != nil {
		return nil, fmt.Errorf("멤버 데이터베이스 초기화 실패: %w", err)
	}

	return memberCache, nil
}

// ProvideMemberCacheWithoutValkey - Valkey 없이 멤버 캐시만 구성
func ProvideMemberCacheWithoutValkey(
	ctx context.Context,
	repo *member.Repository,
	logger *slog.Logger,
) (*member.Cache, error) {
	return buildMemberCache(ctx, repo, nil, logger)
}

// ProvideMemberServiceAdapter - 멤버 데이터 제공자 어댑터 생성
func ProvideMemberServiceAdapter(memberCache *member.Cache) *member.ServiceAdapter {
	return member.NewMemberServiceAdapter(memberCache)
}

// ProvideMembersData - 도메인 인터페이스로 바인딩
func ProvideMembersData(adapter *member.ServiceAdapter) domain.MemberDataProvider {
	return adapter
}

// ----------------------------------------------------------------------------
// Holodex Provider
// ----------------------------------------------------------------------------

// ProvideHolodexAPIKeys - 설정에서 API 키 추출
func ProvideHolodexAPIKeys(cfg *config.Config) []string {
	return cfg.Holodex.APIKeys
}

// ProvideScraperService - 스크래퍼 서비스 생성
func ProvideScraperService(
	cacheSvc *cache.Service,
	members *member.ServiceAdapter,
	logger *slog.Logger,
) *holodex.ScraperService {
	return holodex.NewScraperService(cacheSvc, members, logger)
}

// ProvideHolodexService - Holodex API 서비스 생성
func ProvideHolodexService(
	apiKeys []string,
	cacheSvc *cache.Service,
	scraper *holodex.ScraperService,
	logger *slog.Logger,
) (*holodex.Service, error) {
	svc, err := holodex.NewHolodexService(apiKeys, cacheSvc, scraper, logger)
	if err != nil {
		return nil, fmt.Errorf("holodex 서비스 생성 실패: %w", err)
	}
	return svc, nil
}

// ----------------------------------------------------------------------------
// 프로필 및 매칭 Provider
// ----------------------------------------------------------------------------

// ProvideProfileService - 프로필 서비스 생성 (번역 사전 로드 포함)
func ProvideProfileService(
	ctx context.Context,
	cacheSvc *cache.Service,
	members *member.ServiceAdapter,
	logger *slog.Logger,
) (*member.ProfileService, error) {
	svc, err := member.NewProfileService(cacheSvc, members, logger)
	if err != nil {
		return nil, fmt.Errorf("프로필 서비스 생성 실패: %w", err)
	}
	svc.PreloadTranslations(ctx)
	return svc, nil
}

// ProvideMemberMatcher - 멤버 매칭 서비스 생성
func ProvideMemberMatcher(
	ctx context.Context,
	membersData domain.MemberDataProvider,
	cacheSvc *cache.Service,
	holodex *holodex.Service,
	logger *slog.Logger,
) *matcher.MemberMatcher {
	// selector는 nil (Gemini AI 채널 선택 미사용)
	return matcher.NewMemberMatcher(ctx, membersData, cacheSvc, holodex, nil, logger)
}

// ----------------------------------------------------------------------------
// 알림 Provider
// ----------------------------------------------------------------------------

// ProvideAlarmService - 알림 서비스 생성
func ProvideAlarmService(
	cfg *config.Config,
	cacheSvc *cache.Service,
	holodex *holodex.Service,
	logger *slog.Logger,
) *notification.AlarmService {
	return notification.NewAlarmService(cacheSvc, holodex, logger, cfg.Notification.AdvanceMinutes)
}

// ----------------------------------------------------------------------------
// YouTube Provider
// ----------------------------------------------------------------------------

// ProvideYouTubeStatsRepository - YouTube 통계 저장소 생성
func ProvideYouTubeStatsRepository(
	postgres *database.PostgresService,
	logger *slog.Logger,
) *youtube.StatsRepository {
	return youtube.NewYouTubeStatsRepository(postgres, logger)
}

// YouTubeStack - YouTube 관련 서비스 묶음 (선택적 활성화)
type YouTubeStack struct {
	Service   *youtube.Service
	Scheduler *youtube.Scheduler
	StatsRepo *youtube.StatsRepository
}

// ProvideYouTubeStack - YouTube 서비스 스택 생성
func ProvideYouTubeStack(
	ctx context.Context,
	cfg *config.Config,
	cacheSvc *cache.Service,
	members *member.ServiceAdapter,
	statsRepo *youtube.StatsRepository,
	logger *slog.Logger,
) *YouTubeStack {
	if !cfg.YouTube.EnableQuotaBuilding || cfg.YouTube.APIKey == "" {
		logger.Info("YouTube 쿼터 빌딩 비활성화; 통계 저장소만 사용 가능")
		return &YouTubeStack{StatsRepo: statsRepo}
	}

	svc, err := youtube.NewYouTubeService(ctx, cfg.YouTube.APIKey, cacheSvc, logger)
	if err != nil {
		logger.Warn("YouTube 서비스 초기화 실패 (선택적 기능)", slog.Any("error", err))
		return &YouTubeStack{StatsRepo: statsRepo}
	}

	scheduler := youtube.NewScheduler(svc, cacheSvc, statsRepo, members, logger)
	logger.Info("YouTube 쿼터 빌딩 활성화",
		slog.String("mode", "API Key"),
		slog.Int("daily_target", 9192))

	return &YouTubeStack{
		Service:   svc,
		Scheduler: scheduler,
		StatsRepo: statsRepo,
	}
}

// ----------------------------------------------------------------------------
// 애플리케이션 서비스 Provider
// ----------------------------------------------------------------------------

// ProvideActivityLogger - 활동 로거 생성
func ProvideActivityLogger(cfg *config.Config, logger *slog.Logger) *activity.Logger {
	// Logging.Dir에서 활동 로그 경로 생성
	logDir := cfg.Logging.Dir
	if logDir == "" {
		logDir = "logs"
	}
	activityLogPath := logDir + "/activity.log"
	return activity.NewActivityLogger(activityLogPath, logger)
}

// ProvideSettingsService - 설정 서비스 생성
func ProvideSettingsService(logger *slog.Logger) *settings.Service {
	return settings.NewSettingsService("settings.json", logger)
}

// ProvideACLService - 접근 제어 서비스 생성 (PostgreSQL 영구화)
func ProvideACLService(
	cfg *config.Config,
	postgres *database.PostgresService,
	cacheSvc *cache.Service,
	logger *slog.Logger,
) (*acl.Service, error) {
	svc, err := acl.NewACLService(
		postgres,
		cacheSvc,
		logger,
		cfg.Kakao.ACLEnabled,
		cfg.Kakao.Rooms,
	)
	if err != nil {
		return nil, fmt.Errorf("ACL 서비스 생성 실패: %w", err)
	}
	return svc, nil
}

// ----------------------------------------------------------------------------
// 메시지 스택 Provider
// ----------------------------------------------------------------------------

// MessageStack - 메시지 어댑터와 포매터 묶음
type MessageStack struct {
	Adapter   *adapter.MessageAdapter
	Formatter *adapter.ResponseFormatter
}

// ProvideMessageStack - 메시지 어댑터 및 포매터 생성
func ProvideMessageStack(cfg *config.Config) *MessageStack {
	msgAdapter, formatter := bootstrap.NewMessageStack(cfg.Bot.Prefix)
	return &MessageStack{
		Adapter:   msgAdapter,
		Formatter: formatter,
	}
}

// ----------------------------------------------------------------------------
// Bot 의존성 조립
// ----------------------------------------------------------------------------

// ProvideBotDependencies - 모든 의존성을 bot.Dependencies로 조립
func ProvideBotDependencies(
	cfg *config.Config,
	logger *slog.Logger,
	irisClient iris.Client,
	msgStack *MessageStack,
	cacheSvc *cache.Service,
	postgres *database.PostgresService,
	memberRepo *member.Repository,
	memberCache *member.Cache,
	holodex *holodex.Service,
	profiles *member.ProfileService,
	alarm *notification.AlarmService,
	memberMatcher *matcher.MemberMatcher,
	membersData domain.MemberDataProvider,
	ytStack *YouTubeStack,
	activityLogger *activity.Logger,
	settingsSvc *settings.Service,
	aclSvc *acl.Service,
) *bot.Dependencies {
	return &bot.Dependencies{
		Config:           cfg,
		Logger:           logger,
		Client:           irisClient,
		MessageAdapter:   msgStack.Adapter,
		Formatter:        msgStack.Formatter,
		Cache:            cacheSvc,
		Postgres:         postgres,
		MemberRepo:       memberRepo,
		MemberCache:      memberCache,
		Holodex:          holodex,
		Profiles:         profiles,
		Alarm:            alarm,
		Matcher:          memberMatcher,
		MembersData:      membersData,
		Service:          ytStack.Service,
		Scheduler:        ytStack.Scheduler,
		YouTubeStatsRepo: ytStack.StatsRepo,
		Activity:         activityLogger,
		Settings:         settingsSvc,
		ACL:              aclSvc,
	}
}
