package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/processinglock"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
)

// ProcessingLockService 는 타입이다.
type ProcessingLockService struct {
	service *processinglock.Service
}

// NewProcessingLockService 는 동작을 수행한다.
func NewProcessingLockService(client valkey.Client, logger *slog.Logger) *ProcessingLockService {
	return &ProcessingLockService{
		service: processinglock.New(
			client,
			logger,
			processingKey,
			time.Duration(tsconfig.RedisProcessingTTLSeconds)*time.Second,
		),
	}
}

// StartProcessing 는 동작을 수행한다.
func (s *ProcessingLockService) StartProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Start(ctx, chatID); err != nil {
		return tserrors.RedisError{Operation: "processing_start", Err: err}
	}
	return nil
}

// FinishProcessing 는 동작을 수행한다.
func (s *ProcessingLockService) FinishProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Finish(ctx, chatID); err != nil {
		return tserrors.RedisError{Operation: "processing_finish", Err: err}
	}
	return nil
}

// IsProcessing 는 동작을 수행한다.
func (s *ProcessingLockService) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	processing, err := s.service.IsProcessing(ctx, chatID)
	if err != nil {
		return false, tserrors.RedisError{Operation: "processing_exists", Err: err}
	}
	return processing, nil
}
