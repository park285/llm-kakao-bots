package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httpserver"
)

// BackgroundTask 는 타입이다.
type BackgroundTask struct {
	Name        string
	ErrorLogKey string
	Run         func(ctx context.Context) error
}

// RunHTTPServer 는 동작을 수행한다.
func RunHTTPServer(
	ctx context.Context,
	logger *slog.Logger,
	bot string,
	server *http.Server,
	shutdownTimeout time.Duration,
	backgroundTasks ...BackgroundTask,
) error {
	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gctx := errgroup.WithContext(signalCtx)

	for _, task := range backgroundTasks {
		t := task
		if t.Run == nil {
			continue
		}

		g.Go(func() error {
			if err := t.Run(gctx); err != nil {
				logKey := t.ErrorLogKey
				if logKey == "" {
					logKey = "background_task_failed"
				}
				logger.Error(logKey, "task", t.Name, "err", err)
				return fmt.Errorf("%s failed: %w", t.Name, err)
			}
			return nil
		})
	}

	logger.Info("server_start", "bot", bot, "addr", server.Addr)
	g.Go(func() error {
		if err := httpserver.Serve(gctx, server, shutdownTimeout); err != nil {
			return fmt.Errorf("http server serve failed: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("run http server failed: %w", err)
	}
	return nil
}
