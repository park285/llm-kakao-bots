package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// persistAlarmAsync: 알람을 DB에 비동기로 저장한다. (Write-Through)
// 사용자 응답을 지연시키지 않기 위해 goroutine으로 실행한다.
//
//nolint:contextcheck // Async 작업은 caller context와 독립적으로 실행되어야 함
func (as *AlarmService) persistAlarmAsync(alarm *domain.Alarm) {
	if as.alarmRepo == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := as.alarmRepo.Add(ctx, alarm); err != nil {
			as.logger.Warn("Failed to persist alarm to DB (async)",
				slog.String("room_id", alarm.RoomID),
				slog.String("user_id", alarm.UserID),
				slog.String("channel_id", alarm.ChannelID),
				slog.Any("error", err),
			)
		}
	}()
}

// removeAlarmAsync: 알람을 DB에서 비동기로 삭제한다. (Write-Through)
//
//nolint:contextcheck // Async 작업은 caller context와 독립적으로 실행되어야 함
func (as *AlarmService) removeAlarmAsync(roomID, userID, channelID string) {
	if as.alarmRepo == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := as.alarmRepo.Remove(ctx, roomID, userID, channelID); err != nil {
			as.logger.Warn("Failed to remove alarm from DB (async)",
				slog.String("room_id", roomID),
				slog.String("user_id", userID),
				slog.String("channel_id", channelID),
				slog.Any("error", err),
			)
		}
	}()
}

// clearUserAlarmsAsync: 사용자의 모든 알람을 DB에서 비동기로 삭제한다. (Write-Through)
//
//nolint:contextcheck // Async 작업은 caller context와 독립적으로 실행되어야 함
func (as *AlarmService) clearUserAlarmsAsync(roomID, userID string) {
	if as.alarmRepo == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := as.alarmRepo.ClearByUser(ctx, roomID, userID); err != nil {
			as.logger.Warn("Failed to clear user alarms from DB (async)",
				slog.String("room_id", roomID),
				slog.String("user_id", userID),
				slog.Any("error", err),
			)
		}
	}()
}

// WarmCacheFromDB: 앱 시작 시 DB에서 모든 알람을 로드하여 Valkey 캐시를 워밍한다.
// 이 메서드는 앱 시작 시 한 번만 호출되며, 이후 런타임 중에는 Valkey만 사용한다.
func (as *AlarmService) WarmCacheFromDB(ctx context.Context) error {
	if as.alarmRepo == nil {
		as.logger.Info("Alarm repository not configured, skipping cache warming")
		return nil
	}

	alarms, err := as.alarmRepo.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("load alarms from DB: %w", err)
	}

	if len(alarms) == 0 {
		as.logger.Info("No alarms found in DB, cache warming skipped")
		return nil
	}

	// 1. 알람 데이터를 Valkey에 캐싱
	for _, alarm := range alarms {
		alarmKey := as.getAlarmKey(alarm.RoomID, alarm.UserID)
		registryKey := alarm.RegistryKey()
		channelSubsKey := as.channelSubscribersKey(alarm.ChannelID)

		// 사용자별 알람 채널 목록
		if _, err := as.cache.SAdd(ctx, alarmKey, []string{alarm.ChannelID}); err != nil {
			as.logger.Warn("Failed to warm alarm cache",
				slog.String("alarm_key", alarmKey),
				slog.Any("error", err),
			)
		}

		// 전체 사용자 레지스트리
		_, _ = as.cache.SAdd(ctx, AlarmRegistryKey, []string{registryKey})

		// 채널별 구독자 목록
		_, _ = as.cache.SAdd(ctx, channelSubsKey, []string{registryKey})

		// 채널 레지스트리
		_, _ = as.cache.SAdd(ctx, AlarmChannelRegistryKey, []string{alarm.ChannelID})

		// 멤버 이름 캐싱
		if alarm.MemberName != "" {
			_ = as.CacheMemberName(ctx, alarm.ChannelID, alarm.MemberName)
		}

		// 방/유저 이름 캐싱
		if alarm.RoomName != "" {
			_ = as.cache.HSet(ctx, RoomNamesCacheKey, alarm.RoomID, alarm.RoomName)
		}
		if alarm.UserName != "" {
			_ = as.cache.HSet(ctx, UserNamesCacheKey, alarm.UserID, alarm.UserName)
		}
	}

	as.logger.Info("Cache warmed from DB",
		slog.Int("alarms_loaded", len(alarms)),
	)

	return nil
}
