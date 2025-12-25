package notification

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/conc/pool"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// 알람 키 상수 목록.
const (
	// AlarmKeyPrefix: 알림 데이터 Redis 키 접두사
	AlarmKeyPrefix = "alarm:"
	// AlarmRegistryKey: 알림이 설정된 모든 사용자/채팅방 목록을 추적하는 Set 키
	AlarmRegistryKey            = "alarm:registry"
	AlarmChannelRegistryKey     = "alarm:channel_registry"
	ChannelSubscribersKeyPrefix = "alarm:channel_subscribers:"
	MemberNameKey               = "member_names"
	RoomNamesCacheKey           = "alarm:room_names"
	UserNamesCacheKey           = "alarm:user_names"
	NotifiedKeyPrefix           = "notified:"
	NextStreamKeyPrefix         = "alarm:next_stream:"
)

// NotifiedData: 알림 중복 발송 방지를 위해 기록하는 알림 이력 정보
type NotifiedData struct {
	StartScheduled string `json:"start_scheduled"`
	NotifiedAt     string `json:"notified_at"`
	MinutesUntil   int    `json:"minutes_until"`
}

// AlarmService: 방송 알림(Alarm)을 관리하고, 예정된 방송을 주기적으로 체크하여 알림을 발송하는 서비스
type AlarmService struct {
	cache           *cache.Service
	holodex         *holodex.Service
	logger          *slog.Logger
	targetMinutes   []int
	baseConcurrency int  // 기본 동시성
	maxConcurrency  int  // 최대 동시성
	autoscale       bool // 자동 스케일링 활성화
	cacheMutex      sync.RWMutex
}

// NewAlarmService: 새로운 AlarmService 인스턴스를 생성하고 설정(목표 알림 시간 등)을 초기화한다.
func NewAlarmService(cache *cache.Service, holodex *holodex.Service, logger *slog.Logger, advanceMinutes []int) *AlarmService {
	targetMinutes := buildTargetMinutes(advanceMinutes)

	return &AlarmService{
		cache:           cache,
		holodex:         holodex,
		logger:          logger,
		targetMinutes:   targetMinutes,
		baseConcurrency: 15,   // 최소 동시성
		maxConcurrency:  50,   // 최대 동시성
		autoscale:       true, // 자동 스케일링 활성화
	}
}

func buildTargetMinutes(advanceMinutes []int) []int {
	filtered := make([]int, 0, len(advanceMinutes))
	seen := make(map[int]struct{})

	for _, minute := range advanceMinutes {
		if minute <= 0 {
			continue
		}
		if _, exists := seen[minute]; exists {
			continue
		}
		seen[minute] = struct{}{}
		filtered = append(filtered, minute)
	}

	if len(filtered) == 0 {
		return []int{5, 3, 1}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i] > filtered[j]
	})

	if _, hasFallback := seen[1]; !hasFallback {
		filtered = append(filtered, 1)
	}

	return filtered
}

// AddAlarm: 특정 채팅방(사용자)에 대해 특정 멤버(채널)의 방송 알림을 추가한다.
func (as *AlarmService) AddAlarm(ctx context.Context, roomID, userID, channelID, memberName, roomName, userName string) (bool, error) {
	alarmKey := as.getAlarmKey(roomID, userID)
	added, err := as.cache.SAdd(ctx, alarmKey, []string{channelID})
	if err != nil {
		as.logger.Error("Failed to add alarm", slog.Any("error", err))
		return false, fmt.Errorf("add alarm: %w", err)
	}

	registryKey := as.getRegistryKey(roomID, userID)
	if _, err := as.cache.SAdd(ctx, AlarmRegistryKey, []string{registryKey}); err != nil {
		as.logger.Warn("Failed to add to registry", slog.Any("error", err))
	}

	channelSubsKey := as.channelSubscribersKey(channelID)
	if _, err := as.cache.SAdd(ctx, channelSubsKey, []string{registryKey}); err != nil {
		as.logger.Warn("Failed to add channel subscriber", slog.Any("error", err))
	}

	if _, err := as.cache.SAdd(ctx, AlarmChannelRegistryKey, []string{channelID}); err != nil {
		as.logger.Warn("Failed to add to channel registry", slog.Any("error", err))
	}

	if err := as.CacheMemberName(ctx, channelID, memberName); err != nil {
		as.logger.Warn("Failed to cache member name", slog.Any("error", err))
	}

	// 방/유저 이름 캐싱
	if roomName != "" {
		_ = as.cache.HSet(ctx, RoomNamesCacheKey, roomID, roomName)
	}
	if userName != "" {
		_ = as.cache.HSet(ctx, UserNamesCacheKey, userID, userName)
	}

	as.logger.Info("Alarm added",
		slog.String("room_id", roomID),
		slog.String("room_name", roomName),
		slog.String("user_id", userID),
		slog.String("user_name", userName),
		slog.String("channel_id", channelID),
		slog.String("member_name", memberName),
	)

	return added > 0, nil
}

// RemoveAlarm: 특정 채팅방(사용자)에서 특정 멤버(채널)의 방송 알림을 해제한다.
func (as *AlarmService) RemoveAlarm(ctx context.Context, roomID, userID, channelID string) (bool, error) {
	alarmKey := as.getAlarmKey(roomID, userID)
	removed, err := as.cache.SRem(ctx, alarmKey, []string{channelID})
	if err != nil {
		as.logger.Error("Failed to remove alarm", slog.Any("error", err))
		return false, fmt.Errorf("remove alarm: %w", err)
	}

	registryKey := as.getRegistryKey(roomID, userID)
	channelSubsKey := as.channelSubscribersKey(channelID)

	if _, errSRem := as.cache.SRem(ctx, channelSubsKey, []string{registryKey}); errSRem != nil {
		as.logger.Warn("Failed to remove from channel subscribers", slog.Any("error", errSRem))
	}

	remainingSubs, err := as.cache.SMembers(ctx, channelSubsKey)
	if err != nil {
		as.logger.Warn("Failed to get remaining subscribers", slog.Any("error", err))
	}
	if err == nil && len(remainingSubs) == 0 {
		_, _ = as.cache.SRem(ctx, AlarmChannelRegistryKey, []string{channelID})
		_ = as.cache.Del(ctx, channelSubsKey)
	}

	remainingAlarms, err := as.cache.SMembers(ctx, alarmKey)
	if err == nil && len(remainingAlarms) == 0 {
		_, _ = as.cache.SRem(ctx, AlarmRegistryKey, []string{registryKey})
		as.logger.Info("User removed from registry (no alarms left)",
			slog.String("room_id", roomID),
			slog.String("user_id", userID),
		)
	}

	as.logger.Info("Alarm removed",
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.String("channel_id", channelID),
	)

	return removed > 0, nil
}

// GetUserAlarms: 해당 사용자가 현재 구독 중인 모든 채널 ID 목록을 반환한다.
func (as *AlarmService) GetUserAlarms(ctx context.Context, roomID, userID string) ([]string, error) {
	alarmKey := as.getAlarmKey(roomID, userID)
	channelIDs, err := as.cache.SMembers(ctx, alarmKey)
	if err != nil {
		as.logger.Error("Failed to get user alarms", slog.Any("error", err))
		return []string{}, fmt.Errorf("get user alarms: %w", err)
	}
	return channelIDs, nil
}

// ClearUserAlarms: 해당 사용자의 모든 알림 설정을 삭제(초기화)한다.
func (as *AlarmService) ClearUserAlarms(ctx context.Context, roomID, userID string) (int, error) {
	alarms, err := as.GetUserAlarms(ctx, roomID, userID)
	if err != nil {
		return 0, err
	}

	if len(alarms) == 0 {
		return 0, nil
	}

	alarmKey := as.getAlarmKey(roomID, userID)
	removed, err := as.cache.SRem(ctx, alarmKey, alarms)
	if err != nil {
		as.logger.Error("Failed to clear user alarms", slog.Any("error", err))
		return 0, fmt.Errorf("clear user alarms: %w", err)
	}

	registryKey := as.getRegistryKey(roomID, userID)

	for _, channelID := range alarms {
		channelSubsKey := as.channelSubscribersKey(channelID)
		_, _ = as.cache.SRem(ctx, channelSubsKey, []string{registryKey})

		remainingSubs, err := as.cache.SMembers(ctx, channelSubsKey)
		if err == nil && len(remainingSubs) == 0 {
			_, _ = as.cache.SRem(ctx, AlarmChannelRegistryKey, []string{channelID})
			_ = as.cache.Del(ctx, channelSubsKey)
		}
	}

	_, _ = as.cache.SRem(ctx, AlarmRegistryKey, []string{registryKey})

	as.logger.Info("All alarms cleared",
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.Int("count", int(removed)),
	)

	return int(removed), nil
}

// CheckUpcomingStreams: 구독된 채널들의 예정 방송을 확인하고, 알림 조건(설정된 예고 시간)에 맞으면 알림 메시지를 생성한다.
// Worker Pool을 사용하여 병렬로 채널 정보를 조회한다.
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

// CacheMemberName: 채널 ID에 해당하는 멤버 이름을 Redis에 캐싱한다. (표시 이름 최적화)
func (as *AlarmService) CacheMemberName(ctx context.Context, channelID, memberName string) error {
	if err := as.cache.HSet(ctx, MemberNameKey, channelID, memberName); err != nil {
		return fmt.Errorf("cache member name: %w", err)
	}
	return nil
}

// GetMemberName: 캐시된 멤버 이름을 조회한다. 없으면 빈 문자열을 반환한다.
func (as *AlarmService) GetMemberName(ctx context.Context, channelID string) (string, error) {
	name, err := as.cache.HGet(ctx, MemberNameKey, channelID)
	if err != nil {
		return "", fmt.Errorf("get member name: %w", err)
	}
	return name, nil
}

// SetRoomName sets a display name for a room ID
func (as *AlarmService) SetRoomName(ctx context.Context, roomID, roomName string) error {
	if err := as.cache.HSet(ctx, RoomNamesCacheKey, roomID, roomName); err != nil {
		return fmt.Errorf("set room name: %w", err)
	}
	return nil
}

// SetUserName sets a display name for a user ID
func (as *AlarmService) SetUserName(ctx context.Context, userID, userName string) error {
	if err := as.cache.HSet(ctx, UserNamesCacheKey, userID, userName); err != nil {
		return fmt.Errorf("set user name: %w", err)
	}
	return nil
}

// MarkAsNotified: 해당 방송(streamID)에 대해 특정 시점(minutesUntil)의 알림을 발송했음을 기록한다.
func (as *AlarmService) MarkAsNotified(ctx context.Context, streamID string, startScheduled time.Time, minutesUntil int) error {
	notifiedKey := NotifiedKeyPrefix + streamID
	notifiedData := NotifiedData{
		StartScheduled: startScheduled.Format(time.RFC3339),
		NotifiedAt:     time.Now().Format(time.RFC3339),
		MinutesUntil:   minutesUntil,
	}

	if err := as.cache.Set(ctx, notifiedKey, notifiedData, constants.CacheTTL.NotificationSent); err != nil {
		as.logger.Warn("Failed to mark as notified",
			slog.String("stream_id", streamID),
			slog.Any("error", err),
		)
		return fmt.Errorf("mark as notified: %w", err)
	}

	return nil
}

// isAlreadyNotified checks if a notification was already sent for this stream
func (as *AlarmService) isAlreadyNotified(ctx context.Context, streamID string) bool {
	notifiedKey := NotifiedKeyPrefix + streamID
	var notifiedData NotifiedData
	err := as.cache.Get(ctx, notifiedKey, &notifiedData)
	return err == nil && notifiedData.StartScheduled != ""
}

// GetNextStreamInfo: 특정 채널의 다음 방송 정보(예정 또는 라이브)를 캐시에서 조회한다.
func (as *AlarmService) GetNextStreamInfo(ctx context.Context, channelID string) (*domain.NextStreamInfo, error) {
	as.cacheMutex.RLock()
	defer as.cacheMutex.RUnlock()

	key := NextStreamKeyPrefix + channelID
	data, err := as.cache.HGetAll(ctx, key)
	if err != nil {
		as.logger.Error("Failed to get next stream info from cache",
			slog.String("channel_id", channelID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("get next stream info: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	info := &domain.NextStreamInfo{
		Status:  domain.NextStreamStatus(util.TrimSpace(data["status"])),
		VideoID: util.TrimSpace(data["video_id"]),
		Title:   util.TrimSpace(data["title"]),
	}

	if !info.Status.IsValid() {
		as.logger.Warn("Unexpected cache status",
			slog.String("channel_id", channelID),
			slog.String("status", info.Status.String()),
		)
		return nil, nil
	}

	startScheduledStr := util.TrimSpace(data["start_scheduled"])
	if startScheduledStr != "" {
		scheduledDate, err := time.Parse(time.RFC3339, startScheduledStr)
		if err != nil {
			as.logger.Error("Failed to parse scheduled time",
				slog.String("channel_id", channelID),
				slog.String("start_scheduled", startScheduledStr),
				slog.Any("error", err),
			)
			return nil, nil
		}
		info.StartScheduled = &scheduledDate
	}

	if info.Status.IsUpcoming() {
		if startScheduledStr == "" || info.Title == "" || info.VideoID == "" || info.StartScheduled == nil {
			as.logger.Error("Incomplete cache data for upcoming stream",
				slog.String("channel_id", channelID),
				slog.Bool("has_title", info.Title != ""),
				slog.Bool("has_start", startScheduledStr != ""),
				slog.Bool("has_video_id", info.VideoID != ""),
			)
			return nil, nil
		}
	}

	return info, nil
}

func (as *AlarmService) refreshNextStreamCache(ctx context.Context, channelID string, streams []*domain.Stream) {
	if err := as.writeNextStreamCache(ctx, channelID, streams); err != nil {
		as.logger.Warn("Failed to update next stream cache", slog.String("channel_id", channelID), slog.Any("error", err))
	}
}

func (as *AlarmService) findLiveStream(streams []*domain.Stream) *domain.Stream {
	for _, s := range streams {
		if s != nil && s.IsLive() {
			return s
		}
	}
	return nil
}

func (as *AlarmService) nextUpcomingStream(streams []*domain.Stream) *domain.Stream {
	for _, s := range streams {
		if s != nil && s.IsUpcoming() && s.StartScheduled != nil {
			return s
		}
	}
	return nil
}

func (as *AlarmService) cacheLiveStream(ctx context.Context, key string, stream *domain.Stream) error {
	fields := map[string]interface{}{
		"title":    stream.Title,
		"video_id": stream.ID,
		"status":   "live",
	}
	if stream.StartScheduled != nil {
		fields["start_scheduled"] = stream.StartScheduled.Format(time.RFC3339)
	}

	if err := as.cache.HMSet(ctx, key, fields); err != nil {
		as.logger.Error("Failed to cache live stream", slog.String("stream_id", stream.ID), slog.Any("error", err))
		return fmt.Errorf("cache live stream: %w", err)
	}

	_ = as.cache.Expire(ctx, key, constants.CacheTTL.NextStreamInfo)
	return nil
}

func (as *AlarmService) cacheUpcomingStream(ctx context.Context, key string, stream *domain.Stream) error {
	fields := map[string]interface{}{
		"title":           stream.Title,
		"start_scheduled": stream.StartScheduled.Format(time.RFC3339),
		"video_id":        stream.ID,
		"status":          "upcoming",
	}

	if err := as.cache.HMSet(ctx, key, fields); err != nil {
		as.logger.Error("Failed to cache upcoming stream", slog.String("stream_id", stream.ID), slog.Any("error", err))
		return fmt.Errorf("cache upcoming stream: %w", err)
	}

	_ = as.cache.Expire(ctx, key, constants.CacheTTL.NextStreamInfo)
	return nil
}

func (as *AlarmService) cacheStatus(ctx context.Context, key, status string) error {
	if err := as.cache.HMSet(ctx, key, map[string]interface{}{"status": status}); err != nil {
		as.logger.Error("Failed to set cache status", slog.String("status", status), slog.Any("error", err))
		return fmt.Errorf("cache status: %w", err)
	}

	_ = as.cache.Expire(ctx, key, constants.CacheTTL.NextStreamInfo)
	return nil
}

func (as *AlarmService) shouldPreserveCache(ctx context.Context, key string, streams []*domain.Stream) bool {
	existing, err := as.cache.HGetAll(ctx, key)
	if err != nil || len(existing) == 0 || existing["status"] != "upcoming" {
		return false
	}

	cachedVideoID := existing["video_id"]
	if cachedVideoID == "" {
		return false
	}

	for _, s := range streams {
		if s != nil && s.ID == cachedVideoID && s.IsUpcoming() {
			_ = as.cache.Expire(ctx, key, constants.CacheTTL.NextStreamInfo)
			return true
		}
	}

	return false
}

func (as *AlarmService) writeNextStreamCache(ctx context.Context, channelID string, streams []*domain.Stream) error {
	as.cacheMutex.Lock()
	defer as.cacheMutex.Unlock()

	key := NextStreamKeyPrefix + channelID

	if len(streams) == 0 {
		return as.cacheStatus(ctx, key, "no_upcoming")
	}

	if liveStream := as.findLiveStream(streams); liveStream != nil {
		return as.cacheLiveStream(ctx, key, liveStream)
	}

	upcomingStream := as.nextUpcomingStream(streams)

	if upcomingStream == nil || upcomingStream.StartScheduled == nil {
		if as.shouldPreserveCache(ctx, key, streams) {
			return nil
		}
		return as.cacheStatus(ctx, key, "time_unknown")
	}
	return as.cacheUpcomingStream(ctx, key, upcomingStream)
}

func (as *AlarmService) getAlarmKey(roomID, userID string) string {
	return AlarmKeyPrefix + roomID + ":" + userID
}

func (as *AlarmService) getRegistryKey(roomID, userID string) string {
	return roomID + ":" + userID
}

func (as *AlarmService) channelSubscribersKey(channelID string) string {
	return ChannelSubscribersKeyPrefix + channelID
}

func splitRegistryKey(key string) []string {
	return strings.SplitN(key, ":", 2)
}

// AlarmEntry represents a single alarm for admin display
// calculateConcurrency 채널 수에 따라 최적 동시성 계산
func (as *AlarmService) calculateConcurrency(channelCount int) int {
	if !as.autoscale {
		return as.baseConcurrency
	}

	// 채널 수의 30%를 동시성으로 사용 (최소/최대값 적용)
	optimal := channelCount * 30 / 100

	if optimal < as.baseConcurrency {
		return as.baseConcurrency
	}
	if optimal > as.maxConcurrency {
		return as.maxConcurrency
	}

	return optimal
}

// AlarmEntry: 관리자 대시보드 표시용 알림 정보 구조체
type AlarmEntry struct {
	RoomID     string `json:"roomId"`
	RoomName   string `json:"roomName"`
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	ChannelID  string `json:"channelId"`
	MemberName string `json:"memberName"`
}

// GetAllAlarmKeys returns all alarms for admin dashboard
func (as *AlarmService) GetAllAlarmKeys(ctx context.Context) ([]*AlarmEntry, error) {
	registryKeys, err := as.cache.SMembers(ctx, AlarmRegistryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get alarm registry: %w", err)
	}

	// 이름 맵 미리 로드
	roomNamesMap, _ := as.cache.HGetAll(ctx, RoomNamesCacheKey)
	userNamesMap, _ := as.cache.HGetAll(ctx, UserNamesCacheKey)

	alarms := make([]*AlarmEntry, 0)

	for _, registryKey := range registryKeys {
		parts := splitRegistryKey(registryKey)
		if len(parts) != 2 {
			continue
		}

		roomID, userID := parts[0], parts[1]
		alarmKey := as.getAlarmKey(roomID, userID)

		channelIDs, err := as.cache.SMembers(ctx, alarmKey)
		if err != nil {
			continue
		}

		for _, channelID := range channelIDs {
			memberName, _ := as.GetMemberName(ctx, channelID)

			// 이름 조회 (없으면 ID 그대로)
			roomName := roomNamesMap[roomID]
			if roomName == "" {
				roomName = roomID
			}

			userName := userNamesMap[userID]
			if userName == "" {
				userName = userID
			}

			alarms = append(alarms, &AlarmEntry{
				RoomID:     roomID,
				RoomName:   roomName,
				UserID:     userID,
				UserName:   userName,
				ChannelID:  channelID,
				MemberName: memberName,
			})
		}
	}

	return alarms, nil
}
