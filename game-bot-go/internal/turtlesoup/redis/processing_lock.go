package redis

import (
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/processinglock"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

// ProcessingLockService: 채팅방 단위의 처리 중 상태를 관리하는 락 서비스
// common/processinglock.DomainService의 별칭입니다.
type ProcessingLockService = processinglock.DomainService

// NewProcessingLockService: 새로운 ProcessingLockService 인스턴스를 생성합니다.
func NewProcessingLockService(client valkey.Client, logger *slog.Logger) *ProcessingLockService {
	return processinglock.NewDomainService(
		client,
		logger,
		processingKey,
		time.Duration(tsconfig.RedisProcessingTTLSeconds)*time.Second,
	)
}
