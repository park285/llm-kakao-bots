package mq

import (
	"log/slog"

	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
)

// StreamMessageHandler: commonmq.StreamMessageHandler alias
type StreamMessageHandler = commonmq.StreamMessageHandler

// NewStreamMessageHandler: 공통 스트림 핸들러를 바다거북 스프 서비스용으로 생성합니다.
func NewStreamMessageHandler(gameMessageService *GameMessageService, logger *slog.Logger) *StreamMessageHandler {
	return commonmq.NewStreamMessageHandler(gameMessageService, logger)
}
