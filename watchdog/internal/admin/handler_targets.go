package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	watchdog "llm-watchdog/internal/core"
)

// registerTargetHandlers 컨테이너 타겟 관련 핸들러 등록
func registerTargetHandlers(api *gin.RouterGroup, w *watchdog.Watchdog, logger *slog.Logger) {
	api.GET("/targets", handleListTargets(w))
	api.GET("/targets/:name", handleGetTarget(w))
	api.PUT("/targets/:name/managed", handleSetManaged(w, logger))
	api.POST("/targets/:name/restart", handleRestart(w, logger))
	api.POST("/targets/:name/stop", handleStop(w, logger))
	api.POST("/targets/:name/start", handleStart(w, logger))
	api.POST("/targets/:name/pause", handlePause(w, logger))
	api.POST("/targets/:name/resume", handleResume(w, logger))
}

// handleListTargets 모든 타겟 상태 조회
func handleListTargets(w *watchdog.Watchdog) gin.HandlerFunc {
	return func(c *gin.Context) {
		statuses, err := w.ListTargetsStatus(c.Request.Context())
		if err != nil {
			writeAPIError(c, http.StatusServiceUnavailable, "docker_error", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"generatedAt": timeNow(),
			"targets":     statuses,
		})
	}
}

// handleGetTarget 특정 타겟 상태 조회
func handleGetTarget(w *watchdog.Watchdog) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// handleSetManaged 관리 대상 설정/해제
func handleSetManaged(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// handleRestart 컨테이너 재시작
func handleRestart(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// handleStop 컨테이너 중지
func handleStop(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// handleStart 컨테이너 시작
func handleStart(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// handlePause 모니터링 일시정지
func handlePause(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// handleResume 모니터링 재개
func handleResume(w *watchdog.Watchdog, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
