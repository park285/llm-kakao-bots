package handler

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	domain "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

type fakeLLMClient struct {
	chatFn       func(ctx context.Context, req gemini.Request) (string, string, error)
	structuredFn func(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error)
}

func (f fakeLLMClient) Chat(ctx context.Context, req gemini.Request) (string, string, error) {
	if f.chatFn == nil {
		return "", "gemini-3-test", nil
	}
	return f.chatFn(ctx, req)
}

func (f fakeLLMClient) Structured(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error) {
	if f.structuredFn == nil {
		return map[string]any{}, "gemini-3-test", nil
	}
	return f.structuredFn(ctx, req, schema)
}

func newTestTurtleSoupHandler(t *testing.T, client LLMClient) (*TurtleSoupHandler, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	mini := miniredis.RunT(t)
	cfg := &config.Config{
		SessionStore: config.SessionStoreConfig{URL: "redis://" + mini.Addr(), Enabled: true, DisableCache: true},
		Session:      config.SessionConfig{SessionTTLMinutes: 1, HistoryMaxPairs: 2},
		Guard:        config.GuardConfig{Enabled: false},
		Gemini:       config.GeminiConfig{DefaultModel: "gemini-3-test"},
	}

	store, err := session.NewStore(cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
		mini.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	injectionGuard, err := guard.NewGuard(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create guard: %v", err)
	}

	prompts, err := domain.NewPrompts()
	if err != nil {
		t.Fatalf("failed to load prompts: %v", err)
	}

	loader, err := domain.NewPuzzleLoader()
	if err != nil {
		t.Fatalf("failed to load puzzles: %v", err)
	}

	handler := NewTurtleSoupHandler(cfg, client, injectionGuard, store, prompts, loader, logger)
	router := gin.New()
	handler.RegisterRoutes(router)
	return handler, router
}

func TestTurtleSoupAnswerHistory(t *testing.T) {
	client := fakeLLMClient{
		structuredFn: func(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error) {
			if strings.Contains(req.Prompt, "q2") {
				return map[string]any{"answer": "아니오", "important": true}, "gemini-3-test", nil
			}
			if strings.Contains(req.Prompt, "q1") {
				return map[string]any{"answer": "예", "important": true}, "gemini-3-test", nil
			}
			return map[string]any{"answer": "예", "important": false}, "gemini-3-test", nil
		},
	}
	_, router := newTestTurtleSoupHandler(t, client)

	reqBody1, _ := json.Marshal(map[string]any{
		"chat_id":   "c1",
		"namespace": "ns",
		"scenario":  "scenario",
		"solution":  "solution",
		"question":  "q1",
	})
	req1 := httptest.NewRequest(http.MethodPost, "/api/turtle-soup/answers", bytes.NewBuffer(reqBody1))
	req1.Header.Set("Content-Type", "application/json")
	resp1 := httptest.NewRecorder()
	router.ServeHTTP(resp1, req1)
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp1.Code)
	}

	var out1 TurtleSoupAnswerResponse
	if err := json.Unmarshal(resp1.Body.Bytes(), &out1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out1.QuestionCount != 1 {
		t.Fatalf("expected question_count 1, got %d", out1.QuestionCount)
	}
	if len(out1.History) != 1 || out1.History[0].Question != "q1" {
		t.Fatalf("unexpected history: %+v", out1.History)
	}
	if out1.Answer != "예, 중요한 질문입니다!" {
		t.Fatalf("unexpected answer: %q", out1.Answer)
	}

	reqBody2, _ := json.Marshal(map[string]any{
		"chat_id":   "c1",
		"namespace": "ns",
		"scenario":  "scenario",
		"solution":  "solution",
		"question":  "q2",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/turtle-soup/answers", bytes.NewBuffer(reqBody2))
	req2.Header.Set("Content-Type", "application/json")
	resp2 := httptest.NewRecorder()
	router.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.Code)
	}

	var out2 TurtleSoupAnswerResponse
	if err := json.Unmarshal(resp2.Body.Bytes(), &out2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out2.QuestionCount != 2 {
		t.Fatalf("expected question_count 2, got %d", out2.QuestionCount)
	}
	if len(out2.History) != 2 || out2.History[0].Question != "q1" || out2.History[1].Question != "q2" {
		t.Fatalf("unexpected history: %+v", out2.History)
	}
	if out2.Answer != "아니오 하지만 중요한 질문입니다!" {
		t.Fatalf("unexpected answer: %q", out2.Answer)
	}
}

func TestTurtleSoupHintStructured(t *testing.T) {
	client := fakeLLMClient{
		structuredFn: func(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error) {
			return map[string]any{"hint": "힌트입니다"}, "gemini-3-test", nil
		},
	}
	_, router := newTestTurtleSoupHandler(t, client)

	reqBody, _ := json.Marshal(map[string]any{
		"scenario": "scenario",
		"solution": "solution",
		"level":    2,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/turtle-soup/hints", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var out TurtleSoupHintResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.Hint != "힌트입니다" || out.Level != 2 {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestTurtleSoupValidateReturnsEnum(t *testing.T) {
	client := fakeLLMClient{
		structuredFn: func(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error) {
			return map[string]any{"result": "CLOSE"}, "gemini-3-test", nil
		},
	}
	_, router := newTestTurtleSoupHandler(t, client)

	reqBody, _ := json.Marshal(map[string]any{
		"solution":      "solution",
		"player_answer": "answer",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/turtle-soup/validations", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var out TurtleSoupValidateResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.Result != "CLOSE" {
		t.Fatalf("unexpected result: %q", out.Result)
	}
}

func TestTurtleSoupRewriteStructured(t *testing.T) {
	client := fakeLLMClient{
		structuredFn: func(ctx context.Context, req gemini.Request, schema map[string]any) (map[string]any, string, error) {
			return map[string]any{"scenario": "new scenario", "solution": "new solution"}, "gemini-3-test", nil
		},
	}
	_, router := newTestTurtleSoupHandler(t, client)

	reqBody, _ := json.Marshal(map[string]any{
		"title":      "title",
		"scenario":   "old scenario",
		"solution":   "old solution",
		"difficulty": 3,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/turtle-soup/rewrites", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var out TurtleSoupRewriteResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.Scenario != "new scenario" || out.Solution != "new solution" {
		t.Fatalf("unexpected rewrite: %+v", out)
	}
	if out.OriginalScenario != "old scenario" || out.OriginalSolution != "old solution" {
		t.Fatalf("unexpected original fields: %+v", out)
	}
}

func TestTurtleSoupPuzzleEndpoints(t *testing.T) {
	client := fakeLLMClient{}
	_, router := newTestTurtleSoupHandler(t, client)

	randomReq := httptest.NewRequest(http.MethodGet, "/api/turtle-soup/puzzles/random?difficulty=1", nil)
	randomResp := httptest.NewRecorder()
	router.ServeHTTP(randomResp, randomReq)
	if randomResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", randomResp.Code)
	}

	var puzzle domain.PuzzlePreset
	if err := json.Unmarshal(randomResp.Body.Bytes(), &puzzle); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if puzzle.Difficulty != 1 || puzzle.Title == "" || puzzle.Question == "" || puzzle.Answer == "" {
		t.Fatalf("unexpected puzzle: %+v", puzzle)
	}

	reloadReq := httptest.NewRequest(http.MethodPost, "/api/turtle-soup/puzzles/reload", nil)
	reloadResp := httptest.NewRecorder()
	router.ServeHTTP(reloadResp, reloadReq)
	if reloadResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", reloadResp.Code)
	}

	var reloadPayload map[string]any
	if err := json.Unmarshal(reloadResp.Body.Bytes(), &reloadPayload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if value, ok := reloadPayload["success"].(bool); !ok || !value {
		t.Fatalf("expected success true")
	}
}
