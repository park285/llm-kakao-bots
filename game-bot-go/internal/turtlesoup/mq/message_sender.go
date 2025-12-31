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

// MessageSender 는 타입이다.
type MessageSender struct {
	msgProvider *messageprovider.Provider
	publish     func(ctx context.Context, msg mqmsg.OutboundMessage) error
}

// NewMessageSender 는 동작을 수행한다.
func NewMessageSender(msgProvider *messageprovider.Provider, publish func(ctx context.Context, msg mqmsg.OutboundMessage) error) *MessageSender {
	return &MessageSender{
		msgProvider: msgProvider,
		publish:     publish,
	}
}

// SendFinal 는 동작을 수행한다.
func (s *MessageSender) SendFinal(ctx context.Context, message mqmsg.InboundMessage, text string) error {
	if err := commonmq.SendFinalChunked(ctx, s.publish, message.ChatID, text, message.ThreadID, tsconfig.KakaoMessageMaxLength); err != nil {
		return fmt.Errorf("send final failed: %w", err)
	}
	return nil
}

// SendWaiting 는 동작을 수행한다.
func (s *MessageSender) SendWaiting(ctx context.Context, message mqmsg.InboundMessage, command Command) error {
	if err := commonmq.SendWaitingFromCommand(ctx, s.publish, s.msgProvider, message.ChatID, message.ThreadID, command); err != nil {
		return fmt.Errorf("send waiting failed: %w", err)
	}
	return nil
}

// SendError 는 동작을 수행한다.
func (s *MessageSender) SendError(ctx context.Context, message mqmsg.InboundMessage, mapping ErrorMapping) error {
	return s.publish(ctx, mqmsg.NewError(message.ChatID, s.msgProvider.Get(mapping.Key, mapping.Params...), message.ThreadID))
}

// SendLockError 는 동작을 수행한다.
func (s *MessageSender) SendLockError(ctx context.Context, message mqmsg.InboundMessage, holderName *string) error {
	if holderName != nil && *holderName != "" {
		text := s.msgProvider.Get(tsmessages.LockRequestInProgressWithHolder, messageprovider.P("holder", *holderName))
		return s.publish(ctx, mqmsg.NewError(message.ChatID, text, message.ThreadID))
	}

	text := s.msgProvider.Get(tsmessages.LockRequestInProgress)
	return s.publish(ctx, mqmsg.NewError(message.ChatID, text, message.ThreadID))
}
