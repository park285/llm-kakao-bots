package processinglock

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

// DomainService: 도메인별 설정(키 함수, TTL)을 주입받아 공통 락 로직을 제공합니다.
// twentyq, turtlesoup 등 각 도메인에서 이 구조체를 사용하여 중복 코드를 제거합니다.
type DomainService struct {
	service *Service
}

// NewDomainService: 도메인별 설정으로 DomainService를 생성합니다.
func NewDomainService(
	client valkey.Client,
	logger *slog.Logger,
	keyFunc KeyFunc,
	ttl time.Duration,
) *DomainService {
	return &DomainService{
		service: New(client, logger, keyFunc, ttl),
	}
}

// StartProcessing: 처리 락을 획득합니다. 이미 처리 중이면 에러를 반환합니다.
func (s *DomainService) StartProcessing(ctx context.Context, chatID string) error {
	if err := WrapStartProcessingError(chatID, s.service.Start(ctx, chatID)); err != nil {
		return fmt.Errorf("start processing failed: %w", err)
	}
	return nil
}

// FinishProcessing: 처리 락을 해제합니다.
func (s *DomainService) FinishProcessing(ctx context.Context, chatID string) error {
	if err := WrapFinishProcessingError(s.service.Finish(ctx, chatID)); err != nil {
		return fmt.Errorf("finish processing failed: %w", err)
	}
	return nil
}

// IsProcessing: 현재 처리가 진행 중인지 확인합니다.
func (s *DomainService) IsProcessing(ctx context.Context, chatID string) (bool, error) {
	processing, err := s.service.IsProcessing(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("check processing failed: %w", WrapIsProcessingError(err))
	}
	return processing, nil
}
