package domain

import (
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// StreamStatus: 방송 상태(진행 중, 예정, 종료)를 나타내는 열거형
type StreamStatus string

// StreamStatus 상수 목록.
const (
	// StreamStatusLive: 방송 진행 중
	StreamStatusLive StreamStatus = "live"
	// StreamStatusUpcoming: 방송 예정
	StreamStatusUpcoming StreamStatus = "upcoming"
	// StreamStatusPast: 방송 종료됨
	StreamStatusPast StreamStatus = "past"
)

func (s StreamStatus) String() string {
	return string(s)
}

// IsValid: 방송 상태 값이 유효한지 검증한다.
func (s StreamStatus) IsValid() bool {
	switch s {
	case StreamStatusLive, StreamStatusUpcoming, StreamStatusPast:
		return true
	default:
		return false
	}
}

// Stream: Holodex 등에서 수집한 방송(스트림) 상세 정보
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

// IsLive: 방송이 현재 진행 중('live')인지 확인한다.
func (s *Stream) IsLive() bool {
	if s == nil {
		return false
	}
	return s.Status == StreamStatusLive
}

// IsUpcoming: 방송이 예정('upcoming') 상태인지 확인한다.
func (s *Stream) IsUpcoming() bool {
	if s == nil {
		return false
	}
	return s.Status == StreamStatusUpcoming
}

// IsPast: 방송이 종료('past')되었는지 확인한다.
func (s *Stream) IsPast() bool {
	if s == nil {
		return false
	}
	return s.Status == StreamStatusPast
}

// GetYouTubeURL: 방송 시청을 위한 YouTube URL을 반환한다. (Link 필드가 없으면 ID로 생성)
func (s *Stream) GetYouTubeURL() string {
	if s == nil {
		return ""
	}
	if s.Link != nil && *s.Link != "" {
		return *s.Link
	}
	return "https://youtube.com/watch?v=" + s.ID
}

// TimeUntilStart: 예정된 방송 시작 시각까지 남은 시간을 Duration으로 반환한다.
// 이미 시작 시간이 지났거나 예정 시간이 없으면 nil을 반환한다.
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

// MinutesUntilStart: 방송 시작까지 남은 시간을 '분' 단위(올림)로 계산하여 반환한다.
func (s *Stream) MinutesUntilStart() int {
	return util.MinutesUntilCeil(s.StartScheduled, time.Now())
}
