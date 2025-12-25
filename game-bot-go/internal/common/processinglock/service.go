package processinglock

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

// KeyFunc: 채팅방 ID를 기반으로 락 키를 생성하는 함수 타입
type KeyFunc func(chatID string) string

// Service: Redis를 사용하여 동시 처리를 제어하는 락 서비스
type Service struct {
	client  valkey.Client
	logger  *slog.Logger
	keyFunc KeyFunc
	ttl     time.Duration
}

// New: 새로운 Service 인스턴스를 생성한다.
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

// Start: 처리 락을 설정한다. (이미 존재하더라도 덮어씀, TTL 갱신)
func (s *Service) Start(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Set().Key(key).Value("1").Ex(s.ttl).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("set processing lock failed: %w", err)
	}
	s.logger.Debug("processing_started", "chat_id", chatID)
	return nil
}

// Finish: 처리 락을 해제한다.
func (s *Service) Finish(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("delete processing lock failed: %w", err)
	}
	s.logger.Debug("processing_finished", "chat_id", chatID)
	return nil
}

// IsProcessing: 현재 처리가 진행 중인지(락이 존재하는지) 확인한다.
func (s *Service) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, fmt.Errorf("check processing lock exists failed: %w", err)
	}
	return n > 0, nil
}
