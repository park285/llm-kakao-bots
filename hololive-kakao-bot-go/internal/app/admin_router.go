package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/server"
)

// ProvideAdminAddr: 관리자 서버가 리슨할 주소를 반환합니다.
func ProvideAdminAddr(cfg *config.Config) string {
	return fmt.Sprintf(":%d", cfg.Server.Port)
}

// ProvideAdminServer: 관리자용 HTTP 서버 인스턴스를 생성합니다.
// H2C(HTTP/2 Cleartext)를 기본으로 사용하여 멀티플렉싱과 헤더 압축 이점을 제공한다.
func ProvideAdminServer(addr string, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           server.WrapH2C(router),
		ReadHeaderTimeout: constants.ServerTimeout.ReadHeader,
		ReadTimeout:       constants.ServerTimeout.Read,
		WriteTimeout:      constants.ServerTimeout.Write,
		IdleTimeout:       constants.ServerTimeout.Idle,
		MaxHeaderBytes:    constants.ServerTimeout.MaxHeaderBytes,
	}
}

// ProvideAdminRouter: hololive-bot 도메인 전용 관리자 API를 서빙하는 Gin 라우터를 설정합니다.
// Docker, Traces, UI 서빙은 admin-dashboard가 담당합니다.
func ProvideAdminRouter(
	ctx context.Context,
	cfg *config.Config,
	logger *slog.Logger,
	adminHandler *server.AdminHandler,
) (*gin.Engine, error) {
	router, err := newAdminRouter(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	registerAdminRoutes(router, adminHandler)

	return router, nil
}

func newAdminRouter(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if err := router.SetTrustedProxies(constants.ServerConfig.TrustedProxies); err != nil {
		return nil, fmt.Errorf("failed to set trusted proxies: %w", err)
	}
	router.TrustedPlatform = gin.PlatformCloudflare

	// OTel 미들웨어: 활성화된 경우 모든 HTTP 요청을 추적함 (가장 앞에 배치)
	if cfg.Telemetry.Enabled {
		serviceName := cfg.Telemetry.ServiceName
		if serviceName == "" {
			serviceName = "hololive-bot"
		}
		router.Use(otelgin.Middleware(serviceName))
		logger.Info("otel_http_middleware_enabled", slog.String("service", serviceName))
	}

	router.Use(gin.Recovery())
	router.Use(server.LoggerMiddleware(ctx, logger,
		"/health",
		"/metrics",        // Prometheus 메트릭 폴링 (15초 간격)
		"/api/holo/ws/*",  // WebSocket 시스템 통계 스트리밍
	))
	router.Use(cors.New(newAdminCORSConfig()))
	router.Use(server.SecurityHeadersMiddleware())
	router.Use(server.ClientHintsMiddleware()) // Client Hints 요청 (실제 기기 정보 수집)

	registerAdminHealthRoutes(router)
	return router, nil
}

func newAdminCORSConfig() cors.Config {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = constants.CORSConfig.AllowOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = constants.CORSConfig.AllowMethods
	corsConfig.AllowHeaders = constants.CORSConfig.AllowHeaders
	return corsConfig
}

func registerAdminHealthRoutes(router *gin.Engine) {
	// Health check 엔드포인트 (작은 응답)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "ok",
			"goroutines": runtime.NumGoroutine(),
		})
	})

	// Prometheus 메트릭 (장기 히스토리 분석용)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func registerAdminRoutes(
	router *gin.Engine,
	adminHandler *server.AdminHandler,
) {
	// hololive-bot 도메인 API (admin-dashboard에서 인증 후 프록시)
	holoAPI := router.Group("/api/holo")
	holoAPI.GET("/members", adminHandler.GetMembers)
	holoAPI.POST("/members", adminHandler.AddMember)
	holoAPI.POST("/members/:id/aliases", adminHandler.AddAlias)
	holoAPI.DELETE("/members/:id/aliases", adminHandler.RemoveAlias)
	holoAPI.PATCH("/members/:id/graduation", adminHandler.SetGraduation)
	holoAPI.PATCH("/members/:id/channel", adminHandler.UpdateChannelID)
	holoAPI.PATCH("/members/:id/name", adminHandler.UpdateMemberName)

	holoAPI.GET("/alarms", adminHandler.GetAlarms)
	holoAPI.DELETE("/alarms", adminHandler.DeleteAlarm)

	holoAPI.GET("/rooms", adminHandler.GetRooms)
	holoAPI.POST("/rooms", adminHandler.AddRoom)
	holoAPI.DELETE("/rooms", adminHandler.RemoveRoom)
	holoAPI.POST("/rooms/acl", adminHandler.SetACL)

	holoAPI.GET("/stats", adminHandler.GetStats)
	holoAPI.GET("/stats/channels", adminHandler.GetChannelStats)
	holoAPI.GET("/streams/live", adminHandler.GetLiveStreams)
	holoAPI.GET("/streams/upcoming", adminHandler.GetUpcomingStreams)

	holoAPI.GET("/logs", adminHandler.GetLogs)
	holoAPI.GET("/settings", adminHandler.GetSettings)
	holoAPI.POST("/settings", adminHandler.UpdateSettings)
	holoAPI.POST("/names/room", adminHandler.SetRoomName)
	holoAPI.POST("/names/user", adminHandler.SetUserName)

	// 마일스톤 API
	holoAPI.GET("/milestones", adminHandler.GetMilestones)
	holoAPI.GET("/milestones/near", adminHandler.GetNearMilestoneMembers)
	holoAPI.GET("/milestones/stats", adminHandler.GetMilestoneStats)

	// WebSocket 라우트 (실시간 스트리밍)
	holoAPI.GET("/ws/system-stats", adminHandler.StreamSystemStats)
}
