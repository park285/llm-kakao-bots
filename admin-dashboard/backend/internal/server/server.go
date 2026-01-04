// Package server: HTTP 서버 및 라우팅
package server

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/crypto/bcrypt"

	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/auth"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/config"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/docker"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/logs"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/metrics"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/middleware"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/proxy"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/ssr"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/static"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/status"
	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/traces"
)

// Server: HTTP 서버
type Server struct {
	engine          *gin.Engine
	cfg             *config.Config
	logger          *slog.Logger
	sessions        auth.SessionProvider
	rateLimiter     *auth.LoginRateLimiter
	dockerSvc       *docker.Service
	tracesClient    *traces.Client
	botProxies      *proxy.BotProxies
	statusCollector *status.Collector
	ssrInjector     *ssr.Injector
	ssrConfig       ssr.Config
}

// New: 서버 생성
func New(
	cfg *config.Config,
	logger *slog.Logger,
	sessions auth.SessionProvider,
	dockerSvc *docker.Service,
	tracesClient *traces.Client,
	botProxies *proxy.BotProxies,
	statusCollector *status.Collector,
) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// OTel 미들웨어: 활성화된 경우 모든 HTTP 요청을 추적함 (가장 앞에 배치)
	if cfg.OTELEnabled {
		serviceName := strings.TrimSpace(cfg.OTELServiceName)
		if serviceName == "" {
			serviceName = "admin-dashboard"
		}
		engine.Use(otelgin.Middleware(serviceName))
		if logger != nil {
			logger.Info("otel_http_middleware_enabled", slog.String("service", serviceName))
		}
	}

	engine.Use(gin.Recovery())
	engine.Use(auth.SecurityHeadersMiddleware())

	// CORS 설정
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://admin.capu.blog", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 압축: Cloudflare Tunnel Edge에서 Brotli/Gzip 처리 (서버 CPU 자원 보호)

	// ETag: API GET 응답에 조건부 요청 지원 (304 Not Modified)
	engine.Use(middleware.ETag())

	// Early Hints: 비활성화 - Cloudflare Tunnel과 호환 문제로 임시 비활성화
	// TODO: Cloudflare Tunnel 환경에서 103 응답이 제대로 전달되는지 확인 필요
	// engine.Use(middleware.EarlyHints(nil))

	// Static 캐시 미들웨어
	engine.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})

	// SSR 설정
	ssrConfig := ssr.DefaultConfig()
	ssrInjector := ssr.NewInjector(dockerSvc, cfg.HoloBotURL, logger)

	// HTML 캐시 로드: 임베디드 우선, 파일시스템 폴백
	if static.HasEmbedded() {
		if htmlData, err := static.IndexHTML(); err == nil {
			ssrInjector.LoadHTMLFromBytes(htmlData)
			logger.Info("ssr_using_embedded_static")
		}
	} else {
		if err := ssrInjector.LoadHTMLCache(ssrConfig.IndexPath); err != nil {
			logger.Warn("ssr_html_cache_failed", slog.Any("error", err))
		} else {
			logger.Info("ssr_html_cache_loaded", slog.String("path", ssrConfig.IndexPath))
		}
	}

	s := &Server{
		engine:          engine,
		cfg:             cfg,
		logger:          logger,
		sessions:        sessions,
		rateLimiter:     auth.NewLoginRateLimiter(),
		dockerSvc:       dockerSvc,
		tracesClient:    tracesClient,
		botProxies:      botProxies,
		statusCollector: statusCollector,
		ssrInjector:     ssrInjector,
		ssrConfig:       ssrConfig,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.engine.Group("/admin/api")

	// 도메인별 라우터 설정
	s.setupAuthRoutes(api)

	// 인증 필요 라우트
	authenticated := api.Group("")
	authenticated.Use(auth.AuthMiddleware(s.sessions, s.cfg.AdminSecretKey, s.cfg.ForceHTTPS))

	s.setupDockerRoutes(authenticated)
	s.setupLogsRoutes(authenticated)
	s.setupTracesRoutes(authenticated)
	s.setupStatusRoutes(authenticated)
	s.setupProxyRoutes(authenticated)

	// Health & Static
	s.setupHealthRoute()
	s.setupMetricsRoute()
	s.setupStaticRoutes()
}

// setupAuthRoutes: 인증 관련 라우트 (미들웨어 없음)
func (s *Server) setupAuthRoutes(api *gin.RouterGroup) {
	authGroup := api.Group("/auth")
	authGroup.POST("/login", s.handleLogin)
	authGroup.POST("/logout", s.handleLogout)
	authGroup.POST("/heartbeat", s.handleHeartbeat)
}

// setupDockerRoutes: Docker 컨테이너 관리 라우트
func (s *Server) setupDockerRoutes(authenticated *gin.RouterGroup) {
	dockerGroup := authenticated.Group("/docker")
	dockerGroup.GET("/health", s.handleDockerHealth)
	dockerGroup.GET("/containers", s.handleDockerContainers)
	dockerGroup.POST("/containers/:name/restart", s.handleDockerRestart)
	dockerGroup.POST("/containers/:name/stop", s.handleDockerStop)
	dockerGroup.POST("/containers/:name/start", s.handleDockerStart)
	dockerGroup.GET("/containers/:name/logs/stream", s.handleDockerLogStream)
}

// setupLogsRoutes: 시스템 로그 라우트
func (s *Server) setupLogsRoutes(authenticated *gin.RouterGroup) {
	logsGroup := authenticated.Group("/logs")
	logsGroup.GET("/files", s.handleLogFiles)
	logsGroup.GET("", s.handleSystemLogs)
}

// setupTracesRoutes: Jaeger 분산 추적 라우트
func (s *Server) setupTracesRoutes(authenticated *gin.RouterGroup) {
	tracesGroup := authenticated.Group("/traces")
	tracesGroup.GET("/health", s.handleTracesHealth)
	tracesGroup.GET("/services", s.handleTracesServices)
	tracesGroup.GET("/operations/:service", s.handleTracesOperations)
	tracesGroup.GET("", s.handleTracesSearch)
	tracesGroup.GET("/:traceId", s.handleTraceDetail)
	tracesGroup.GET("/dependencies", s.handleTracesDependencies)
	tracesGroup.GET("/metrics/:service", s.handleTracesMetrics)
}

// setupStatusRoutes: 통합 시스템 상태 라우트
func (s *Server) setupStatusRoutes(authenticated *gin.RouterGroup) {
	statusGroup := authenticated.Group("/status")
	statusGroup.GET("", s.handleAggregatedStatus)

	// WebSocket: 실시간 시스템 리소스 스트리밍 (CPU, Memory, Goroutines)
	// 기존 /admin/api/holo/ws/system-stats → /admin/api/ws/system-stats로 이관
	wsGroup := authenticated.Group("/ws")
	wsGroup.GET("/system-stats", s.handleSystemStatsStream)
}

// setupProxyRoutes: 도메인 봇 프록시 라우트
func (s *Server) setupProxyRoutes(authenticated *gin.RouterGroup) {
	if s.botProxies == nil {
		return
	}

	// 도메인별 프록시
	authenticated.Any("/holo/*path", s.botProxies.ProxyHolo)
	authenticated.Any("/twentyq/*path", s.botProxies.ProxyTwentyQ)
	authenticated.Any("/turtle/*path", s.botProxies.ProxyTurtle)
}

// setupHealthRoute: 헬스체크 라우트 (인증 없음)
func (s *Server) setupHealthRoute() {
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// setupMetricsRoute: Prometheus 메트릭 라우트 (옵션: API 키 보호)
func (s *Server) setupMetricsRoute() {
	s.engine.GET("/metrics",
		metrics.APIKeyAuth(s.cfg.MetricsAPIKey),
		gin.WrapH(promhttp.Handler()),
	)
}

// setupStaticRoutes: React SPA 정적 파일 서빙
func (s *Server) setupStaticRoutes() {
	// 임베디드 FS 사용 여부
	useEmbedded := static.HasEmbedded()

	// Static assets (/assets/*)
	if useEmbedded {
		assetsFS, err := static.Assets()
		if err == nil {
			s.engine.StaticFS("/assets", http.FS(assetsFS))
		}
	} else {
		s.engine.Static("/assets", s.ssrConfig.AssetsDir)
	}

	// Favicon
	s.engine.GET("/favicon.svg", func(c *gin.Context) {
		c.Header("Cache-Control", s.ssrConfig.CacheControlFavicon)
		if useEmbedded {
			if data, err := static.Favicon(); err == nil {
				c.Data(200, "image/svg+xml", data)
				return
			}
		}
		c.File(s.ssrConfig.FaviconPath)
	})

	// SSR 데이터 주입 핸들러
	serveWithSSR := func(c *gin.Context) {
		c.Header("Cache-Control", s.ssrConfig.CacheControlHTML)

		// 인증 상태 확인
		isAuthenticated := s.checkAuthFromCookie(c)
		sessionCookie, _ := c.Cookie("admin_session")

		// SSR 데이터 주입 시도
		html, err := s.ssrInjector.InjectForPath(c.Request.Context(), c.Request.URL.Path, isAuthenticated, sessionCookie)
		if err != nil || len(html) == 0 {
			// 폴백: 캐시된 HTML 또는 파일 시스템
			if cachedHTML := s.ssrInjector.GetHTMLCache(); len(cachedHTML) > 0 {
				c.Data(200, "text/html; charset=utf-8", cachedHTML)
				return
			}
			c.File(s.ssrConfig.IndexPath)
			return
		}

		c.Data(200, "text/html; charset=utf-8", html)
	}

	// 루트 경로
	s.engine.GET("/", serveWithSSR)

	// SPA Fallback (NoRoute)
	s.engine.NoRoute(serveWithSSR)
}

// checkAuthFromCookie: 세션 쿠키에서 인증 상태 확인 (SSR 전용)
func (s *Server) checkAuthFromCookie(c *gin.Context) bool {
	signedSessionID, err := c.Cookie("admin_session")
	if err != nil || signedSessionID == "" {
		return false
	}

	sessionID, valid := auth.ValidateSessionSignature(signedSessionID, s.cfg.AdminSecretKey)
	if !valid {
		return false
	}

	return s.sessions.ValidateSession(c.Request.Context(), sessionID)
}

// HTTPServer: net/http.Server 인스턴스를 반환합니다.
func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:              ":" + s.cfg.Port,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// ===== Auth Handlers =====

// handleLogin godoc
// @Summary      User login
// @Description  Authenticate with username and password. Returns session cookie on success.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login credentials"
// @Success      200      {object}  LoginResponse
// @Failure      400      {object}  ErrorResponse  "Invalid request body"
// @Failure      429      {object}  ErrorResponse  "Too many login attempts"
// @Router       /auth/login [post]
func (s *Server) handleLogin(c *gin.Context) {
	ip := c.ClientIP()

	allowed, remaining := s.rateLimiter.IsAllowed(ip)
	if !allowed {
		s.logger.Warn("Login rate limited", slog.String("ip", ip))
		c.Header("Retry-After", strconv.Itoa(int(remaining.Seconds())))
		c.JSON(429, gin.H{"error": "Too many login attempts", "retry_after": remaining.Seconds()})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if req.Username != s.cfg.AdminUser {
		s.handleLoginFailure(c, ip, req.Username, "invalid_username")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.AdminPassHash), []byte(req.Password)); err != nil {
		s.handleLoginFailure(c, ip, req.Username, "invalid_password")
		return
	}

	s.rateLimiter.RecordSuccess(ip)

	session, err := s.sessions.CreateSession(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to create session", slog.Any("error", err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session store unavailable"})
		return
	}

	signedSessionID := auth.SignSessionID(session.ID, s.cfg.AdminSecretKey)
	auth.SetSecureCookie(c, auth.SessionCookieName, signedSessionID, 0, s.cfg.ForceHTTPS)

	s.logger.Info("Admin logged in", slog.String("username", req.Username), slog.String("ip", ip))
	c.JSON(200, gin.H{"status": "ok", "message": "Login successful"})
}

func (s *Server) handleLoginFailure(c *gin.Context, ip, username, reason string) {
	failCount := s.rateLimiter.RecordFailure(ip)

	s.logger.Warn("Failed login attempt",
		slog.String("username", username),
		slog.String("ip", ip),
		slog.String("reason", reason),
		slog.Int("fail_count", failCount),
	)

	delay := time.Duration(failCount) * 500 * time.Millisecond
	if delay > 3*time.Second {
		delay = 3 * time.Second
	}
	time.Sleep(delay)

	c.JSON(200, gin.H{"success": false, "error": "Authentication failed"})
}

// handleLogout godoc
// @Summary      User logout
// @Description  Invalidate session and clear cookies
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  StatusResponse
// @Router       /auth/logout [post]
func (s *Server) handleLogout(c *gin.Context) {
	signedSessionID, _ := c.Cookie(auth.SessionCookieName)
	if signedSessionID != "" {
		if sessionID, valid := auth.ValidateSessionSignature(signedSessionID, s.cfg.AdminSecretKey); valid {
			s.sessions.DeleteSession(c.Request.Context(), sessionID)
		}
	}

	auth.ClearSecureCookie(c, auth.SessionCookieName, s.cfg.ForceHTTPS)
	c.JSON(200, gin.H{"status": "ok", "message": "Logout successful"})
}

// handleHeartbeat godoc
// @Summary      Session heartbeat
// @Description  Keep session alive and optionally rotate session token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        request  body      HeartbeatRequest  false  "Heartbeat options"
// @Success      200      {object}  HeartbeatResponse
// @Failure      401      {object}  ErrorResponse  "Session expired or invalid"
// @Router       /auth/heartbeat [post]
func (s *Server) handleHeartbeat(c *gin.Context) {
	signedSessionID, err := c.Cookie(auth.SessionCookieName)
	if err != nil || signedSessionID == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	sessionID, valid := auth.ValidateSessionSignature(signedSessionID, s.cfg.AdminSecretKey)
	if !valid {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Idle bool `json:"idle"`
	}
	_ = c.ShouldBindJSON(&req)

	ctx := c.Request.Context()
	refreshed, absoluteExpired, err := s.sessions.RefreshSessionWithValidation(ctx, sessionID, req.Idle)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	if absoluteExpired {
		auth.ClearSecureCookie(c, auth.SessionCookieName, s.cfg.ForceHTTPS)
		c.JSON(401, gin.H{"error": "Session expired", "absolute_expired": true})
		return
	}

	if req.Idle && !refreshed {
		c.JSON(200, gin.H{"status": "idle", "idle_rejected": true})
		return
	}

	if !refreshed {
		auth.ClearSecureCookie(c, auth.SessionCookieName, s.cfg.ForceHTTPS)
		c.JSON(401, gin.H{"error": "Session expired"})
		return
	}

	response := gin.H{"status": "ok"}

	if s.cfg.SessionTokenRotation {
		newSession, rotateErr := s.sessions.RotateSession(ctx, sessionID)
		if rotateErr == nil {
			newSignedSessionID := auth.SignSessionID(newSession.ID, s.cfg.AdminSecretKey)
			auth.SetSecureCookie(c, auth.SessionCookieName, newSignedSessionID, 0, s.cfg.ForceHTTPS)
			response["rotated"] = true
			response["absolute_expires_at"] = newSession.AbsoluteExpiresAt.Unix()
		}
	}

	c.JSON(200, response)
}

// ===== Docker Handlers =====

// handleDockerHealth godoc
// @Summary      Docker health check
// @Description  Check if Docker daemon is accessible
// @Tags         docker
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  DockerHealthResponse
// @Router       /docker/health [get]
func (s *Server) handleDockerHealth(c *gin.Context) {
	if s.dockerSvc == nil {
		c.JSON(http.StatusOK, gin.H{"status": "unavailable", "available": false})
		return
	}
	available := s.dockerSvc.Available(c.Request.Context())
	dockerStatus := "ok"
	if !available {
		dockerStatus = "unavailable"
	}
	c.JSON(http.StatusOK, gin.H{"status": dockerStatus, "available": available})
}

// handleDockerContainers godoc
// @Summary      List containers
// @Description  Get all managed Docker containers with status and resource usage
// @Tags         docker
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  ContainerListResponse
// @Failure      503  {object}  ErrorResponse  "Docker service unavailable"
// @Router       /docker/containers [get]
func (s *Server) handleDockerContainers(c *gin.Context) {
	if s.dockerSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker service not available"})
		return
	}
	containers, err := s.dockerSvc.ListContainers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "containers": containers})
}

// handleDockerRestart godoc
// @Summary      Restart container
// @Description  Restart a managed Docker container by name
// @Tags         docker
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        name  path      string  true  "Container name"
// @Success      200   {object}  StatusResponse
// @Failure      404   {object}  ErrorResponse  "Container not found"
// @Failure      503   {object}  ErrorResponse  "Docker service unavailable"
// @Router       /docker/containers/{name}/restart [post]
func (s *Server) handleDockerRestart(c *gin.Context) {
	if s.dockerSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker service not available"})
		return
	}
	name := c.Param("name")
	if !s.dockerSvc.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	if err := s.dockerSvc.RestartContainer(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Container restart initiated"})
}

// handleDockerStop godoc
// @Summary      Stop container
// @Description  Stop a managed Docker container by name
// @Tags         docker
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        name  path      string  true  "Container name"
// @Success      200   {object}  StatusResponse
// @Failure      404   {object}  ErrorResponse  "Container not found"
// @Failure      503   {object}  ErrorResponse  "Docker service unavailable"
// @Router       /docker/containers/{name}/stop [post]
func (s *Server) handleDockerStop(c *gin.Context) {
	if s.dockerSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker service not available"})
		return
	}
	name := c.Param("name")
	if !s.dockerSvc.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	if err := s.dockerSvc.StopContainer(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Container stopped"})
}

// handleDockerStart godoc
// @Summary      Start container
// @Description  Start a stopped Docker container by name
// @Tags         docker
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        name  path      string  true  "Container name"
// @Success      200   {object}  StatusResponse
// @Failure      404   {object}  ErrorResponse  "Container not found"
// @Failure      503   {object}  ErrorResponse  "Docker service unavailable"
// @Router       /docker/containers/{name}/start [post]
func (s *Server) handleDockerStart(c *gin.Context) {
	if s.dockerSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker service not available"})
		return
	}
	name := c.Param("name")
	if !s.dockerSvc.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	if err := s.dockerSvc.StartContainer(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Container started"})
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS는 별도 미들웨어에서 처리
	},
}

func (s *Server) handleDockerLogStream(c *gin.Context) {
	if s.dockerSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker service not available"})
		return
	}

	name := c.Param("name")
	if !s.dockerSvc.IsManaged(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ctx := c.Request.Context()
	logReader, err := s.dockerSvc.GetLogStream(ctx, name)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = logReader.Close() }()

	header := make([]byte, 8)
	buf := make([]byte, 4096)
	const maxLogChunkSize = 1 << 20

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, err := io.ReadFull(logReader, header)
		if err != nil {
			return
		}

		size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])
		if size <= 0 {
			continue
		}
		if size > maxLogChunkSize {
			_, _ = io.CopyN(io.Discard, logReader, int64(size))
			continue
		}
		if size > cap(buf) {
			buf = make([]byte, size)
		}
		payload := buf[:size]

		n, err := io.ReadFull(logReader, payload)
		if err != nil {
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, payload[:n]); err != nil {
			return
		}
	}
}

// ===== System Logs Handlers =====

// handleLogFiles godoc
// @Summary      List log files
// @Description  Get available system log files
// @Tags         logs
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  LogFilesResponse
// @Router       /logs/files [get]
func (s *Server) handleLogFiles(c *gin.Context) {
	files := logs.ListLogFiles()
	c.JSON(http.StatusOK, gin.H{"status": "ok", "files": files})
}

// handleSystemLogs godoc
// @Summary      Get system logs
// @Description  Read last N lines from a system log file
// @Tags         logs
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        file   query     string  false  "Log file key"  default(combined)
// @Param        lines  query     int     false  "Number of lines to fetch"  default(200)  maximum(1000)
// @Success      200    {object}  SystemLogsResponse
// @Failure      400    {object}  ErrorResponse  "Invalid log file"
// @Router       /logs [get]
func (s *Server) handleSystemLogs(c *gin.Context) {
	fileKey := c.Query("file")
	if fileKey == "" {
		fileKey = "combined"
	}

	logPath, ok := logs.GetLogFilePath(fileKey)
	if !ok {
		c.JSON(400, gin.H{"error": "Invalid log file", "allowed_keys": logs.GetLogFileKeys()})
		return
	}

	linesStr := c.Query("lines")
	lines := 200
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil {
			lines = n
		}
	}
	if lines > logs.MaxLogLines {
		lines = logs.MaxLogLines
	}
	if lines < 1 {
		lines = 1
	}

	logLines, err := logs.TailFile(logPath, lines)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(200, gin.H{"status": "ok", "file": fileKey, "lines": []string{}, "error": "Log file not found"})
			return
		}
		s.logger.Error("Failed to read log file", slog.String("file", logPath), slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to read log file"})
		return
	}

	c.JSON(200, gin.H{"status": "ok", "file": fileKey, "lines": logLines, "count": len(logLines)})
}

// ===== Traces Handlers =====

// handleTracesHealth godoc
// @Summary      Jaeger health check
// @Description  Check if Jaeger Query API is accessible
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  TracesHealthResponse
// @Failure      503  {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces/health [get]
func (s *Server) handleTracesHealth(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"status": "unavailable", "available": false})
		return
	}
	available := s.tracesClient.Available(c.Request.Context())
	if !available {
		c.JSON(503, gin.H{"status": "unavailable", "available": false})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "available": true})
}

// handleTracesServices godoc
// @Summary      List traced services
// @Description  Get all services that have emitted traces
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  ServicesResponse
// @Failure      503  {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces/services [get]
func (s *Server) handleTracesServices(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"error": "Jaeger service unavailable"})
		return
	}
	services, err := s.tracesClient.GetServices(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch from Jaeger", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "services": services})
}

// handleTracesOperations godoc
// @Summary      List operations
// @Description  Get all operations for a specific service
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        service  path      string  true  "Service name"
// @Success      200      {object}  OperationsResponse
// @Failure      400      {object}  ErrorResponse  "Service is required"
// @Failure      503      {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces/operations/{service} [get]
func (s *Server) handleTracesOperations(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"error": "Jaeger service unavailable"})
		return
	}
	service := c.Param("service")
	if service == "" {
		c.JSON(400, gin.H{"error": "Invalid parameter: service is required"})
		return
	}
	operations, err := s.tracesClient.GetOperations(c.Request.Context(), service)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch from Jaeger", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "service": service, "operations": operations})
}

// handleTracesSearch godoc
// @Summary      Search traces
// @Description  Search for traces matching given criteria
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        service      query     string  true   "Service name"
// @Param        operation    query     string  false  "Operation name"
// @Param        lookback     query     string  false  "Lookback duration"  default(1h)
// @Param        minDuration  query     string  false  "Minimum duration (e.g., 100ms)"
// @Param        maxDuration  query     string  false  "Maximum duration (e.g., 1s)"
// @Param        limit        query     int     false  "Maximum traces to return"  default(20)  maximum(100)
// @Success      200          {object}  TracesSearchResponse
// @Failure      400          {object}  ErrorResponse  "Service is required"
// @Failure      503          {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces [get]
func (s *Server) handleTracesSearch(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"error": "Jaeger service unavailable"})
		return
	}

	service := c.Query("service")
	if service == "" {
		c.JSON(400, gin.H{"error": "Invalid parameter: service is required"})
		return
	}

	params := traces.TraceSearchParams{
		Service:     service,
		Operation:   c.Query("operation"),
		Lookback:    c.DefaultQuery("lookback", "1h"),
		MinDuration: c.Query("minDuration"),
		MaxDuration: c.Query("maxDuration"),
	}

	// Parse tag query parameters (Jaeger v2 format: tag=key:value)
	if tagValues, exists := c.GetQueryArray("tag"); exists && len(tagValues) > 0 {
		params.Tags = make(map[string]string)
		for _, tv := range tagValues {
			if idx := strings.Index(tv, ":"); idx > 0 {
				params.Tags[tv[:idx]] = tv[idx+1:]
			}
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = limit
		}
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	result, err := s.tracesClient.SearchTraces(c.Request.Context(), params)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch from Jaeger", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "traces": result.Traces, "total": result.Total, "limit": result.Limit})
}

// handleTraceDetail godoc
// @Summary      Get trace detail
// @Description  Get full detail of a specific trace including all spans
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        traceId  path      string  true  "Trace ID"
// @Success      200      {object}  TraceDetailResponse
// @Failure      400      {object}  ErrorResponse  "Trace ID required"
// @Failure      404      {object}  ErrorResponse  "Trace not found"
// @Failure      503      {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces/{traceId} [get]
func (s *Server) handleTraceDetail(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"error": "Jaeger service unavailable"})
		return
	}

	traceID := c.Param("traceId")
	if traceID == "" {
		c.JSON(400, gin.H{"error": "Invalid parameter: traceId is required"})
		return
	}

	detail, err := s.tracesClient.GetTrace(c.Request.Context(), traceID)
	if err != nil {
		if errors.Is(err, traces.ErrTraceNotFound) {
			c.JSON(404, gin.H{"error": "Trace not found", "traceId": traceID})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to fetch from Jaeger", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "traceId": detail.TraceID, "spans": detail.Spans, "processes": detail.Processes})
}

// handleTracesDependencies godoc
// @Summary      Get service dependencies
// @Description  Get service dependency graph from traced calls
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        lookback  query     string  false  "Lookback duration"  default(24h)
// @Success      200       {object}  DependenciesResponse
// @Failure      503       {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces/dependencies [get]
func (s *Server) handleTracesDependencies(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"error": "Jaeger service unavailable"})
		return
	}

	lookback := c.DefaultQuery("lookback", "24h")
	result, err := s.tracesClient.GetDependencies(c.Request.Context(), lookback)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch dependencies from Jaeger", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "dependencies": result.Dependencies, "count": len(result.Dependencies)})
}

// handleTracesMetrics godoc
// @Summary      Get service metrics (SPM)
// @Description  Get Service Performance Monitoring metrics (RED: Rate, Error, Duration)
// @Tags         traces
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        service           path      string  true   "Service name"
// @Param        quantile          query     string  false  "Latency quantile"  default(0.95)
// @Param        spanKind          query     string  false  "Span kind filter"  default(server)
// @Param        lookback          query     string  false  "Lookback duration"  default(1h)
// @Param        step              query     string  false  "Time series step"
// @Param        ratePer           query     string  false  "Rate calculation period"  default(minute)
// @Param        groupByOperation  query     bool    false  "Group metrics by operation"
// @Success      200               {object}  MetricsResponse
// @Failure      400               {object}  ErrorResponse  "Service is required"
// @Failure      503               {object}  ErrorResponse  "Jaeger unavailable"
// @Router       /traces/metrics/{service} [get]
func (s *Server) handleTracesMetrics(c *gin.Context) {
	if s.tracesClient == nil {
		c.JSON(503, gin.H{"error": "Jaeger service unavailable"})
		return
	}

	service := c.Param("service")
	if service == "" {
		c.JSON(400, gin.H{"error": "Invalid parameter: service is required"})
		return
	}

	params := traces.MetricsParams{
		Service:   service,
		Quantile:  c.DefaultQuery("quantile", "0.95"),
		SpanKind:  c.DefaultQuery("spanKind", "server"),
		Lookback:  c.DefaultQuery("lookback", "1h"),
		Step:      c.Query("step"),
		RatePer:   c.DefaultQuery("ratePer", "minute"),
		GroupByOp: c.Query("groupByOperation") == "true",
	}

	result, err := s.tracesClient.GetServiceMetrics(c.Request.Context(), params)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch metrics from Jaeger", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"status":     "ok",
		"service":    result.Service,
		"metrics":    result.Metrics,
		"operations": result.Operations,
		"latencies":  result.Latencies,
		"calls":      result.Calls,
		"errors":     result.Errors,
	})
}

// ===== Status Handlers =====

// handleAggregatedStatus godoc
// @Summary      통합 시스템 상태
// @Description  모든 서비스(Admin, Holo Bot, Game Bots, LLM Server)의 상태를 집계하여 반환
// @Tags         status
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  status.AggregatedStatus
// @Router       /status [get]
func (s *Server) handleAggregatedStatus(c *gin.Context) {
	if s.statusCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Status collector not initialized"})
		return
	}

	result := s.statusCollector.GetAggregatedStatus(c.Request.Context())
	c.JSON(http.StatusOK, result)
}

// handleSystemStatsStream: WebSocket을 통해 시스템 리소스 사용량을 실시간 스트리밍합니다.
// 2초마다 CPU/메모리/고루틴 통계를 전송합니다.
func (s *Server) handleSystemStatsStream(c *gin.Context) {
	if s.statusCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Status collector not initialized"})
		return
	}

	// WebSocket 업그레이드
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ctx := c.Request.Context()
	statsChan := make(chan *status.SystemStats, 1)

	// 별도 goroutine에서 stats 스트리밍
	go s.statusCollector.StreamSystemStats(ctx, statsChan)

	for {
		select {
		case <-ctx.Done():
			return
		case stats, ok := <-statsChan:
			if !ok {
				return
			}
			if err := conn.WriteJSON(stats); err != nil {
				return
			}
		}
	}
}
