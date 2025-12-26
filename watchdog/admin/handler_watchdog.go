package admin

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"llm-watchdog/watchdog"

	"github.com/gin-gonic/gin"
)

// registerWatchdogHandlers watchdog 상태 및 설정 관련 핸들러 등록
func registerWatchdogHandlers(api *gin.RouterGroup, w *watchdog.Watchdog, logger *slog.Logger) {
	api.GET("/watchdog/status", handleWatchdogStatus(w))
	api.PUT("/watchdog/enabled", handleWatchdogEnabled(w))
	api.POST("/watchdog/check-now", handleWatchdogCheckNow(w, logger))
	api.POST("/watchdog/reload-config", handleWatchdogReloadConfig(w, logger))
}

// handleWatchdogStatus watchdog 상태 조회
func handleWatchdogStatus(w *watchdog.Watchdog) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := w.GetConfig()
		startedAt := w.GetStartedAt()
		uptime := int64(time.Since(startedAt).Seconds())
		c.JSON(http.StatusOK, gin.H{
			"startedAt":         startedAt,
			"uptimeSec":         uptime,
			"enabled":           cfg.Enabled,
			"configSource":      w.GetConfigSource(),
			"configPath":        w.GetConfigPath(),
			"containers":        cfg.Containers,
			"intervalSec":       cfg.IntervalSeconds,
			"maxFailures":       cfg.MaxFailures,
			"retryChecks":       cfg.RetryChecks,
			"retryIntervalSec":  cfg.RetryIntervalSeconds,
			"cooldownSec":       cfg.CooldownSeconds,
			"restartTimeoutSec": cfg.RestartTimeoutSec,
			"dockerSocket":      cfg.DockerSocket,
			"useEvents":         cfg.UseEvents,
			"statusReportSec":   cfg.StatusReportSeconds,
			"verbose":           cfg.VerboseLogging,
		})
	}
}

// handleWatchdogEnabled watchdog 활성화/비활성화 토글
func handleWatchdogEnabled(w *watchdog.Watchdog) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Enabled bool   `json:"enabled"`
			Reason  string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAPIError(c, http.StatusBadRequest, "invalid_json", "Invalid JSON")
			return
		}

		adminEmail := getAdminEmail(c)
		req.Reason = strings.TrimSpace(req.Reason)
		if req.Reason == "" {
			req.Reason = "admin_api_toggle"
		}

		w.SetEnabled(req.Enabled, adminEmail, req.Reason)
		c.JSON(http.StatusOK, gin.H{
			"enabled": req.Enabled,
			"status":  "ok",
		})
	}
}

// handleWatchdogCheckNow 즉시 헬스체크 트리거
func handleWatchdogCheckNow(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info("admin_action", "action", "watchdog_check_now", "admin_email", getAdminEmail(c))
		w.TriggerHealthCheck()
		c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
	}
}

// handleWatchdogReloadConfig 설정 리로드
func handleWatchdogReloadConfig(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info("admin_action", "action", "watchdog_reload_config", "admin_email", getAdminEmail(c))
		result, err := w.ReloadConfigFromFile(c.Request.Context())
		if err != nil {
			writeAPIError(c, http.StatusBadRequest, "reload_failed", err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
