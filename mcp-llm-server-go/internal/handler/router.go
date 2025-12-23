package handler

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/middleware"
)

// NewRouter 는 HTTP 라우터를 구성한다.
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
	router.Use(
		middleware.RequestID(),
		middleware.RequestLogger(logger),
		gin.Recovery(),
		middleware.APIKeyAuth(cfg),
		middleware.RateLimit(cfg),
	)

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
