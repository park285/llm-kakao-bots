package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
)

// SetRoomName: 방 ID에 대한 표시 이름을 설정합니다.
func (h *APIHandler) SetRoomName(c *gin.Context) {
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

	// AlarmService를 통해 Valkey에 저장함
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

// SetUserName: 사용자 ID에 대한 표시 이름을 설정합니다.
func (h *APIHandler) SetUserName(c *gin.Context) {
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

	// AlarmService를 통해 Valkey에 저장함
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

// GetLogs: 활동 로그를 반환합니다.
func (h *APIHandler) GetLogs(c *gin.Context) {
	logs, err := h.activity.GetRecentLogs(100)
	if err != nil {
		h.logger.Error("Failed to get logs", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get logs"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "logs": logs})
}

// GetSettings: 현재 설정을 반환합니다.
func (h *APIHandler) GetSettings(c *gin.Context) {
	s := h.settings.Get()
	c.JSON(200, gin.H{"status": "ok", "settings": s})
}

// UpdateSettings: 설정을 업데이트합니다.
func (h *APIHandler) UpdateSettings(c *gin.Context) {
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
