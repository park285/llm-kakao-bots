package mq

import (
	"context"
	"fmt"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
)

// ReplyPublisher: 봇의 응답 메시지를 전용 스트림으로 발행하는 발행자
type ReplyPublisher struct {
	publisher *StreamPublisher
}

// NewReplyPublisher: 새로운 ReplyPublisher 인스턴스를 생성합니다.
func NewReplyPublisher(publisher *StreamPublisher) *ReplyPublisher {
	return &ReplyPublisher{publisher: publisher}
}

// Publish: 응답 메시지를 출력 스트림에 발행합니다.
func (p *ReplyPublisher) Publish(ctx context.Context, message mqmsg.OutboundMessage) error {
	if _, err := p.publisher.Publish(ctx, message.ToStreamValues()); err != nil {
		return fmt.Errorf("publish reply message failed: %w", err)
	}
	return nil
}
