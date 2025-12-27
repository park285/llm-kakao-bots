package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lockutil"
	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	qassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
)

type lockMode int

const (
	lockModeWrite lockMode = iota
	lockModeRead
)

func (m lockMode) String() string {
	switch m {
	case lockModeWrite:
		return "WRITE"
	case lockModeRead:
		return "READ"
	default:
		return "UNKNOWN"
	}
}

// LockManager: Redis Lua 스크립트를 활용하여 읽기/쓰기 분산 락을 관리하는 컴포넌트
type LockManager struct {
	client valkey.Client
	logger *slog.Logger

	registry         *luautil.Registry
	redisCallTimeout time.Duration
}

// NewLockManager: 새로운 LockManager 인스턴스를 생성하고 Redis 클라이언트를 설정한다.
func NewLockManager(client valkey.Client, logger *slog.Logger) *LockManager {
	registry := luautil.NewRegistry([]luautil.Script{
		{Name: luautil.ScriptLockAcquireRead, Source: qassets.LockAcquireReadLua},
		{Name: luautil.ScriptLockAcquireWrite, Source: qassets.LockAcquireWriteLua},
		{Name: luautil.ScriptLockRelease, Source: qassets.LockReleaseLua},
		{Name: luautil.ScriptLockReleaseRead, Source: qassets.LockReleaseReadLua},
		{Name: luautil.ScriptLockRenewRead, Source: qassets.LockRenewReadLua},
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

// TryAcquireSharedLock: 공유 락(읽기 락 성격) 획득을 시도한다. (1회 시도, SET NX 사용)
func (m *LockManager) TryAcquireSharedLock(ctx context.Context, lockKey string, ttlSeconds int64) (bool, error) {
	acquired, err := lockutil.TryAcquireSharedLock(ctx, m.client, lockKey, ttlSeconds)
	if err != nil {
		return false, fmt.Errorf("try acquire shared lock: %w", err)
	}
	return acquired, nil
}

// ReleaseSharedLock: 공유 락을 해제한다. (DEL 사용)
func (m *LockManager) ReleaseSharedLock(ctx context.Context, lockKey string) error {
	if err := lockutil.ReleaseSharedLock(ctx, m.client, lockKey); err != nil {
		return fmt.Errorf("release shared lock: %w", err)
	}
	return nil
}

// WithLock: 배타적 락(Write Lock)을 획득한 상태에서 작업을 수행한다. (재진입 가능, Context Scope)
func (m *LockManager) WithLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error {
	return m.withRWLock(ctx, chatID, holderName, lockModeWrite, block)
}

// WithReadLock: 공유 락(Read Lock)을 획득한 상태에서 작업을 수행한다. (재진입 가능, Context Scope)
func (m *LockManager) WithReadLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error {
	return m.withRWLock(ctx, chatID, holderName, lockModeRead, block)
}

// 락 획득 재시도 설정
// acquireWithRetry 락 획득을 exponential backoff로 재시도.
// 경합 상황에서 즉시 실패 대신 짧은 간격으로 재시도하여 성공률 향상.
