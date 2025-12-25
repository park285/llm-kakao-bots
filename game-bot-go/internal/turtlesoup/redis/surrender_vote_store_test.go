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
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	chatID := prefix + "room_vote"

	vote := tsmodel.SurrenderVote{
		Initiator:       "user1",
		EligiblePlayers: []string{"user1", "user2"},
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
	if len(got.EligiblePlayers) != 2 {
		t.Errorf("expected 2 eligible players, got %d", len(got.EligiblePlayers))
	}
}

func TestSurrenderVoteStore_Approve(t *testing.T) {
	store, client := newTestSurrenderVoteStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	chatID := prefix + "room_vote_approve"

	vote := tsmodel.SurrenderVote{
		Initiator:       "user1",
		EligiblePlayers: []string{"user1", "user2"},
		Approvals:       []string{"user1"},
	}
	if err := store.Save(ctx, chatID, vote); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	updated, err := store.Approve(ctx, chatID, "user2")
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if updated == nil {
		t.Fatal("expected vote, got nil")
	}
	if len(updated.Approvals) != 2 {
		t.Errorf("expected 2 approvals, got %d", len(updated.Approvals))
	}

	// Persisted?
	got, err := store.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected vote, got nil")
	}
	if len(got.Approvals) != 2 {
		t.Errorf("expected persistence, got %d", len(got.Approvals))
	}
}

func TestSurrenderVoteStore_Clear(t *testing.T) {
	store, client := newTestSurrenderVoteStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	ctx := context.Background()
	chatID := prefix + "room_vote_clear"
	store.Save(ctx, chatID, tsmodel.SurrenderVote{})

	if err := store.Clear(ctx, chatID); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get(ctx, chatID)
	if got != nil {
		t.Error("expected nil vote")
	}
}
