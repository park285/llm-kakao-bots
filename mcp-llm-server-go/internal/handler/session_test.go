package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/goccy/go-json"
	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/metrics"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

func TestSessionHandlerCreateGetDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mini := miniredis.RunT(t)
	cfg := &config.Config{
		SessionStore: config.SessionStoreConfig{URL: "redis://" + mini.Addr(), Enabled: true, DisableCache: true},
		Session:      config.SessionConfig{SessionTTLMinutes: 1, HistoryMaxPairs: 1},
		Guard:        config.GuardConfig{Enabled: false},
		Gemini:       config.GeminiConfig{DefaultModel: "gemini-3-test"},
	}

	store, err := session.NewStore(cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	client, err := gemini.NewClient(cfg, metrics.NewStore(), nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	manager := session.NewManager(store, client, cfg, logger)
	g, err := guard.NewGuard(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create guard: %v", err)
	}

	handler := NewSessionHandler(manager, g, logger)
	router := gin.New()
	handler.RegisterRoutes(router)

	createReq := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	var info session.Info
	if err := json.Unmarshal(createResp.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if info.ID == "" {
		t.Fatalf("expected session id")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/sessions/"+info.ID, nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+info.ID, nil)
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteResp.Code)
	}

	getMissing := httptest.NewRequest(http.MethodGet, "/api/sessions/"+info.ID, nil)
	getMissingResp := httptest.NewRecorder()
	router.ServeHTTP(getMissingResp, getMissing)
	if getMissingResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", getMissingResp.Code)
	}
}
