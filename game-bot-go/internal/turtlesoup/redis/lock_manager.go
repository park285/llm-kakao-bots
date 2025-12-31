package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lockutil"
	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	tsassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/assets"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

// LockManager: Redis와 Lua 스크립트를 사용하여 분산 락(Distributed Lock)을 구현한 관리자
// Reentrancy(재진입)를 지원하며, Context 스코프 기반의 자동 락 관리를 제공합니다.
type LockManager struct {
	client valkey.Client
	logger *slog.Logger

	registry         *luautil.Registry
	redisCallTimeout time.Duration
}

// NewLockManager: 새로운 LockManager 인스턴스를 생성합니다.
func NewLockManager(client valkey.Client, logger *slog.Logger) *LockManager {
	registry := luautil.NewRegistry([]luautil.Script{
		{Name: luautil.ScriptTurtleLockRelease, Source: tsassets.LockReleaseLua},
	})

	preloadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := registry.Preload(preloadCtx, client); err != nil && logger != nil {
		logger.Warn("lua_preload_failed", "component", "turtlesoup_lock_manager", "err", err)
	}
	return &LockManager{
		client:           client,
		logger:           logger,
		registry:         registry,
		redisCallTimeout: 5 * time.Second,
	}
}

// TryAcquireSharedLock: 여러 사용자가 동시에 접근 가능한 '공유 락(Shared Lock)' 획득을 시도합니다. (주로 읽기 작업용)
// 이미 락이 존재하더라도 획득 성공으로 간주하거나, 별도의 키를 사용하여 동시성을 제어합니다.
func (m *LockManager) TryAcquireSharedLock(ctx context.Context, lockKey string, ttlSeconds int64) (bool, error) {
	acquired, err := lockutil.TryAcquireSharedLock(ctx, m.client, lockKey, ttlSeconds)
	if err != nil {
		return false, fmt.Errorf("try acquire shared lock: %w", err)
	}
	return acquired, nil
}

// ReleaseSharedLock: 공유 락을 해제합니다. (DEL)
func (m *LockManager) ReleaseSharedLock(ctx context.Context, lockKey string) error {
	if err := lockutil.ReleaseSharedLock(ctx, m.client, lockKey); err != nil {
		return fmt.Errorf("release shared lock: %w", err)
	}
	return nil
}

// WithLock: 배타적 락(Write Lock)을 획득한 상태에서 주어진 함수(block)를 실행합니다.
// 락 획득 실패 시 에러를 반환하며, 실행 완료 후 자동으로 락을 해제합니다. 재진입(Reentry)을 지원합니다.
func (m *LockManager) WithLock(ctx context.Context, sessionID string, holderName *string, block func(ctx context.Context) error) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id is empty")
	}

	scope, ok := lockutil.ScopeFromContext(ctx)
	if !ok {
		scope = lockutil.NewScope()
		ctx = lockutil.WithScope(ctx, scope)
	}

	key := lockKey(sessionID)
	if scope.IncrementIfHeld(key) {
		defer scope.Decrement(key)
		return block(ctx)
	}

	timeoutSeconds := int64(tsconfig.RedisLockTimeoutSeconds)
	lockTTLDuration := lockutil.TTLDurationFromSeconds(int64(tsconfig.RedisLockTTLSeconds))

	acquireCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	token, err := lockutil.NewToken()
	if err != nil {
		return fmt.Errorf("generate lock token failed: %w", err)
	}

	holderValue := buildHolderValue(token, holderName)

	m.logger.Debug("lock_attempting", "session_id", sessionID, "timeout_seconds", timeoutSeconds)

	acquired, acquireErr := m.acquire(acquireCtx, sessionID, token, holderValue, lockTTLDuration)
	if acquireErr != nil {
		return acquireErr
	}
	if !acquired {
		currentHolder, holderErr := m.getHolder(ctx, sessionID)
		if holderErr != nil {
			m.logger.Warn("lock_holder_read_failed", "err", holderErr, "session_id", sessionID)
		}
		return cerrors.LockError{
			SessionID:   sessionID,
			HolderName:  currentHolder,
			Description: "failed to acquire lock",
		}
	}

	scope.Set(key, lockutil.HeldLock{Token: token, Count: 1})
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
		defer releaseCancel()

		releaseToken, shouldRelease := scope.ReleaseIfLast(key)
		if !shouldRelease {
			return
		}
		if err := m.release(releaseCtx, sessionID, releaseToken.Token); err != nil {
			m.logger.Warn("lock_release_failed", "err", err, "session_id", sessionID)
		} else {
			m.logger.Debug("lock_released", "session_id", sessionID)
		}
	}()

	m.logger.Debug("lock_acquired", "session_id", sessionID)
	return block(ctx)
}

// 락 획득 재시도 설정
// acquire: Redis의 SET NX 명령과 고유 토큰을 사용하여 락 획득을 시도합니다.
// 실패 시 지수 백오프(Exponential Backoff) 전략으로 재시도합니다.
