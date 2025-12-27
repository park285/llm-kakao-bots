package lockutil

import "time"

// TTLMillisFromSeconds 는 TTL 초를 밀리초로 변환한다.
func TTLMillisFromSeconds(seconds int64) int64 {
	return seconds * 1000
}

// TTLDurationFromSeconds 는 TTL 초를 time.Duration 으로 변환한다.
func TTLDurationFromSeconds(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}
