package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/watchdog"
)

// WatchdogProxyHandler handles watchdog API proxy requests.
type WatchdogProxyHandler struct {
	client   *watchdog.Client
	logger   *slog.Logger
	activity *activity.Logger
}

// NewWatchdogProxyHandler creates a new watchdog proxy handler.
func NewWatchdogProxyHandler(watchdogURL string, logger *slog.Logger, activity *activity.Logger) *WatchdogProxyHandler {
	return &WatchdogProxyHandler{
		client:   watchdog.NewClient(watchdogURL, logger),
		logger:   logger,
		activity: activity,
	}
}

// GetContainers returns all Docker containers from watchdog.
func (h *WatchdogProxyHandler) GetContainers(c *gin.Context) {
	ctx := c.Request.Context()

	containers, err := h.client.GetContainers(ctx)
	if err != nil {
		h.logger.Error("Failed to get containers from watchdog", slog.Any("error", err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "Failed to connect to watchdog",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"containers": containers.Containers,
	})
}

// GetManagedTargets returns managed container targets from watchdog.
func (h *WatchdogProxyHandler) GetManagedTargets(c *gin.Context) {
	ctx := c.Request.Context()

	targets, err := h.client.GetManagedTargets(ctx)
	if err != nil {
		h.logger.Error("Failed to get targets from watchdog", slog.Any("error", err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "Failed to connect to watchdog",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"targets": targets,
	})
}

// RestartContainer restarts a container via watchdog.
func (h *WatchdogProxyHandler) RestartContainer(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Container name is required"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
		Force  bool   `json:"force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "Admin dashboard restart"
		req.Force = false
	}

	ctx := c.Request.Context()
	result, err := h.client.RestartContainer(ctx, name, req.Reason, req.Force)
	if err != nil {
		h.logger.Error("Failed to restart container",
			slog.String("container", name),
			slog.Any("error", err),
		)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "Failed to restart container",
			"detail": err.Error(),
		})
		return
	}

	h.logger.Info("Container restart initiated via admin",
		slog.String("container", name),
		slog.String("reason", req.Reason),
	)

	h.activity.Log("container_restart", "Container restart: "+name, map[string]any{
		"container": name,
		"reason":    req.Reason,
		"force":     req.Force,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": result.Message,
	})
}

// CheckHealth checks if watchdog is available.
func (h *WatchdogProxyHandler) CheckHealth(c *gin.Context) {
	ctx := c.Request.Context()
	available := h.client.IsAvailable(ctx)

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"available": available,
	})
}
