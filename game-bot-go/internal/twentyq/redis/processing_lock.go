package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/processinglock"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
)

// ProcessingLockService 는 타입이다.
type ProcessingLockService struct {
	service *processinglock.Service
	client  valkey.Client
}

// NewProcessingLockService 는 동작을 수행한다.
func NewProcessingLockService(client valkey.Client, logger *slog.Logger) *ProcessingLockService {
	return &ProcessingLockService{
		service: processinglock.New(
			client,
			logger,
			processingKey,
			time.Duration(qconfig.RedisProcessingTTLSeconds)*time.Second,
		),
		client: client,
	}
}

// StartProcessing 는 동작을 수행한다.
func (s *ProcessingLockService) StartProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Start(ctx, chatID); err != nil {
		return qerrors.RedisError{Operation: "processing_start", Err: err}
	}
	return nil
}

// FinishProcessing 는 동작을 수행한다.
func (s *ProcessingLockService) FinishProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Finish(ctx, chatID); err != nil {
		return qerrors.RedisError{Operation: "processing_finish", Err: err}
	}
	return nil
}

// IsProcessing 는 동작을 수행한다.
func (s *ProcessingLockService) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	processing, err := s.service.IsProcessing(ctx, chatID)
	if err != nil {
		return false, qerrors.RedisError{Operation: "processing_exists", Err: err}
	}
	return processing, nil
}

// compile-time check
var _ = valkeyx.IsNil
