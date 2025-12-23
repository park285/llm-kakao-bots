package mq

import (
	"context"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/textutil"
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
	chunks := textutil.ChunkByLines(text, tsconfig.KakaoMessageMaxLength)
	if len(chunks) == 0 {
		return s.publish(ctx, mqmsg.NewFinal(message.ChatID, "", message.ThreadID))
	}

	for idx, chunk := range chunks {
		isLast := idx == len(chunks)-1
		if isLast {
			if err := s.publish(ctx, mqmsg.NewFinal(message.ChatID, chunk, message.ThreadID)); err != nil {
				return err
			}
			continue
		}
		if err := s.publish(ctx, mqmsg.NewWaiting(message.ChatID, chunk, message.ThreadID)); err != nil {
			return err
		}
	}
	return nil
}

// SendWaiting 는 동작을 수행한다.
func (s *MessageSender) SendWaiting(ctx context.Context, message mqmsg.InboundMessage, command Command) error {
	key := command.WaitingMessageKey()
	if key == nil {
		return nil
	}
	return s.publish(ctx, mqmsg.NewWaiting(message.ChatID, s.msgProvider.Get(*key), message.ThreadID))
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
