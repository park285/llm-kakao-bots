package mq

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/valkey-io/valkey-go"
)

// StreamPublisherConfig 는 타입이다.
type StreamPublisherConfig struct {
	Stream string

	MaxLen int64
}

// StreamPublisher 는 타입이다.
type StreamPublisher struct {
	client valkey.Client
	logger *slog.Logger
	cfg    StreamPublisherConfig
}

// NewStreamPublisher 는 동작을 수행한다.
func NewStreamPublisher(client valkey.Client, logger *slog.Logger, cfg StreamPublisherConfig) *StreamPublisher {
	return &StreamPublisher{
		client: client,
		logger: logger,
		cfg:    cfg,
	}
}

// Publish 는 동작을 수행한다.
func (p *StreamPublisher) Publish(ctx context.Context, values map[string]any) (string, error) {
	// Build field-value pairs
	fieldValues := make([]string, 0, len(values)*2)
	for k, v := range values {
		fieldValues = append(fieldValues, k, fmt.Sprint(v))
	}

	if len(fieldValues) < 2 {
		return "", fmt.Errorf("no values to publish")
	}

	// Use Arbitrary command for flexibility with MAXLEN ~
	var args []string
	if p.cfg.MaxLen > 0 {
		args = append(args, "MAXLEN", "~", fmt.Sprintf("%d", p.cfg.MaxLen))
	}
	args = append(args, "*")
	args = append(args, fieldValues...)

	cmd := p.client.B().Arbitrary("XADD").Keys(p.cfg.Stream).Args(args...).Build()

	id, err := p.client.Do(ctx, cmd).ToString()
	if err != nil {
		return "", fmt.Errorf("xadd failed stream=%s: %w", p.cfg.Stream, err)
	}

	p.logger.Debug("message_published", "stream", p.cfg.Stream, "id", id)
	return id, nil
}
