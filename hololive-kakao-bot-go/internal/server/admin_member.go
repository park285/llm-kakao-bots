package server

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// aliasRequest represents a unified alias request
type aliasRequest struct {
	Type  string `json:"type" binding:"required,oneof=ko ja"`
	Alias string `json:"alias" binding:"required,min=1"`
}

// handleAliasOperation processes alias add/remove operations with shared logic
func (h *AdminHandler) handleAliasOperation(
	c *gin.Context,
	repoFunc func(context.Context, int, string, string) error,
	operationName string,
) {
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("Invalid member ID", slog.String("id", c.Param("id")), slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid member ID"})
		return
	}

	var req aliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	if err := repoFunc(ctx, memberID, req.Type, req.Alias); err != nil {
		h.logger.Error("Failed to "+operationName+" alias",
			slog.Int("member_id", memberID),
			slog.String("type", req.Type),
			slog.String("alias", req.Alias),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Failed to " + operationName + " alias"})
		return
	}

	if err := h.memberCache.InvalidateAliasCache(ctx, req.Alias); err != nil {
		h.logger.Warn("Failed to invalidate alias cache", slog.Any("error", err))
	}

	h.logger.Info("Alias "+operationName,
		slog.Int("member_id", memberID),
		slog.String("type", req.Type),
		slog.String("alias", req.Alias),
	)

	h.activity.Log("member_alias_"+operationName, fmt.Sprintf("Member alias %s: %s (ID: %d)", operationName, req.Alias, memberID), map[string]any{
		"member_id": memberID,
		"type":      req.Type,
		"alias":     req.Alias,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Alias " + operationName + " successfully",
	})
}

// AddAlias adds an alias to a member
func (h *AdminHandler) AddAlias(c *gin.Context) {
	h.handleAliasOperation(c, h.repo.AddAlias, "add")
}

// RemoveAlias removes an alias from a member
func (h *AdminHandler) RemoveAlias(c *gin.Context) {
	h.handleAliasOperation(c, h.repo.RemoveAlias, "remove")
}

// SetGraduation 는 졸업 상태를 갱신한다.
//
//nolint:dupl // Similar patterns for different update operations
func (h *AdminHandler) SetGraduation(c *gin.Context) {
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("Invalid member ID", slog.String("id", c.Param("id")), slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid member ID"})
		return
	}

	var req struct {
		IsGraduated bool `json:"isGraduated"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	if err := h.repo.SetGraduation(ctx, memberID, req.IsGraduated); err != nil {
		h.logger.Error("Failed to set graduation status",
			slog.Int("member_id", memberID),
			slog.Bool("is_graduated", req.IsGraduated),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Failed to set graduation status"})
		return
	}

	if err := h.memberCache.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh cache after graduation update", slog.Any("error", err))
	}

	h.logger.Info("Graduation status updated",
		slog.Int("member_id", memberID),
		slog.Bool("is_graduated", req.IsGraduated),
	)

	statusStr := "graduated"
	if !req.IsGraduated {
		statusStr = "active"
	}
	h.activity.Log("member_graduation", fmt.Sprintf("Member status changed to %s (ID: %d)", statusStr, memberID), map[string]any{
		"member_id":    memberID,
		"is_graduated": req.IsGraduated,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Graduation status updated successfully",
	})
}

// UpdateChannelID 는 채널 ID를 갱신한다.
//
//nolint:dupl // Similar patterns for different update operations
func (h *AdminHandler) UpdateChannelID(c *gin.Context) {
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("Invalid member ID", slog.String("id", c.Param("id")), slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid member ID"})
		return
	}

	var req struct {
		ChannelID string `json:"channelId" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	if err := h.repo.UpdateChannelID(ctx, memberID, req.ChannelID); err != nil {
		h.logger.Error("Failed to update channel ID",
			slog.Int("member_id", memberID),
			slog.String("channel_id", req.ChannelID),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Failed to update channel ID"})
		return
	}

	if err := h.memberCache.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh cache after channel ID update", slog.Any("error", err))
	}

	h.logger.Info("Channel ID updated",
		slog.Int("member_id", memberID),
		slog.String("channel_id", req.ChannelID),
	)

	h.activity.Log("member_channel_update", fmt.Sprintf("Member channel ID updated to %s (ID: %d)", req.ChannelID, memberID), map[string]any{
		"member_id":  memberID,
		"channel_id": req.ChannelID,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Channel ID updated successfully",
	})
}

// GetMembers returns all members as JSON
func (h *AdminHandler) GetMembers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	members, err := h.repo.GetAllMembers(ctx)
	if err != nil {
		h.logger.Error("Failed to get members", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get members"})
		return
	}

	c.JSON(200, gin.H{
		"status":  "ok",
		"members": members,
	})
}

// AddMember adds a new member
func (h *AdminHandler) AddMember(c *gin.Context) {
	var req domain.Member
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.repo.CreateMember(ctx, &req); err != nil {
		h.logger.Error("Failed to add member", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to add member"})
		return
	}

	if err := h.memberCache.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh member cache", slog.Any("error", err))
	}

	h.activity.Log("member_add", "Member added: "+req.Name, map[string]any{"name": req.Name})

	c.JSON(200, gin.H{"status": "ok", "message": "Member added successfully"})
}
