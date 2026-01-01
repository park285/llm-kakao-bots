package mq

import (
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
)

// ReplyPublisher: commonmq.ReplyPublisher alias
type ReplyPublisher = commonmq.ReplyPublisher

// NewReplyPublisher: 공통 응답 발행자를 바다거북 스프 서비스용으로 생성합니다.
func NewReplyPublisher(publisher *commonmq.StreamPublisher) *ReplyPublisher {
	return commonmq.NewReplyPublisher(publisher)
}
