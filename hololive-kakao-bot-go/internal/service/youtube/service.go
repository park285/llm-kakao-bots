package youtube

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
)

// Service: YouTube API와 상호작용하여 채널 및 영상 정보를 제공하는 서비스
type Service struct {
	service    *youtube.Service
	cache      *cache.Service
	logger     *slog.Logger
	quotaUsed  int
	quotaMu    sync.Mutex
	quotaReset time.Time
}

// NewService: YouTube 서비스 인스턴스를 생성한다.
func NewYouTubeService(ctx context.Context, apiKey string, cache *cache.Service, logger *slog.Logger) (*Service, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("YouTube API key is required")
	}

	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	ys := &Service{
		service:    service,
		cache:      cache,
		logger:     logger,
		quotaUsed:  0,
		quotaReset: getNextQuotaReset(),
	}

	logger.Info("YouTube backup service initialized",
		slog.Time("quotaReset", ys.quotaReset))

	return ys, nil
}

func getNextQuotaReset() time.Time {
	pt, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Now().In(pt)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, pt)
	return next
}

func (ys *Service) checkQuota(cost int) error {
	ys.quotaMu.Lock()
	defer ys.quotaMu.Unlock()

	now := time.Now()
	if now.After(ys.quotaReset) {
		ys.quotaUsed = 0
		ys.quotaReset = getNextQuotaReset()
		ys.logger.Info("YouTube API quota auto-reset",
			slog.Time("nextReset", ys.quotaReset))
	}

	if ys.quotaUsed+cost > (constants.YouTubeConfig.DailyQuotaLimit - constants.YouTubeConfig.QuotaSafetyMargin) {
		return &QuotaExceededError{
			Used:      ys.quotaUsed,
			Limit:     constants.YouTubeConfig.DailyQuotaLimit,
			Requested: cost,
			ResetTime: ys.quotaReset,
		}
	}

	return nil
}

func (ys *Service) consumeQuota(cost int) {
	ys.quotaMu.Lock()
	defer ys.quotaMu.Unlock()

	ys.quotaUsed += cost
	remaining := constants.YouTubeConfig.DailyQuotaLimit - ys.quotaUsed

	ys.logger.Debug("YouTube API quota consumed",
		slog.Int("cost", cost),
		slog.Int("used", ys.quotaUsed),
		slog.Int("remaining", remaining),
		slog.Float64("usagePercent", float64(ys.quotaUsed)/float64(constants.YouTubeConfig.DailyQuotaLimit)*100))

	if remaining < constants.YouTubeConfig.QuotaSafetyMargin {
		ys.logger.Warn("YouTube API quota running low",
			slog.Int("remaining", remaining),
			slog.Time("resetTime", ys.quotaReset))
	}
}

// 개별 채널의 예정된 스트림을 가져와서 결과에 추가
func (ys *Service) fetchChannelUpcoming(ctx context.Context, channelID string, allStreams *[]*domain.Stream, mu *sync.Mutex, actualCost *int, costMu *sync.Mutex, errChan chan<- error) {
	streams, err := ys.getChannelUpcomingStreams(ctx, channelID)
	if err != nil {
		errChan <- fmt.Errorf("channel %s: %w", channelID, err)
		return
	}

	mu.Lock()
	*allStreams = append(*allStreams, streams...)
	mu.Unlock()

	costMu.Lock()
	*actualCost += constants.YouTubeConfig.SearchQuotaCost
	costMu.Unlock()
}

// FetchUpcomingStreams: 지정된 채널들의 예정된 방송(라이브 예정) 목록을 조회한다.
func (ys *Service) GetUpcomingStreams(ctx context.Context, channelIDs []string) ([]*domain.Stream, error) {
	if len(channelIDs) > constants.YouTubeConfig.MaxChannelsPerCall {
		ys.logger.Warn("Too many channels requested, limiting to max",
			slog.Int("requested", len(channelIDs)),
			slog.Int("limited", constants.YouTubeConfig.MaxChannelsPerCall))
		channelIDs = channelIDs[:constants.YouTubeConfig.MaxChannelsPerCall]
	}

	sortedIDs := make([]string, len(channelIDs))
	copy(sortedIDs, channelIDs)
	slices.Sort(sortedIDs)
	cacheKey := fmt.Sprintf("youtube:upcoming:%s", strings.Join(sortedIDs, ","))
	if cached, found := ys.cache.GetStreams(ctx, cacheKey); found {
		ys.logger.Debug("YouTube cache hit (backup avoided)",
			slog.Int("streams", len(cached)))
		return cached, nil
	}

	estimatedCost := len(channelIDs) * constants.YouTubeConfig.SearchQuotaCost
	if err := ys.checkQuota(estimatedCost); err != nil {
		return nil, err
	}

	ys.logger.Info("Fetching from YouTube API (BACKUP MODE)",
		slog.Int("channels", len(channelIDs)),
		slog.Int("estimatedCost", estimatedCost))

	var allStreams []*domain.Stream
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(channelIDs))

	semaphore := make(chan struct{}, constants.YouTubeConfig.MaxConcurrentRequests)

	actualCost := 0
	costMu := sync.Mutex{}

	for _, channelID := range channelIDs {
		wg.Add(1)
		go func(chID string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			ys.fetchChannelUpcoming(ctx, chID, &allStreams, &mu, &actualCost, &costMu, errChan)
		}(channelID)
	}

	wg.Wait()
	close(errChan)

	ys.consumeQuota(actualCost)

	errors := make([]error, 0)
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		ys.logger.Warn("Some YouTube API calls failed",
			slog.Int("failures", len(errors)),
			slog.Int("successes", len(channelIDs)-len(errors)))
	}

	if len(allStreams) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("all YouTube API calls failed: %d errors", len(errors))
	}

	ys.cache.SetStreams(ctx, cacheKey, allStreams, constants.YouTubeConfig.CacheExpiration)

	ys.logger.Info("YouTube API backup completed",
		slog.Int("channels", len(channelIDs)),
		slog.Int("streams", len(allStreams)),
		slog.Int("quotaUsed", actualCost))

	return allStreams, nil
}

// processUpcomingStreams: YouTube API 응답에서 예정된 방송 정보를 추출하고 가공한다.
func (ys *Service) getChannelUpcomingStreams(ctx context.Context, channelID string) ([]*domain.Stream, error) {
	call := ys.service.Search.List([]string{"snippet"}).
		ChannelId(channelID).
		Type("video").
		EventType("upcoming").
		MaxResults(int64(constants.YouTubeConfig.SearchMaxResults)).
		Order("date")

	response, err := call.Context(ctx).Do()
	if err != nil {
		apiErr := &googleapi.Error{}
		if errors.As(err, &apiErr) {
			if apiErr.Code == 403 {
				return nil, &QuotaExceededError{
					Used:      ys.quotaUsed,
					Limit:     constants.YouTubeConfig.DailyQuotaLimit,
					Requested: constants.YouTubeConfig.SearchQuotaCost,
					ResetTime: ys.quotaReset,
				}
			}
		}
		return nil, fmt.Errorf("YouTube API error: %w", err)
	}

	streams := make([]*domain.Stream, 0, len(response.Items))
	for _, item := range response.Items {
		if item.Id == nil || item.Id.VideoId == "" {
			continue
		}

		stream := &domain.Stream{
			ID:        item.Id.VideoId,
			Title:     item.Snippet.Title,
			ChannelID: channelID,
			Status:    domain.StreamStatusUpcoming,
			Link:      stringPtr(fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.Id.VideoId)),
			Thumbnail: extractThumbnail(item.Snippet.Thumbnails),
		}

		if item.Snippet.PublishedAt != "" {
			if startTime, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt); err == nil {
				stream.StartScheduled = &startTime
			}
		}

		if item.Snippet.ChannelTitle != "" {
			stream.Channel = &domain.Channel{
				ID:   channelID,
				Name: item.Snippet.ChannelTitle,
			}
		}

		streams = append(streams, stream)
	}

	return streams, nil
}

func extractThumbnail(thumbnails *youtube.ThumbnailDetails) *string {
	if thumbnails == nil {
		return nil
	}

	if thumbnails.Maxres != nil && thumbnails.Maxres.Url != "" {
		return &thumbnails.Maxres.Url
	}
	if thumbnails.High != nil && thumbnails.High.Url != "" {
		return &thumbnails.High.Url
	}
	if thumbnails.Medium != nil && thumbnails.Medium.Url != "" {
		return &thumbnails.Medium.Url
	}
	if thumbnails.Default != nil && thumbnails.Default.Url != "" {
		return &thumbnails.Default.Url
	}

	return nil
}

// GetQuotaStatus: 현재 API 할당량 사용량, 잔여량, 초기화 예정 시간을 반환한다.
func (ys *Service) GetQuotaStatus() (used int, remaining int, resetTime time.Time) {
	ys.quotaMu.Lock()
	defer ys.quotaMu.Unlock()

	if time.Now().After(ys.quotaReset) {
		return 0, constants.YouTubeConfig.DailyQuotaLimit, getNextQuotaReset()
	}

	return ys.quotaUsed, constants.YouTubeConfig.DailyQuotaLimit - ys.quotaUsed, ys.quotaReset
}

// IsQuotaAvailable: 지정된 채널 수만큼 조회할 수 있는 API 할당량이 남아있는지 확인한다.
func (ys *Service) IsQuotaAvailable(channelCount int) bool {
	estimatedCost := channelCount * constants.YouTubeConfig.SearchQuotaCost
	err := ys.checkQuota(estimatedCost)
	return err == nil
}

// QuotaExceededError: API 할당량 초과 시 발생하는 에러 구조체
type QuotaExceededError struct {
	Used      int
	Limit     int
	Requested int
	ResetTime time.Time
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("YouTube API quota exceeded: used %d/%d (requested %d more), resets at %s",
		e.Used, e.Limit, e.Requested, e.ResetTime.Format(time.RFC3339))
}

func stringPtr(s string) *string {
	return &s
}

// FetchChannelStats: 지정된 채널들의 통계(구독자 수, 조회수 등)를 조회한다.
func (ys *Service) GetChannelStatistics(ctx context.Context, channelIDs []string) (map[string]*ChannelStats, error) {
	if len(channelIDs) == 0 {
		return make(map[string]*ChannelStats), nil
	}

	cost := len(channelIDs) * constants.YouTubeConfig.ChannelsQuotaCost
	if err := ys.checkQuota(cost); err != nil {
		return nil, err
	}

	result := make(map[string]*ChannelStats)

	batchSize := 50
	for i := 0; i < len(channelIDs); i += batchSize {
		end := i + batchSize
		if end > len(channelIDs) {
			end = len(channelIDs)
		}

		batch := channelIDs[i:end]

		call := ys.service.Channels.List([]string{"statistics", "snippet"}).
			Id(batch...)

		response, err := call.Context(ctx).Do()
		if err != nil {
			ys.logger.Error("Failed to fetch channel statistics",
				slog.Int("batch_size", len(batch)),
				slog.Any("error", err))
			continue
		}

		for _, channel := range response.Items {
			stats := &ChannelStats{
				ChannelID:       channel.Id,
				ChannelTitle:    channel.Snippet.Title,
				SubscriberCount: channel.Statistics.SubscriberCount,
				VideoCount:      channel.Statistics.VideoCount,
				ViewCount:       channel.Statistics.ViewCount,
				Timestamp:       time.Now(),
			}
			result[channel.Id] = stats
		}
	}

	ys.consumeQuota(cost)

	ys.logger.Info("Channel statistics fetched",
		slog.Int("channels", len(channelIDs)),
		slog.Int("results", len(result)),
		slog.Int("quota_used", cost))

	return result, nil
}

// GetRecentVideos: 특정 채널의 최근 업로드된 비디오 목록을 조회한다.
func (ys *Service) GetRecentVideos(ctx context.Context, channelID string, maxResults int64) ([]string, error) {
	if err := ys.checkQuota(constants.YouTubeConfig.SearchQuotaCost); err != nil {
		return nil, err
	}

	call := ys.service.Search.List([]string{"id"}).
		ChannelId(channelID).
		Type("video").
		Order("date").
		MaxResults(maxResults)

	response, err := call.Context(ctx).Do()
	if err != nil {
		ys.logger.Error("Failed to fetch recent videos",
			slog.String("channel", channelID),
			slog.Any("error", err))
		return nil, fmt.Errorf("YouTube search error: %w", err)
	}

	videoIDs := make([]string, 0, len(response.Items))
	for _, item := range response.Items {
		if item.Id != nil && item.Id.VideoId != "" {
			videoIDs = append(videoIDs, item.Id.VideoId)
		}
	}

	ys.consumeQuota(constants.YouTubeConfig.SearchQuotaCost)

	ys.logger.Debug("Recent videos fetched",
		slog.String("channel", channelID),
		slog.Int("count", len(videoIDs)))

	return videoIDs, nil
}

// ChannelStats: API로부터 조회된 단일 채널의 통계 정보
type ChannelStats struct {
	ChannelID       string
	ChannelTitle    string
	SubscriberCount uint64
	VideoCount      uint64
	ViewCount       uint64
	Timestamp       time.Time
}
