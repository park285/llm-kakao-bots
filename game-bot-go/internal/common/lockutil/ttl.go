package lockutil

import "time"

// TTLMillisFromSeconds: TTL 초를 밀리초로 변환합니다.
func TTLMillisFromSeconds(seconds int64) int64 {
	return seconds * 1000
}

// TTLDurationFromSeconds: TTL 초를 time.Duration으로 변환합니다.
func TTLDurationFromSeconds(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}
