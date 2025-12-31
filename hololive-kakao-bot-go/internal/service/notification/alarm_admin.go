package notification

import (
	"context"
	"fmt"
)

// AlarmEntry: 관리자 대시보드 표시용 알림 정보 구조체
type AlarmEntry struct {
	RoomID     string `json:"roomId"`
	RoomName   string `json:"roomName"`
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	ChannelID  string `json:"channelId"`
	MemberName string `json:"memberName"`
}

// GetAllAlarmKeys: 관리자 대시보드용 모든 알람 정보를 반환합니다.
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

// GetDistinctRooms: 알람이 설정된 고유한 방 ID 목록을 반환합니다.
func (as *AlarmService) GetDistinctRooms(ctx context.Context) ([]string, error) {
	registryKeys, err := as.cache.SMembers(ctx, AlarmRegistryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get alarm registry: %w", err)
	}

	roomSet := make(map[string]struct{})
	for _, registryKey := range registryKeys {
		parts := splitRegistryKey(registryKey)
		if len(parts) != 2 {
			continue
		}
		roomSet[parts[0]] = struct{}{}
	}

	rooms := make([]string, 0, len(roomSet))
	for roomID := range roomSet {
		rooms = append(rooms, roomID)
	}

	return rooms, nil
}
