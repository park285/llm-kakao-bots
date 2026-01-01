package notification

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// CheckUpcomingStreams: 구독된 채널들의 예정 방송을 확인하고, 알림 조건(설정된 예고 시간)에 맞으면 알림 메시지를 생성합니다.
// Worker Pool을 사용하여 병렬로 채널 정보를 조회합니다.
func (as *AlarmService) CheckUpcomingStreams(ctx context.Context) ([]*domain.AlarmNotification, error) {
	channelIDs, err := as.cache.SMembers(ctx, AlarmChannelRegistryKey)
	if err != nil {
		as.logger.Error("Failed to get channel registry", slog.Any("error", err))
		return nil, fmt.Errorf("check upcoming streams: %w", err)
	}

	if len(channelIDs) == 0 {
		return []*domain.AlarmNotification{}, nil
	}

	// 동적 동시성 계산
	concurrency := as.calculateConcurrency(len(channelIDs))

	as.logger.Debug("Alarm check starting",
		slog.Int("channels", len(channelIDs)),
		slog.Int("concurrency", concurrency),
	)

	p := pool.New().WithMaxGoroutines(concurrency)
	now := time.Now()

	results := make([]*channelCheckResult, len(channelIDs))
	resultsMu := sync.Mutex{}

	for idx, channelID := range channelIDs {
		idx, channelID := idx, channelID
		p.Go(func() {
			result := as.checkChannel(ctx, channelID)
			resultsMu.Lock()
			results[idx] = result
			resultsMu.Unlock()
		})
	}

	p.Wait()

	notifications := make([]*domain.AlarmNotification, 0)

	for _, result := range results {
		if result == nil || len(result.subscribers) == 0 {
			continue
		}

		as.triggerCacheRefresh(ctx, result.channelID, result.streams)

		if len(result.streams) == 0 {
			continue
		}

		upcomingStreams := as.filterUpcomingStreams(result.streams, now)

		for _, stream := range upcomingStreams {
			roomNotifs, err := as.createNotification(ctx, stream, result.channelID, result.subscribers)
			if err != nil {
				as.logger.Warn("Failed to create notification", slog.Any("error", err))
				continue
			}

			if len(roomNotifs) > 0 {
				as.logger.Info("Alarm notifications created",
					slog.String("channel", stream.ChannelName),
					slog.Int("minutes_until", roomNotifs[0].MinutesUntil),
					slog.Int("rooms", len(roomNotifs)),
				)
				notifications = append(notifications, roomNotifs...)
			}
		}
	}

	return notifications, nil
}

type channelCheckResult struct {
	channelID   string
	subscribers []string
	streams     []*domain.Stream
}

func (as *AlarmService) checkChannel(ctx context.Context, channelID string) *channelCheckResult {
	channelSubsKey := as.channelSubscribersKey(channelID)
	subscribers, err := as.cache.SMembers(ctx, channelSubsKey)
	if err != nil {
		as.logger.Warn("Failed to get subscribers", slog.String("channel_id", channelID), slog.Any("error", err))
		return &channelCheckResult{channelID: channelID, subscribers: []string{}, streams: []*domain.Stream{}}
	}

	// 채널 구독자 수 로그 (필요시 활성화)
	// as.logger.Info("Channel subscribers", slog.String("channel_id", channelID), slog.Int("count", len(subscribers)))

	if len(subscribers) == 0 {
		_, _ = as.cache.SRem(ctx, AlarmChannelRegistryKey, []string{channelID})
		as.logger.Info("Channel removed from registry (no subscribers)", slog.String("channel_id", channelID))
		return &channelCheckResult{channelID: channelID, subscribers: []string{}, streams: []*domain.Stream{}}
	}

	streams, err := as.holodex.GetChannelSchedule(ctx, channelID, 24, true)
	if err != nil {
		as.logger.Warn("Failed to get channel schedule",
			slog.String("channel_id", channelID),
			slog.Any("error", err),
		)
		return &channelCheckResult{channelID: channelID, subscribers: subscribers, streams: []*domain.Stream{}}
	}

	return &channelCheckResult{
		channelID:   channelID,
		subscribers: subscribers,
		streams:     streams,
	}
}

func (as *AlarmService) filterUpcomingStreams(streams []*domain.Stream, now time.Time) []*domain.Stream {
	filtered := make([]*domain.Stream, 0, len(streams))

	for _, stream := range streams {
		if !stream.IsUpcoming() || stream.StartScheduled == nil {
			continue
		}

		secondsUntil := int(stream.StartScheduled.Sub(now).Seconds())
		minutesUntil := util.MinutesUntilCeil(stream.StartScheduled, now)

		shouldNotify := false
		for _, target := range as.targetMinutes {
			if minutesUntil == target {
				shouldNotify = true
				break
			}
		}

		if secondsUntil > 0 && shouldNotify {
			filtered = append(filtered, stream)
		}
	}

	return filtered
}

func (as *AlarmService) triggerCacheRefresh(parent context.Context, channelID string, streams []*domain.Stream) {
	if parent == nil {
		return
	}

	select {
	case <-parent.Done():
		return
	default:
	}

	streamsCopy := make([]*domain.Stream, len(streams))
	copy(streamsCopy, streams)

	go func(p context.Context, chID string, data []*domain.Stream) {
		if p != nil {
			select {
			case <-p.Done():
				return
			default:
			}
		}

		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout.AlarmService)
		defer cancel()

		as.refreshNextStreamCache(ctxWithTimeout, chID, data)
	}(parent, channelID, streamsCopy)
}

func (as *AlarmService) createNotification(ctx context.Context, stream *domain.Stream, channelID string, subscriberKeys []string) ([]*domain.AlarmNotification, error) {
	if stream.StartScheduled == nil {
		return []*domain.AlarmNotification{}, nil
	}

	minutesUntil := stream.MinutesUntilStart()
	if minutesUntil < 0 {
		return []*domain.AlarmNotification{}, nil
	}

	scheduleChangeMsg := as.detectScheduleChange(ctx, stream)

	// 이미 알림을 보냈고 일정 변경이 없으면 중복 발송 방지
	if scheduleChangeMsg == "" && as.isAlreadyNotified(ctx, stream.ID) {
		as.logger.Debug("Skipping duplicate notification",
			slog.String("stream_id", stream.ID),
			slog.String("channel", stream.ChannelName),
			slog.Int("minutes_until", minutesUntil),
		)
		return []*domain.AlarmNotification{}, nil
	}

	usersByRoom, keysToRemove := as.validateAndGroupSubscribers(ctx, channelID, subscriberKeys)

	channelSubsKey := as.channelSubscribersKey(channelID)
	if len(keysToRemove) > 0 {
		_, _ = as.cache.SRem(ctx, channelSubsKey, keysToRemove)
	}

	if len(usersByRoom) == 0 {
		_, _ = as.cache.SRem(ctx, AlarmChannelRegistryKey, []string{channelID})
		_ = as.cache.Del(ctx, channelSubsKey)
		return []*domain.AlarmNotification{}, nil
	}

	channel, err := as.holodex.GetChannel(ctx, channelID)
	if err != nil || channel == nil {
		as.logger.Warn("Failed to get channel", slog.String("channel_id", channelID), slog.Any("error", err))
		return []*domain.AlarmNotification{}, nil
	}

	notifications := make([]*domain.AlarmNotification, 0, len(usersByRoom))
	for roomID, users := range usersByRoom {
		notifications = append(notifications, domain.NewAlarmNotification(
			roomID,
			channel,
			stream,
			minutesUntil,
			users,
			scheduleChangeMsg,
		))
	}

	return notifications, nil
}

// 스트림 일정 변경 감지 및 변경 메시지 반환
func (as *AlarmService) detectScheduleChange(ctx context.Context, stream *domain.Stream) string {
	notifiedKey := NotifiedKeyPrefix + stream.ID
	var notifiedData NotifiedData

	err := as.cache.Get(ctx, notifiedKey, &notifiedData)
	if err != nil || notifiedData.StartScheduled == "" {
		return ""
	}

	savedTime, err := time.Parse(time.RFC3339, notifiedData.StartScheduled)
	if err != nil {
		return ""
	}

	currentTime := *stream.StartScheduled
	if savedTime.Unix() == currentTime.Unix() {
		return ""
	}

	as.logger.Info("Schedule changed, resetting notification",
		slog.String("stream_id", stream.ID),
		slog.String("old", notifiedData.StartScheduled),
		slog.String("new", stream.StartScheduled.Format(time.RFC3339)))

	return formatScheduleChangeMessage(savedTime, currentTime)
}

func formatScheduleChangeMessage(savedTime, currentTime time.Time) string {
	diff := currentTime.Sub(savedTime)
	if diff == 0 {
		return ""
	}

	minutesChanged := int(math.Ceil(math.Abs(diff.Minutes())))
	isDelayed := diff > 0

	if minutesChanged > 0 {
		if isDelayed {
			return fmt.Sprintf("일정이 %d분 늦춰졌습니다.", minutesChanged)
		}
		return fmt.Sprintf("일정이 %d분 앞당겨졌습니다.", minutesChanged)
	}

	if isDelayed {
		return "일정이 잠시 늦춰졌습니다."
	}
	return "일정이 잠시 앞당겨졌습니다."
}

// 구독자 검증 및 룸별 그룹화
func (as *AlarmService) validateAndGroupSubscribers(ctx context.Context, channelID string, subscriberKeys []string) (map[string][]string, []string) {
	usersByRoom := make(map[string][]string)
	keysToRemove := make([]string, 0)

	for _, registryKey := range subscriberKeys {
		parts := splitRegistryKey(registryKey)
		if len(parts) != 2 {
			as.logger.Warn("Invalid registry key", slog.String("key", registryKey))
			keysToRemove = append(keysToRemove, registryKey)
			continue
		}

		room, user := parts[0], parts[1]
		userAlarmKey := as.getAlarmKey(room, user)

		stillSubscribed, isMemberErr := as.cache.SIsMember(ctx, userAlarmKey, channelID)
		if isMemberErr != nil || !stillSubscribed {
			keysToRemove = append(keysToRemove, registryKey)
			continue
		}

		usersByRoom[room] = append(usersByRoom[room], user)
	}

	return usersByRoom, keysToRemove
}
