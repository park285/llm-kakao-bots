package mq

import (
	"context"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/textutil"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// MessageSender: 게임 관련 메시지(응답, 대기, 에러 등)를 생성하고 발행(Publish)하는 컴포넌트
type MessageSender struct {
	msgProvider *messageprovider.Provider
	publish     func(ctx context.Context, msg mqmsg.OutboundMessage) error
}

// NewMessageSender: 주어진 발행 함수를 사용하여 새로운 MessageSender 인스턴스를 생성합니다.
func NewMessageSender(msgProvider *messageprovider.Provider, publish func(ctx context.Context, msg mqmsg.OutboundMessage) error) *MessageSender {
	return &MessageSender{
		msgProvider: msgProvider,
		publish:     publish,
	}
}

// SendFinal: 최종 처리 결과(답변)를 전송합니다. 메시지가 길 경우 분할 전송합니다.
func (s *MessageSender) SendFinal(ctx context.Context, message mqmsg.InboundMessage, text string) error {
	chunks := textutil.ChunkByLines(text, qconfig.KakaoMessageMaxLength)
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

// SendWaiting: 명령어 처리가 시작되었음을 알리는 대기 메시지(예: ~가 생각 중입니다)를 전송합니다.
func (s *MessageSender) SendWaiting(ctx context.Context, message mqmsg.InboundMessage, command Command) error {
	key := command.WaitingMessageKey()
	if key == nil {
		return nil
	}
	return s.publish(ctx, mqmsg.NewWaiting(message.ChatID, s.msgProvider.Get(*key), message.ThreadID))
}

// SendError: 발생한 에러에 매핑된 사용자 메시지를 전송합니다.
// 게임 관련 에러는 final 타입으로 전송하여 Iris의 에러 이모지 추가를 방지합니다.
func (s *MessageSender) SendError(ctx context.Context, message mqmsg.InboundMessage, mapping ErrorMapping) error {
	return s.publish(ctx, mqmsg.NewFinal(message.ChatID, s.msgProvider.Get(mapping.Key, mapping.Params...), message.ThreadID))
}

// SendLockError: 락 획득 실패(다른 요청 처리 중) 시 안내 메시지를 전송합니다.
func (s *MessageSender) SendLockError(ctx context.Context, message mqmsg.InboundMessage) error {
	return s.publish(ctx, mqmsg.NewError(message.ChatID, s.msgProvider.Get(qmessages.LockRequestInProgress), message.ThreadID))
}
