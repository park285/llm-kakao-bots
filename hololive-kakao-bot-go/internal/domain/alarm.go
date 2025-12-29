package domain

import "time"

// Alarm: 특정 채팅방(user)이 특정 멤버(channel)의 방송 알림을 구독한 정보
type Alarm struct {
	ID         int       `json:"id,omitempty"`          // DB 기본 키
	RoomID     string    `json:"room_id"`               // 카카오톡 방 ID
	UserID     string    `json:"user_id"`               // 카카오톡 사용자 ID
	ChannelID  string    `json:"channel_id"`            // YouTube 채널 ID
	MemberName string    `json:"member_name,omitempty"` // 표시용 멤버 이름
	RoomName   string    `json:"room_name,omitempty"`   // 방 이름 (캐싱용)
	UserName   string    `json:"user_name,omitempty"`   // 사용자 이름 (캐싱용)
	CreatedAt  time.Time `json:"created_at"`
}

// RegistryKey: room:user 형식의 레지스트리 키를 반환한다.
func (a *Alarm) RegistryKey() string {
	return a.RoomID + ":" + a.UserID
}

// NewAlarm: 새로운 알림 구독 객체를 생성한다.
func NewAlarm(roomID, userID, channelID, memberName string) *Alarm {
	return &Alarm{
		RoomID:     roomID,
		UserID:     userID,
		ChannelID:  channelID,
		MemberName: memberName,
		CreatedAt:  time.Now(),
	}
}

// AlarmNotification: 방송 시작 임박 등의 이벤트로 인해 발송될 알림 메시지 정보
// 여러 사용자(Users)에게 동일한 내용이 전송될 수 있다.
type AlarmNotification struct {
	RoomID                string   `json:"room_id"`
	Channel               *Channel `json:"channel"`
	Stream                *Stream  `json:"stream"`
	MinutesUntil          int      `json:"minutes_until"`
	Users                 []string `json:"users"`
	ScheduleChangeMessage string   `json:"schedule_change_message,omitempty"`
}

// NewAlarmNotification: 알림 발송을 위한 새로운 Notification 객체를 생성한다.
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

// UserCount: 이 알림을 수신하게 될 사용자의 수를 반환한다.
func (n *AlarmNotification) UserCount() int {
	return len(n.Users)
}
