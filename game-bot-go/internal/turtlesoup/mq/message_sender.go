package mq

import (
	"context"
	"fmt"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// MessageSender: MQ로 게임 관련 메시지를 발행하는 서비스입니다.
// 공통 BaseMessageSender를 내부적으로 사용하여 핵심 로직을 위임합니다.
type MessageSender struct {
	base *commonmq.BaseMessageSender
}

// NewMessageSender: MessageSender 인스턴스를 생성합니다.
func NewMessageSender(msgProvider *messageprovider.Provider, publish func(ctx context.Context, msg mqmsg.OutboundMessage) error) *MessageSender {
	return &MessageSender{
		base: commonmq.NewBaseMessageSender(msgProvider, publish, commonmq.MessageSenderConfig{
			MessageMaxLength:       tsconfig.KakaoMessageMaxLength,
			LockErrorKey:           tsmessages.LockRequestInProgress,
			LockErrorWithHolderKey: tsmessages.LockRequestInProgressWithHolder,
			UseFinalForErrors:      false, // turtlesoup는 에러를 Error 타입으로 전송
		}),
	}
}

// SendFinal: 최종 응답 메시지를 발송합니다. 긴 메시지는 청크로 분할합니다.
func (s *MessageSender) SendFinal(ctx context.Context, message mqmsg.InboundMessage, text string) error {
	if err := s.base.SendFinal(ctx, message.ChatID, text, message.ThreadID); err != nil {
		return fmt.Errorf("send final message: %w", err)
	}
	return nil
}

// SendWaiting: 처리 중이라는 대기 상태 메시지를 발송합니다.
func (s *MessageSender) SendWaiting(ctx context.Context, message mqmsg.InboundMessage, command Command) error {
	if err := s.base.SendWaiting(ctx, message.ChatID, message.ThreadID, command); err != nil {
		return fmt.Errorf("send waiting message: %w", err)
	}
	return nil
}

// SendError: 에러 메시지를 발송합니다.
func (s *MessageSender) SendError(ctx context.Context, message mqmsg.InboundMessage, mapping ErrorMapping) error {
	if err := s.base.SendError(ctx, message.ChatID, message.ThreadID, mapping.Key, mapping.Params...); err != nil {
		return fmt.Errorf("send error message: %w", err)
	}
	return nil
}

// SendLockError: 락 경합 시 에러 메시지를 발송합니다.
func (s *MessageSender) SendLockError(ctx context.Context, message mqmsg.InboundMessage, holderName *string) error {
	if err := s.base.SendLockError(ctx, message.ChatID, message.ThreadID, holderName); err != nil {
		return fmt.Errorf("send lock error message: %w", err)
	}
	return nil
}
