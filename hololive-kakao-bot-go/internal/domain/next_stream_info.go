package domain

import "time"

// NextStreamStatus 는 타입이다.
type NextStreamStatus string

// NextStreamStatus 상수 목록.
const (
	// NextStreamStatusLive 는 상수다.
	NextStreamStatusLive        NextStreamStatus = "live"
	NextStreamStatusUpcoming    NextStreamStatus = "upcoming"
	NextStreamStatusNoUpcoming  NextStreamStatus = "no_upcoming"
	NextStreamStatusTimeUnknown NextStreamStatus = "time_unknown"
)

func (s NextStreamStatus) String() string {
	return string(s)
}

// IsLive 는 동작을 수행한다.
func (s NextStreamStatus) IsLive() bool {
	return s == NextStreamStatusLive
}

// IsUpcoming 는 동작을 수행한다.
func (s NextStreamStatus) IsUpcoming() bool {
	return s == NextStreamStatusUpcoming
}

// IsValid 는 동작을 수행한다.
func (s NextStreamStatus) IsValid() bool {
	switch s {
	case NextStreamStatusLive, NextStreamStatusUpcoming, NextStreamStatusNoUpcoming, NextStreamStatusTimeUnknown:
		return true
	default:
		return false
	}
}

// NextStreamInfo 는 타입이다.
type NextStreamInfo struct {
	Status         NextStreamStatus
	VideoID        string
	Title          string
	StartScheduled *time.Time
}
