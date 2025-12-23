package mq

import (
	"log/slog"

	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
)

// StreamMessageHandler 는 타입이다.
type StreamMessageHandler = commonmq.StreamMessageHandler

// NewStreamMessageHandler 는 동작을 수행한다.
func NewStreamMessageHandler(gameMessageService *GameMessageService, logger *slog.Logger) *StreamMessageHandler {
	return commonmq.NewStreamMessageHandler(gameMessageService, logger)
}
