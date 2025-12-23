package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

func newTestSessionStore(t *testing.T) (*SessionStore, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSessionStore(client, logger), client
}

func TestSessionStore_SaveAndLoad(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	sessionID := prefix + "sess_1"

	gameState := tsmodel.GameState{
		SessionID: sessionID,
		UserID:    "user_1",
		ChatID:    "room_1",
		IsSolved:  true,
		HintsUsed: 5,
	}

	if err := store.SaveGameState(ctx, gameState); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.LoadGameState(ctx, sessionID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded state, got nil")
	}
	if loaded.SessionID != sessionID {
		t.Errorf("expected sessionID %s, got %s", sessionID, loaded.SessionID)
	}
	if !loaded.IsSolved {
		t.Error("expected IsSolved true")
	}
	if loaded.HintsUsed != 5 {
		t.Errorf("expected 5 hints used, got %d", loaded.HintsUsed)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	sessionID := prefix + "sess_del"

	store.SaveGameState(ctx, tsmodel.GameState{SessionID: sessionID})

	if err := store.DeleteSession(ctx, sessionID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	loaded, err := store.LoadGameState(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestSessionStore_Exists(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	sessionID := prefix + "sess_exists"

	exists, err := store.SessionExists(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected not exists")
	}

	store.SaveGameState(ctx, tsmodel.GameState{SessionID: sessionID})
	exists, err = store.SessionExists(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected exists")
	}
}

func TestSessionStore_RefreshTTL(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	sessionID := prefix + "sess_ttl"
	store.SaveGameState(ctx, tsmodel.GameState{SessionID: sessionID})

	ok, err := store.RefreshTTL(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected refresh success")
	}
}
