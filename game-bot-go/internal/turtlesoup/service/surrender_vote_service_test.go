package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

type voteTestEnv struct {
	svc            *SurrenderVoteService
	sessionManager *GameSessionManager
	voteStore      *tsredis.SurrenderVoteStore
	sessionStore   *tsredis.SessionStore
	client         valkey.Client
}

func setupVoteTestEnv(t *testing.T) *voteTestEnv {
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	sessionStore := tsredis.NewSessionStore(client, logger)
	lockManager := tsredis.NewLockManager(client, logger)
	voteStore := tsredis.NewSurrenderVoteStore(client, logger)

	sessionManager := NewGameSessionManager(sessionStore, lockManager)
	svc := NewSurrenderVoteService(sessionManager, voteStore)

	return &voteTestEnv{
		svc:            svc,
		sessionManager: sessionManager,
		voteStore:      voteStore,
		sessionStore:   sessionStore,
		client:         client,
	}
}

func (e *voteTestEnv) teardown(t *testing.T) {
	testhelper.CleanupTestKeys(t, e.client, tsconfig.RedisKeyPrefix+":")
	e.client.Close()
}

func TestSurrenderVoteService_ResolvePlayers(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_players"

	// 1. Session missing
	_, err := env.svc.ResolvePlayers(ctx, chatID)
	if err == nil {
		t.Error("expected error for missing session")
	}

	// 2. Session exists, single user
	err = env.sessionStore.SaveGameState(ctx, tsmodel.GameState{
		SessionID: chatID,
		UserID:    "user1",
		Players:   nil,
	})
	if err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	players, err := env.svc.ResolvePlayers(ctx, chatID)
	if err != nil {
		t.Fatalf("ResolvePlayers failed: %v", err)
	}
	if len(players) != 1 || players[0] != "user1" {
		t.Errorf("expected [user1], got %v", players)
	}

	// 3. Session exists, multiple players
	err = env.sessionStore.SaveGameState(ctx, tsmodel.GameState{
		SessionID: chatID,
		UserID:    "user1",
		Players:   []string{"user1", "user2", "user3"},
	})
	if err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	players, err = env.svc.ResolvePlayers(ctx, chatID)
	if err != nil {
		t.Fatalf("ResolvePlayers multi failed: %v", err)
	}
	if len(players) != 3 {
		t.Errorf("expected 3 players, got %d", len(players))
	}
}

func TestSurrenderVoteService_RequireSession(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_require_sess"

	// 1. Session missing
	err := env.svc.RequireSession(ctx, chatID)
	if err == nil {
		t.Error("expected error for missing session")
	}

	// 2. Session exists
	err = env.sessionStore.SaveGameState(ctx, tsmodel.GameState{
		SessionID: chatID,
		UserID:    "user1",
	})
	if err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	err = env.svc.RequireSession(ctx, chatID)
	if err != nil {
		t.Errorf("RequireSession failed: %v", err)
	}
}

func TestSurrenderVoteService_StartVote_Immediate(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_vote_imm"

	// Single player -> 100% approval immediately
	res, err := env.svc.StartVote(ctx, chatID, "user1", []string{"user1"})
	if err != nil {
		t.Fatalf("StartVote failed: %v", err)
	}

	if res.Type != VoteStartImmediate {
		t.Errorf("expected VoteStartImmediate, got %v", res.Type)
	}
	if !res.Vote.IsApproved() {
		t.Error("vote should be approved")
	}
}

func TestSurrenderVoteService_StartVote_Started(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_vote_start"

	// 3 players -> Initiator is 1/3 (33%) -> Not enough (need >50% i.e. 2/3)
	res, err := env.svc.StartVote(ctx, chatID, "user1", []string{"user1", "user2", "user3"})
	if err != nil {
		t.Fatalf("StartVote failed: %v", err)
	}

	if res.Type != VoteStartStarted {
		t.Errorf("expected VoteStartStarted, got %v", res.Type)
	}

	// Check store
	vote, err := env.voteStore.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("Get vote failed: %v", err)
	}
	if vote == nil {
		t.Error("vote should be persisted")
	}
}

func TestSurrenderVoteService_Approve(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_approve"
	players := []string{"user1", "user2", "user3"}

	// Start vote
	_, err := env.svc.StartVote(ctx, chatID, "user1", players)
	if err != nil {
		t.Fatalf("StartVote failed: %v", err)
	}

	// 1. Approve by already voted (Initiator)
	res, err := env.svc.Approve(ctx, chatID, "user1")
	if err != nil {
		t.Fatalf("Approve user1 failed: %v", err)
	}
	if res.Type != VoteApprovalAlreadyVoted {
		t.Errorf("expected VoteApprovalAlreadyVoted, got %v", res.Type)
	}

	// 2. Approve by ineligible
	res, err = env.svc.Approve(ctx, chatID, "user4")
	if err != nil {
		t.Fatalf("Approve user4 failed: %v", err)
	}
	if res.Type != VoteApprovalNotEligible {
		t.Errorf("expected VoteApprovalNotEligible, got %v", res.Type)
	}

	// 3. Approve by user2 -> Progress (Current: 2/3)
	res, err = env.svc.Approve(ctx, chatID, "user2")
	if err != nil {
		t.Fatalf("Approve user2 failed: %v", err)
	}
	if res.Type != VoteApprovalProgress {
		t.Errorf("expected VoteApprovalProgress, got %v", res.Type)
	}

	// 4. Approve by user3 -> Completed (Current: 3/3)
	res, err = env.svc.Approve(ctx, chatID, "user3")
	if err != nil {
		t.Fatalf("Approve user3 failed: %v", err)
	}
	if res.Type != VoteApprovalCompleted {
		t.Errorf("expected VoteApprovalCompleted, got %v", res.Type)
	}

	// 5. Verify cleared
	vote, err := env.voteStore.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("Get vote failed: %v", err)
	}
	if vote != nil {
		t.Error("vote should be cleared after completion")
	}
}

func TestSurrenderVoteService_Clear(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_clear"

	_, err := env.svc.StartVote(ctx, chatID, "user1", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("StartVote failed: %v", err)
	}

	if err := env.svc.Clear(ctx, chatID); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	vote, _ := env.voteStore.Get(ctx, chatID)
	if vote != nil {
		t.Error("vote should be cleared")
	}
}

func TestSurrenderVoteService_ActiveVote(t *testing.T) {
	env := setupVoteTestEnv(t)
	defer env.teardown(t)
	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "chat_active"

	// 1. No vote
	v, err := env.svc.ActiveVote(ctx, chatID)
	if err != nil {
		t.Fatalf("ActiveVote failed: %v", err)
	}
	if v != nil {
		t.Error("expected no active vote")
	}

	// 2. Start valid vote
	_, _ = env.svc.StartVote(ctx, chatID, "user1", []string{"user1", "user2", "user3"})

	v, err = env.svc.ActiveVote(ctx, chatID)
	if err != nil {
		t.Fatalf("ActiveVote check failed: %v", err)
	}
	if v == nil {
		t.Error("expected active vote")
	}
	if v.Initiator != "user1" {
		t.Errorf("unexpected initiator: %s", v.Initiator)
	}
}
