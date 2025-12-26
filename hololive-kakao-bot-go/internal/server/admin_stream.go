package server

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

const channelStatsCacheKey = "admin:channel_stats"
const channelStatsCacheTTL = 10 * time.Minute

// GetLiveStreams returns currently live streams
func (h *AdminHandler) GetLiveStreams(c *gin.Context) {
	ctx := c.Request.Context()
	streams, err := h.holodex.GetLiveStreams(ctx)
	if err != nil {
		h.logger.Error("Failed to get live streams", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get live streams"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "streams": streams})
}

// GetUpcomingStreams returns upcoming streams
func (h *AdminHandler) GetUpcomingStreams(c *gin.Context) {
	ctx := c.Request.Context()
	streams, err := h.holodex.GetUpcomingStreams(ctx, 24)
	if err != nil {
		h.logger.Error("Failed to get upcoming streams", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get upcoming streams"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "streams": streams})
}

// GetChannelStats returns channel statistics (cached for 10 minutes)
func (h *AdminHandler) GetChannelStats(c *gin.Context) {
	ctx := c.Request.Context()
	if h.youtube == nil {
		c.JSON(503, gin.H{"error": "YouTube service not available"})
		return
	}

	// 캐시 확인
	if h.valkeyCache != nil {
		var cachedStats map[string]*youtube.ChannelStats
		if err := h.valkeyCache.Get(ctx, channelStatsCacheKey, &cachedStats); err == nil && cachedStats != nil {
			h.logger.Debug("Channel stats cache hit", slog.Int("count", len(cachedStats)))
			c.JSON(200, gin.H{"status": "ok", "stats": cachedStats})
			return
		}
	}

	members, err := h.repo.GetAllMembers(ctx)
	if err != nil {
		h.logger.Error("Failed to get members", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get members"})
		return
	}

	var channelIDs []string
	for _, m := range members {
		if m.ChannelID != "" && !m.IsGraduated {
			channelIDs = append(channelIDs, m.ChannelID)
		}
	}

	stats, err := h.youtube.GetChannelStatistics(ctx, channelIDs)
	if err != nil {
		h.logger.Error("Failed to get channel stats", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get channel stats"})
		return
	}

	// 캐시 저장
	if h.valkeyCache != nil && stats != nil {
		if err := h.valkeyCache.Set(ctx, channelStatsCacheKey, stats, channelStatsCacheTTL); err != nil {
			h.logger.Warn("Failed to cache channel stats", slog.Any("error", err))
		}
	}

	c.JSON(200, gin.H{"status": "ok", "stats": stats})
}
