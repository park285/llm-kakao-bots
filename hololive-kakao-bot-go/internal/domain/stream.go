package domain

import (
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// StreamStatus 는 타입이다.
type StreamStatus string

// StreamStatus 상수 목록.
const (
	// StreamStatusLive 는 상수다.
	StreamStatusLive     StreamStatus = "live"
	StreamStatusUpcoming StreamStatus = "upcoming"
	StreamStatusPast     StreamStatus = "past"
)

func (s StreamStatus) String() string {
	return string(s)
}

// IsValid 는 동작을 수행한다.
func (s StreamStatus) IsValid() bool {
	switch s {
	case StreamStatusLive, StreamStatusUpcoming, StreamStatusPast:
		return true
	default:
		return false
	}
}

// Stream 는 타입이다.
type Stream struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	ChannelID      string       `json:"channel_id"`
	ChannelName    string       `json:"channel_name"`
	Status         StreamStatus `json:"status"`
	StartScheduled *time.Time   `json:"start_scheduled,omitempty"`
	StartActual    *time.Time   `json:"start_actual,omitempty"`
	Duration       *int         `json:"duration,omitempty"` // seconds
	Thumbnail      *string      `json:"thumbnail,omitempty"`
	Link           *string      `json:"link,omitempty"`
	TopicID        *string      `json:"topic_id,omitempty"`
	Channel        *Channel     `json:"channel,omitempty"`
}

// IsLive 는 동작을 수행한다.
func (s *Stream) IsLive() bool {
	if s == nil {
		return false
	}
	return s.Status == StreamStatusLive
}

// IsUpcoming 는 동작을 수행한다.
func (s *Stream) IsUpcoming() bool {
	if s == nil {
		return false
	}
	return s.Status == StreamStatusUpcoming
}

// IsPast 는 동작을 수행한다.
func (s *Stream) IsPast() bool {
	if s == nil {
		return false
	}
	return s.Status == StreamStatusPast
}

// GetYouTubeURL 는 동작을 수행한다.
func (s *Stream) GetYouTubeURL() string {
	if s == nil {
		return ""
	}
	if s.Link != nil && *s.Link != "" {
		return *s.Link
	}
	return "https://youtube.com/watch?v=" + s.ID
}

// TimeUntilStart 는 동작을 수행한다.
func (s *Stream) TimeUntilStart() *time.Duration {
	if s.StartScheduled == nil {
		return nil
	}

	now := time.Now()
	if s.StartScheduled.Before(now) {
		return nil
	}

	duration := s.StartScheduled.Sub(now)
	return &duration
}

// MinutesUntilStart 는 동작을 수행한다.
func (s *Stream) MinutesUntilStart() int {
	return util.MinutesUntilCeil(s.StartScheduled, time.Now())
}
