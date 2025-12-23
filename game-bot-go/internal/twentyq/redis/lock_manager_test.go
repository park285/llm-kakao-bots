package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
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

func TestLockManager_ReadWriteExclusion(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_rw"

	// 1. Acquire Write Lock
	startCh := make(chan struct{})
	releaseCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		close(startCh)
		lm.WithLock(ctx, chatID, nil, func(ctx context.Context) error {
			<-releaseCh
			return nil
		})
		close(doneCh)
	}()

	<-startCh
	time.Sleep(50 * time.Millisecond) // Wait for lock acquisition

	// 2. Try to Acquire Read Lock (Should Timeout)
	timeoutCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	err := lm.WithReadLock(timeoutCtx, chatID, nil, func(ctx context.Context) error {
		return nil
	})

	// Should fail with deadline exceeded or acquisition failure
	if err == nil {
		t.Error("WithReadLock should fail (timeout) while Write Lock is held")
	}

	// 3. Release Write Lock
	close(releaseCh)
	<-doneCh
	time.Sleep(20 * time.Millisecond)

	// 4. Try Acquire Read Lock (Should Succeed)
	err = lm.WithReadLock(ctx, chatID, nil, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("WithReadLock failed after Write Lock released: %v", err)
	}
}
