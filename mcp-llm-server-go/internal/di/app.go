package di

import (
	"log/slog"
	"net/http"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

// App 은 애플리케이션 구성 요소를 묶는다.
type App struct {
	Server          *http.Server
	Logger          *slog.Logger
	Config          *config.Config
	SessionStore    *session.Store
	UsageRepository *usage.Repository
	UsageRecorder   *usage.Recorder
}

// NewApp 은 App 인스턴스를 생성한다.
func NewApp(
	server *http.Server,
	logger *slog.Logger,
	cfg *config.Config,
	sessionStore *session.Store,
	usageRepository *usage.Repository,
	usageRecorder *usage.Recorder,
) *App {
	return &App{
		Server:          server,
		Logger:          logger,
		Config:          cfg,
		SessionStore:    sessionStore,
		UsageRepository: usageRepository,
		UsageRecorder:   usageRecorder,
	}
}

// Close 앱 리소스 정리
func (a *App) Close() {
	if a.SessionStore != nil {
		a.SessionStore.Close()
	}
	if a.UsageRecorder != nil {
		a.UsageRecorder.Close()
	}
	if a.UsageRepository != nil {
		a.UsageRepository.Close()
	}
}
