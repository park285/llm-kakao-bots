package notification

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/alarm"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
)

// NewAlarmService: 새로운 AlarmService 인스턴스를 생성하고 설정(목표 알림 시간 등)을 초기화합니다.
func NewAlarmService(
	cacheSvc *cache.Service,
	holodexSvc *holodex.Service,
	alarmRepo *alarm.Repository,
	logger *slog.Logger,
	advanceMinutes []int,
) *AlarmService {
	targetMinutes := buildTargetMinutes(advanceMinutes)

	return &AlarmService{
		cache:           cacheSvc,
		holodex:         holodexSvc,
		alarmRepo:       alarmRepo,
		logger:          logger,
		targetMinutes:   targetMinutes,
		baseConcurrency: 15,
		maxConcurrency:  50,
		autoscale:       true,
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

	slices.SortFunc(filtered, func(a, b int) int { return b - a })

	if _, hasFallback := seen[1]; !hasFallback {
		filtered = append(filtered, 1)
	}

	return filtered
}

// AddAlarm: 특정 채팅방(사용자)에 대해 특정 멤버(채널)의 방송 알림을 추가합니다.
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

	// 비동기 DB 저장 (Write-Through, non-blocking)
	as.persistAlarmAsync(&domain.Alarm{
		RoomID:     roomID,
		UserID:     userID,
		ChannelID:  channelID,
		MemberName: memberName,
		RoomName:   roomName,
		UserName:   userName,
	})

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

// RemoveAlarm: 특정 채팅방(사용자)에서 특정 멤버(채널)의 방송 알림을 해제합니다.
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

	// 비동기 DB 삭제 (Write-Through, non-blocking)
	as.removeAlarmAsync(roomID, userID, channelID)

	as.logger.Info("Alarm removed",
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.String("channel_id", channelID),
	)

	return removed > 0, nil
}

// GetUserAlarms: 해당 사용자가 현재 구독 중인 모든 채널 ID 목록을 반환합니다.
func (as *AlarmService) GetUserAlarms(ctx context.Context, roomID, userID string) ([]string, error) {
	alarmKey := as.getAlarmKey(roomID, userID)
	channelIDs, err := as.cache.SMembers(ctx, alarmKey)
	if err != nil {
		as.logger.Error("Failed to get user alarms", slog.Any("error", err))
		return []string{}, fmt.Errorf("get user alarms: %w", err)
	}
	return channelIDs, nil
}

// ClearUserAlarms: 해당 사용자의 모든 알림 설정을 삭제(초기화)합니다.
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

	// 비동기 DB 삭제 (Write-Through, non-blocking)
	as.clearUserAlarmsAsync(roomID, userID)

	as.logger.Info("All alarms cleared",
		slog.String("room_id", roomID),
		slog.String("user_id", userID),
		slog.Int("count", int(removed)),
	)

	return int(removed), nil
}
