package mq

import (
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
)

// ReplyPublisher: commonmq.ReplyPublisher alias
type ReplyPublisher = commonmq.ReplyPublisher

// NewReplyPublisher: 새로운 ReplyPublisher 인스턴스를 생성한다.
func NewReplyPublisher(publisher *commonmq.StreamPublisher) *ReplyPublisher {
	return commonmq.NewReplyPublisher(publisher)
}
