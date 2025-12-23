package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// ServerApp 는 타입이다.
type ServerApp struct {
	Bot             string
	Logger          *slog.Logger
	Server          *http.Server
	ShutdownTimeout time.Duration
	BackgroundTasks []BackgroundTask
}

// NewServerApp 는 동작을 수행한다.
func NewServerApp(
	bot string,
	logger *slog.Logger,
	server *http.Server,
	shutdownTimeout time.Duration,
	backgroundTasks ...BackgroundTask,
) *ServerApp {
	return &ServerApp{
		Bot:             bot,
		Logger:          logger,
		Server:          server,
		ShutdownTimeout: shutdownTimeout,
		BackgroundTasks: backgroundTasks,
	}
}

// Run 는 동작을 수행한다.
func (a *ServerApp) Run(ctx context.Context) error {
	if a == nil {
		return nil
	}
	return RunHTTPServer(
		ctx,
		a.Logger,
		a.Bot,
		a.Server,
		a.ShutdownTimeout,
		a.BackgroundTasks...,
	)
}
