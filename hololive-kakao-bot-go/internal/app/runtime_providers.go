package app

import (
	"context"
	"fmt"
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
	"github.com/kapu/hololive-kakao-bot-go/internal/service/system"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// ProvideBot: 봇 인스턴스를 생성하여 제공함
func ProvideBot(deps *bot.Dependencies) (*bot.Bot, error) {
	created, err := bot.NewBot(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	return created, nil
}

// ProvideValkeyMQConsumer: 메시지 큐(Valkey) 컨슈머를 생성하여 제공합니다.
func ProvideValkeyMQConsumer(
	ctx context.Context,
	mqCfg mq.ValkeyMQConfig,
	logger *slog.Logger,
	kakaoBot *bot.Bot,
	cacheSvc *cache.Service,
) (*mq.ValkeyMQConsumer, error) {
	consumer := mq.NewValkeyMQConsumer(ctx, mqCfg, logger, kakaoBot, cacheSvc)
	if consumer == nil {
		return nil, fmt.Errorf("failed to create valkey MQ consumer")
	}
	return consumer, nil
}

// ProvideSystemCollector: 시스템 리소스 수집기를 생성하여 제공합니다.
func ProvideSystemCollector(cfg *config.Config) *system.Collector {
	endpoints := []system.ServiceEndpoint{
		{Name: "llm-server", URL: cfg.Services.LLMServerHealthURL},
		{Name: "twentyq", URL: cfg.Services.GameBotTwentyQHealthURL},
		{Name: "turtlesoup", URL: cfg.Services.GameBotTurtleHealthURL},
	}
	return system.NewCollector(endpoints)
}

// ProvideAdminHandler: 관리자 API 핸들러를 생성하여 제공한다. 모든 서비스 의존성을 주입받는다.
func ProvideAdminHandler(
	repo *member.Repository,
	memberCache *member.Cache,
	valkeyCache *cache.Service,
	alarm *notification.AlarmService,
	holodexSvc *holodex.Service,
	youtubeSvc *youtube.Service,
	statsRepo *youtube.StatsRepository,
	activityLogger *activity.Logger,
	settingsSvc *settings.Service,
	aclSvc *acl.Service,
	systemSvc *system.Collector,
	logger *slog.Logger,
) *server.AdminHandler {
	return server.NewAdminHandler(
		repo,
		memberCache,
		valkeyCache,
		alarm,
		holodexSvc,
		youtubeSvc,
		statsRepo,
		activityLogger,
		settingsSvc,
		aclSvc,
		systemSvc,
		logger,
	)
}

// ProvideYouTubeService: YouTube 서비스 인스턴스를 제공합니다.
func ProvideYouTubeService(ytStack *YouTubeStack) *youtube.Service {
	return ytStack.Service
}

// ProvideYouTubeScheduler: YouTube 스케줄러 인스턴스를 제공합니다.
func ProvideYouTubeScheduler(deps *bot.Dependencies) *youtube.Scheduler {
	return deps.Scheduler
}
