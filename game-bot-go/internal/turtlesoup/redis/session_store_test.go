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

func TestPendingMessageStore_KeyPrefix(t *testing.T) {
	client := testhelper.NewTestValkeyClient(t)
	defer client.Close()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	store := NewPendingMessageStore(client, logger)
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	chatID := prefix + "room1"

	res, err := store.Enqueue(ctx, chatID, tsmodel.PendingMessage{UserID: "user_1", Content: "hello", Timestamp: 123})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if res != EnqueueSuccess {
		t.Fatalf("expected EnqueueSuccess, got %s", res)
	}

	dataKey := tsconfig.RedisKeyPendingPrefix + ":data:{" + chatID + "}"
	orderKey := tsconfig.RedisKeyPendingPrefix + ":order:{" + chatID + "}"

	exists, err := client.Do(ctx, client.B().Exists().Key(dataKey, orderKey).Build()).AsInt64()
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if exists != 2 {
		t.Fatalf("expected pending keys to exist, got %d", exists)
	}

	dequeued, err := store.Dequeue(ctx, chatID)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if dequeued.Status != DequeueSuccess {
		t.Fatalf("expected DequeueSuccess, got %s", dequeued.Status)
	}
	if dequeued.Message == nil {
		t.Fatal("expected message")
	}
	if dequeued.Message.UserID != "user_1" {
		t.Fatalf("expected user_1, got %s", dequeued.Message.UserID)
	}
	if dequeued.Message.Content != "hello" {
		t.Fatalf("expected hello content, got %s", dequeued.Message.Content)
	}

	res, err = store.Enqueue(ctx, chatID, tsmodel.PendingMessage{UserID: "user_2", Content: "hello2", Timestamp: 124})
	if err != nil {
		t.Fatalf("enqueue second failed: %v", err)
	}
	if res != EnqueueSuccess {
		t.Fatalf("expected EnqueueSuccess, got %s", res)
	}

	if err := store.Clear(ctx, chatID); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	existsAfter, err := client.Do(ctx, client.B().Exists().Key(dataKey, orderKey).Build()).AsInt64()
	if err != nil {
		t.Fatalf("exists after clear failed: %v", err)
	}
	if existsAfter != 0 {
		t.Fatalf("expected pending keys deleted, got %d", existsAfter)
	}
}
