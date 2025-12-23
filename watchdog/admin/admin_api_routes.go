package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llm-watchdog/watchdog"

	"github.com/gin-gonic/gin"
)

type restartRequest struct {
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

type stopRequest struct {
	TimeoutSeconds int    `json:"timeoutSeconds"`
	Reason         string `json:"reason"`
}

type startRequest struct {
	Reason string `json:"reason"`
}

type managedRequest struct {
	Managed *bool  `json:"managed"`
	Reason  string `json:"reason"`
}

func registerAdminAPIRoutes(router *gin.Engine, w *watchdog.Watchdog, logger *slog.Logger) {
	api := router.Group("/admin/api/v1")
	api.Use(noCacheHeaders)

	api.GET("/watchdog/status", func(c *gin.Context) {
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
			"cooldownSec":       cfg.CooldownSeconds,
			"restartTimeoutSec": cfg.RestartTimeoutSec,
			"dockerSocket":      cfg.DockerSocket,
			"useEvents":         cfg.UseEvents,
			"statusReportSec":   cfg.StatusReportSeconds,
			"verbose":           cfg.VerboseLogging,
		})
	})

	api.PUT("/watchdog/enabled", func(c *gin.Context) {
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
	})

	api.GET("/docker/containers", func(c *gin.Context) {
		containers, err := w.ListDockerContainers(c.Request.Context())
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"generatedAt": time.Now(),
			"containers":  containers,
		})
	})

	api.POST("/watchdog/check-now", func(c *gin.Context) {
		logger.Info("admin_action", "action", "watchdog_check_now", "admin_email", getAdminEmail(c))
		w.TriggerHealthCheck()
		c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
	})

	api.POST("/watchdog/reload-config", func(c *gin.Context) {
		logger.Info("admin_action", "action", "watchdog_reload_config", "admin_email", getAdminEmail(c))
		result, err := w.ReloadConfigFromFile(c.Request.Context())
		if err != nil {
			writeAPIError(c, http.StatusBadRequest, "reload_failed", err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	api.GET("/targets", func(c *gin.Context) {
		statuses, err := w.ListTargetsStatus(c.Request.Context())
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"generatedAt": time.Now(),
			"targets":     statuses,
		})
	})

	api.GET("/targets/:name", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		status, err := w.GetTargetStatus(c.Request.Context(), name)
		if err != nil {
			if errors.Is(err, watchdog.ErrContainerNotManaged) {
				writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
				return
			}
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, status)
	})

	api.PUT("/targets/:name/managed", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		if name == "" {
			writeAPIError(c, http.StatusBadRequest, "invalid_name", "컨테이너 이름이 비어있습니다.")
			return
		}

		var req managedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAPIError(c, http.StatusBadRequest, "invalid_json", "요청 JSON이 유효하지 않습니다.")
			return
		}
		if req.Managed == nil {
			writeAPIError(c, http.StatusBadRequest, "missing_field", "managed 필드가 필요합니다.")
			return
		}

		req.Reason = strings.TrimSpace(req.Reason)
		if req.Reason == "" {
			if *req.Managed {
				req.Reason = "managed_enable"
			} else {
				req.Reason = "managed_disable"
			}
		}

		adminEmail := getAdminEmail(c)
		logger.Info("admin_action",
			"action", "target_managed",
			"container", name,
			"managed", *req.Managed,
			"admin_email", adminEmail,
			"reason", req.Reason,
		)

		reloadResult, err := w.SetTargetManaged(c.Request.Context(), name, *req.Managed, adminEmail, req.Reason)
		if err != nil {
			if errors.Is(err, watchdog.ErrConfigPathNotSet) {
				writeAPIError(c, http.StatusBadRequest, "config_path_not_set", err.Error())
				return
			}
			writeAPIError(c, http.StatusInternalServerError, "managed_update_failed", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"container": name,
			"managed":   *req.Managed,
			"reload":    reloadResult,
		})
	})

	api.POST("/targets/:name/restart", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		var req restartRequest
		_ = c.ShouldBindJSON(&req)
		req.Reason = strings.TrimSpace(req.Reason)
		if req.Reason == "" {
			req.Reason = "manual_restart"
		}

		adminEmail := getAdminEmail(c)
		logger.Info("admin_action", "action", "restart", "container", name, "admin_email", adminEmail, "force", req.Force, "reason", req.Reason)
		ok, msg, err := w.RequestRestart(c.Request.Context(), name, watchdog.RestartByManual, req.Reason, adminEmail, req.Force)
		if err != nil {
			if errors.Is(err, watchdog.ErrContainerNotManaged) {
				writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
				return
			}
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"accepted": ok, "status": msg})
	})

	api.POST("/targets/:name/stop", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		var req stopRequest
		_ = c.ShouldBindJSON(&req)
		req.Reason = strings.TrimSpace(req.Reason)
		adminEmail := getAdminEmail(c)
		logger.Info("admin_action", "action", "stop", "container", name, "admin_email", adminEmail, "timeout_seconds", req.TimeoutSeconds, "reason", req.Reason)

		if err := w.StopContainer(c.Request.Context(), name, req.TimeoutSeconds, adminEmail, req.Reason); err != nil {
			if errors.Is(err, watchdog.ErrContainerNotManaged) {
				writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
				return
			}
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.POST("/targets/:name/start", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		var req startRequest
		_ = c.ShouldBindJSON(&req)
		req.Reason = strings.TrimSpace(req.Reason)
		adminEmail := getAdminEmail(c)
		logger.Info("admin_action", "action", "start", "container", name, "admin_email", adminEmail, "reason", req.Reason)

		if err := w.StartContainer(c.Request.Context(), name, adminEmail, req.Reason); err != nil {
			if errors.Is(err, watchdog.ErrContainerNotManaged) {
				writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
				return
			}
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.POST("/targets/:name/pause", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		logger.Info("admin_action", "action", "pause", "container", name, "admin_email", getAdminEmail(c))
		if err := w.PauseMonitoring(name); err != nil {
			if errors.Is(err, watchdog.ErrContainerNotManaged) {
				writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
				return
			}
			writeAPIError(c, http.StatusInternalServerError, "pause_failed", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.POST("/targets/:name/resume", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		logger.Info("admin_action", "action", "resume", "container", name, "admin_email", getAdminEmail(c))
		if err := w.ResumeMonitoring(name); err != nil {
			if errors.Is(err, watchdog.ErrContainerNotManaged) {
				writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
				return
			}
			writeAPIError(c, http.StatusInternalServerError, "resume_failed", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.GET("/events", func(c *gin.Context) {
		limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
		events := w.SnapshotEvents(limit)
		c.JSON(http.StatusOK, gin.H{
			"events": events,
		})
	})

	api.GET("/targets/:name/logs", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		_, ok := w.GetState(name)
		if !ok {
			writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
			return
		}

		tail, _ := strconv.Atoi(strings.TrimSpace(c.Query("tail")))
		if tail <= 0 {
			tail = 200
		}
		if tail > 2000 {
			tail = 2000
		}

		timestamps := true
		if strings.TrimSpace(c.Query("timestamps")) == "false" {
			timestamps = false
		}

		reader, isTTY, err := w.OpenDockerLogs(c.Request.Context(), name, tail, false, timestamps)
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		defer reader.Close()

		reader = watchdog.WrapDockerLogsReader(reader, isTTY)
		body, err := ioReadAllLimit(reader, 5*1024*1024)
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.Data(http.StatusOK, "text/plain; charset=utf-8", body)
	})

	api.GET("/targets/:name/logs/stream", func(c *gin.Context) {
		name := watchdog.CanonicalContainerName(c.Param("name"))
		_, ok := w.GetState(name)
		if !ok {
			writeAPIError(c, http.StatusNotFound, "not_found", "관리 대상 컨테이너가 아닙니다.")
			return
		}

		tail, _ := strconv.Atoi(strings.TrimSpace(c.Query("tail")))
		if tail <= 0 {
			tail = 200
		}
		if tail > 2000 {
			tail = 2000
		}

		reader, isTTY, err := w.OpenDockerLogs(c.Request.Context(), name, tail, true, true)
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		defer reader.Close()

		reader = watchdog.WrapDockerLogsReader(reader, isTTY)

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)
		c.Writer.Flush()

		adminEmail := getAdminEmail(c)
		logger.Info("admin_action", "action", "logs_stream", "container", name, "admin_email", adminEmail, "tail", tail)

		_ = watchdog.StreamLines(c.Request.Context(), reader, func(line string) error {
			payload, _ := json.Marshal(gin.H{
				"container": name,
				"log":       line,
				"at":        time.Now(),
			})
			_, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
			if err != nil {
				return err
			}
			c.Writer.Flush()
			return nil
		})
	})
}
