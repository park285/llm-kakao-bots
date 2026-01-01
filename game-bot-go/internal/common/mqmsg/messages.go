package mqmsg

import (
	"errors"
	"fmt"
	"strings"
)

// MQ 메시지 파싱 에러 목록.
var (
	// ErrMissingChatID: 채팅방 ID가 누락된 경우 반환되는 에러입니다.
	ErrMissingChatID  = errors.New("missing chat id")
	ErrMissingContent = errors.New("missing content")
	ErrMissingUserID  = errors.New("missing user id")
)

// InboundMessage: MQ에서 수신한 인바운드 메시지 구조체입니다.
type InboundMessage struct {
	ChatID   string
	UserID   string
	Content  string
	ThreadID *string
	Sender   *string
}

// OutboundType: 아웃바운드 메시지의 유형을 나타냅니다 (waiting, final, error).
type OutboundType string

// OutboundType 상수 목록.
const (
	// OutboundWaiting: 처리 중임을 알리는 대기 메시지입니다.
	OutboundWaiting OutboundType = "waiting"
	OutboundFinal   OutboundType = "final"
	OutboundError   OutboundType = "error"
)

// OutboundMessage: MQ로 발행할 아웃바운드 메시지 구조체입니다.
type OutboundMessage struct {
	ChatID   string
	Text     string
	ThreadID *string
	Type     OutboundType
}

// NewWaiting: 처리 중 상태의 대기 메시지를 생성합니다.
func NewWaiting(chatID string, text string, threadID *string) OutboundMessage {
	return OutboundMessage{ChatID: chatID, Text: text, ThreadID: threadID, Type: OutboundWaiting}
}

// NewFinal: 최종 응답 메시지를 생성합니다.
func NewFinal(chatID string, text string, threadID *string) OutboundMessage {
	return OutboundMessage{ChatID: chatID, Text: text, ThreadID: threadID, Type: OutboundFinal}
}

// NewError: 에러 응답 메시지를 생성합니다.
func NewError(chatID string, text string, threadID *string) OutboundMessage {
	return OutboundMessage{ChatID: chatID, Text: text, ThreadID: threadID, Type: OutboundError}
}

// ToStreamValues: 메시지를 Redis Stream 발행용 map으로 변환합니다.
func (m OutboundMessage) ToStreamValues() map[string]any {
	values := map[string]any{
		"chatId": m.ChatID,
		"text":   m.Text,
		"type":   string(m.Type),
	}
	if m.ThreadID != nil && strings.TrimSpace(*m.ThreadID) != "" {
		values["threadId"] = strings.TrimSpace(*m.ThreadID)
	}
	return values
}

// ParseInboundMessage: Redis Stream 필드에서 인바운드 메시지를 파싱합니다.
func ParseInboundMessage(fields map[string]string) (InboundMessage, error) {
	chatID := strings.TrimSpace(fields["room"])
	if chatID == "" {
		return InboundMessage{}, ErrMissingChatID
	}

	content := strings.TrimSpace(fields["text"])
	if content == "" {
		return InboundMessage{}, ErrMissingContent
	}

	sender := strings.TrimSpace(fields["sender"])
	userID := strings.TrimSpace(fields["userId"])
	if userID == "" {
		return InboundMessage{}, ErrMissingUserID
	}

	var threadIDPtr *string
	if threadID := strings.TrimSpace(fields["threadId"]); threadID != "" {
		threadIDPtr = &threadID
	}

	var senderPtr *string
	if sender != "" {
		senderPtr = &sender
	}

	return InboundMessage{
		ChatID:   chatID,
		UserID:   userID,
		Content:  content,
		ThreadID: threadIDPtr,
		Sender:   senderPtr,
	}, nil
}

func (m InboundMessage) String() string {
	threadID := ""
	if m.ThreadID != nil {
		threadID = *m.ThreadID
	}
	sender := ""
	if m.Sender != nil {
		sender = *m.Sender
	}
	return fmt.Sprintf("chatId=%s userId=%s threadId=%s sender=%s", m.ChatID, m.UserID, threadID, sender)
}
