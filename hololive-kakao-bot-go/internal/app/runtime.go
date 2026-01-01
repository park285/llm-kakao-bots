package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/bot"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/mq"
	"github.com/kapu/hololive-kakao-bot-go/internal/server"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// BotRuntime: 봇 애플리케이션의 전체 실행 환경 및 상태를 관리하는 구조체
type BotRuntime struct {
	Config *config.Config
	Logger *slog.Logger

	Bot        *bot.Bot
	MQConsumer *mq.ValkeyMQConsumer
	Scheduler  *youtube.Scheduler

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

// BuildRuntime: 설정과 로거를 기반으로 봇 런타임 환경을 구성하고 모든 의존성을 초기화합니다.
func BuildRuntime(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*BotRuntime, error) {
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
		return nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}
	runtime.cleanup = cleanup

	return runtime, nil
}

// StartAdminServer: 관리자용 웹 서버를 비동기적으로 시작합니다.
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
				r.Logger.Error("HTTP server error", slog.Any("error", err))
			}
		}
	}()
}

// ShutdownAdminServer: 관리자용 웹 서버를 안전하게 종료합니다.
func (r *BotRuntime) ShutdownAdminServer(ctx context.Context) error {
	if r == nil || r.AdminServer == nil {
		return nil
	}
	if err := r.AdminServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown failed: %w", err)
	}
	return nil
}

// Start: 봇의 모든 구성 요소(스케줄러, 알람 체커, MQ 컨슈머, 관리자 서버)를 시작합니다.
func (r *BotRuntime) Start(ctx context.Context, errCh chan<- error) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// 활성화된 경우 YouTube 스케줄러 시작 (ingestion 기능)
	if r.Scheduler != nil {
		r.Scheduler.Start(ctx)
		if r.Logger != nil {
			r.Logger.Info("YouTube ingestion scheduler started")
		}
	}

	// 백그라운드에서 알림 체커 시작
	if r.Bot != nil {
		go func() {
			if err := r.Bot.Start(ctx); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					if r.Logger != nil {
						r.Logger.Info("Bot alarm checker stopped (context done)")
					}
				} else if r.Logger != nil {
					r.Logger.Error("Bot alarm checker error", slog.Any("error", err))
				}
			}
		}()
	}

	if r.MQConsumer != nil {
		r.MQConsumer.Start(ctx)
	}

	r.StartAdminServer(errCh)
	if r.Logger != nil && r.AdminAddr != "" {
		r.Logger.Info("Admin HTTP server started", slog.String("addr", r.AdminAddr))
	}
}

// Shutdown: 봇의 모든 구성 요소를 안전하게 종료하고 리소스를 정리합니다.
func (r *BotRuntime) Shutdown(ctx context.Context) {
	if r == nil {
		return
	}

	// YouTube 스케줄러 중지
	if r.Scheduler != nil {
		r.Scheduler.Stop()
		if r.Logger != nil {
			r.Logger.Info("YouTube ingestion scheduler stopped")
		}
	}

	if err := r.ShutdownAdminServer(ctx); err != nil {
		if r.Logger != nil {
			r.Logger.Error("HTTP server shutdown error", slog.Any("error", err))
		}
	}
	if r.Bot != nil {
		if err := r.Bot.Shutdown(ctx); err != nil {
			if r.Logger != nil {
				r.Logger.Error("Error during shutdown", slog.Any("error", err))
			}
		}
	}
}

// Run: 봇 애플리케이션을 실행하고 종료 신호(SIGINT, SIGTERM)를 대기한다. (블로킹)
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
			r.Logger.Info("Received shutdown signal", slog.String("signal", sig.String()))
		}
	case err := <-errCh:
		if r.Logger != nil {
			r.Logger.Error("Server error", slog.Any("error", err))
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
