package mq

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
)

// InboundMessageHandler: 인바운드 메시지 처리를 위한 인터페이스
type InboundMessageHandler interface {
	HandleMessage(ctx context.Context, message mqmsg.InboundMessage)
}

// StreamMessageHandler: Redis 스트림으로부터 수신된 로우(Raw) 메시지를 파싱하여 비즈니스 로직 처리가 가능한 형태(mqmsg.InboundMessage)로 변환하고 처리 모듈로 전달합니다.
type StreamMessageHandler struct {
	handler InboundMessageHandler
	logger  *slog.Logger
}

// NewStreamMessageHandler: 새로운 StreamMessageHandler 인스턴스를 생성합니다.
func NewStreamMessageHandler(handler InboundMessageHandler, logger *slog.Logger) *StreamMessageHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StreamMessageHandler{
		handler: handler,
		logger:  logger,
	}
}

// HandleStreamMessage: XMessage(Redis Stream Message)를 받아 필수 필드(room, text 등)를 추출하여 InboundMessage로 변환한 뒤 핸들러에게 전달합니다.
func (h *StreamMessageHandler) HandleStreamMessage(ctx context.Context, message XMessage) error {
	fields := make(map[string]string, 5)
	for k, v := range message.Values {
		switch k {
		case "room", "text", "sender", "threadId", "userId":
			fields[k] = v
		}
	}

	inbound, err := mqmsg.ParseInboundMessage(fields)
	if err != nil {
		h.logger.Warn("message_parsing_failed", "id", message.ID, "err", err)
		return nil
	}

	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logger.Debug("message_received", "id", message.ID, "chat_id", inbound.ChatID, "user_id", inbound.UserID)
	}
	if h.handler != nil {
		h.handler.HandleMessage(ctx, inbound)
	}
	return nil
}
