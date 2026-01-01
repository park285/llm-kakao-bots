package mq

import (
	"context"
	"fmt"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// MessageSender: 게임 관련 메시지(응답, 대기, 에러 등)를 생성하고 발행(Publish)하는 컴포넌트
// 공통 BaseMessageSender를 내부적으로 사용하여 핵심 로직을 위임합니다.
type MessageSender struct {
	base *commonmq.BaseMessageSender
}

// NewMessageSender: 주어진 발행 함수를 사용하여 새로운 MessageSender 인스턴스를 생성합니다.
func NewMessageSender(msgProvider *messageprovider.Provider, publish func(ctx context.Context, msg mqmsg.OutboundMessage) error) *MessageSender {
	return &MessageSender{
		base: commonmq.NewBaseMessageSender(msgProvider, publish, commonmq.MessageSenderConfig{
			MessageMaxLength:  qconfig.KakaoMessageMaxLength,
			LockErrorKey:      qmessages.LockRequestInProgress,
			UseFinalForErrors: true, // twentyq는 게임 에러를 Final로 전송
		}),
	}
}

// SendFinal: 최종 처리 결과(답변)를 전송합니다. 메시지가 길 경우 분할 전송합니다.
func (s *MessageSender) SendFinal(ctx context.Context, message mqmsg.InboundMessage, text string) error {
	if err := s.base.SendFinal(ctx, message.ChatID, text, message.ThreadID); err != nil {
		return fmt.Errorf("send final message: %w", err)
	}
	return nil
}

// SendWaiting: 명령어 처리가 시작되었음을 알리는 대기 메시지(예: ~가 생각 중입니다)를 전송합니다.
func (s *MessageSender) SendWaiting(ctx context.Context, message mqmsg.InboundMessage, command Command) error {
	if err := s.base.SendWaiting(ctx, message.ChatID, message.ThreadID, command); err != nil {
		return fmt.Errorf("send waiting message: %w", err)
	}
	return nil
}

// SendError: 발생한 에러에 매핑된 사용자 메시지를 전송합니다.
// 게임 관련 에러는 final 타입으로 전송하여 Iris의 에러 이모지 추가를 방지합니다.
func (s *MessageSender) SendError(ctx context.Context, message mqmsg.InboundMessage, mapping ErrorMapping) error {
	if err := s.base.SendError(ctx, message.ChatID, message.ThreadID, mapping.Key, mapping.Params...); err != nil {
		return fmt.Errorf("send error message: %w", err)
	}
	return nil
}

// SendLockError: 락 획득 실패(다른 요청 처리 중) 시 안내 메시지를 전송합니다.
func (s *MessageSender) SendLockError(ctx context.Context, message mqmsg.InboundMessage) error {
	if err := s.base.SendLockError(ctx, message.ChatID, message.ThreadID, nil); err != nil {
		return fmt.Errorf("send lock error message: %w", err)
	}
	return nil
}
