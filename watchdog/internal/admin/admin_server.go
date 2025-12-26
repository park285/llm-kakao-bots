package admin

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	watchdog "llm-watchdog/internal/core"
)

//go:embed dist/*
var uiDist embed.FS

// RunServer 는 동작을 수행한다.
func RunServer(ctx context.Context, adminCfg Config, w *watchdog.Watchdog, logger *slog.Logger) error {
	if err := adminCfg.ValidateForEnable(); err != nil {
		return err
	}

	allowlist, err := newIPAllowlist(adminCfg.AllowedIPs)
	if err != nil {
		return fmt.Errorf("admin allowed ips invalid: %w", err)
	}
	if allowlist == nil {
		return fmt.Errorf("WATCHDOG_ADMIN_ALLOWED_IPS is required")
	}

	router := setupRouter(adminCfg, w, logger, allowlist)
	setupStaticAssets(router, logger)

	handler := http.Handler(router)
	if adminCfg.UseH2C {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

	server := buildHTTPServer(adminCfg, handler)
	return runServerLifecycle(ctx, server, adminCfg, logger)
}

func setupRouter(adminCfg Config, w *watchdog.Watchdog, logger *slog.Logger, allowlist *ipAllowlist) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		noCacheHeaders(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.Use(allowlist.middleware())

	var apiMiddlewares []gin.HandlerFunc
	hasCFAccess := strings.TrimSpace(adminCfg.CFAccessTeamDomain) != "" || strings.TrimSpace(adminCfg.CFAccessAUD) != ""
	if hasCFAccess {
		verifier, verifierErr := newCFAccessVerifier(adminCfg, logger)
		if verifierErr == nil {
			apiMiddlewares = append(apiMiddlewares, verifier.middleware())
			logger.Info("cf_access_enabled")
		} else {
			logger.Error("cf_access_verifier_init_failed", "err", verifierErr)
		}
	} else {
		logger.Debug("cf_access_disabled")
	}

	registerAdminAPIRoutes(router, w, logger, apiMiddlewares...)
	return router
}

func setupStaticAssets(router *gin.Engine, logger *slog.Logger) {
	distFS, err := fs.Sub(uiDist, "dist")
	if err != nil {
		logger.Error("dist_fs_sub_failed", "err", err)
		return
	}

	router.GET("/", func(c *gin.Context) {
		c.FileFromFS("/", http.FS(distFS))
	})

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/admin/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if _, err := distFS.Open(cleanPath); err == nil {
			c.FileFromFS(cleanPath, http.FS(distFS))
			return
		}

		c.FileFromFS("/", http.FS(distFS))
	})
}

func buildHTTPServer(adminCfg Config, handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:              adminCfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: adminCfg.ReadHeaderTimeout,
		IdleTimeout:       adminCfg.IdleTimeout,
	}
	if !adminCfg.UseH2C {
		server.ReadTimeout = adminCfg.ReadTimeout
		server.WriteTimeout = adminCfg.WriteTimeout
	}
	return server
}

func runServerLifecycle(ctx context.Context, server *http.Server, adminCfg Config, logger *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("admin_server_start", "addr", adminCfg.Addr, "h2c", adminCfg.UseH2C)
		err := server.ListenAndServe()
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), adminCfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("admin_server_shutdown_failed", "err", err)
		} else {
			logger.Info("admin_server_shutdown_ok")
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("admin server failed: %w", err)
		}
		return nil
	}
}
