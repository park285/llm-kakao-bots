package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/processinglock"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

// ProcessingLockService: 채팅방 단위의 처리 중 상태를 관리하는 락 서비스 (common/processinglock 래퍼)
type ProcessingLockService struct {
	service *processinglock.Service
}

// NewProcessingLockService: 새로운 ProcessingLockService 인스턴스를 생성한다.
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

// StartProcessing: 처리를 시작하고 락을 설정한다.
func (s *ProcessingLockService) StartProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Start(ctx, chatID); err != nil {
		return cerrors.RedisError{Operation: "processing_start", Err: err}
	}
	return nil
}

// FinishProcessing: 처리를 완료하고 락을 해제한다.
func (s *ProcessingLockService) FinishProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Finish(ctx, chatID); err != nil {
		return cerrors.RedisError{Operation: "processing_finish", Err: err}
	}
	return nil
}

// IsProcessing: 현재 처리가 진행 중인지 확인한다.
func (s *ProcessingLockService) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	processing, err := s.service.IsProcessing(ctx, chatID)
	if err != nil {
		return false, cerrors.RedisError{Operation: "processing_exists", Err: err}
	}
	return processing, nil
}
