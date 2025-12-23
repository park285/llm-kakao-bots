package app

import (
	"fmt"
	"net"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/mq"
	"github.com/kapu/hololive-kakao-bot-go/internal/server"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// ----------------------------------------------------------------------------
// BotRuntime Provider
// ----------------------------------------------------------------------------

// AdminCredentials 는 타입이다.
type AdminCredentials struct {
	User     string
	PassHash string
}

// ProvideBot 는 동작을 수행한다.
func ProvideBot(deps *bot.Dependencies) (*bot.Bot, error) {
	created, err := bot.NewBot(deps)
	if err != nil {
		return nil, fmt.Errorf("봇 생성 실패: %w", err)
	}
	return created, nil
}

// ProvideValkeyMQConsumer 는 동작을 수행한다.
func ProvideValkeyMQConsumer(
	mqCfg mq.ValkeyMQConfig,
	logger *zap.Logger,
	kakaoBot *bot.Bot,
	cacheSvc *cache.Service,
) (*mq.ValkeyMQConsumer, error) {
	consumer := mq.NewValkeyMQConsumer(mqCfg, logger, kakaoBot, cacheSvc)
	if consumer == nil {
		return nil, fmt.Errorf("valkey MQ consumer 생성 실패")
	}
	return consumer, nil
}

// ProvideSessionStore 는 동작을 수행한다.
func ProvideSessionStore(cacheSvc *cache.Service, logger *zap.Logger) *server.ValkeySessionStore {
	return server.NewValkeySessionStore(cacheSvc.GetClient(), logger)
}

// ProvideLoginRateLimiter 는 동작을 수행한다.
func ProvideLoginRateLimiter() *server.LoginRateLimiter {
	return server.NewLoginRateLimiter()
}

// ProvideSecurityConfig 는 동작을 수행한다.
func ProvideSecurityConfig(cfg *config.Config) *server.SecurityConfig {
	return &server.SecurityConfig{
		SessionSecret: cfg.Server.SessionSecret,
		ForceHTTPS:    cfg.Server.ForceHTTPS,
	}
}

// ProvideAdminCredentials 는 동작을 수행한다.
func ProvideAdminCredentials(cfg *config.Config) AdminCredentials {
	return AdminCredentials{
		User:     cfg.Server.AdminUser,
		PassHash: cfg.Server.AdminPassHash,
	}
}

// ProvideAdminHandler 는 동작을 수행한다.
func ProvideAdminHandler(
	repo *member.Repository,
	memberCache *member.Cache,
	valkeyCache *cache.Service,
	alarm *notification.AlarmService,
	holodexSvc *holodex.Service,
	youtubeSvc *youtube.Service,
	activityLogger *activity.Logger,
	settingsSvc *settings.Service,
	aclSvc *acl.Service,
	cfg *config.Config,
	sessions *server.ValkeySessionStore,
	rateLimiter *server.LoginRateLimiter,
	securityCfg *server.SecurityConfig,
	adminCreds AdminCredentials,
	logger *zap.Logger,
) *server.AdminHandler {
	return server.NewAdminHandler(
		repo,
		memberCache,
		valkeyCache,
		alarm,
		holodexSvc,
		youtubeSvc,
		activityLogger,
		settingsSvc,
		aclSvc,
		cfg,
		sessions,
		rateLimiter,
		securityCfg,
		adminCreds.User,
		adminCreds.PassHash,
		logger,
	)
}

// ProvideAdminAllowedCIDRs 는 동작을 수행한다.
func ProvideAdminAllowedCIDRs(cfg *config.Config) ([]*net.IPNet, error) {
	allowed, err := server.NewIPAllowList(cfg.Server.AdminAllowedIPs)
	if err != nil {
		return nil, fmt.Errorf("admin allowlist 생성 실패: %w", err)
	}
	return allowed, nil
}

// ProvideWatchdogProxyHandler - Watchdog API 프록시 핸들러 생성
func ProvideWatchdogProxyHandler(cfg *config.Config, logger *zap.Logger, activity *activity.Logger) *server.WatchdogProxyHandler {
	watchdogURL := cfg.WatchdogURL
	if watchdogURL == "" {
		watchdogURL = "http://llm-watchdog:30002" // Docker 네트워크 기본값
	}
	return server.NewWatchdogProxyHandler(watchdogURL, logger, activity)
}

// ProvideYouTubeService 는 동작을 수행한다.
func ProvideYouTubeService(ytStack *YouTubeStack) *youtube.Service {
	return ytStack.Service
}

// ProvideYouTubeScheduler 는 동작을 수행한다.
func ProvideYouTubeScheduler(deps *bot.Dependencies) *youtube.Scheduler {
	return deps.Scheduler
}
