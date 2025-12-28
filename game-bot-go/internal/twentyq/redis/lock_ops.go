package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lockutil"
	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func (m *LockManager) withLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat id is empty")
	}

	scope, ok := lockutil.ScopeFromContext(ctx)
	if !ok {
		scope = lockutil.NewScope()
		ctx = lockutil.WithScope(ctx, scope)
	}

	handled, err := m.handleReentry(ctx, scope, chatID, block)
	if handled {
		return err
	}

	token, err := lockutil.NewToken()
	if err != nil {
		return fmt.Errorf("generate lock token failed: %w", err)
	}

	ttlMillis := lockutil.TTLMillisFromSeconds(int64(qconfig.RedisLockTTLSeconds))

	acquired, acquireErr := m.acquireWithRetry(ctx, chatID, token, ttlMillis)
	if acquireErr != nil {
		return acquireErr
	}
	if !acquired {
		return cerrors.LockError{
			SessionID:   chatID,
			HolderName:  holderName,
			Description: "failed to acquire lock after retries",
		}
	}

	key := lockKey(chatID)

	renewCancel := m.startRenewWatchdog(ctx, chatID, token, ttlMillis)
	defer m.releaseIfLast(ctx, scope, key, chatID)

	scope.Set(key, lockutil.HeldLock{
		Token:     token,
		Count:     1,
		StopRenew: renewCancel,
	})

	m.logger.Debug("lock_acquired", "chat_id", chatID)
	return block(ctx)
}

func (m *LockManager) handleReentry(
	ctx context.Context,
	scope *lockutil.Scope,
	chatID string,
	block func(ctx context.Context) error,
) (handled bool, err error) {
	key := lockKey(chatID)
	if scope.IncrementIfHeld(key) {
		defer scope.Decrement(key)
		return true, block(ctx)
	}
	return false, nil
}

func (m *LockManager) releaseIfLast(ctx context.Context, scope *lockutil.Scope, key string, chatID string) {
	held, shouldRelease := scope.ReleaseIfLast(key)
	if !shouldRelease {
		return
	}

	if held.StopRenew != nil {
		held.StopRenew()
	}

	releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
	defer releaseCancel()

	if err := m.release(releaseCtx, chatID, held.Token); err != nil {
		m.logger.Warn("lock_release_failed", "err", err, "chat_id", chatID)
		return
	}
	m.logger.Debug("lock_released", "chat_id", chatID)
}

func (m *LockManager) acquire(ctx context.Context, chatID string, token string, ttlMillis int64) (bool, error) {
	writeKey := lockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockAcquire, []string{writeKey}, []string{token, ttlArg})
	if err != nil {
		return false, fmt.Errorf("lock acquire script missing: %w", err)
	}
	n, err := valkeyx.ParseLuaInt64(resp)
	if err != nil {
		return false, wrapRedisError("lock_acquire", err)
	}
	return n == 1, nil
}

const (
	lockRetryMaxAttempts   = 3
	lockRetryInitialDelay  = 50 * time.Millisecond
	lockRetryMaxDelay      = 500 * time.Millisecond
	lockRetryDelayMultiply = 2
)

func (m *LockManager) acquireWithRetry(
	ctx context.Context,
	chatID string,
	token string,
	ttlMillis int64,
) (bool, error) {
	delay := lockRetryInitialDelay

	for attempt := 0; attempt < lockRetryMaxAttempts; attempt++ {
		acquired, err := m.acquire(ctx, chatID, token, ttlMillis)
		if err != nil {
			return false, err
		}
		if acquired {
			if attempt > 0 {
				m.logger.Debug("lock_acquired_after_retry", "chat_id", chatID, "attempt", attempt+1)
			}
			return true, nil
		}

		if attempt == lockRetryMaxAttempts-1 {
			break
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return false, fmt.Errorf("lock acquire canceled: %w", ctx.Err())
		case <-timer.C:
			delay = min(delay*lockRetryDelayMultiply, lockRetryMaxDelay)
		}
	}

	m.logger.Debug("lock_acquire_failed_after_retries", "chat_id", chatID, "attempts", lockRetryMaxAttempts)
	return false, nil
}

func (m *LockManager) renew(ctx context.Context, chatID string, token string, ttlMillis int64) (bool, error) {
	writeKey := lockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockRenewWrite, []string{writeKey}, []string{token, ttlArg})
	if err != nil {
		return false, fmt.Errorf("lock renew script missing: %w", err)
	}
	n, err := valkeyx.ParseLuaInt64(resp)
	if err != nil {
		return false, wrapRedisError("lock_renew", err)
	}
	return n == 1, nil
}

func (m *LockManager) release(ctx context.Context, chatID string, token string) error {
	writeKey := lockKey(chatID)

	resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockRelease, []string{writeKey}, []string{token})
	if err != nil {
		return fmt.Errorf("lock release script missing: %w", err)
	}
	if err := resp.Error(); err != nil {
		return wrapRedisError("lock_release", err)
	}
	return nil
}

func wrapRedisError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", valkeyx.WrapRedisError(operation, err))
}

func (m *LockManager) startRenewWatchdog(ctx context.Context, chatID string, token string, ttlMillis int64) context.CancelFunc {
	const (
		minIntervalSeconds = 1
		divisor            = 3
	)

	intervalMillis := ttlMillis / divisor
	interval := time.Duration(intervalMillis) * time.Millisecond
	if interval < time.Duration(minIntervalSeconds)*time.Second {
		interval = time.Duration(minIntervalSeconds) * time.Second
	}

	renewCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				callCtx, callCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
				renewed, err := m.renew(callCtx, chatID, token, ttlMillis)
				callCancel()
				if err != nil {
					m.logger.Warn("lock_renew_failed", "chat_id", chatID, "err", err)
					return
				}
				if !renewed {
					m.logger.Warn("lock_renew_rejected", "chat_id", chatID)
					return
				}
			}
		}
	}()

	return cancel
}
