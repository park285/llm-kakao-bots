package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/server"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/docker"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
)

// ProvideAdminAddr: 관리자 서버가 리슨할 주소를 반환한다.
func ProvideAdminAddr(cfg *config.Config) string {
	return fmt.Sprintf(":%d", cfg.Server.Port)
}

// ProvideAdminServer: 관리자용 HTTP 서버 인스턴스를 생성한다.
// H2C(HTTP/2 Cleartext)를 기본으로 사용하여 멀티플렉싱과 헤더 압축 이점을 제공한다.
func ProvideAdminServer(addr string, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           server.WrapH2C(router),
		ReadHeaderTimeout: constants.ServerTimeout.ReadHeader,
		IdleTimeout:       constants.ServerTimeout.Idle,
	}
}

// ProvideAdminRouter: 관리자 API 및 UI를 서빙하는 Gin 라우터를 설정하고 제공한다.
func ProvideAdminRouter(
	ctx context.Context,
	logger *slog.Logger,
	adminHandler *server.AdminHandler,
	dockerSvc *docker.Service,
	sessions *server.ValkeySessionStore,
	securityCfg *server.SecurityConfig,
	allowedCIDRs []*net.IPNet,
	memberRepo *member.Repository,
	settingsSvc *settings.Service,
) (*gin.Engine, error) {
	router, err := newAdminRouter(ctx, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Valkey session store initialized")

	adminIPGuard := server.AdminIPAllowMiddleware(allowedCIDRs, logger)
	if len(allowedCIDRs) == 0 {
		logger.Error("Admin IP allowlist is empty; denying all admin requests")
	} else {
		logger.Info("Admin IP allowlist applied", slog.Int("cidr_count", len(allowedCIDRs)))
	}

	//nolint:contextcheck // gin 미들웨어는 c.Request.Context()로 context 전달
	adminAuth := server.AdminAuthMiddleware(sessions, securityCfg.SessionSecret, securityCfg.ForceHTTPS)
	registerAdminRoutes(router, adminIPGuard, adminAuth, adminHandler)

	// Docker 컨테이너 관리 API (선택적)
	dockerHandler := server.NewDockerHandler(dockerSvc)
	registerDockerRoutes(router, adminIPGuard, adminAuth, dockerHandler)
	if dockerSvc != nil {
		logger.Info("Docker management API enabled")
	}

	// SSR 데이터 인젝터 생성 및 HTML 캐시
	ssrInjector := server.NewSSRDataInjector(memberRepo, settingsSvc, dockerSvc)
	if htmlData, err := os.ReadFile(constants.AdminUIConfig.IndexPath); err == nil {
		ssrInjector.SetHTMLCache(htmlData)
		logger.Info("SSR data injector enabled", slog.String("path", constants.AdminUIConfig.IndexPath))
	} else {
		logger.Warn("SSR data injection disabled: failed to cache HTML", slog.Any("error", err))
	}

	//nolint:contextcheck // gin 핸들러는 c.Request.Context()로 context 전달
	registerAdminUIRoutes(router, ssrInjector, sessions, securityCfg.SessionSecret)

	return router, nil
}

func newAdminRouter(ctx context.Context, logger *slog.Logger) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if err := router.SetTrustedProxies(constants.ServerConfig.TrustedProxies); err != nil {
		return nil, fmt.Errorf("failed to set trusted proxies: %w", err)
	}
	router.TrustedPlatform = gin.PlatformCloudflare
	router.Use(gin.Recovery())
	router.Use(server.LoggerMiddleware(ctx, logger, "/health")) // HTTP 접속 로깅 (/health 제외)
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
	// Health check 엔드포인트 (Gzip 비활성화 - 작은 응답)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "ok",
			"goroutines": runtime.NumGoroutine(),
		})
	})
}

func registerAdminRoutes(
	router *gin.Engine,
	adminIPGuard gin.HandlerFunc,
	adminAuth gin.HandlerFunc,
	adminHandler *server.AdminHandler,
) {
	// 공개 관리자 API 라우트 (인증 불필요)
	router.POST("/admin/api/login", adminIPGuard, adminHandler.HandleLogin)
	router.GET("/admin/api/logout", adminIPGuard, adminHandler.HandleLogout)

	// 보호된 관리자 API 라우트 (인증 필요)
	adminAPI := router.Group("/admin/api", adminIPGuard, adminAuth)
	adminAPI.POST("/heartbeat", adminHandler.HandleHeartbeat) // 세션 TTL 갱신
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
	adminAPI.PATCH("/members/:id/name", adminHandler.UpdateMemberName)
	adminAPI.GET("/streams/live", adminHandler.GetLiveStreams)
	adminAPI.GET("/streams/upcoming", adminHandler.GetUpcomingStreams)
	adminAPI.GET("/stats/channels", adminHandler.GetChannelStats)
	adminAPI.GET("/logs", adminHandler.GetLogs)
	adminAPI.GET("/settings", adminHandler.GetSettings)
	adminAPI.POST("/settings", adminHandler.UpdateSettings)

	// 마일스톤 API
	adminAPI.GET("/milestones", adminHandler.GetMilestones)
	adminAPI.GET("/milestones/near", adminHandler.GetNearMilestoneMembers)
	adminAPI.GET("/milestones/stats", adminHandler.GetMilestoneStats)

	// WebSocket 라우트 (실시간 스트리밍)
	adminAPI.GET("/ws/system-stats", adminHandler.StreamSystemStats)
}

func registerDockerRoutes(
	router *gin.Engine,
	adminIPGuard gin.HandlerFunc,
	adminAuth gin.HandlerFunc,
	dockerHandler *server.DockerHandler,
) {
	dockerAPI := router.Group("/admin/api/docker", adminIPGuard, adminAuth)
	dockerAPI.GET("/health", dockerHandler.GetHealth)
	dockerAPI.GET("/containers", dockerHandler.GetContainers)
	dockerAPI.POST("/containers/:name/restart", dockerHandler.RestartContainer)
	dockerAPI.POST("/containers/:name/stop", dockerHandler.StopContainer)
	dockerAPI.POST("/containers/:name/start", dockerHandler.StartContainer)

	// WebSocket 라우트 (실시간 로그 스트리밍)
	dockerAPI.GET("/containers/:name/logs/stream", dockerHandler.StreamLogs)
}

func registerAdminUIRoutes(
	router *gin.Engine,
	ssrInjector *server.SSRDataInjector,
	sessions *server.ValkeySessionStore,
	sessionSecret string,
) {
	// Serve React SPA (프로덕션용 React 빌드)
	router.Static(constants.AdminUIConfig.AssetsRoute, constants.AdminUIConfig.AssetsDir)

	// SSR 데이터 주입 핸들러 (인증 상태 확인 후 데이터 프리페칭)
	serveWithSSR := func(c *gin.Context) {
		c.Header("Cache-Control", constants.AdminUIConfig.CacheControlHTML)

		// 인증 상태 확인 (세션 쿠키에서)
		isAuthenticated := checkAuthFromCookie(c, sessions, sessionSecret)

		// SSR 데이터 주입 시도
		html, err := ssrInjector.InjectForPath(c.Request.Context(), c.Request.URL.Path, isAuthenticated)
		if err != nil || len(html) == 0 {
			// 폴백: 원본 HTML 파일 서빙
			c.File(constants.AdminUIConfig.IndexPath)
			return
		}

		c.Data(200, "text/html; charset=utf-8", html)
	}

	// HTML은 항상 최신 버전을 받도록 캐시 금지
	router.GET("/", serveWithSSR)

	// Favicon 서빙 (NoRoute에 걸리지 않도록 명시적 처리)
	router.GET(constants.AdminUIConfig.FaviconRoute, func(c *gin.Context) {
		c.Header("Cache-Control", constants.AdminUIConfig.CacheControlFavicon)
		c.File(constants.AdminUIConfig.FaviconPath)
	})

	router.NoRoute(serveWithSSR)
}

// checkAuthFromCookie: 세션 쿠키에서 인증 상태를 확인합니다. (SSR 전용)
// 인증 미들웨어와 동일한 로직이지만 HTTP 오류 대신 bool을 반환합니다.
func checkAuthFromCookie(c *gin.Context, sessions *server.ValkeySessionStore, sessionSecret string) bool {
	signedSessionID, err := c.Cookie("admin_session")
	if err != nil || signedSessionID == "" {
		return false
	}

	// HMAC 서명 검증
	sessionID, valid := server.ValidateSessionSignature(signedSessionID, sessionSecret)
	if !valid {
		return false
	}

	// Valkey 세션 존재 여부 확인
	return sessions.ValidateSession(c.Request.Context(), sessionID)
}
