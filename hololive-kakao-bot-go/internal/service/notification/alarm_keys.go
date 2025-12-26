package notification

import "strings"

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
