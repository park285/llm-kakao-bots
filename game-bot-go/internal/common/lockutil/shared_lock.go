package lockutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

// NewToken: 락 식별을 위한 임의 토큰을 생성합니다.
func NewToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("rand read failed: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// TryAcquireSharedLock: 공유 락을 획득합니다. (SET NX)
func TryAcquireSharedLock(ctx context.Context, client valkey.Client, lockKey string, ttlSeconds int64) (bool, error) {
	lockKey = strings.TrimSpace(lockKey)
	if lockKey == "" {
		return false, fmt.Errorf("lock key is empty")
	}
	if ttlSeconds <= 0 {
		return false, fmt.Errorf("invalid ttlSeconds: %d", ttlSeconds)
	}

	cmd := client.B().Set().Key(lockKey).Value("1").Nx().Ex(time.Duration(ttlSeconds) * time.Second).Build()
	err := client.Do(ctx, cmd).Error()
	if err != nil {
		if valkeyx.IsNil(err) {
			return false, nil
		}
		return false, cerrors.RedisError{Operation: "shared_lock_acquire", Err: err}
	}
	return true, nil
}

// ReleaseSharedLock: 공유 락을 해제합니다. (DEL)
func ReleaseSharedLock(ctx context.Context, client valkey.Client, lockKey string) error {
	lockKey = strings.TrimSpace(lockKey)
	if lockKey == "" {
		return fmt.Errorf("lock key is empty")
	}
	cmd := client.B().Del().Key(lockKey).Build()
	if err := client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "shared_lock_release", Err: err}
	}
	return nil
}
