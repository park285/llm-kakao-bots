package mq

import (
	"log/slog"

	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
)

// StreamMessageHandler: commonmq.StreamMessageHandler alias
type StreamMessageHandler = commonmq.StreamMessageHandler

// NewStreamMessageHandler: 새로운 StreamMessageHandler 인스턴스를 생성합니다.
func NewStreamMessageHandler(gameMessageService *GameMessageService, logger *slog.Logger) *StreamMessageHandler {
	return commonmq.NewStreamMessageHandler(gameMessageService, logger)
}
