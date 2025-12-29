package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// ServerApp: HTTP 서버와 백그라운드 작업을 포함하는 애플리케이션 실행 단위입니다.
type ServerApp struct {
	Bot             string
	Logger          *slog.Logger
	Server          *http.Server
	ShutdownTimeout time.Duration
	BackgroundTasks []BackgroundTask
}

// NewServerApp: 새로운 ServerApp 인스턴스를 생성합니다.
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

// Run: 애플리케이션(HTTP 서버 및 백그라운드 작업)을 실행합니다.
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
