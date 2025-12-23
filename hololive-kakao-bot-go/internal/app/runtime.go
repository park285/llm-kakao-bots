package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/mq"
	"github.com/kapu/hololive-kakao-bot-go/internal/server"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// BotRuntime 는 타입이다.
type BotRuntime struct {
	Config *config.Config
	Logger *zap.Logger

	Bot              *bot.Bot
	MQConsumer       *mq.ValkeyMQConsumer
	Scheduler *youtube.Scheduler

	AdminHandler      *server.AdminHandler
	Sessions          *server.ValkeySessionStore
	SecurityConfig    *server.SecurityConfig
	AdminAllowedCIDRs []*net.IPNet
	AdminRouter       *gin.Engine
	AdminAddr         string
	AdminServer       *http.Server

	cleanup func()
}

// Close - 런타임 리소스 정리 (DB, 캐시 연결 해제)
func (r *BotRuntime) Close() {
	if r != nil && r.cleanup != nil {
		r.cleanup()
	}
}

// BuildRuntime 는 동작을 수행한다.
func BuildRuntime(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*BotRuntime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger must not be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	runtime, cleanup, err := InitializeBotRuntime(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("런타임 초기화 실패: %w", err)
	}
	runtime.cleanup = cleanup

	return runtime, nil
}

// StartAdminServer 는 동작을 수행한다.
func (r *BotRuntime) StartAdminServer(errCh chan<- error) {
	if r == nil || r.AdminServer == nil {
		return
	}

	go func() {
		if err := r.AdminServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errCh != nil {
				errCh <- fmt.Errorf("HTTP server error: %w", err)
				return
			}
			if r.Logger != nil {
				r.Logger.Error("HTTP server error", zap.Error(err))
			}
		}
	}()
}

// ShutdownAdminServer 는 동작을 수행한다.
func (r *BotRuntime) ShutdownAdminServer(ctx context.Context) error {
	if r == nil || r.AdminServer == nil {
		return nil
	}
	if err := r.AdminServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown failed: %w", err)
	}
	return nil
}

// Start 는 동작을 수행한다.
func (r *BotRuntime) Start(ctx context.Context, errCh chan<- error) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Start YouTube scheduler if enabled (ingestion feature)
	if r.Scheduler != nil {
		r.Scheduler.Start(ctx)
		if r.Logger != nil {
			r.Logger.Info("YouTube ingestion scheduler started")
		}
	}

	// Start alarm checker in background
	if r.Bot != nil {
		go func() {
			if err := r.Bot.Start(ctx); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					if r.Logger != nil {
						r.Logger.Info("Bot alarm checker stopped (context done)")
					}
				} else if r.Logger != nil {
					r.Logger.Error("Bot alarm checker error", zap.Error(err))
				}
			}
		}()
	}

	if r.MQConsumer != nil {
		r.MQConsumer.Start(ctx)
	}

	r.StartAdminServer(errCh)
	if r.Logger != nil && r.AdminAddr != "" {
		r.Logger.Info("Admin HTTP server started", zap.String("addr", r.AdminAddr))
	}
}

// Shutdown 는 동작을 수행한다.
func (r *BotRuntime) Shutdown(ctx context.Context) {
	if r == nil {
		return
	}

	// Stop YouTube scheduler
	if r.Scheduler != nil {
		r.Scheduler.Stop()
		if r.Logger != nil {
			r.Logger.Info("YouTube ingestion scheduler stopped")
		}
	}

	if err := r.ShutdownAdminServer(ctx); err != nil {
		if r.Logger != nil {
			r.Logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}
	if r.Bot != nil {
		if err := r.Bot.Shutdown(ctx); err != nil {
			if r.Logger != nil {
				r.Logger.Error("Error during shutdown", zap.Error(err))
			}
		}
	}
}

// Run 는 동작을 수행한다.
func (r *BotRuntime) Run() {
	if r == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	errCh := make(chan error, 1)
	r.Start(ctx, errCh)
	if r.Logger != nil {
		r.Logger.Info("Bot started, waiting for signals...")
	}

	select {
	case sig := <-sigCh:
		if r.Logger != nil {
			r.Logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		}
	case err := <-errCh:
		if r.Logger != nil {
			r.Logger.Error("Server error", zap.Error(err))
		}
	}

	if r.Logger != nil {
		r.Logger.Info("Shutting down gracefully...")
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), constants.AppTimeout.Shutdown)
	defer shutdownCancel()

	r.Shutdown(shutdownCtx)

	if r.Logger != nil {
		r.Logger.Info("Shutdown complete")
	}
}
