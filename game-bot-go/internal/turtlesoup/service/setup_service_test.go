package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	json "github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

func TestPrepareNewGame(t *testing.T) {
	client := testhelper.NewTestValkeyClient(t)
	defer client.Close()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	sessionStore := tsredis.NewSessionStore(client, logger)
	lockManager := tsredis.NewLockManager(client, logger)
	sessionManager := NewGameSessionManager(sessionStore, lockManager)
	dedupStore := tsredis.NewPuzzleDedupStore(client, logger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session_id": "sess_llm_1",
			})
			return
		}
		if r.URL.Path == "/api/turtle-soup/puzzles" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"title":      "Test Puzzle",
				"scenario":   "A man walks into a bar...",
				"solution":   "He was blind.",
				"category":   "mystery",
				"difficulty": 3,
				"hints":      []string{"Hint 1"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	llmClient, err := llmrest.New(llmrest.Config{
		BaseURL: server.URL,
		Timeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	puzzleCfg := tsconfig.PuzzleConfig{
		RewriteEnabled: false,
	}
	puzzleService := NewPuzzleService(llmClient, puzzleCfg, dedupStore, logger)

	setupService := NewGameSetupService(llmClient, puzzleService, sessionManager, logger)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		sessionID := prefix + "session_1"
		res, err := setupService.PrepareNewGame(ctx, sessionID, "user_1", "chat_1", nil, nil, nil)
		if err != nil {
			t.Fatalf("PrepareNewGame failed: %v", err)
		}
		if res.State.SessionID != sessionID {
			t.Errorf("expected %s, got %s", sessionID, res.State.SessionID)
		}
		if res.Puzzle.Title != "Test Puzzle" {
			t.Errorf("expected Test Puzzle, got %s", res.Puzzle.Title)
		}
	})

	t.Run("AlreadyExists_NotSolved", func(t *testing.T) {
		sessionID := prefix + "session_existing"
		setupService.PrepareNewGame(ctx, sessionID, "user_1", "chat_2", nil, nil, nil)

		_, err := setupService.PrepareNewGame(ctx, sessionID, "user_1", "chat_2", nil, nil, nil)
		if err == nil {
			t.Error("expected error for existing active session")
		}
	})

	t.Run("AlreadyExists_Solved", func(t *testing.T) {
		// 서브테스트 전에 dedup 키 정리 (global 키)
		testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":puzzle:")

		sessionID := prefix + "session_solved"
		chatID := prefix + "chat_solved"
		err := sessionStore.SaveGameState(ctx, tsmodel.GameState{
			SessionID: sessionID,
			UserID:    "user_1",
			IsSolved:  true,
		})
		if err != nil {
			t.Fatal(err)
		}

		res, err := setupService.PrepareNewGame(ctx, sessionID, "user_1", chatID, nil, nil, nil)
		if err != nil {
			t.Fatalf("PrepareNewGame failed for solved session: %v", err)
		}
		if res.State.SessionID != sessionID {
			t.Errorf("expected %s, got %s", sessionID, res.State.SessionID)
		}
	})

	t.Run("PuzzleGenerationError", func(t *testing.T) {
		serverErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer serverErr.Close()

		clientErr, _ := llmrest.New(llmrest.Config{BaseURL: serverErr.URL})
		svcErr := NewGameSetupService(clientErr, NewPuzzleService(clientErr, puzzleCfg, dedupStore, logger), sessionManager, logger)

		_, err := svcErr.PrepareNewGame(ctx, prefix+"session_err", "user_1", "chat_err", nil, nil, nil)
		if err == nil {
			t.Error("expected error due to puzzle generation failure")
		}
	})

}
