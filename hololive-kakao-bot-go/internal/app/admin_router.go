package app

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/server"
)

// ProvideAdminAddr: 관리자 서버가 리슨할 주소를 반환한다.
func ProvideAdminAddr(cfg *config.Config) string {
	return fmt.Sprintf(":%d", cfg.Server.Port)
}

// ProvideAdminServer: 관리자용 HTTP 서버 인스턴스를 생성한다.
func ProvideAdminServer(addr string, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: constants.ServerTimeout.ReadHeader,
		IdleTimeout:       constants.ServerTimeout.Idle,
	}
}

// ProvideAdminRouter: 관리자 API 및 UI를 서빙하는 Gin 라우터를 설정하고 제공한다.
func ProvideAdminRouter(
	logger *slog.Logger,
	adminHandler *server.AdminHandler,
	sessions *server.ValkeySessionStore,
	securityCfg *server.SecurityConfig,
	allowedCIDRs []*net.IPNet,
) (*gin.Engine, error) {
	router, err := newAdminRouter(logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Valkey session store initialized")

	adminIPGuard := server.AdminIPAllowMiddleware(allowedCIDRs, logger)
	logger.Info("Admin IP allowlist applied", slog.Int("cidr_count", len(allowedCIDRs)))

	adminAuth := server.AdminAuthMiddleware(sessions, securityCfg.SessionSecret)
	registerAdminRoutes(router, adminIPGuard, adminAuth, adminHandler)
	registerAdminUIRoutes(router)

	return router, nil
}

func newAdminRouter(logger *slog.Logger) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if err := router.SetTrustedProxies(constants.ServerConfig.TrustedProxies); err != nil {
		return nil, fmt.Errorf("trusted proxies 설정 실패: %w", err)
	}
	router.TrustedPlatform = gin.PlatformCloudflare
	router.Use(gin.Recovery())
	router.Use(server.LoggerMiddleware(logger, "/health")) // HTTP 접속 로깅 (/health 제외)
	router.Use(cors.New(newAdminCORSConfig()))
	router.Use(server.SecurityHeadersMiddleware())
	router.Use(newAdminGzipMiddleware())
	router.Use(newAdminStaticCacheMiddleware())

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

func newAdminGzipMiddleware() gin.HandlerFunc {
	return gzip.Gzip(gzip.DefaultCompression, gzip.WithCustomShouldCompressFn(func(c *gin.Context) bool {
		// Static assets는 항상 압축
		if strings.HasPrefix(c.Request.URL.Path, constants.AdminUIConfig.AssetsURLPrefix) {
			return true
		}
		// Health check, 작은 API 응답은 압축 제외
		if c.Request.URL.Path == "/health" {
			return false
		}
		// 기본적으로 압축 허용 (실제 Content-Length는 응답 후 결정됨)
		return true
	}))
}

func newAdminStaticCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, constants.AdminUIConfig.AssetsURLPrefix) {
			c.Header("Cache-Control", constants.AdminUIConfig.CacheControlAssets)
		}
		c.Next()
	}
}

func registerAdminHealthRoutes(router *gin.Engine) {
	// Health check endpoint (Gzip 비활성화 - 작은 응답)
	router.GET("/health", func(c *gin.Context) {
		c.Data(200, "application/json", []byte(`{"status":"ok"}`))
	})
}

func registerAdminRoutes(
	router *gin.Engine,
	adminIPGuard gin.HandlerFunc,
	adminAuth gin.HandlerFunc,
	adminHandler *server.AdminHandler,
) {
	// Public admin API routes (no auth required)
	router.POST("/admin/api/login", adminIPGuard, adminHandler.HandleLogin)
	router.GET("/admin/api/logout", adminIPGuard, adminHandler.HandleLogout)

	// Protected admin API routes (auth required)
	adminAPI := router.Group("/admin/api", adminIPGuard, adminAuth)
	adminAPI.GET("/members", adminHandler.GetMembers)
	adminAPI.POST("/members", adminHandler.AddMember)
	adminAPI.POST("/members/:id/aliases", adminHandler.AddAlias)
	adminAPI.DELETE("/members/:id/aliases", adminHandler.RemoveAlias)
	adminAPI.GET("/alarms", adminHandler.GetAlarms)
	adminAPI.DELETE("/alarms", adminHandler.DeleteAlarm)
	adminAPI.GET("/rooms", adminHandler.GetRooms)
	adminAPI.POST("/rooms", adminHandler.AddRoom)
	adminAPI.DELETE("/rooms", adminHandler.RemoveRoom)
	adminAPI.POST("/rooms/acl", adminHandler.SetACL)
	adminAPI.GET("/stats", adminHandler.GetStats)
	adminAPI.POST("/names/room", adminHandler.SetRoomName)
	adminAPI.POST("/names/user", adminHandler.SetUserName)
	adminAPI.PATCH("/members/:id/graduation", adminHandler.SetGraduation)
	adminAPI.PATCH("/members/:id/channel", adminHandler.UpdateChannelID)
	adminAPI.GET("/streams/live", adminHandler.GetLiveStreams)
	adminAPI.GET("/streams/upcoming", adminHandler.GetUpcomingStreams)
	adminAPI.GET("/stats/channels", adminHandler.GetChannelStats)
	adminAPI.GET("/logs", adminHandler.GetLogs)
	adminAPI.GET("/settings", adminHandler.GetSettings)
	adminAPI.POST("/settings", adminHandler.UpdateSettings)
}

func registerAdminUIRoutes(router *gin.Engine) {
	// Serve React SPA (프로덕션용 React 빌드)
	router.Static(constants.AdminUIConfig.AssetsRoute, constants.AdminUIConfig.AssetsDir)

	// HTML은 항상 최신 버전을 받도록 캐시 금지 (업데이트 시 구버전 HTML 캐싱으로 인한 chunk mismatch 방지)
	router.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", constants.AdminUIConfig.CacheControlHTML)
		c.File(constants.AdminUIConfig.IndexPath)
	})

	// Favicon 서빙 (NoRoute에 걸리지 않도록 명시적 처리)
	router.GET(constants.AdminUIConfig.FaviconRoute, func(c *gin.Context) {
		c.Header("Cache-Control", constants.AdminUIConfig.CacheControlFavicon) // 24시간 캐시
		c.File(constants.AdminUIConfig.FaviconPath)
	})

	router.NoRoute(func(c *gin.Context) {
		c.Header("Cache-Control", constants.AdminUIConfig.CacheControlHTML)
		c.File(constants.AdminUIConfig.IndexPath)
	})
}
