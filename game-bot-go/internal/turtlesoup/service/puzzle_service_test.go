package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
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
	client     valkey.Client
	ts         *httptest.Server
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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.callCount++
		if env.mocks.err {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path

		if strings.Contains(path, "/api/turtle-soup/puzzles/random") {
			if env.mocks.preset != nil {
				json.NewEncoder(w).Encode(env.mocks.preset)
			} else {
				diff := 1
				if d := r.URL.Query().Get("difficulty"); d != "" {
					diff, _ = strconv.Atoi(d)
				}
				json.NewEncoder(w).Encode(llmrest.TurtleSoupPuzzlePresetResponse{
					Title:      ptr.String("Preset Title"),
					Question:   ptr.String("Preset Question"),
					Answer:     ptr.String("Preset Answer"),
					Difficulty: &diff,
				})
			}
			return
		}

		if strings.Contains(path, "/api/turtle-soup/puzzles") {
			if env.mocks.generate != nil {
				json.NewEncoder(w).Encode(env.mocks.generate)
			} else {
				json.NewEncoder(w).Encode(llmrest.TurtleSoupPuzzleGenerationResponse{
					Title:      "Gen Title",
					Scenario:   "Gen Scenario",
					Solution:   "Gen Solution",
					Category:   "mystery",
					Difficulty: 1,
					Hints:      []string{"h1", "h2"},
				})
			}
			return
		}

		if strings.Contains(path, "/api/turtle-soup/rewrites") {
			if env.mocks.rewrite != nil {
				json.NewEncoder(w).Encode(env.mocks.rewrite)
			} else {
				json.NewEncoder(w).Encode(llmrest.TurtleSoupRewriteResponse{
					Scenario: "Rewritten Scenario",
					Solution: "Rewritten Solution",
				})
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	env.ts = ts

	llmClient, err := llmrest.New(llmrest.Config{
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("llm client init failed: %v", err)
	}

	cfg := tsconfig.PuzzleConfig{RewriteEnabled: rewriteEnabled}
	env.svc = NewPuzzleService(llmClient, cfg, dedupStore, logger)

	return env
}

func (e *puzzleTestEnv) teardown(t *testing.T) {
	testhelper.CleanupTestKeys(t, e.client, tsconfig.RedisKeyPrefix+":")
	e.client.Close()
	e.ts.Close()
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

	env.ts.Close() // Kill server to force connection errors

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
