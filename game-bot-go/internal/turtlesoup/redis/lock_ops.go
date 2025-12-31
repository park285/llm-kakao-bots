package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

const (
	lockRetryInitialDelay  = 50 * time.Millisecond
	lockRetryMaxDelay      = 500 * time.Millisecond
	lockRetryDelayMultiply = 2
)

func (m *LockManager) acquire(ctx context.Context, sessionID string, token string, holderValue string, ttl time.Duration) (bool, error) {
	key := lockKey(sessionID)
	holderKey := lockHolderKey(sessionID)

	delay := lockRetryInitialDelay

	for {
		cmd := m.client.B().Set().Key(key).Value(token).Nx().Ex(ttl).Build()
		err := m.client.Do(ctx, cmd).Error()
		if err != nil {
			if valkeyx.IsNil(err) {
				// SET NX failed (key exists)
				if ctx.Err() != nil {
					return false, nil
				}

				// Exponential backoff: 50ms → 100ms → 200ms → ... 最大 500ms
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
					delay = min(delay*lockRetryDelayMultiply, lockRetryMaxDelay)
					continue
				case <-ctx.Done():
					timer.Stop()
					return false, nil
				}
			}
			if ctx.Err() != nil {
				return false, nil
			}
			return false, wrapRedisError("lock_acquire", err)
		}

		// Lock 획득 완료
		break
	}

	holderCmd := m.client.B().Set().Key(holderKey).Value(holderValue).Ex(ttl).Build()
	if err := m.client.Do(ctx, holderCmd).Error(); err != nil {
		releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
		defer releaseCancel()
		_ = m.release(releaseCtx, sessionID, token)
		return false, wrapRedisError("lock_set_holder", err)
	}

	return true, nil
}

func (m *LockManager) release(ctx context.Context, sessionID string, token string) error {
	key := lockKey(sessionID)
	holderKey := lockHolderKey(sessionID)

	resp, err := m.registry.Exec(ctx, m.client, luautil.ScriptTurtleLockRelease, []string{key, holderKey}, []string{token})
	if err != nil {
		return fmt.Errorf("lock release script missing: %w", err)
	}
	if err := resp.Error(); err != nil {
		return wrapRedisError("lock_release", err)
	}
	return nil
}

func (m *LockManager) getHolder(ctx context.Context, sessionID string) (*string, error) {
	holderKey := lockHolderKey(sessionID)
	cmd := m.client.B().Get().Key(holderKey).Build()
	value, err := m.client.Do(ctx, cmd).ToString()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, wrapRedisError("lock_get_holder", err)
	}

	_, name := parseHolderValue(value)
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	return &name, nil
}

func buildHolderValue(token string, holderName *string) string {
	name := "다른 사용자"
	if holderName != nil && strings.TrimSpace(*holderName) != "" {
		name = strings.TrimSpace(*holderName)
	}
	return token + "|" + name
}

func parseHolderValue(raw string) (token string, name string) {
	raw = strings.TrimSpace(raw)
	delim := strings.Index(raw, "|")
	if delim <= 0 {
		return "", raw
	}
	return raw[:delim], raw[delim+1:]
}

func wrapRedisError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", valkeyx.WrapRedisError(operation, err))
}
