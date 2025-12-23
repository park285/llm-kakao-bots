package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func newTestSurrenderVoteStore(t *testing.T) (*SurrenderVoteStore, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSurrenderVoteStore(client, logger), client
}

func TestSurrenderVoteStore_SaveAndGet(t *testing.T) {
	store, client := newTestSurrenderVoteStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_vote"

	vote := qmodel.SurrenderVote{
		Initiator:       "user1",
		EligiblePlayers: []string{"user1", "user2", "user3"},
		CreatedAt:       time.Now().UnixMilli(),
		Approvals:       []string{"user1"},
	}

	if err := store.Save(ctx, chatID, vote); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected vote, got nil")
	}
	if got.Initiator != "user1" {
		t.Errorf("expected initiator user1, got %s", got.Initiator)
	}
	if len(got.EligiblePlayers) != 3 {
		t.Errorf("expected 3 eligible players, got %d", len(got.EligiblePlayers))
	}
}

func TestSurrenderVoteStore_Exists(t *testing.T) {
	store, client := newTestSurrenderVoteStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_vote_exists"

	exists, err := store.Exists(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected vote not exists")
	}

	store.Save(ctx, chatID, qmodel.SurrenderVote{})
	exists, err = store.Exists(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected vote exists")
	}
}

func TestSurrenderVoteStore_Clear(t *testing.T) {
	store, client := newTestSurrenderVoteStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_vote_clear"
	store.Save(ctx, chatID, qmodel.SurrenderVote{})

	if err := store.Clear(ctx, chatID); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	got, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil vote after clear")
	}
}

func TestSurrenderVoteStore_Approve(t *testing.T) {
	store, client := newTestSurrenderVoteStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_vote_approve"

	// Setup vote
	vote := qmodel.SurrenderVote{
		Initiator:       "user1",
		EligiblePlayers: []string{"user1", "user2", "user3"},
		CreatedAt:       time.Now().UnixMilli(),
		Approvals:       []string{"user1"},
	}
	store.Save(ctx, chatID, vote)

	// 1. Valid Approval
	updated, err := store.Approve(ctx, chatID, "user2")
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if len(updated.Approvals) != 2 {
		t.Errorf("expected 2 approvals, got %d", len(updated.Approvals))
	}

	// Verify persistence
	got, _ := store.Get(ctx, chatID)
	if len(got.Approvals) != 2 {
		t.Errorf("expected 2 approvals persisted, got %d", len(got.Approvals))
	}

	// 2. Duplicate Approval (Idempotent check in model)
	updated2, err := store.Approve(ctx, chatID, "user2")
	if err != nil {
		t.Fatalf("duplicate approve failed: %v", err)
	}
	if len(updated2.Approvals) != 2 {
		t.Errorf("expected approval count to remain 2, got %d", len(updated2.Approvals))
	}

	// 3. Not Eligible User
	_, err = store.Approve(ctx, chatID, "user4")
	if err == nil {
		t.Error("expected error for non-eligible user")
	}
}
