package processinglock

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

// KeyFunc 는 타입이다.
type KeyFunc func(chatID string) string

// Service 는 타입이다.
type Service struct {
	client  valkey.Client
	logger  *slog.Logger
	keyFunc KeyFunc
	ttl     time.Duration
}

// New 는 동작을 수행한다.
func New(client valkey.Client, logger *slog.Logger, keyFunc KeyFunc, ttl time.Duration) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		client:  client,
		logger:  logger,
		keyFunc: keyFunc,
		ttl:     ttl,
	}
}

// Start 는 동작을 수행한다.
func (s *Service) Start(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Set().Key(key).Value("1").Ex(s.ttl).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("set processing lock failed: %w", err)
	}
	s.logger.Debug("processing_started", "chat_id", chatID)
	return nil
}

// Finish 는 동작을 수행한다.
func (s *Service) Finish(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("delete processing lock failed: %w", err)
	}
	s.logger.Debug("processing_finished", "chat_id", chatID)
	return nil
}

// IsProcessing 는 동작을 수행한다.
func (s *Service) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, fmt.Errorf("check processing lock exists failed: %w", err)
	}
	return n > 0, nil
}
