package app

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// Container 는 타입이다.
type Container struct {
	Config *config.Config
	Logger *zap.Logger

	botDeps *bot.Dependencies
	cleanup func()
}

// Close - 컨테이너 리소스 정리 (DB, 캐시 연결 해제)
func (c *Container) Close() {
	if c != nil && c.cleanup != nil {
		c.cleanup()
	}
}

// NewBot 는 동작을 수행한다.
func (c *Container) NewBot() (*bot.Bot, error) {
	if c == nil || c.botDeps == nil {
		return nil, fmt.Errorf("bot dependencies not initialized")
	}
	b, err := bot.NewBot(c.botDeps)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot instance: %w", err)
	}
	return b, nil
}

// GetYouTubeScheduler 는 동작을 수행한다.
func (c *Container) GetYouTubeScheduler() *youtube.Scheduler {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.Scheduler
}

// GetMemberRepo 는 동작을 수행한다.
func (c *Container) GetMemberRepo() *member.Repository {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.MemberRepo
}

// GetMemberCache 는 동작을 수행한다.
func (c *Container) GetMemberCache() *member.Cache {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.MemberCache
}

// GetAlarmService 는 동작을 수행한다.
func (c *Container) GetAlarmService() *notification.AlarmService {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.Alarm
}

// GetCache 는 동작을 수행한다.
func (c *Container) GetCache() *cache.Service {
	if c.botDeps == nil {
		return nil
	}
	return c.botDeps.Cache
}

// GetHolodexService 는 동작을 수행한다.
func (c *Container) GetHolodexService() *holodex.Service {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.Holodex
}

// GetYouTubeService 는 동작을 수행한다.
func (c *Container) GetYouTubeService() *youtube.Service {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.Service
}

// GetActivityLogger 는 동작을 수행한다.
func (c *Container) GetActivityLogger() *activity.Logger {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.Activity
}

// GetSettingsService 는 동작을 수행한다.
func (c *Container) GetSettingsService() *settings.Service {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.Settings
}

// GetACLService 는 동작을 수행한다.
func (c *Container) GetACLService() *acl.Service {
	if c == nil || c.botDeps == nil {
		return nil
	}
	return c.botDeps.ACL
}

// Build 는 동작을 수행한다.
func Build(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger must not be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Wire 생성 의존성 주입 사용
	deps, cleanup, err := InitializeBotDependencies(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("의존성 초기화 실패: %w", err)
	}

	return &Container{
		Config:  cfg,
		Logger:  logger,
		botDeps: deps,
		cleanup: cleanup,
	}, nil
}
