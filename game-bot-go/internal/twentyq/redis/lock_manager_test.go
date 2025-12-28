package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lockutil"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func newTestLockManager(t *testing.T) (*LockManager, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewLockManager(client, logger), client
}

func TestLockManager_WithLock(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_lock"

	// 1. Acquire Lock
	executed := false
	err := lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
		executed = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}
	if !executed {
		t.Error("block was not executed")
	}
}

func TestLockManager_WithLock_ReleasesAfterLongBlock(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	lm.redisCallTimeout = 20 * time.Millisecond

	ctx := context.Background()
	chatID := prefix + "room_release"

	err := lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
		time.Sleep(2 * lm.redisCallTimeout)
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	executedSecond := false
	err = lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
		executedSecond = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock should succeed after release, got: %v", err)
	}
	if !executedSecond {
		t.Error("second block was not executed")
	}
}

func TestLockManager_ConcurrentLocks(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_concurrent"

	// Acquire Write Lock in goroutine
	startCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		close(startCh)
		lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond) // Hold lock
			return nil
		})
		close(doneCh)
	}()

	<-startCh
	time.Sleep(20 * time.Millisecond) // Ensure goroutine got lock

	// Wait for goroutine
	<-doneCh
}

func TestLockManager_Reentry(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_reentry"

	executedInner := false
	err := lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
		// Re-enter with same lock
		return lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
			executedInner = true
			return nil
		})
	})

	if err != nil {
		t.Fatalf("Reentry failed: %v", err)
	}
	if !executedInner {
		t.Error("Inner block not executed")
	}
}

func TestLockManager_AcquireExclusive(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_acquire"

	token1, err := lockutil.NewToken()
	if err != nil {
		t.Fatalf("token1 failed: %v", err)
	}
	token2, err := lockutil.NewToken()
	if err != nil {
		t.Fatalf("token2 failed: %v", err)
	}

	ttlMillis := lockutil.TTLMillisFromSeconds(int64(qconfig.RedisLockTTLSeconds))

	acquired, err := lm.acquire(ctx, chatID, token1, ttlMillis)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if !acquired {
		t.Fatal("expected first acquire to succeed")
	}

	acquired, err = lm.acquire(ctx, chatID, token2, ttlMillis)
	if err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	if acquired {
		t.Fatal("expected second acquire to fail while lock is held")
	}
}

func TestLockManager_ReleaseIgnoresMismatchedToken(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_release_token"

	token, err := lockutil.NewToken()
	if err != nil {
		t.Fatalf("token failed: %v", err)
	}
	ttlMillis := lockutil.TTLMillisFromSeconds(int64(qconfig.RedisLockTTLSeconds))

	acquired, err := lm.acquire(ctx, chatID, token, ttlMillis)
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	if err := lm.release(ctx, chatID, "wrong-token"); err != nil {
		t.Fatalf("release with wrong token failed: %v", err)
	}

	held, err := lm.acquire(ctx, chatID, "second-token", ttlMillis)
	if err != nil {
		t.Fatalf("reacquire failed: %v", err)
	}
	if held {
		t.Fatal("expected lock to remain held after wrong-token release")
	}
}
