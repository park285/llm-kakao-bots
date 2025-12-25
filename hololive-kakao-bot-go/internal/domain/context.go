package domain

import "time"

// CommandContext: 명령어 실행 시 필요한 컨텍스트 정보(채팅방, 사용자, 메시지 내용, 타임스탬프 등)를 담는 구조체
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

// NewCommandContext: 새로운 CommandContext 인스턴스를 생성한다. Sender 필드는 하위 호환성을 위해 UserName으로 설정된다.
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
