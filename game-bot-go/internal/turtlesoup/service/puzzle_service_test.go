package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/valkey-io/valkey-go"
	"google.golang.org/grpc"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

type puzzleMockResponses struct {
	generate *llmrest.TurtleSoupPuzzleGenerationResponse
	preset   *llmrest.TurtleSoupPuzzlePresetResponse
	rewrite  *llmrest.TurtleSoupRewriteResponse
	err      bool
}

type puzzleTestEnv struct {
	svc        *PuzzleService
	llmClient  *llmrest.Client
	stopLLM    func()
	client     valkey.Client
	dedupStore *tsredis.PuzzleDedupStore
	mocks      puzzleMockResponses
	callCount  int
}

func setupPuzzleTestEnv(t *testing.T, rewriteEnabled bool) *puzzleTestEnv {
	client := testhelper.NewTestValkeyClient(t)
	testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")
	testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPuzzleGlobal)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dedupStore := tsredis.NewPuzzleDedupStore(client, logger)

	env := &puzzleTestEnv{client: client, dedupStore: dedupStore}

	stub := &turtlesoupLLMGRPCStub{
		callCount: &env.callCount,
		hasError: func() bool {
			return env.mocks.err
		},
		generatePuzzle: func() *llmrest.TurtleSoupPuzzleGenerationResponse {
			if env.mocks.generate != nil {
				return env.mocks.generate
			}
			return &llmrest.TurtleSoupPuzzleGenerationResponse{
				Title:      "Gen Title",
				Scenario:   "Gen Scenario",
				Solution:   "Gen Solution",
				Category:   "mystery",
				Difficulty: 1,
				Hints:      []string{"h1", "h2"},
			}
		},
		getRandomPuzzle: func() *llmrest.TurtleSoupPuzzlePresetResponse {
			if env.mocks.preset != nil {
				return env.mocks.preset
			}
			diff := 1
			return &llmrest.TurtleSoupPuzzlePresetResponse{
				Title:      ptr.String("Preset Title"),
				Question:   ptr.String("Preset Question"),
				Answer:     ptr.String("Preset Answer"),
				Difficulty: &diff,
			}
		},
		rewriteScenario: func() *llmrest.TurtleSoupRewriteResponse {
			if env.mocks.rewrite != nil {
				return env.mocks.rewrite
			}
			return &llmrest.TurtleSoupRewriteResponse{Scenario: "Rewritten Scenario", Solution: "Rewritten Solution"}
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

	cfg := tsconfig.PuzzleConfig{RewriteEnabled: rewriteEnabled}
	env.svc = NewPuzzleService(llmClient, cfg, dedupStore, logger)

	return env
}

func (e *puzzleTestEnv) teardown(t *testing.T) {
	testhelper.CleanupTestKeys(t, e.client, tsconfig.RedisKeyPrefix+":")
	if e.llmClient != nil {
		_ = e.llmClient.Close()
	}
	e.client.Close()
	if e.stopLLM != nil {
		e.stopLLM()
	}
}

func TestPuzzleService_GeneratePuzzle_Success(t *testing.T) {
	env := setupPuzzleTestEnv(t, false)
	defer env.teardown(t)

	ctx := context.Background()
	chatID := testhelper.UniqueTestPrefix(t) + "chat_gen_success"

	puzzle, err := env.svc.GeneratePuzzle(ctx, PuzzleGenerationRequest{}, chatID)
	if err != nil {
		t.Fatalf("GeneratePuzzle failed: %v", err)
	}

	if puzzle.Title != "Gen Title" {
		t.Errorf("expected title 'Gen Title', got '%s'", puzzle.Title)
	}
}

func TestPuzzleService_GeneratePuzzle_DedupRetry(t *testing.T) {
	env := setupPuzzleTestEnv(t, false)
	defer env.teardown(t)

	ctx := context.Background()
	chatID := testhelper.UniqueTestPrefix(t) + "chat_dedup"

	// 1. First generation success
	res1, err := env.svc.GeneratePuzzle(ctx, PuzzleGenerationRequest{}, chatID)
	if err != nil {
		t.Fatalf("First GeneratePuzzle failed: %v", err)
	}

	// 2. Second generation should retry if it returns same content
	env.callCount = 0
	res2, err := env.svc.GeneratePuzzle(ctx, PuzzleGenerationRequest{}, chatID)
	if err != nil {
		t.Fatalf("Second GeneratePuzzle failed: %v", err)
	}

	if res2.Title == res1.Title {
		t.Error("expected different puzzle (fallback to preset), got same title")
	}
	if res2.Title != "Preset Title" {
		t.Errorf("expected preset title 'Preset Title', got '%s'", res2.Title)
	}

	if env.callCount < 2 {
		t.Errorf("expected multiple calls due to retry, got %d", env.callCount)
	}
}

func TestPuzzleService_GeneratePuzzle_DeepErrors(t *testing.T) {
	env := setupPuzzleTestEnv(t, false)
	defer env.teardown(t)

	env.stopLLM() // Kill server to force connection errors

	ctx := context.Background()
	_, err := env.svc.GeneratePuzzle(ctx, PuzzleGenerationRequest{}, testhelper.UniqueTestPrefix(t)+"chat_err")
	if err == nil {
		t.Error("expected error when server is down")
	}
}

func TestPuzzleService_RewriteLogic(t *testing.T) {
	env := setupPuzzleTestEnv(t, true)
	defer env.teardown(t)

	env.mocks.generate = &llmrest.TurtleSoupPuzzleGenerationResponse{
		Title: "", // Empty title invalid
	}

	ctx := context.Background()
	puzzle, err := env.svc.GeneratePuzzle(ctx, PuzzleGenerationRequest{}, testhelper.UniqueTestPrefix(t)+"chat_rewrite")
	if err != nil {
		t.Fatalf("GeneratePuzzle failed: %v", err)
	}

	if puzzle.Scenario != "Rewritten Scenario" {
		t.Errorf("expected rewritten scenario, got '%s'", puzzle.Scenario)
	}
}
