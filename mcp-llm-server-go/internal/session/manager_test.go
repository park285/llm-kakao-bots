package session

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/metrics"
)

func TestManagerCreateGetDelete(t *testing.T) {
	store, _ := newTestStore(t, 1)
	cfg := &config.Config{
		Session:      config.SessionConfig{SessionTTLMinutes: 1, HistoryMaxPairs: 1},
		SessionStore: config.SessionStoreConfig{URL: "", Enabled: true},
	}

	client, err := gemini.NewClient(cfg, metrics.NewStore(), nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	manager := NewManager(store, client, cfg, logger)
	info, err := manager.Create(context.Background(), CreateSessionRequest{SystemPrompt: "sys", Model: "m1"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if info.ID == "" {
		t.Fatalf("expected session id")
	}

	loaded, err := manager.Get(context.Background(), info.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.ID != info.ID || loaded.Model != "m1" {
		t.Fatalf("unexpected session info")
	}

	if err := manager.Delete(context.Background(), info.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if _, err := store.GetSession(context.Background(), info.ID); err == nil {
		t.Fatalf("expected session not found")
	}

	meta := Meta{ID: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.CreateSession(context.Background(), meta); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if count := manager.Count(context.Background()); count != 1 {
		t.Fatalf("expected session count to be 1, got %d", count)
	}
}
