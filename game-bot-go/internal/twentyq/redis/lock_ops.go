package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lockutil"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func (m *LockManager) withRWLock(ctx context.Context, chatID string, holderName *string, mode lockMode, block func(ctx context.Context) error) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat id is empty")
	}

	scope, ok := lockScopeFromContext(ctx)
	if !ok {
		scope = newLockScope()
		ctx = withLockScope(ctx, scope)
	}

	handled, err := m.handleReentry(ctx, scope, chatID, holderName, mode, block)
	if handled {
		return err
	}

	token, err := lockutil.NewToken()
	if err != nil {
		return fmt.Errorf("generate lock token failed: %w", err)
	}

	ttlMillis := int64(qconfig.RedisLockTTLSeconds) * 1000

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

	scope.set(key, heldLock{
		mode:      mode,
		token:     token,
		count:     1,
		stopRenew: renewCancel,
	})

	m.logger.Debug("lock_acquired", "chat_id", chatID, "mode", mode.String())
	return block(ctx)
}

func (m *LockManager) handleReentry(
	ctx context.Context,
	scope *lockScope,
	chatID string,
	holderName *string,
	mode lockMode,
	block func(ctx context.Context) error,
) (handled bool, err error) {
	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)

	switch mode {
	case lockModeWrite:
		if scope.incrementIfHeld(writeKey) {
			defer scope.decrement(writeKey)
			return true, block(ctx)
		}
		if scope.isHeld(readKey) {
			return true, cerrors.LockError{
				SessionID:   chatID,
				HolderName:  holderName,
				Description: "write lock requested while read lock held",
			}
		}
		return false, nil
	case lockModeRead:
		if scope.incrementIfHeld(writeKey) {
			defer scope.decrement(writeKey)
			return true, block(ctx)
		}
		if scope.incrementIfHeld(readKey) {
			defer scope.decrement(readKey)
			return true, block(ctx)
		}
		return false, nil
	default:
		return true, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) releaseIfLast(ctx context.Context, scope *lockScope, key string, chatID string, ttlMillis int64) {
	held, shouldRelease := scope.releaseIfLast(key)
	if !shouldRelease {
		return
	}

	if held.stopRenew != nil {
		held.stopRenew()
	}

	releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
	defer releaseCancel()

	if err := m.release(releaseCtx, chatID, held.mode, held.token, ttlMillis); err != nil {
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
	if err := m.loadScripts(ctx); err != nil {
		return false, err
	}

	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		cmd := m.client.B().Evalsha().Sha1(m.acquireWriteSHA).Numkeys(2).Key(writeKey, readKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.acquire(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_acquire_write", Err: err}
		}
		return n == 1, nil
	case lockModeRead:
		cmd := m.client.B().Evalsha().Sha1(m.acquireReadSHA).Numkeys(2).Key(writeKey, readKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.acquire(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_acquire_read", Err: err}
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
	if err := m.loadScripts(ctx); err != nil {
		return false, err
	}

	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		cmd := m.client.B().Evalsha().Sha1(m.renewWriteSHA).Numkeys(1).Key(writeKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.renew(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_renew_write", Err: err}
		}
		return n == 1, nil
	case lockModeRead:
		cmd := m.client.B().Evalsha().Sha1(m.renewReadSHA).Numkeys(1).Key(readKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.renew(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_renew_read", Err: err}
		}
		return n == 1, nil
	default:
		return false, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) release(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) error {
	if err := m.loadScripts(ctx); err != nil {
		return err
	}

	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		cmd := m.client.B().Evalsha().Sha1(m.releaseWriteSHA).Numkeys(1).Key(writeKey).Arg(token).Build()
		if err := m.client.Do(ctx, cmd).Error(); err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.release(ctx, chatID, mode, token, ttlMillis)
			}
			return cerrors.RedisError{Operation: "lock_release_write", Err: err}
		}
		return nil
	case lockModeRead:
		cmd := m.client.B().Evalsha().Sha1(m.releaseReadSHA).Numkeys(1).Key(readKey).Arg(token, ttlArg).Build()
		if err := m.client.Do(ctx, cmd).Error(); err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.release(ctx, chatID, mode, token, ttlMillis)
			}
			return cerrors.RedisError{Operation: "lock_release_read", Err: err}
		}
		return nil
	default:
		return fmt.Errorf("unknown lock mode: %d", mode)
	}
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
