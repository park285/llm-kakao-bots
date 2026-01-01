package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"
	"google.golang.org/grpc"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
	tssecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/security"
)

type mockResponses struct {
	guardMalicious bool
	puzzle         *llmrest.TurtleSoupPuzzleGenerationResponse
	answer         *llmrest.TurtleSoupAnswerResponse
	validation     *llmrest.TurtleSoupValidateResponse
	hint           *llmrest.TurtleSoupHintResponse
}

type testEnv struct {
	svc          *GameService
	llmClient    *llmrest.Client
	stopLLM      func()
	client       valkey.Client
	sessionStore *tsredis.SessionStore
	mocks        mockResponses
	t            *testing.T
	prefix       string
}

func setupTestEnv(t *testing.T) *testEnv {
	client := testhelper.NewTestValkeyClient(t)
	testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")
	testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPuzzleGlobal)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	sessionStore := tsredis.NewSessionStore(client, logger)
	lockManager := tsredis.NewLockManager(client, logger)
	dedupStore := tsredis.NewPuzzleDedupStore(client, logger)

	sessionManager := NewGameSessionManager(sessionStore, lockManager)

	prefix := testhelper.UniqueTestPrefix(t)
	env := &testEnv{client: client, sessionStore: sessionStore, t: t, prefix: prefix}
	stub := &turtlesoupLLMGRPCStub{
		guardMalicious: func() bool {
			return env.mocks.guardMalicious
		},
		generatePuzzle: func() *llmrest.TurtleSoupPuzzleGenerationResponse {
			if env.mocks.puzzle != nil {
				return env.mocks.puzzle
			}
			return &llmrest.TurtleSoupPuzzleGenerationResponse{
				Title:      "Test Puzzle",
				Scenario:   "A man walks into a bar...",
				Solution:   "He was thirsty.",
				Category:   "mystery",
				Difficulty: 1,
			}
		},
		answerQuestion: func() *llmrest.TurtleSoupAnswerResponse {
			if env.mocks.answer != nil {
				return env.mocks.answer
			}
			return &llmrest.TurtleSoupAnswerResponse{
				Answer:        "No",
				History:       []llmrest.TurtleSoupHistoryItem{{Question: "Is it food?", Answer: "No"}},
				QuestionCount: 1,
			}
		},
		validateSolution: func() *llmrest.TurtleSoupValidateResponse {
			if env.mocks.validation != nil {
				return env.mocks.validation
			}
			return &llmrest.TurtleSoupValidateResponse{Result: "NO"}
		},
		generateHint: func() *llmrest.TurtleSoupHintResponse {
			if env.mocks.hint != nil {
				return env.mocks.hint
			}
			return &llmrest.TurtleSoupHintResponse{Hint: "This is a hint", Level: 1}
		},
	}

	baseURL, stop := testhelper.StartTestGRPCServer(t, func(s *grpc.Server) {
		llmv1.RegisterLLMServiceServer(s, stub)
	})
	env.stopLLM = stop

	llmClient, err := llmrest.New(llmrest.Config{
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("llm client init failed: %v", err)
	}
	env.llmClient = llmClient

	puzzleConfig := tsconfig.PuzzleConfig{RewriteEnabled: false}
	puzzleService := NewPuzzleService(llmClient, puzzleConfig, dedupStore, logger)
	setupService := NewGameSetupService(llmClient, puzzleService, sessionManager, logger)
	injectionGuard := tssecurity.NewMcpInjectionGuard(llmClient, logger)

	env.svc = NewGameService(llmClient, sessionManager, setupService, injectionGuard, logger)

	return env
}

func (e *testEnv) chatID(suffix string) string {
	return e.prefix + suffix
}

func (e *testEnv) teardown() {
	testhelper.CleanupTestKeys(e.t, e.client, tsconfig.RedisKeyPrefix+":")
	if e.llmClient != nil {
		_ = e.llmClient.Close()
	}
	e.client.Close()
	if e.stopLLM != nil {
		e.stopLLM()
	}
}

func TestGameService_StartGame(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess1"
	userID := "user1"
	chatID := env.chatID("chat1")

	state, err := env.svc.StartGame(ctx, sessionID, userID, chatID, nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	if state.SessionID != sessionID {
		t.Errorf("expected session ID %s, got %s", sessionID, state.SessionID)
	}
	if state.Puzzle == nil {
		t.Error("expected puzzle to be set")
	}

	loaded, err := env.sessionStore.LoadGameState(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if loaded == nil {
		t.Error("session not found in store")
	}
}

func TestGameService_AskQuestion(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess2"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat2"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	question := "Is it alive?"
	env.mocks.answer = &llmrest.TurtleSoupAnswerResponse{
		Answer:        "No",
		History:       []llmrest.TurtleSoupHistoryItem{{Question: question, Answer: "No"}},
		QuestionCount: 1,
	}

	state, result, err := env.svc.AskQuestion(ctx, sessionID, question)
	if err != nil {
		t.Fatalf("AskQuestion failed: %v", err)
	}

	if result.Answer != "No" {
		t.Errorf("expected answer 'No', got %s", result.Answer)
	}
	if state.QuestionCount != 1 {
		t.Errorf("expected question count 1, got %d", state.QuestionCount)
	}
}

func TestGameService_SubmitSolution_Correct(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess3"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat3"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	env.mocks.validation = &llmrest.TurtleSoupValidateResponse{
		Result: "YES",
	}

	state, validation, err := env.svc.SubmitSolution(ctx, sessionID, "The correct answer")
	if err != nil {
		t.Fatalf("SubmitSolution failed: %v", err)
	}

	if validation != tsmodel.ValidationYes {
		t.Errorf("expected ValidationYes, got %s", validation)
	}

	time.Sleep(50 * time.Millisecond)
	exists, _ := env.sessionStore.SessionExists(ctx, sessionID)
	if exists {
		t.Error("session should be deleted after correct solution")
	}
	if state.Puzzle == nil {
		t.Error("state puzzle should be present in return even if session deleted")
	}
}

func TestGameService_SubmitSolution_Incorrect(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess4"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat4"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	env.mocks.validation = &llmrest.TurtleSoupValidateResponse{
		Result: "NO",
	}

	_, validation, err := env.svc.SubmitSolution(ctx, sessionID, "Wrong answer")
	if err != nil {
		t.Fatalf("SubmitSolution failed: %v", err)
	}

	if validation != tsmodel.ValidationNo {
		t.Errorf("expected ValidationNo, got %s", validation)
	}

	exists, _ := env.sessionStore.SessionExists(ctx, sessionID)
	if !exists {
		t.Error("session should NOT be deleted after incorrect solution")
	}
}

func TestGameService_RequestHint(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess5"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat5"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	env.mocks.hint = &llmrest.TurtleSoupHintResponse{
		Hint:  "Specific Hint",
		Level: 1,
	}

	state, hint, err := env.svc.RequestHint(ctx, sessionID)
	if err != nil {
		t.Fatalf("RequestHint failed: %v", err)
	}

	if hint != "Specific Hint" {
		t.Errorf("expected hint 'Specific Hint', got %s", hint)
	}
	if state.HintsUsed != 1 {
		t.Errorf("expected hints used 1, got %d", state.HintsUsed)
	}
}

func TestGameService_RegisterPlayer(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess6"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat6"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	err = env.svc.RegisterPlayer(ctx, sessionID, "user2")
	if err != nil {
		t.Fatalf("RegisterPlayer failed: %v", err)
	}

	loaded, err := env.svc.GetGameState(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetGameState failed: %v", err)
	}

	if len(loaded.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(loaded.Players))
	}
	if loaded.Players[0] != "user1" || loaded.Players[1] != "user2" {
		t.Errorf("unexpected players list: %v", loaded.Players)
	}

	err = env.svc.RegisterPlayer(ctx, sessionID, "user2")
	if err != nil {
		t.Fatalf("RegisterPlayer duplicate failed: %v", err)
	}
	loaded, _ = env.svc.GetGameState(ctx, sessionID)
	if len(loaded.Players) != 2 {
		t.Errorf("expected 2 players after duplicate add, got %d", len(loaded.Players))
	}
}

func TestGameService_Surrender(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess7"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat7"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	res, err := env.svc.Surrender(ctx, sessionID)
	if err != nil {
		t.Fatalf("Surrender failed: %v", err)
	}

	if res.Solution == "" {
		t.Error("expected solution in surrender result")
	}

	exists, _ := env.sessionStore.SessionExists(ctx, sessionID)
	if exists {
		t.Error("session should be deleted after surrender")
	}
}

func TestGameService_IsMalicious(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess8"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat8"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	env.mocks.guardMalicious = true

	_, _, err = env.svc.AskQuestion(ctx, sessionID, "Bad question")
	if err == nil {
		t.Error("expected error for malicious question")
	}
	var injectionErr cerrors.InputInjectionError
	if !errors.As(err, &injectionErr) {
		t.Fatalf("expected InputInjectionError, got: %v", err)
	}

	_, _, err = env.svc.SubmitSolution(ctx, sessionID, "Bad answer")
	if err == nil {
		t.Error("expected error for malicious answer")
	}
}

func TestGameService_SubmitAnswer(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess_ans"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat_ans"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	env.mocks.validation = &llmrest.TurtleSoupValidateResponse{Result: "NO"}

	res, err := env.svc.SubmitAnswer(ctx, sessionID, "Some answer")
	if err != nil {
		t.Fatalf("SubmitAnswer failed: %v", err)
	}

	if res.Result != tsmodel.ValidationNo {
		t.Errorf("expected ValidationNo, got %s", res.Result)
	}
	if res.QuestionCount != 0 {
		t.Errorf("expected QuestionCount 0, got %d", res.QuestionCount)
	}
}

func TestGameService_EndGame(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	ctx := context.Background()
	sessionID := testhelper.UniqueTestPrefix(t) + "sess_end"

	_, err := env.svc.StartGame(ctx, sessionID, "user1", env.chatID("chat_end"), nil, nil, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	exists, _ := env.sessionStore.SessionExists(ctx, sessionID)
	if !exists {
		t.Fatal("session should exist before EndGame")
	}

	err = env.svc.EndGame(ctx, sessionID)
	if err != nil {
		t.Fatalf("EndGame failed: %v", err)
	}

	exists, _ = env.sessionStore.SessionExists(ctx, sessionID)
	if exists {
		t.Error("session should be deleted after EndGame")
	}
}
