package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	qassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
)

// LockManager: Redis Lua 스크립트를 활용하여 배타적 락을 관리하는 컴포넌트
type LockManager struct {
	client valkey.Client
	logger *slog.Logger

	registry         *luautil.Registry
	redisCallTimeout time.Duration
}

// NewLockManager: 새로운 LockManager 인스턴스를 생성하고 Redis 클라이언트를 설정한다.
func NewLockManager(client valkey.Client, logger *slog.Logger) *LockManager {
	registry := luautil.NewRegistry([]luautil.Script{
		{Name: luautil.ScriptLockAcquire, Source: qassets.LockAcquireLua},
		{Name: luautil.ScriptLockRelease, Source: qassets.LockReleaseLua},
		{Name: luautil.ScriptLockRenewWrite, Source: qassets.LockRenewWriteLua},
	})
	if err := registry.Preload(context.Background(), client); err != nil && logger != nil {
		logger.Warn("lua_preload_failed", "component", "twentyq_lock_manager", "err", err)
	}
	return &LockManager{
		client:           client,
		logger:           logger,
		registry:         registry,
		redisCallTimeout: 5 * time.Second,
	}
}

// WithLock: 배타적 락을 획득한 상태에서 작업을 수행한다. (재진입 가능, Context Scope)
func (m *LockManager) WithLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error {
	return m.withLock(ctx, chatID, holderName, block)
}

// 락 획득 재시도 설정
// acquireWithRetry 락 획득을 exponential backoff로 재시도.
// 경합 상황에서 즉시 실패 대신 짧은 간격으로 재시도하여 성공률 향상.
