package mq

import (
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
)

// ReplyPublisher 는 타입이다.
type ReplyPublisher = commonmq.ReplyPublisher

// NewReplyPublisher 는 동작을 수행한다.
func NewReplyPublisher(publisher *commonmq.StreamPublisher) *ReplyPublisher {
	return commonmq.NewReplyPublisher(publisher)
}
