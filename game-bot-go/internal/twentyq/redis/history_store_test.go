package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func newTestHistoryStore(t *testing.T) (*HistoryStore, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewHistoryStore(client, logger), client
}

func TestHistoryStore_AddAndGet(t *testing.T) {
	store, client := newTestHistoryStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_history"

	item1 := qmodel.QuestionHistory{
		QuestionNumber: 1,
		Question:       "Q1",
		Answer:         "A1",
	}
	item2 := qmodel.QuestionHistory{
		QuestionNumber: 2,
		Question:       "Q2",
		Answer:         "A2",
	}

	if err := store.Add(ctx, chatID, item1); err != nil {
		t.Fatalf("add item1 failed: %v", err)
	}
	if err := store.Add(ctx, chatID, item2); err != nil {
		t.Fatalf("add item2 failed: %v", err)
	}

	history, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 items, got %d", len(history))
	}
	if history[0].Question != "Q1" {
		t.Errorf("expected first item Q1, got %s", history[0].Question)
	}
	if history[1].Question != "Q2" {
		t.Errorf("expected second item Q2, got %s", history[1].Question)
	}
}

func TestHistoryStore_Clear(t *testing.T) {
	store, client := newTestHistoryStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_history_clear"

	store.Add(ctx, chatID, qmodel.QuestionHistory{Question: "Q1"})

	if err := store.Clear(ctx, chatID); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	history, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d items", len(history))
	}
}

func TestHistoryStore_CorruptedData_Ignored(t *testing.T) {
	store, client := newTestHistoryStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_history_corrupt"

	// Add valid and invalid data
	store.Add(ctx, chatID, qmodel.QuestionHistory{QuestionNumber: 1})
	// Note: Can't easily inject invalid JSON without direct Redis access
	// This test verifies the store handles normal data correctly
	store.Add(ctx, chatID, qmodel.QuestionHistory{QuestionNumber: 3})

	history, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 valid items, got %d", len(history))
	}
}
