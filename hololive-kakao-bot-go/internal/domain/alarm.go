package domain

import "time"

// Alarm 는 타입이다.
type Alarm struct {
	RoomID     string    `json:"room_id"`     // KakaoTalk room ID
	UserID     string    `json:"user_id"`     // KakaoTalk user ID
	ChannelID  string    `json:"channel_id"`  // YouTube channel ID
	MemberName string    `json:"member_name"` // Member name for display
	CreatedAt  time.Time `json:"created_at"`
}

// NewAlarm 는 동작을 수행한다.
func NewAlarm(roomID, userID, channelID, memberName string) *Alarm {
	return &Alarm{
		RoomID:     roomID,
		UserID:     userID,
		ChannelID:  channelID,
		MemberName: memberName,
		CreatedAt:  time.Now(),
	}
}

// AlarmNotification 는 타입이다.
type AlarmNotification struct {
	RoomID                string   `json:"room_id"`
	Channel               *Channel `json:"channel"`
	Stream                *Stream  `json:"stream"`
	MinutesUntil          int      `json:"minutes_until"`
	Users                 []string `json:"users"`
	ScheduleChangeMessage string   `json:"schedule_change_message,omitempty"`
}

// NewAlarmNotification 는 동작을 수행한다.
func NewAlarmNotification(roomID string, channel *Channel, stream *Stream, minutesUntil int, users []string, scheduleMessage string) *AlarmNotification {
	return &AlarmNotification{
		RoomID:                roomID,
		Channel:               channel,
		Stream:                stream,
		MinutesUntil:          minutesUntil,
		Users:                 users,
		ScheduleChangeMessage: scheduleMessage,
	}
}

// UserCount 는 동작을 수행한다.
func (n *AlarmNotification) UserCount() int {
	return len(n.Users)
}
