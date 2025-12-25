package holodex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
	"github.com/kapu/hololive-kakao-bot-go/pkg/errors"
)

const searchChannelsCacheKeyPrefix = "search_channels:"

// ChannelRaw: Holodex API로부터 수신한 채널 정보의 Raw 데이터 구조체
type ChannelRaw struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	EnglishName     *string `json:"english_name,omitempty"`
	Photo           *string `json:"photo,omitempty"`
	Twitter         *string `json:"twitter,omitempty"`
	VideoCount      *int    `json:"video_count,omitempty"`
	SubscriberCount *int    `json:"subscriber_count,omitempty"`
	Org             *string `json:"org,omitempty"`
	Suborg          *string `json:"suborg,omitempty"`
	Group           *string `json:"group,omitempty"`
}

// StreamRaw: Holodex API로부터 수신한 방송(스트림) 정보의 Raw 데이터 구조체
type StreamRaw struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	ChannelID      *string             `json:"channel_id,omitempty"`
	Status         domain.StreamStatus `json:"status"`
	StartScheduled *string             `json:"start_scheduled,omitempty"`
	StartActual    *string             `json:"start_actual,omitempty"`
	Duration       *int                `json:"duration,omitempty"`
	Link           *string             `json:"link,omitempty"`
	Thumbnail      *string             `json:"thumbnail,omitempty"`
	TopicID        *string             `json:"topic_id,omitempty"`
	Channel        *ChannelRaw         `json:"channel,omitempty"`
}

// Service: Holodex External API와 통신하여 채널 및 스트림 정보를 가져오는 클라이언트 서비스
// 캐싱 및 스크래핑 폴백(Fallback) 기능을 포함한다.
type Service struct {
	requester Requester
	cache     *cache.Service
	scraper   *ScraperService
	logger    *slog.Logger
}

// NewHolodexService: 새로운 Holodex API 서비스 인스턴스를 생성한다. (API Key 검증 포함)
func NewHolodexService(apiKeys []string, cache *cache.Service, scraper *ScraperService, logger *slog.Logger) (*Service, error) {
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("at least one Holodex API key is required")
	}

	logger.Info("Holodex API key pool configured", slog.Int("active_keys", len(apiKeys)))

	httpClient := &http.Client{
		Timeout: constants.APIConfig.HolodexTimeout,
	}

	requester := NewHolodexAPIClient(httpClient, apiKeys, logger)

	return &Service{
		requester: requester,
		cache:     cache,
		scraper:   scraper,
		logger:    logger,
	}, nil
}

// GetLiveStreams: 현재 진행 중인('live') 모든 Hololive 스트림 목록을 조회한다. (캐시 적용)
func (h *Service) GetLiveStreams(ctx context.Context) ([]*domain.Stream, error) {
	cacheKey := "live_streams"

	var cached []*domain.Stream
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		return cached, nil
	}

	params := url.Values{}
	params.Set("org", "Hololive")
	params.Set("status", "live")
	params.Set("type", "stream")

	body, err := h.requester.DoRequest(ctx, "GET", "/live", params)
	if err != nil {
		h.logger.Error("Failed to get live streams", slog.Any("error", err))
		return nil, fmt.Errorf("get live streams: %w", err)
	}

	var rawStreams []StreamRaw
	if err := json.Unmarshal(body, &rawStreams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal live streams: %w", err)
	}

	streams := h.mapStreamsResponse(rawStreams)
	filtered := h.filterHololiveStreams(streams)

	_ = h.cache.Set(ctx, cacheKey, filtered, constants.CacheTTL.LiveStreams)

	return filtered, nil
}

// GetUpcomingStreams: 향후 예정된('upcoming') Hololive 스트림 목록을 조회한다. (최대 hours 시간까지, 캐시 적용)
func (h *Service) GetUpcomingStreams(ctx context.Context, hours int) ([]*domain.Stream, error) {
	cacheKey := fmt.Sprintf("upcoming_streams_%d", hours)

	var cached []*domain.Stream
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		return cached, nil
	}

	params := url.Values{}
	params.Set("org", "Hololive")
	params.Set("status", "upcoming")
	params.Set("type", "stream")
	params.Set("max_upcoming_hours", fmt.Sprintf("%d", util.Min(hours, 168)))
	params.Set("order", "asc")
	params.Set("orderby", "start_scheduled")

	body, err := h.requester.DoRequest(ctx, "GET", "/live", params)
	if err != nil {
		h.logger.Error("Failed to get upcoming streams", slog.Any("error", err))
		return nil, fmt.Errorf("get upcoming streams: %w", err)
	}

	var rawStreams []StreamRaw
	if err := json.Unmarshal(body, &rawStreams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal upcoming streams: %w", err)
	}

	streams := h.mapStreamsResponse(rawStreams)
	filtered := h.filterHololiveStreams(streams)
	upcoming := h.filterUpcomingStreams(filtered)

	_ = h.cache.Set(ctx, cacheKey, upcoming, constants.CacheTTL.UpcomingStreams)

	return upcoming, nil
}

// GetChannelSchedule: 특정 채널의 방송 일정(예정된 방송)을 조회한다.
// includeLive가 true이면 현재 진행 중인 방송도 포함한다.
func (h *Service) GetChannelSchedule(ctx context.Context, channelID string, hours int, includeLive bool) ([]*domain.Stream, error) {
	cacheKey := fmt.Sprintf("channel_schedule_%s_%d_%t", channelID, hours, includeLive)

	var cached []*domain.Stream
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		copied := make([]*domain.Stream, len(cached))
		for i, stream := range cached {
			streamCopy := *stream
			if stream.StartScheduled != nil {
				t := *stream.StartScheduled
				streamCopy.StartScheduled = &t
			}
			if stream.StartActual != nil {
				t := *stream.StartActual
				streamCopy.StartActual = &t
			}
			copied[i] = &streamCopy
		}

		if includeLive {
			return copied, nil
		}
		return h.filterUpcomingStreams(copied), nil
	}

	var statuses []domain.StreamStatus
	if includeLive {
		statuses = []domain.StreamStatus{domain.StreamStatusLive, domain.StreamStatusUpcoming}
	} else {
		statuses = []domain.StreamStatus{domain.StreamStatusUpcoming}
	}

	allStreams := make([]*domain.Stream, 0)

	for _, status := range statuses {
		params := url.Values{}
		params.Set("channel_id", channelID)
		params.Set("status", string(status))
		params.Set("type", "stream")
		params.Set("max_upcoming_hours", fmt.Sprintf("%d", hours))

		body, err := h.requester.DoRequest(ctx, "GET", "/live", params)
		if err != nil {
			h.logger.Error("Failed to get channel schedule",
				slog.String("channel_id", channelID),
				slog.String("status", string(status)),
				slog.Any("error", err),
			)

			if h.shouldUseFallback(err) && h.scraper != nil {
				h.logger.Warn("Using scraper fallback for channel schedule",
					slog.String("channel_id", channelID),
					slog.Any("error", err))

				return h.scraper.FetchChannel(ctx, channelID)
			}

			return nil, fmt.Errorf("get channel schedule: %w", err)
		}

		var rawStreams []StreamRaw
		if err := json.Unmarshal(body, &rawStreams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal channel schedule: %w", err)
		}

		streams := h.mapStreamsResponse(rawStreams)
		allStreams = append(allStreams, streams...)
	}

	hololiveOnly := h.filterHololiveStreams(allStreams)

	slices.SortFunc(hololiveOnly, func(a, b *domain.Stream) int {
		aTime := int64(0)
		bTime := int64(0)
		if a.StartScheduled != nil {
			aTime = a.StartScheduled.Unix()
		}
		if b.StartScheduled != nil {
			bTime = b.StartScheduled.Unix()
		}
		if aTime < bTime {
			return -1
		}
		if aTime > bTime {
			return 1
		}
		return 0
	})

	result := hololiveOnly
	if !includeLive {
		result = h.filterUpcomingStreams(hololiveOnly)
	}

	_ = h.cache.Set(ctx, cacheKey, result, constants.CacheTTL.ChannelSchedule)

	return result, nil
}

// SearchChannels: 채널 이름 검색 쿼리를 통해 해당하는 Hololive 채널 목록을 조회한다.
func (h *Service) SearchChannels(ctx context.Context, query string) ([]*domain.Channel, error) {
	cacheKey := buildSearchChannelsCacheKey(query)

	var cached []*domain.Channel
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		return cached, nil
	}

	query = util.TrimSpace(query)
	params := url.Values{}
	params.Set("org", "Hololive")
	params.Set("name", query)
	params.Set("limit", "50")

	body, err := h.requester.DoRequest(ctx, "GET", "/channels", params)
	if err != nil {
		h.logger.Error("Failed to search channels", slog.String("query", query), slog.Any("error", err))
		return nil, fmt.Errorf("search channels: %w", err)
	}

	var rawChannels []ChannelRaw
	if err := json.Unmarshal(body, &rawChannels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search channels: %w", err)
	}

	channels := h.mapChannelsResponse(rawChannels)

	h.logger.Debug("Holodex API search results",
		slog.String("query", query),
		slog.Int("total_results", len(channels)),
	)

	filtered := make([]*domain.Channel, 0, len(channels))
	for _, ch := range channels {
		if ch.Org != nil && *ch.Org == "Hololive" && !h.isHolostarsChannel(ch) {
			filtered = append(filtered, ch)
		}
	}

	h.logger.Debug("After HOLOSTARS filter", slog.Int("count", len(filtered)))

	_ = h.cache.Set(ctx, cacheKey, filtered, constants.CacheTTL.ChannelSearch)

	return filtered, nil
}

func buildSearchChannelsCacheKey(query string) string {
	normalized := util.Normalize(query)
	if normalized == "" {
		return searchChannelsCacheKeyPrefix + "empty"
	}

	sum := sha256.Sum256([]byte(normalized))
	return searchChannelsCacheKeyPrefix + hex.EncodeToString(sum[:])
}

// GetChannel: 채널 ID로 특정 채널의 상세 정보를 조회한다.
func (h *Service) GetChannel(ctx context.Context, channelID string) (*domain.Channel, error) {
	cacheKey := fmt.Sprintf("channel_%s", channelID)

	var cached domain.Channel
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached.ID != "" {
		return &cached, nil
	}

	body, err := h.requester.DoRequest(ctx, "GET", "/channels/"+channelID, nil)
	if err != nil {
		apiErr := &errors.APIError{}
		if stdErrors.As(err, &apiErr) {
			return nil, nil
		}
		h.logger.Error("Failed to get channel", slog.String("channel_id", channelID), slog.Any("error", err))
		return nil, fmt.Errorf("get channel: %w", err)
	}

	var rawChannel ChannelRaw
	if err := json.Unmarshal(body, &rawChannel); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel: %w", err)
	}

	channel := h.mapChannelResponse(&rawChannel)
	_ = h.cache.Set(ctx, cacheKey, channel, constants.CacheTTL.ChannelInfo)

	return channel, nil
}

func (h *Service) mapStreamsResponse(rawStreams []StreamRaw) []*domain.Stream {
	streams := make([]*domain.Stream, 0, len(rawStreams))
	for _, raw := range rawStreams {
		stream := h.mapStreamResponse(&raw)
		if stream != nil {
			streams = append(streams, stream)
		}
	}
	return streams
}

func (h *Service) mapStreamResponse(raw *StreamRaw) *domain.Stream {
	stream := &domain.Stream{
		ID:        raw.ID,
		Title:     raw.Title,
		Status:    raw.Status,
		Duration:  raw.Duration,
		Thumbnail: raw.Thumbnail,
		Link:      raw.Link,
		TopicID:   raw.TopicID,
	}

	// 썸네일 URL이 없으면 유튜브 기본 썸네일 URL 생성 (mqdefault.jpg - 320x180)
	if stream.Thumbnail == nil || *stream.Thumbnail == "" {
		thumbURL := fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", raw.ID)
		stream.Thumbnail = &thumbURL
	}

	// Link가 없으면 유튜브 영상 URL 생성
	if stream.Link == nil || *stream.Link == "" {
		linkURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", raw.ID)
		stream.Link = &linkURL
	}

	// ChannelID 설정 (빈 문자열 체크)
	if raw.ChannelID != nil && *raw.ChannelID != "" {
		stream.ChannelID = *raw.ChannelID
	} else if raw.Channel != nil && raw.Channel.ID != "" {
		stream.ChannelID = raw.Channel.ID
	} else {
		// ChannelID가 없으면 invalid 데이터로 간주
		h.logger.Warn("Stream missing ChannelID - skipping",
			slog.String("stream_id", raw.ID),
			slog.String("title", raw.Title))
		return nil
	}

	// ChannelName 설정 (빈 문자열 체크)
	if raw.Channel != nil && raw.Channel.Name != "" {
		stream.ChannelName = raw.Channel.Name
	} else {
		// ChannelName이 없어도 ChannelID가 있으면 허용
		h.logger.Debug("Stream missing ChannelName, will use ChannelID",
			slog.String("stream_id", raw.ID),
			slog.String("channel_id", stream.ChannelID))
	}

	if raw.StartScheduled != nil && *raw.StartScheduled != "" {
		if t, err := time.Parse(time.RFC3339, *raw.StartScheduled); err == nil {
			stream.StartScheduled = &t
		}
	}

	if raw.StartActual != nil && *raw.StartActual != "" {
		if t, err := time.Parse(time.RFC3339, *raw.StartActual); err == nil {
			stream.StartActual = &t
		}
	}

	if raw.Channel != nil {
		stream.Channel = h.mapChannelResponse(raw.Channel)
	}

	return stream
}

func (h *Service) mapChannelsResponse(rawChannels []ChannelRaw) []*domain.Channel {
	channels := make([]*domain.Channel, len(rawChannels))
	for i, raw := range rawChannels {
		channels[i] = h.mapChannelResponse(&raw)
	}
	return channels
}

func (h *Service) mapChannelResponse(raw *ChannelRaw) *domain.Channel {
	return &domain.Channel{
		ID:              raw.ID,
		Name:            raw.Name,
		EnglishName:     raw.EnglishName,
		Photo:           raw.Photo,
		Twitter:         raw.Twitter,
		VideoCount:      raw.VideoCount,
		SubscriberCount: raw.SubscriberCount,
		Org:             raw.Org,
		Suborg:          raw.Suborg,
		Group:           raw.Group,
	}
}

func (h *Service) filterHololiveStreams(streams []*domain.Stream) []*domain.Stream {
	filtered := make([]*domain.Stream, 0, len(streams))

	for _, stream := range streams {
		if stream.Channel == nil {
			h.logger.Debug("Filtered out stream without channel info", slog.String("id", stream.ID))
			continue
		}

		channel := stream.Channel

		if channel.Org == nil || *channel.Org != "Hololive" {
			org := ""
			if channel.Org != nil {
				org = *channel.Org
			}
			h.logger.Debug("Filtered out non-Hololive stream",
				slog.String("channel", stream.ChannelName),
				slog.String("org", org),
			)
			continue
		}

		if h.isHolostarsChannel(channel) {
			h.logger.Debug("Filtered out HOLOSTARS stream", slog.String("channel", stream.ChannelName))
			continue
		}

		filtered = append(filtered, stream)
	}

	return filtered
}

func (h *Service) filterUpcomingStreams(streams []*domain.Stream) []*domain.Stream {
	now := time.Now()
	filtered := make([]*domain.Stream, 0, len(streams))

	for _, stream := range streams {
		if stream.StartActual != nil {
			continue
		}

		if stream.StartScheduled != nil && stream.StartScheduled.After(now) {
			filtered = append(filtered, stream)
		} else if stream.StartScheduled == nil {
			filtered = append(filtered, stream)
		}
	}

	return filtered
}

func (h *Service) isHolostarsChannel(channel *domain.Channel) bool {
	if channel == nil {
		return false
	}

	upper := func(s *string) string {
		if s == nil {
			return ""
		}
		return strings.ToUpper(*s)
	}

	return strings.Contains(upper(channel.Suborg), "HOLOSTARS") ||
		strings.Contains(strings.ToUpper(channel.Name), "HOLOSTARS") ||
		strings.Contains(upper(channel.EnglishName), "HOLOSTARS")
}

func (h *Service) shouldUseFallback(err error) bool {
	if err == nil {
		return false
	}

	if h.requester != nil && h.requester.IsCircuitOpen() {
		return true
	}

	apiErr := &errors.APIError{}
	if stdErrors.As(err, &apiErr) {
		if apiErr.StatusCode >= 500 {
			return true
		}
		if apiErr.StatusCode == 503 {
			return true
		}
	}

	keyRotationError := &errors.KeyRotationError{}
	return stdErrors.As(err, &keyRotationError)
}
