package mqmsg

import (
	"errors"
	"fmt"
	"strings"
)

// MQ 메시지 파싱 에러 목록.
var (
	// ErrMissingChatID 는 패키지 변수다.
	ErrMissingChatID  = errors.New("missing chat id")
	ErrMissingContent = errors.New("missing content")
	ErrMissingUserID  = errors.New("missing user id")
)

// InboundMessage 는 타입이다.
type InboundMessage struct {
	ChatID   string
	UserID   string
	Content  string
	ThreadID *string
	Sender   *string
}

// OutboundType 는 타입이다.
type OutboundType string

// OutboundType 상수 목록.
const (
	// OutboundWaiting 는 상수다.
	OutboundWaiting OutboundType = "waiting"
	OutboundFinal   OutboundType = "final"
	OutboundError   OutboundType = "error"
)

// OutboundMessage 는 타입이다.
type OutboundMessage struct {
	ChatID   string
	Text     string
	ThreadID *string
	Type     OutboundType
}

// NewWaiting 는 동작을 수행한다.
func NewWaiting(chatID string, text string, threadID *string) OutboundMessage {
	return OutboundMessage{ChatID: chatID, Text: text, ThreadID: threadID, Type: OutboundWaiting}
}

// NewFinal 는 동작을 수행한다.
func NewFinal(chatID string, text string, threadID *string) OutboundMessage {
	return OutboundMessage{ChatID: chatID, Text: text, ThreadID: threadID, Type: OutboundFinal}
}

// NewError 는 동작을 수행한다.
func NewError(chatID string, text string, threadID *string) OutboundMessage {
	return OutboundMessage{ChatID: chatID, Text: text, ThreadID: threadID, Type: OutboundError}
}

// ToStreamValues 는 동작을 수행한다.
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

// ParseInboundMessage 는 동작을 수행한다.
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
