package service

import (
	"context"
	"strings"
	"testing"

	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func TestRiddleService_HandleSurrenderConsensus_StartsVote(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	chatID := "room_surrender_vote_start"
	userID := "user1"

	if _, err := env.svc.Start(ctx, chatID, userID, []string{"사물"}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if _, err := env.svc.playerStore.Add(ctx, chatID, userID, ""); err != nil {
		t.Fatalf("player add failed: %v", err)
	}
	if _, err := env.svc.playerStore.Add(ctx, chatID, "user2", ""); err != nil {
		t.Fatalf("player add failed: %v", err)
	}

	resp, err := env.svc.HandleSurrenderConsensus(ctx, chatID, userID)
	if err != nil {
		t.Fatalf("HandleSurrenderConsensus failed: %v", err)
	}
	if resp != "Vote Started" {
		t.Errorf("unexpected response: %q", resp)
	}

	vote, err := env.svc.voteStore.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("vote get failed: %v", err)
	}
	if vote == nil {
		t.Fatal("expected vote to be created")
	}
	if len(vote.Approvals) != 1 || vote.Approvals[0] != userID {
		t.Errorf("unexpected approvals: %v", vote.Approvals)
	}
}

func TestRiddleService_HandleSurrenderAgree_CompletesAndClears(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	chatID := "room_surrender_agree"
	userID := "user1"
	otherUser := "user2"

	if _, err := env.svc.Start(ctx, chatID, userID, []string{"사물"}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if _, err := env.svc.playerStore.Add(ctx, chatID, userID, ""); err != nil {
		t.Fatalf("player add failed: %v", err)
	}
	if _, err := env.svc.playerStore.Add(ctx, chatID, otherUser, ""); err != nil {
		t.Fatalf("player add failed: %v", err)
	}

	if _, err := env.svc.HandleSurrenderConsensus(ctx, chatID, userID); err != nil {
		t.Fatalf("HandleSurrenderConsensus failed: %v", err)
	}

	resp, err := env.svc.HandleSurrenderAgree(ctx, chatID, otherUser)
	if err != nil {
		t.Fatalf("HandleSurrenderAgree failed: %v", err)
	}
	if !strings.Contains(resp, "Surrender Result") {
		t.Errorf("unexpected response: %q", resp)
	}

	exists, err := env.svc.sessionStore.Exists(ctx, chatID)
	if err != nil {
		t.Fatalf("session exists check failed: %v", err)
	}
	if exists {
		t.Error("expected session to be cleared")
	}

	hasVote, err := env.svc.voteStore.Exists(ctx, chatID)
	if err != nil {
		t.Fatalf("vote exists check failed: %v", err)
	}
	if hasVote {
		t.Error("expected vote to be cleared")
	}
}

func TestRiddleService_Surrender_ClearsSession(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	chatID := "room_surrender_direct"
	userID := "user1"

	if _, err := env.svc.Start(ctx, chatID, userID, []string{"사물"}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	resp, err := env.svc.Surrender(ctx, chatID)
	if err != nil {
		t.Fatalf("Surrender failed: %v", err)
	}
	if !strings.Contains(resp, "Surrender Result") {
		t.Errorf("unexpected response: %q", resp)
	}

	exists, err := env.svc.sessionStore.Exists(ctx, chatID)
	if err != nil {
		t.Fatalf("session exists check failed: %v", err)
	}
	if exists {
		t.Error("expected session to be cleared")
	}
}

func TestRiddleService_HandleSurrenderConsensus_ExistingVote(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	chatID := "room_surrender_vote_exists"
	userID := "user1"
	otherUser := "user2"

	if _, err := env.svc.Start(ctx, chatID, userID, []string{"사물"}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if _, err := env.svc.playerStore.Add(ctx, chatID, userID, ""); err != nil {
		t.Fatalf("player add failed: %v", err)
	}
	if _, err := env.svc.playerStore.Add(ctx, chatID, otherUser, ""); err != nil {
		t.Fatalf("player add failed: %v", err)
	}

	if _, err := env.svc.HandleSurrenderConsensus(ctx, chatID, userID); err != nil {
		t.Fatalf("HandleSurrenderConsensus failed: %v", err)
	}

	resp, err := env.svc.HandleSurrenderConsensus(ctx, chatID, otherUser)
	if err != nil {
		t.Fatalf("HandleSurrenderConsensus failed: %v", err)
	}
	if !strings.Contains(resp, "Vote In Progress") {
		t.Errorf("unexpected response: %q", resp)
	}

	vote, err := env.svc.voteStore.Get(ctx, chatID)
	if err != nil {
		t.Fatalf("vote get failed: %v", err)
	}
	if vote == nil || len(vote.Approvals) != 1 {
		t.Errorf("unexpected vote approvals: %v", vote)
	}
}

func TestRiddleService_Surrender_HintBlock(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	chatID := "room_surrender_hint"
	userID := "user1"

	if _, err := env.svc.Start(ctx, chatID, userID, []string{"사물"}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := env.svc.historyStore.Add(ctx, chatID, qmodel.QuestionHistory{
		QuestionNumber: -1,
		Answer:         "힌트 내용",
	}); err != nil {
		t.Fatalf("history add failed: %v", err)
	}

	resp, err := env.svc.Surrender(ctx, chatID)
	if err != nil {
		t.Fatalf("Surrender failed: %v", err)
	}
	if !strings.Contains(resp, "Surrender Hint Header 1") {
		t.Errorf("expected hint header, got %q", resp)
	}
	if !strings.Contains(resp, "Surrender Hint 1: 힌트 내용") {
		t.Errorf("expected hint content, got %q", resp)
	}
}
