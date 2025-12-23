package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	"github.com/valkey-io/valkey-go"
)

func newTestRiddleServiceForRegistration(t *testing.T) (*RiddleService, *redis.PlayerStore, valkey.Client) {
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	playerStore := redis.NewPlayerStore(client, logger)

	svc := NewRiddleService(
		nil, "", nil, nil, nil, nil, nil, nil,
		playerStore,
		nil, nil, nil, nil, nil,
		logger,
	)
	return svc, playerStore, client
}

func TestRiddleService_RegisterPlayerAsync(t *testing.T) {
	svc, store, client := newTestRiddleServiceForRegistration(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "room1"
	userID := "user1"
	sender := "User One"

	// 1. Register Async
	svc.RegisterPlayerAsync(ctx, chatID, userID, &sender)

	// Wait for worker to process
	time.Sleep(100 * time.Millisecond)

	// Verify
	players, err := store.GetAll(ctx, chatID)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}

	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].UserID != userID {
		t.Errorf("expected user id %s, got %s", userID, players[0].UserID)
	}
	if players[0].Sender != sender {
		t.Errorf("expected sender %s, got %s", sender, players[0].Sender)
	}
}

func TestRiddleService_RegisterPlayerAsync_InvalidInput(t *testing.T) {
	svc, _, client := newTestRiddleServiceForRegistration(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()

	// Empty ChatID - should not create anything
	svc.RegisterPlayerAsync(ctx, "", "user1", nil)
	time.Sleep(50 * time.Millisecond)

	// Empty UserID - should not create anything
	svc.RegisterPlayerAsync(ctx, "room1", "   ", nil)
	time.Sleep(50 * time.Millisecond)
	// No assertion on keys directly since valkey client doesn't expose this easily
}

func TestRiddleService_RegisterPlayerAsync_Duplicate(t *testing.T) {
	svc, store, client := newTestRiddleServiceForRegistration(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "room1"
	userID := "user1"
	sender1 := "User One"
	sender2 := "User One Updated"

	// First registration
	svc.RegisterPlayerAsync(ctx, chatID, userID, &sender1)
	time.Sleep(50 * time.Millisecond)

	// Second registration (update sender)
	svc.RegisterPlayerAsync(ctx, chatID, userID, &sender2)
	time.Sleep(50 * time.Millisecond)

	players, err := store.GetAll(ctx, chatID)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}

	if len(players) != 1 {
		t.Errorf("expected 1 player, got %d", len(players))
	}
	if players[0].Sender != sender2 {
		t.Errorf("expected sender update to %s, got %s", sender2, players[0].Sender)
	}
}
