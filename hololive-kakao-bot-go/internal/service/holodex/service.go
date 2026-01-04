package holodex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stdErrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"

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
func NewHolodexService(apiKeys []string, cacheSvc *cache.Service, scraper *ScraperService, logger *slog.Logger) (*Service, error) {
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("at least one Holodex API key is required")
	}

	logger.Info("Holodex API key pool configured", slog.Int("active_keys", len(apiKeys)))

	// DefaultTransport 복제: TCP Keep-Alive(30s), TLSHandshakeTimeout(10s), Proxy 지원 등 안전장치 유지
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = constants.HolodexTransportConfig.MaxConnsPerHost
	transport.MaxIdleConnsPerHost = constants.HolodexTransportConfig.MaxIdleConnsPerHost
	transport.IdleConnTimeout = constants.HolodexTransportConfig.IdleConnTimeout
	// HTTP/2 활성화 유지 (DefaultTransport 기본값): Cloudflare가 HTTP/2 응답을 보내므로 필수

	httpClient := &http.Client{
		Timeout:   constants.APIConfig.HolodexTimeout,
		Transport: transport,
	}

	requester := NewHolodexAPIClient(httpClient, apiKeys, logger)

	return &Service{
		requester: requester,
		cache:     cacheSvc,
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
	params.Set("sort", "start_scheduled")

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

// GetChannelSchedule: 특정 채널의 방송 일정(예정된 방송)을 조회합니다.
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

	// Holodex API는 콤마 구분 복수 status를 지원
	// 기존 2회 호출을 단일 호출로 통합하여 latency 및 rate limit 부담 감소
	var statusStr string
	if includeLive {
		statusStr = string(domain.StreamStatusLive) + "," + string(domain.StreamStatusUpcoming)
	} else {
		statusStr = string(domain.StreamStatusUpcoming)
	}

	params := url.Values{}
	params.Set("channel_id", channelID)
	params.Set("status", statusStr)
	params.Set("type", "stream")
	params.Set("max_upcoming_hours", fmt.Sprintf("%d", hours))

	body, err := h.requester.DoRequest(ctx, "GET", "/live", params)
	if err != nil {
		h.logger.Error("Failed to get channel schedule",
			slog.String("channel_id", channelID),
			slog.String("status", statusStr),
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

	allStreams := h.mapStreamsResponse(rawStreams)

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

// SearchChannels: 채널 이름 검색 쿼리를 통해 해당하는 Hololive 채널 목록을 조회합니다.
func (h *Service) SearchChannels(ctx context.Context, query string) ([]*domain.Channel, error) {
	cacheKey := buildSearchChannelsCacheKey(query)

	var cached []*domain.Channel
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		return cached, nil
	}

	query = util.TrimSpace(query)
	params := url.Values{}
	params.Set("org", "Hololive")
	params.Set("type", "vtuber")
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
	normalizedQuery := strings.ToLower(query)
	for _, ch := range channels {
		if ch.Org != nil && *ch.Org == "Hololive" && !h.isHolostarsChannel(ch) {
			// 쿼리가 비어있으면 모든 채널 반환, 아니면 이름 매칭 필터링
			if normalizedQuery == "" {
				filtered = append(filtered, ch)
				continue
			}
			// 채널 이름 또는 영어 이름에 쿼리가 포함되는지 확인
			nameMatch := strings.Contains(strings.ToLower(ch.Name), normalizedQuery)
			englishMatch := ch.EnglishName != nil && strings.Contains(strings.ToLower(*ch.EnglishName), normalizedQuery)
			if nameMatch || englishMatch {
				filtered = append(filtered, ch)
			}
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

// GetChannel: 채널 ID로 특정 채널의 상세 정보를 조회합니다.
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

// GetChannels: 여러 채널 ID로 채널 정보를 배치 조회합니다.
// 캐시를 우선 조회하고, 캐시 미스된 채널은 /channels 리스트 API로 한 번에 조회합니다.
// 기존 N+1 개별 호출 패턴을 단일 호출로 최적화하여 rate limit 부담을 대폭 감소시킵니다.
func (h *Service) GetChannels(ctx context.Context, channelIDs []string) (map[string]*domain.Channel, error) {
	if len(channelIDs) == 0 {
		return make(map[string]*domain.Channel), nil
	}

	result := make(map[string]*domain.Channel, len(channelIDs))
	var missedIDs []string

	// 캐시에서 먼저 조회
	for _, id := range channelIDs {
		cacheKey := fmt.Sprintf("channel_%s", id)
		var cached domain.Channel
		if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached.ID != "" {
			result[id] = &cached
		} else {
			missedIDs = append(missedIDs, id)
		}
	}

	h.logger.Debug("GetChannels cache status",
		slog.Int("total", len(channelIDs)),
		slog.Int("cache_hits", len(result)),
		slog.Int("cache_misses", len(missedIDs)),
	)

	// 캐시 미스가 없으면 바로 반환
	if len(missedIDs) == 0 {
		return result, nil
	}

	// 캐시 미스된 채널을 /channels 리스트 API로 한 번에 조회
	// Holodex /channels API는 ID 필터를 지원하지 않으므로 org로 전체 조회 후 필터링
	allChannels, err := h.fetchHololiveChannelList(ctx)
	if err != nil {
		h.logger.Warn("Failed to fetch channel list, falling back to individual queries",
			slog.Any("error", err),
			slog.Int("missed_count", len(missedIDs)),
		)
		// 폴백: 개별 조회 (기존 방식, 최대 5개 동시)
		return h.fetchChannelsIndividually(ctx, channelIDs, result, missedIDs)
	}

	// 필요한 채널만 결과에 추가하고 캐시 저장
	missedSet := make(map[string]bool, len(missedIDs))
	for _, id := range missedIDs {
		missedSet[id] = true
	}

	for _, ch := range allChannels {
		if missedSet[ch.ID] {
			result[ch.ID] = ch
			// 개별 캐시 저장
			cacheKey := fmt.Sprintf("channel_%s", ch.ID)
			_ = h.cache.Set(ctx, cacheKey, ch, constants.CacheTTL.ChannelInfo)
		}
	}

	h.logger.Info("GetChannels batch complete (optimized)",
		slog.Int("requested", len(channelIDs)),
		slog.Int("returned", len(result)),
		slog.Int("from_list_api", len(result)-len(channelIDs)+len(missedIDs)),
	)

	return result, nil
}

// GetChannelsLiveStatus: 특정 채널들의 현재 생방송/예정 상태를 빠르게 조회합니다.
// /users/live 엔드포인트를 사용하여 캐시된 결과를 반환합니다.
// 주의: org, status, sort 필터링 미지원 - live+upcoming 모두 반환됨
// 사용 시나리오: 알림 체크, 대시보드 상태 표시 등 빠른 상태 확인
func (h *Service) GetChannelsLiveStatus(ctx context.Context, channelIDs []string) ([]*domain.Stream, error) {
	if len(channelIDs) == 0 {
		return []*domain.Stream{}, nil
	}

	// 채널 ID 목록으로 캐시 키 생성
	cacheKey := fmt.Sprintf("channels_live_status_%s", strings.Join(channelIDs, ","))
	var cached []*domain.Stream
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		return cached, nil
	}

	params := url.Values{}
	params.Set("channels", strings.Join(channelIDs, ","))

	body, err := h.requester.DoRequest(ctx, "GET", "/users/live", params)
	if err != nil {
		h.logger.Error("Failed to get channels live status",
			slog.Int("channel_count", len(channelIDs)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("get channels live status: %w", err)
	}

	var rawStreams []StreamRaw
	if err := json.Unmarshal(body, &rawStreams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channels live status: %w", err)
	}

	streams := h.mapStreamsResponse(rawStreams)
	filtered := h.filterHololiveStreams(streams)

	// /users/live는 캐시된 결과이므로 짧은 TTL 적용 (30초)
	_ = h.cache.Set(ctx, cacheKey, filtered, 30*time.Second)

	h.logger.Debug("GetChannelsLiveStatus completed",
		slog.Int("requested_channels", len(channelIDs)),
		slog.Int("streams_found", len(filtered)),
	)

	return filtered, nil
}

// fetchHololiveChannelList: Hololive 채널 목록을 /channels API로 조회합니다.
// 내부 캐시를 사용하여 반복 호출 시 효율을 높입니다.
// Holodex API limit=100 제한으로 인해 페이지네이션을 사용합니다.
func (h *Service) fetchHololiveChannelList(ctx context.Context) ([]*domain.Channel, error) {
	const cacheKey = "hololive_channel_list"

	var cached []*domain.Channel
	if err := h.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		return cached, nil
	}

	var allChannels []*domain.Channel
	const pageSize = 100
	offset := 0

	for {
		params := url.Values{}
		params.Set("org", "Hololive")
		params.Set("type", "vtuber")
		params.Set("limit", fmt.Sprintf("%d", pageSize))
		params.Set("offset", fmt.Sprintf("%d", offset))

		body, err := h.requester.DoRequest(ctx, "GET", "/channels", params)
		if err != nil {
			return nil, fmt.Errorf("fetch hololive channel list (offset=%d): %w", offset, err)
		}

		var rawChannels []ChannelRaw
		if err := json.Unmarshal(body, &rawChannels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal channel list: %w", err)
		}

		channels := h.mapChannelsResponse(rawChannels)
		allChannels = append(allChannels, channels...)

		// 마지막 페이지면 종료
		if len(rawChannels) < pageSize {
			break
		}

		offset += pageSize

		// 무한 루프 방지 (최대 500개)
		if offset >= 500 {
			h.logger.Warn("Pagination limit reached", slog.Int("offset", offset))
			break
		}
	}

	h.logger.Debug("Fetched all Hololive channels", slog.Int("total", len(allChannels)))

	// 5분간 캐시 (채널 정보는 자주 변하지 않음)
	_ = h.cache.Set(ctx, cacheKey, allChannels, 5*time.Minute)

	return allChannels, nil
}

// fetchChannelsIndividually: 개별 /channels/{id} API로 채널을 조회합니다. (폴백용)
func (h *Service) fetchChannelsIndividually(ctx context.Context, channelIDs []string, result map[string]*domain.Channel, missedIDs []string) (map[string]*domain.Channel, error) {
	const maxConcurrent = 5
	semaphore := make(chan struct{}, maxConcurrent)
	resultChan := make(chan struct {
		id      string
		channel *domain.Channel
	}, len(missedIDs))

	for _, id := range missedIDs {
		go func(channelID string) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-ctx.Done():
				resultChan <- struct {
					id      string
					channel *domain.Channel
				}{channelID, nil}
				return
			default:
			}

			channel, err := h.GetChannel(ctx, channelID)
			if err != nil {
				h.logger.Warn("Failed to get channel in batch",
					slog.String("channel_id", channelID),
					slog.Any("error", err),
				)
				resultChan <- struct {
					id      string
					channel *domain.Channel
				}{channelID, nil}
				return
			}
			resultChan <- struct {
				id      string
				channel *domain.Channel
			}{channelID, channel}
		}(id)
	}

	for i := 0; i < len(missedIDs); i++ {
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("batch channel fetch canceled: %w", ctx.Err())
		case r := <-resultChan:
			if r.channel != nil {
				result[r.id] = r.channel
			}
		}
	}

	h.logger.Info("GetChannels batch complete (fallback)",
		slog.Int("requested", len(channelIDs)),
		slog.Int("returned", len(result)),
	)

	return result, nil
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
