package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
)

func newTestHintCountStore(t *testing.T) (*HintCountStore, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewHintCountStore(client, logger), client
}

func TestHintCountStore_IncrementAndGet(t *testing.T) {
	store, client := newTestHintCountStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_hint"

	// 1. Initial Get (should be 0)
	count, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// 2. Increment
	newCount, err := store.Increment(ctx, chatID)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if newCount != 1 {
		t.Errorf("expected 1, got %d", newCount)
	}

	// 3. Get Again
	count, err = store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("Get2 failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestHintCountStore_Delete(t *testing.T) {
	store, client := newTestHintCountStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_del"

	store.Increment(ctx, chatID)

	if err := store.Delete(ctx, chatID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	count, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}
