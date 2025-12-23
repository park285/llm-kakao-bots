package mq

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
)

// InboundMessageHandler 는 타입이다.
type InboundMessageHandler interface {
	HandleMessage(ctx context.Context, message mqmsg.InboundMessage)
}

// StreamMessageHandler 는 타입이다.
type StreamMessageHandler struct {
	handler InboundMessageHandler
	logger  *slog.Logger
}

// NewStreamMessageHandler 는 동작을 수행한다.
func NewStreamMessageHandler(handler InboundMessageHandler, logger *slog.Logger) *StreamMessageHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StreamMessageHandler{
		handler: handler,
		logger:  logger,
	}
}

// HandleStreamMessage 는 동작을 수행한다.
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
