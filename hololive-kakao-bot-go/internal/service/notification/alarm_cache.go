package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

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

// GetMemberNameWithFallback: 2-layer fallback으로 멤버 이름을 조회한다.
// 1. Cache (Valkey) → 2. Database (alarms 테이블)
// DB 조회 성공 시 Valkey에 캐싱하여 다음 요청은 Valkey에서 처리.
func (as *AlarmService) GetMemberNameWithFallback(ctx context.Context, channelID string) string {
	// Layer 1: Valkey Cache (빠름)
	name, err := as.cache.HGet(ctx, MemberNameKey, channelID)
	if err == nil && util.TrimSpace(name) != "" {
		return name
	}

	// Layer 2: alarms 테이블 (영속 저장소)
	if as.alarmRepo != nil {
		displayName, err := as.alarmRepo.GetMemberName(ctx, channelID)
		if err == nil && displayName != "" {
			// DB 조회 성공 시 Valkey에 캐싱 (다음 요청은 빠르게)
			if cacheErr := as.CacheMemberName(ctx, channelID, displayName); cacheErr != nil {
				as.logger.Warn("Failed to cache member name from DB",
					slog.String("channel_id", channelID),
					slog.Any("error", cacheErr),
				)
			}
			as.logger.Debug("Member name resolved from alarms DB",
				slog.String("channel_id", channelID),
				slog.String("name", displayName),
			)
			return displayName
		}
	}

	// 모든 레이어 실패: 채널 ID 반환
	as.logger.Warn("Failed to resolve member name from cache/DB",
		slog.String("channel_id", channelID),
	)
	return channelID
}

// SetRoomName: 방 ID에 대한 표시 이름을 설정합니다.
func (as *AlarmService) SetRoomName(ctx context.Context, roomID, roomName string) error {
	if err := as.cache.HSet(ctx, RoomNamesCacheKey, roomID, roomName); err != nil {
		return fmt.Errorf("set room name: %w", err)
	}
	return nil
}

// SetUserName: 사용자 ID에 대한 표시 이름을 설정합니다.
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

// isAlreadyNotified: 해당 방송에 대해 이미 알림이 발송되었는지 확인함
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
	fields := map[string]any{
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
	fields := map[string]any{
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
	if err := as.cache.HMSet(ctx, key, map[string]any{"status": status}); err != nil {
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
