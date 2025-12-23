package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

func newTestLockManager(t *testing.T) (*LockManager, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewLockManager(client, logger), client
}

func TestLockManager_WithLock_ReleasesAfterLongBlock(t *testing.T) {
	lm, client := newTestLockManager(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	lm.redisCallTimeout = 20 * time.Millisecond

	ctx := context.Background()
	sessionID := prefix + "sess_release"

	err := lm.WithLock(ctx, sessionID, nil, func(ctx context.Context) error {
		time.Sleep(2 * lm.redisCallTimeout)
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	// Verify lock is released by acquiring it again
	executedSecond := false
	err = lm.WithLock(ctx, sessionID, nil, func(ctx context.Context) error {
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
