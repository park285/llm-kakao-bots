package redis

import (
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/processinglock"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// ProcessingLockService: 게임 메시지 처리 중복 방지를 위한 분산 락 서비스
// common/processinglock.DomainService의 별칭입니다.
type ProcessingLockService = processinglock.DomainService

// NewProcessingLockService: 새로운 ProcessingLockService 인스턴스를 생성합니다.
func NewProcessingLockService(client valkey.Client, logger *slog.Logger) *ProcessingLockService {
	return processinglock.NewDomainService(
		client,
		logger,
		processingKey,
		time.Duration(qconfig.RedisProcessingTTLSeconds)*time.Second,
	)
}
