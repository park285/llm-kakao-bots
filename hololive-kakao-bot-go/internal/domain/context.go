package domain

import "time"

// CommandContext 는 타입이다.
type CommandContext struct {
	Room        string // 숫자 Room ID
	RoomName    string // 한글 방 이름
	UserID      string // 숫자 User ID
	UserName    string // 한글 유저 이름
	Sender      string // 발신자 (deprecated, UserName과 동일)
	IsGroupChat bool
	Message     string
	Timestamp   time.Time
}

// NewCommandContext 는 동작을 수행한다.
func NewCommandContext(room, roomName, userID, userName, message string, isGroupChat bool) *CommandContext {
	return &CommandContext{
		Room:        room,
		RoomName:    roomName,
		UserID:      userID,
		UserName:    userName,
		Sender:      userName, // backward compatibility
		IsGroupChat: isGroupChat,
		Message:     message,
		Timestamp:   time.Now(),
	}
}
