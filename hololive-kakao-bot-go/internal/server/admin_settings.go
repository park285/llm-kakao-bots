package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
)

// SetRoomName sets a display name for a room ID
func (h *AdminHandler) SetRoomName(c *gin.Context) {
	var req struct {
		RoomID   string `json:"roomId" binding:"required"`
		RoomName string `json:"roomName" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	// Save to Valkey via AlarmService
	if err := h.alarm.SetRoomName(ctx, req.RoomID, req.RoomName); err != nil {
		h.logger.Error("Failed to set room name", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to set room name"})
		return
	}

	h.logger.Info("Room name set",
		slog.String("room_id", req.RoomID),
		slog.String("room_name", req.RoomName),
	)

	h.activity.Log("name_update", fmt.Sprintf("Room name set: %s -> %s", req.RoomID, req.RoomName), map[string]any{
		"room_id":   req.RoomID,
		"room_name": req.RoomName,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Room name set successfully",
	})
}

// SetUserName sets a display name for a user ID
func (h *AdminHandler) SetUserName(c *gin.Context) {
	var req struct {
		UserID   string `json:"userId" binding:"required"`
		UserName string `json:"userName" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	// Save to Valkey via AlarmService
	if err := h.alarm.SetUserName(ctx, req.UserID, req.UserName); err != nil {
		h.logger.Error("Failed to set user name", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to set user name"})
		return
	}

	h.logger.Info("User name set",
		slog.String("user_id", req.UserID),
		slog.String("user_name", req.UserName),
	)

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "User name set successfully",
	})
}

// GetLogs returns activity logs
func (h *AdminHandler) GetLogs(c *gin.Context) {
	logs, err := h.activity.GetRecentLogs(100)
	if err != nil {
		h.logger.Error("Failed to get logs", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get logs"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "logs": logs})
}

// GetSettings returns current settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	s := h.settings.Get()
	c.JSON(200, gin.H{"status": "ok", "settings": s})
}

// UpdateSettings updates settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req settings.Settings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := h.settings.Update(req); err != nil {
		h.logger.Error("Failed to update settings", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to update settings"})
		return
	}

	h.activity.Log("settings_update", "Settings updated", nil)

	c.JSON(200, gin.H{"status": "ok", "message": "Settings updated"})
}
