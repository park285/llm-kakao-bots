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

	"llm-watchdog/watchdog"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// /health 는 인증 없이 제공 (k8s/docker 헬스체크)
	router.GET("/health", func(c *gin.Context) {
		noCacheHeaders(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.Use(allowlist.middleware())

	var apiMiddlewares []gin.HandlerFunc
	hasCFAccess := strings.TrimSpace(adminCfg.CFAccessTeamDomain) != "" || strings.TrimSpace(adminCfg.CFAccessAUD) != ""
	if hasCFAccess {
		verifier, err := newCFAccessVerifier(adminCfg, logger)
		if err != nil {
			return fmt.Errorf("cf access verifier init failed: %w", err)
		}
		apiMiddlewares = append(apiMiddlewares, verifier.middleware())
		logger.Info("cf_access_enabled")
	} else {
		logger.Info("cf_access_disabled")
	}

	registerAdminAPIRoutes(router, w, logger, apiMiddlewares...)

	// Serve Static Assets (SPA)
	distFS, err := fs.Sub(uiDist, "dist")
	if err != nil {
		logger.Error("dist_fs_sub_failed", "err", err)
	} else {
		// 루트 요청은 디렉터리로 처리하여 index.html 리다이렉트 루프를 피함
		router.GET("/", func(c *gin.Context) {
			c.FileFromFS("/", http.FS(distFS))
		})

		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// If path starts with /admin/api, but not handled, return 404 JSON
			if strings.HasPrefix(path, "/admin/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
				return
			}

			// Try to serve static file
			cleanPath := strings.TrimPrefix(path, "/")
			if _, err := distFS.Open(cleanPath); err == nil {
				c.FileFromFS(cleanPath, http.FS(distFS))
				return
			}

			// React Router fallback: 디렉터리로 처리해 index.html 리다이렉트 루프를 피함
			c.FileFromFS("/", http.FS(distFS))
		})
	}

	handler := http.Handler(router)
	if adminCfg.UseH2C {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

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
