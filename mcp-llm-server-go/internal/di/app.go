package di

import (
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

// App: 애플리케이션 구성 요소를 묶는다.
type App struct {
	Server          *http.Server
	GRPCServer      *grpc.Server
	GRPCListener    net.Listener // TCP 리스너
	GRPCUDSListener net.Listener // UDS 리스너 (선택적)
	Logger          *slog.Logger
	Config          *config.Config
	SessionStore    *session.Store
	UsageRepository *usage.Repository
	UsageRecorder   *usage.Recorder
}

// NewApp: App 인스턴스를 생성합니다.
func NewApp(
	server *http.Server,
	grpcServer *grpc.Server,
	grpcListener net.Listener,
	grpcUDSListener net.Listener,
	logger *slog.Logger,
	cfg *config.Config,
	sessionStore *session.Store,
	usageRepository *usage.Repository,
	usageRecorder *usage.Recorder,
) *App {
	return &App{
		Server:          server,
		GRPCServer:      grpcServer,
		GRPCListener:    grpcListener,
		GRPCUDSListener: grpcUDSListener,
		Logger:          logger,
		Config:          cfg,
		SessionStore:    sessionStore,
		UsageRepository: usageRepository,
		UsageRecorder:   usageRecorder,
	}
}

// Close: 앱 리소스를 정리합니다.
func (a *App) Close() {
	if a.GRPCServer != nil {
		a.GRPCServer.Stop()
	}
	if a.GRPCListener != nil {
		_ = a.GRPCListener.Close()
	}
	if a.GRPCUDSListener != nil {
		_ = a.GRPCUDSListener.Close()
	}
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
