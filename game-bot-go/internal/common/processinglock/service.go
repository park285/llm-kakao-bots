package processinglock

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

type KeyFunc func(chatID string) string

// ErrAlreadyProcessing: 이미 해당 채팅방에서 처리가 진행 중일 때 반환되는 에러
var ErrAlreadyProcessing = errors.New("already processing")

// Service: Redis를 사용하여 동시 처리를 제어하는 락 서비스
type Service struct {
	client  valkey.Client
	logger  *slog.Logger
	keyFunc KeyFunc
	ttl     time.Duration
}

// New: 새로운 Service 인스턴스를 생성합니다.
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

// Start: 처리 락을 획득합니다. (SET NX)
// 이미 락이 존재하면 ErrAlreadyProcessing 을 반환합니다.
func (s *Service) Start(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Set().Key(key).Value("1").Nx().Ex(s.ttl).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		if valkeyx.IsNil(err) {
			return ErrAlreadyProcessing
		}
		return fmt.Errorf("set processing lock failed: %w", err)
	}
	s.logger.Debug("processing_started", "chat_id", chatID)
	return nil
}

// Finish: 처리 락을 해제합니다.
func (s *Service) Finish(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("delete processing lock failed: %w", err)
	}
	s.logger.Debug("processing_finished", "chat_id", chatID)
	return nil
}

// IsProcessing: 현재 처리가 진행 중인지(락이 존재하는지) 확인합니다.
func (s *Service) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	key := s.keyFunc(chatID)
	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, fmt.Errorf("check processing lock exists failed: %w", err)
	}
	return n > 0, nil
}

func WrapStartProcessingError(chatID string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrAlreadyProcessing) {
		return cerrors.LockError{SessionID: chatID, Description: "already processing"}
	}
	return cerrors.RedisError{Operation: "processing_start", Err: err}
}

func WrapFinishProcessingError(err error) error {
	if err == nil {
		return nil
	}
	return cerrors.RedisError{Operation: "processing_finish", Err: err}
}

func WrapIsProcessingError(err error) error {
	if err == nil {
		return nil
	}
	return cerrors.RedisError{Operation: "processing_exists", Err: err}
}
