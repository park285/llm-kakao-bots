package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/processinglock"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

// ProcessingLockService: 게임 메시지 처리 중복 방지를 위한 분산 락 서비스
type ProcessingLockService struct {
	service *processinglock.Service
	client  valkey.Client
}

// NewProcessingLockService: 새로운 ProcessingLockService 인스턴스를 생성한다.
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

// StartProcessing: 메시지 처리를 위한 락을 획득한다. 이미 처리 중이면 에러를 반환한다.
func (s *ProcessingLockService) StartProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Start(ctx, chatID); err != nil {
		return cerrors.RedisError{Operation: "processing_start", Err: err}
	}
	return nil
}

// FinishProcessing: 메시지 처리가 완료되면 락을 해제한다.
func (s *ProcessingLockService) FinishProcessing(ctx context.Context, chatID string) error {
	if err := s.service.Finish(ctx, chatID); err != nil {
		return cerrors.RedisError{Operation: "processing_finish", Err: err}
	}
	return nil
}

// IsProcessing: 현재 해당 채팅방에서 메시지가 처리 중인지 확인한다.
func (s *ProcessingLockService) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	processing, err := s.service.IsProcessing(ctx, chatID)
	if err != nil {
		return false, cerrors.RedisError{Operation: "processing_exists", Err: err}
	}
	return processing, nil
}

// compile-time check
var _ = valkeyx.IsNil
