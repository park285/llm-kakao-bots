package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
)

func newTestPlayerStore(t *testing.T) (*PlayerStore, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewPlayerStore(client, logger), client
}

func TestPlayerStore_AddAndGet(t *testing.T) {
	store, client := newTestPlayerStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_player"

	// 1. Add
	isNew, err := store.Add(ctx, chatID, "user1", "User One")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true for first user")
	}

	// 2. Add Same (Update sender)
	isNew, err = store.Add(ctx, chatID, "user1", "User One Updated")
	if err != nil {
		t.Fatalf("Add2 failed: %v", err)
	}
	if isNew {
		t.Error("expected isNew=false for existing user")
	}

	// 3. Add Another
	isNew, err = store.Add(ctx, chatID, "user2", "User Two")
	if err != nil {
		t.Fatalf("Add3 failed: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true for second user")
	}

	// 4. GetAll
	players, err := store.GetAll(ctx, chatID)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(players) != 2 {
		t.Errorf("expected 2 players, got %d", len(players))
	}

	for _, p := range players {
		if p.UserID == "user1" && p.Sender != "User One Updated" {
			t.Errorf("expected updated sender, got %s", p.Sender)
		}
	}
}

func TestPlayerStore_Clear(t *testing.T) {
	store, client := newTestPlayerStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_clear"

	store.Add(ctx, chatID, "user1", "Sender")

	if err := store.Clear(ctx, chatID); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	players, err := store.GetAll(ctx, chatID)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(players) != 0 {
		t.Error("expected empty")
	}
}
