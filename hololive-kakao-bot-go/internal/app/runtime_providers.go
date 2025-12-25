package app

import (
	"fmt"
	"net"

	"log/slog"

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

// AdminCredentials: 관리자 계정 정보를 담는 구조체 (사용자명, 비밀번호 해시)
type AdminCredentials struct {
	User     string
	PassHash string
}

// ProvideBot: 봇 인스턴스를 생성하여 제공한다. (Wire 의존성 주입용)
func ProvideBot(deps *bot.Dependencies) (*bot.Bot, error) {
	created, err := bot.NewBot(deps)
	if err != nil {
		return nil, fmt.Errorf("봇 생성 실패: %w", err)
	}
	return created, nil
}

// ProvideValkeyMQConsumer: 메시지 큐(Valkey) 컨슈머를 생성하여 제공한다.
func ProvideValkeyMQConsumer(
	mqCfg mq.ValkeyMQConfig,
	logger *slog.Logger,
	kakaoBot *bot.Bot,
	cacheSvc *cache.Service,
) (*mq.ValkeyMQConsumer, error) {
	consumer := mq.NewValkeyMQConsumer(mqCfg, logger, kakaoBot, cacheSvc)
	if consumer == nil {
		return nil, fmt.Errorf("valkey MQ consumer 생성 실패")
	}
	return consumer, nil
}

// ProvideSessionStore: 세션 저장소(Valkey 백엔드)를 생성하여 제공한다.
func ProvideSessionStore(cacheSvc *cache.Service, logger *slog.Logger) *server.ValkeySessionStore {
	return server.NewValkeySessionStore(cacheSvc.GetClient(), logger)
}

// ProvideLoginRateLimiter: 로그인 시도 제한(Rate Limiter)을 생성하여 제공한다.
func ProvideLoginRateLimiter() *server.LoginRateLimiter {
	return server.NewLoginRateLimiter()
}

// ProvideSecurityConfig: 보안 관련 설정(세션 비밀키 등)을 로드하여 제공한다.
func ProvideSecurityConfig(cfg *config.Config) *server.SecurityConfig {
	return &server.SecurityConfig{
		SessionSecret: cfg.Server.SessionSecret,
		ForceHTTPS:    cfg.Server.ForceHTTPS,
	}
}

// ProvideAdminCredentials: 관리자 자격 증명을 설정에서 로드하여 제공한다.
func ProvideAdminCredentials(cfg *config.Config) AdminCredentials {
	return AdminCredentials{
		User:     cfg.Server.AdminUser,
		PassHash: cfg.Server.AdminPassHash,
	}
}

// ProvideAdminHandler: 관리자 API 핸들러를 생성하여 제공한다. 모든 서비스 의존성을 주입받는다.
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
	logger *slog.Logger,
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

// ProvideAdminAllowedCIDRs: 관리자 페이지 접근 허용 IP 대역을 설정에서 로드하고 파싱하여 제공한다.
func ProvideAdminAllowedCIDRs(cfg *config.Config) ([]*net.IPNet, error) {
	allowed, err := server.NewIPAllowList(cfg.Server.AdminAllowedIPs)
	if err != nil {
		return nil, fmt.Errorf("admin allowlist 생성 실패: %w", err)
	}
	return allowed, nil
}

// ProvideWatchdogProxyHandler - Watchdog API 프록시 핸들러 생성
func ProvideWatchdogProxyHandler(cfg *config.Config, logger *slog.Logger, activity *activity.Logger) *server.WatchdogProxyHandler {
	watchdogURL := cfg.WatchdogURL
	if watchdogURL == "" {
		watchdogURL = "http://llm-watchdog:30002" // Docker 네트워크 기본값
	}
	return server.NewWatchdogProxyHandler(watchdogURL, logger, activity)
}

// ProvideYouTubeService: YouTube 서비스 인스턴스를 제공한다.
func ProvideYouTubeService(ytStack *YouTubeStack) *youtube.Service {
	return ytStack.Service
}

// ProvideYouTubeScheduler: YouTube 스케줄러 인스턴스를 제공한다.
func ProvideYouTubeScheduler(deps *bot.Dependencies) *youtube.Scheduler {
	return deps.Scheduler
}
