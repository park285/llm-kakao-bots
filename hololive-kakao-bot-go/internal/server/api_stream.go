package server

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

const channelStatsCacheKey = "admin:channel_stats"
const channelStatsCacheTTL = 10 * time.Minute

// GetLiveStreams: 현재 라이브 방송 중인 스트림 목록을 반환합니다.
func (h *APIHandler) GetLiveStreams(c *gin.Context) {
	ctx := c.Request.Context()
	streams, err := h.holodex.GetLiveStreams(ctx)
	if err != nil {
		h.logger.Error("Failed to get live streams", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get live streams"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "streams": streams})
}

// GetUpcomingStreams: 예정된 스트림 목록을 반환합니다.
func (h *APIHandler) GetUpcomingStreams(c *gin.Context) {
	ctx := c.Request.Context()
	streams, err := h.holodex.GetUpcomingStreams(ctx, 24)
	if err != nil {
		h.logger.Error("Failed to get upcoming streams", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get upcoming streams"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "streams": streams})
}

// GetChannelStats: 채널 통계를 반환합니다. (10분간 캐시됨)
func (h *APIHandler) GetChannelStats(c *gin.Context) {
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

// GetChannel: channelId로 특정 채널의 상세 정보(프로필 이미지 포함)를 반환합니다.
// 배치 조회: channelIds 파라미터로 여러 채널을 한 번에 조회할 수 있습니다.
// - 단일 조회: /api/holo/channels?channelId=UC...
// - 배치 조회: /api/holo/channels?channelIds=UC1,UC2,UC3...
// NOTE: DB에서 직접 조회하여 Holodex API 호출 없이 응답합니다.
func (h *APIHandler) GetChannel(c *gin.Context) {
	ctx := c.Request.Context()

	// 배치 조회 지원: channelIds 파라미터 확인
	channelIDs := c.Query("channelIds")
	if channelIDs != "" {
		ids := splitChannelIDs(channelIDs)
		if len(ids) == 0 {
			c.JSON(400, gin.H{"error": "channelIds parameter is empty or invalid"})
			return
		}

		// 최대 100개로 제한
		if len(ids) > 100 {
			ids = ids[:100]
		}

		// DB에서 직접 조회 (Holodex API 호출 없음)
		channelsMap, err := h.repo.GetMembersWithPhoto(ctx, ids)
		if err != nil {
			h.logger.Error("Failed to get channels from DB", slog.Any("error", err), slog.Int("count", len(ids)))
			c.JSON(500, gin.H{"error": "Failed to get channels"})
			return
		}

		// Map을 슬라이스로 변환
		channels := make([]*ChannelResponse, 0, len(channelsMap))
		for _, member := range channelsMap {
			channels = append(channels, memberToChannelResponse(member))
		}

		c.JSON(200, gin.H{"status": "ok", "channels": channels})
		return
	}

	// 단일 조회 (레거시 호환성)
	channelID := c.Query("channelId")
	if channelID == "" {
		c.JSON(400, gin.H{"error": "channelId or channelIds parameter required"})
		return
	}

	// DB에서 직접 조회 (Holodex API 호출 없음)
	member, err := h.repo.GetMemberWithPhotoByChannelID(ctx, channelID)
	if err != nil {
		h.logger.Error("Failed to get channel from DB", slog.String("channelId", channelID), slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get channel"})
		return
	}

	if member == nil {
		c.JSON(404, gin.H{"error": "Channel not found"})
		return
	}

	c.JSON(200, gin.H{"status": "ok", "channel": memberToChannelResponse(member)})
}

// ChannelResponse: 채널 API 응답 구조체 (기존 Holodex 호환 형식)
type ChannelResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Photo *string `json:"photo,omitempty"`
}

// memberToChannelResponse: domain.Member를 API 응답 형식으로 변환
func memberToChannelResponse(m *domain.Member) *ChannelResponse {
	if m == nil {
		return nil
	}
	resp := &ChannelResponse{
		ID:   m.ChannelID,
		Name: m.Name,
	}
	if m.Photo != "" {
		resp.Photo = &m.Photo
	}
	return resp
}

// splitChannelIDs: 쉼표로 구분된 채널 ID 문자열을 슬라이스로 분리합니다.
func splitChannelIDs(ids string) []string {
	parts := make([]string, 0)
	for _, id := range splitByComma(ids) {
		trimmed := trimSpace(id)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// SearchChannels: 이름으로 채널을 검색합니다.
func (h *APIHandler) SearchChannels(c *gin.Context) {
	ctx := c.Request.Context()
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "q parameter required"})
		return
	}

	channels, err := h.holodex.SearchChannels(ctx, query)
	if err != nil {
		h.logger.Error("Failed to search channels", slog.String("query", query), slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to search channels"})
		return
	}

	c.JSON(200, gin.H{"status": "ok", "channels": channels})
}
