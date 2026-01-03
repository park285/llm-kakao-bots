package handler

import (
	"log/slog"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
)

// NewRouter: HTTP 라우터를 구성합니다.
func NewRouter(
	cfg *config.Config,
	logger *slog.Logger,
	llmHandler *LLMHandler,
	sessionHandler *SessionHandler,
	guardHandler *GuardHandler,
	usageHandler *UsageHandler,
	twentyqHandler *TwentyQHandler,
	turtleSoupHandler *TurtleSoupHandler,
) *gin.Engine {
	setGinMode(cfg.Logging.Level)

	router := gin.New()

	// 미들웨어 체인: OTel이 가장 앞에 있어야 모든 요청을 추적함
	middlewares := []gin.HandlerFunc{
		middleware.RequestID(),
		middleware.RequestLogger(logger),
		gin.Recovery(),
		gzip.Gzip(gzip.DefaultCompression),
		middleware.APIKeyAuth(cfg),
		middleware.RateLimit(cfg),
	}

	// OTel 미들웨어: 활성화된 경우에만 추가 (가장 앞에 배치)
	if cfg.Telemetry.Enabled {
		middlewares = append([]gin.HandlerFunc{
			otelgin.Middleware(cfg.Telemetry.ServiceName),
		}, middlewares...)
	}

	router.Use(middlewares...)

	RegisterHealthRoutes(router, cfg)
	llmHandler.RegisterRoutes(router)
	sessionHandler.RegisterRoutes(router)
	guardHandler.RegisterRoutes(router)
	usageHandler.RegisterRoutes(router)
	twentyqHandler.RegisterRoutes(router)
	turtleSoupHandler.RegisterRoutes(router)

	return router
}

func setGinMode(level string) {
	if strings.EqualFold(strings.TrimSpace(level), "debug") {
		gin.SetMode(gin.DebugMode)
		return
	}
	gin.SetMode(gin.ReleaseMode)
}
