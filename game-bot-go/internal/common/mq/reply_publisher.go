package mq

import (
	"context"
	"fmt"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
)

// ReplyPublisher 는 타입이다.
type ReplyPublisher struct {
	publisher *StreamPublisher
}

// NewReplyPublisher 는 동작을 수행한다.
func NewReplyPublisher(publisher *StreamPublisher) *ReplyPublisher {
	return &ReplyPublisher{publisher: publisher}
}

// Publish 는 동작을 수행한다.
func (p *ReplyPublisher) Publish(ctx context.Context, message mqmsg.OutboundMessage) error {
	if _, err := p.publisher.Publish(ctx, message.ToStreamValues()); err != nil {
		return fmt.Errorf("publish reply message failed: %w", err)
	}
	return nil
}
