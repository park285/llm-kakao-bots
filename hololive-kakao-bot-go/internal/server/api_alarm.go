package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
)

// GetAlarms: 모든 알람을 JSON으로 반환합니다.
func (h *APIHandler) GetAlarms(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	// 모든 알림 레지스트리 키 조회
	alarmKeys, err := h.alarm.GetAllAlarmKeys(ctx)
	if err != nil {
		h.logger.Error("Failed to get alarm keys", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get alarms"})
		return
	}

	c.JSON(200, gin.H{
		"status": "ok",
		"alarms": alarmKeys,
	})
}

// DeleteAlarm: 특정 알람을 삭제합니다.
func (h *APIHandler) DeleteAlarm(c *gin.Context) {
	var req struct {
		RoomID    string `json:"roomId" binding:"required"`
		UserID    string `json:"userId" binding:"required"`
		ChannelID string `json:"channelId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	removed, err := h.alarm.RemoveAlarm(ctx, req.RoomID, req.UserID, req.ChannelID)
	if err != nil {
		h.logger.Error("Failed to delete alarm", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to delete alarm"})
		return
	}

	h.activity.Log("alarm_delete", fmt.Sprintf("Alarm deleted: %s / %s", req.UserID, req.ChannelID), map[string]any{
		"room_id":    req.RoomID,
		"user_id":    req.UserID,
		"channel_id": req.ChannelID,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"removed": removed,
	})
}
