package mq

import (
	"context"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
)

// MessageSenderConfig: 게임별 MessageSender를 설정하는 구조체입니다.
type MessageSenderConfig struct {
	// MessageMaxLength: 단일 메시지 최대 길이 (초과 시 분할 전송)
	MessageMaxLength int

	// LockErrorKey: 락 경합 시 에러 메시지 키
	LockErrorKey string

	// LockErrorWithHolderKey: 락 보유자 정보를 포함한 에러 메시지 키 (선택적)
	LockErrorWithHolderKey string

	// UseFinalForErrors: 에러 메시지에 Final 타입 사용 여부
	// true: mqmsg.NewFinal, false: mqmsg.NewError
	UseFinalForErrors bool
}

// BaseMessageSender: 게임별 MessageSender의 공통 기반 구조체입니다.
// 메시지 발행, 분할 전송, 에러 처리 등의 공통 로직을 제공합니다.
type BaseMessageSender struct {
	msgProvider *messageprovider.Provider
	publish     func(ctx context.Context, msg mqmsg.OutboundMessage) error
	config      MessageSenderConfig
}

// NewBaseMessageSender: 새로운 BaseMessageSender 인스턴스를 생성합니다.
func NewBaseMessageSender(
	msgProvider *messageprovider.Provider,
	publish func(ctx context.Context, msg mqmsg.OutboundMessage) error,
	config MessageSenderConfig,
) *BaseMessageSender {
	return &BaseMessageSender{
		msgProvider: msgProvider,
		publish:     publish,
		config:      config,
	}
}

// SendFinal: 최종 처리 결과(답변)를 전송합니다. 메시지가 길 경우 분할 전송합니다.
func (s *BaseMessageSender) SendFinal(ctx context.Context, chatID string, text string, threadID *string) error {
	return SendFinalChunked(ctx, s.publish, chatID, text, threadID, s.config.MessageMaxLength)
}

// SendWaiting: 명령어 처리가 시작되었음을 알리는 대기 메시지를 전송합니다.
func (s *BaseMessageSender) SendWaiting(ctx context.Context, chatID string, threadID *string, command interface{ WaitingMessageKey() *string }) error {
	return SendWaitingFromCommand(ctx, s.publish, s.msgProvider, chatID, threadID, command)
}

// SendError: 발생한 에러에 매핑된 사용자 메시지를 전송합니다.
// UseFinalForErrors 설정에 따라 Final 또는 Error 타입으로 전송합니다.
func (s *BaseMessageSender) SendError(ctx context.Context, chatID string, threadID *string, key string, params ...messageprovider.Param) error {
	text := s.msgProvider.Get(key, params...)
	if s.config.UseFinalForErrors {
		return s.publish(ctx, mqmsg.NewFinal(chatID, text, threadID))
	}
	return s.publish(ctx, mqmsg.NewError(chatID, text, threadID))
}

// SendLockError: 락 획득 실패(다른 요청 처리 중) 시 안내 메시지를 전송합니다.
// holderName이 제공되고 LockErrorWithHolderKey가 설정된 경우 보유자 정보를 포함합니다.
func (s *BaseMessageSender) SendLockError(ctx context.Context, chatID string, threadID *string, holderName *string) error {
	var text string
	if holderName != nil && *holderName != "" && s.config.LockErrorWithHolderKey != "" {
		text = s.msgProvider.Get(s.config.LockErrorWithHolderKey, messageprovider.P("holder", *holderName))
	} else {
		text = s.msgProvider.Get(s.config.LockErrorKey)
	}
	return s.publish(ctx, mqmsg.NewError(chatID, text, threadID))
}

// MsgProvider: 내부 메시지 프로바이더를 반환합니다.
func (s *BaseMessageSender) MsgProvider() *messageprovider.Provider {
	return s.msgProvider
}

// Publish: 내부 publish 함수를 반환합니다.
func (s *BaseMessageSender) Publish() func(context.Context, mqmsg.OutboundMessage) error {
	return s.publish
}
