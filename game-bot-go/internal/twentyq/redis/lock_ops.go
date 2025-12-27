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

func (m *LockManager) withRWLock(ctx context.Context, chatID string, holderName *string, mode lockMode, block func(ctx context.Context) error) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat id is empty")
	}

	scope, ok := lockutil.ScopeFromContext(ctx)
	if !ok {
		scope = lockutil.NewScope()
		ctx = lockutil.WithScope(ctx, scope)
	}

	handled, err := m.handleReentry(ctx, scope, chatID, holderName, mode, block)
	if handled {
		return err
	}

	token, err := lockutil.NewToken()
	if err != nil {
		return fmt.Errorf("generate lock token failed: %w", err)
	}

	ttlMillis := lockutil.TTLMillisFromSeconds(int64(qconfig.RedisLockTTLSeconds))

	// 락 획득 재시도 (exponential backoff)
	acquired, acquireErr := m.acquireWithRetry(ctx, chatID, mode, token, ttlMillis)
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

	key, err := scopeKey(chatID, mode)
	if err != nil {
		return err
	}

	renewCancel := m.startRenewWatchdog(ctx, chatID, mode, token, ttlMillis)
	defer m.releaseIfLast(ctx, scope, key, chatID, ttlMillis)

	scope.Set(key, lockutil.HeldLock{
		Mode:      int(mode),
		Token:     token,
		Count:     1,
		StopRenew: renewCancel,
	})

	m.logger.Debug("lock_acquired", "chat_id", chatID, "mode", mode.String())
	return block(ctx)
}

func (m *LockManager) handleReentry(
	ctx context.Context,
	scope *lockutil.Scope,
	chatID string,
	holderName *string,
	mode lockMode,
	block func(ctx context.Context) error,
) (handled bool, err error) {
	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)

	switch mode {
	case lockModeWrite:
		if scope.IncrementIfHeld(writeKey) {
			defer scope.Decrement(writeKey)
			return true, block(ctx)
		}
		if scope.IsHeld(readKey) {
			return true, cerrors.LockError{
				SessionID:   chatID,
				HolderName:  holderName,
				Description: "write lock requested while read lock held",
			}
		}
		return false, nil
	case lockModeRead:
		if scope.IncrementIfHeld(writeKey) {
			defer scope.Decrement(writeKey)
			return true, block(ctx)
		}
		if scope.IncrementIfHeld(readKey) {
			defer scope.Decrement(readKey)
			return true, block(ctx)
		}
		return false, nil
	default:
		return true, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) releaseIfLast(ctx context.Context, scope *lockutil.Scope, key string, chatID string, ttlMillis int64) {
	held, shouldRelease := scope.ReleaseIfLast(key)
	if !shouldRelease {
		return
	}

	if held.StopRenew != nil {
		held.StopRenew()
	}

	releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
	defer releaseCancel()

	if err := m.release(releaseCtx, chatID, lockMode(held.Mode), held.Token, ttlMillis); err != nil {
		m.logger.Warn("lock_release_failed", "err", err, "chat_id", chatID)
		return
	}
	m.logger.Debug("lock_released", "chat_id", chatID)
}

func scopeKey(chatID string, mode lockMode) (string, error) {
	switch mode {
	case lockModeWrite:
		return lockKey(chatID), nil
	case lockModeRead:
		return readLockKey(chatID), nil
	default:
		return "", fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) acquire(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) (bool, error) {
	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockAcquireWrite, []string{writeKey, readKey}, []string{token, ttlArg})
		if err != nil {
			return false, fmt.Errorf("lock acquire write script missing: %w", err)
		}
		n, err := valkeyx.ParseLuaInt64(resp)
		if err != nil {
			return false, wrapRedisError("lock_acquire_write", err)
		}
		return n == 1, nil
	case lockModeRead:
		resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockAcquireRead, []string{writeKey, readKey}, []string{token, ttlArg})
		if err != nil {
			return false, fmt.Errorf("lock acquire read script missing: %w", err)
		}
		n, err := valkeyx.ParseLuaInt64(resp)
		if err != nil {
			return false, wrapRedisError("lock_acquire_read", err)
		}
		return n == 1, nil
	default:
		return false, fmt.Errorf("unknown lock mode: %d", mode)
	}
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
	mode lockMode,
	token string,
	ttlMillis int64,
) (bool, error) {
	delay := lockRetryInitialDelay

	for attempt := 0; attempt < lockRetryMaxAttempts; attempt++ {
		acquired, err := m.acquire(ctx, chatID, mode, token, ttlMillis)
		if err != nil {
			return false, err
		}
		if acquired {
			if attempt > 0 {
				m.logger.Debug("lock_acquired_after_retry", "chat_id", chatID, "mode", mode.String(), "attempt", attempt+1)
			}
			return true, nil
		}

		// 마지막 시도면 재시도 없이 종료
		if attempt == lockRetryMaxAttempts-1 {
			break
		}

		// Context 취소 확인 후 대기
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return false, fmt.Errorf("lock acquire canceled: %w", ctx.Err())
		case <-timer.C:
			// 다음 재시도를 위해 delay 증가 (exponential backoff)
			delay = min(delay*lockRetryDelayMultiply, lockRetryMaxDelay)
		}
	}

	m.logger.Debug("lock_acquire_failed_after_retries", "chat_id", chatID, "mode", mode.String(), "attempts", lockRetryMaxAttempts)
	return false, nil
}

func (m *LockManager) renew(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) (bool, error) {
	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockRenewWrite, []string{writeKey}, []string{token, ttlArg})
		if err != nil {
			return false, fmt.Errorf("lock renew write script missing: %w", err)
		}
		n, err := valkeyx.ParseLuaInt64(resp)
		if err != nil {
			return false, wrapRedisError("lock_renew_write", err)
		}
		return n == 1, nil
	case lockModeRead:
		resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockRenewRead, []string{readKey}, []string{token, ttlArg})
		if err != nil {
			return false, fmt.Errorf("lock renew read script missing: %w", err)
		}
		n, err := valkeyx.ParseLuaInt64(resp)
		if err != nil {
			return false, wrapRedisError("lock_renew_read", err)
		}
		return n == 1, nil
	default:
		return false, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) release(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) error {
	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockRelease, []string{writeKey}, []string{token})
		if err != nil {
			return fmt.Errorf("lock release write script missing: %w", err)
		}
		if err := resp.Error(); err != nil {
			return wrapRedisError("lock_release_write", err)
		}
		return nil
	case lockModeRead:
		resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptLockReleaseRead, []string{readKey}, []string{token, ttlArg})
		if err != nil {
			return fmt.Errorf("lock release read script missing: %w", err)
		}
		if err := resp.Error(); err != nil {
			return wrapRedisError("lock_release_read", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func wrapRedisError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", valkeyx.WrapRedisError(operation, err))
}

func (m *LockManager) startRenewWatchdog(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) context.CancelFunc {
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
				renewed, err := m.renew(callCtx, chatID, mode, token, ttlMillis)
				callCancel()
				if err != nil {
					m.logger.Warn("lock_renew_failed", "chat_id", chatID, "mode", mode.String(), "err", err)
					return
				}
				if !renewed {
					m.logger.Warn("lock_renew_rejected", "chat_id", chatID, "mode", mode.String())
					return
				}
			}
		}
	}()

	return cancel
}
