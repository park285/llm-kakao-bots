package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

func newTestStore(t *testing.T, historyMaxPairs int) (*Store, *miniredis.Miniredis) {
	mini := miniredis.RunT(t)
	cfg := &config.Config{
		SessionStore: config.SessionStoreConfig{URL: "redis://" + mini.Addr(), Enabled: true, DisableCache: true},
		Session: config.SessionConfig{
			SessionTTLMinutes: 1,
			HistoryMaxPairs:   historyMaxPairs,
		},
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
		mini.Close()
	})
	return store, mini
}

func TestNewStoreDisabled(t *testing.T) {
	cfg := &config.Config{
		SessionStore: config.SessionStoreConfig{Enabled: false, Required: true},
	}
	if _, err := NewStore(cfg); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewStoreFallsBackToMemoryWhenRedisDisabled(t *testing.T) {
	cfg := &config.Config{
		SessionStore: config.SessionStoreConfig{Enabled: false, Required: false},
		Session: config.SessionConfig{
			SessionTTLMinutes: 1,
			HistoryMaxPairs:   1,
		},
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("expected memory store, got error: %v", err)
	}

	now := time.Now()
	meta := Meta{ID: "s1", SystemPrompt: "sys", Model: "m1", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateSession(context.Background(), meta); err != nil {
		t.Fatalf("create session: %v", err)
	}

	loaded, err := store.GetSession(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.ID != "s1" || loaded.Model != "m1" {
		t.Fatalf("unexpected session: %+v", loaded)
	}

	if err := store.AppendHistory(context.Background(), "s1", llm.HistoryEntry{Role: "user", Content: "one"}); err != nil {
		t.Fatalf("append history: %v", err)
	}
	history, err := store.GetHistory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected history, got %d", len(history))
	}

	if err := store.DeleteSession(context.Background(), "s1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
}

func TestNewStoreMiniredisRequiresDisableCache(t *testing.T) {
	mini := miniredis.RunT(t)
	cfg := &config.Config{
		SessionStore: config.SessionStoreConfig{URL: "redis://" + mini.Addr(), Enabled: true, DisableCache: false},
		Session: config.SessionConfig{
			SessionTTLMinutes: 1,
			HistoryMaxPairs:   1,
		},
	}
	if _, err := NewStore(cfg); err == nil {
		t.Fatalf("expected error")
	} else if !errors.Is(err, valkey.ErrNoCache) {
		t.Fatalf("expected valkey.ErrNoCache, got: %v", err)
	}
}

func TestStoreCRUD(t *testing.T) {
	store, _ := newTestStore(t, 2)

	now := time.Now()
	meta := Meta{ID: "s1", SystemPrompt: "sys", Model: "m1", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateSession(context.Background(), meta); err != nil {
		t.Fatalf("create session: %v", err)
	}

	loaded, err := store.GetSession(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.ID != "s1" || loaded.Model != "m1" {
		t.Fatalf("unexpected session: %+v", loaded)
	}

	loaded.MessageCount = 2
	if err := store.UpdateSession(context.Background(), *loaded); err != nil {
		t.Fatalf("update session: %v", err)
	}

	if err := store.DeleteSession(context.Background(), "s1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if _, err := store.GetSession(context.Background(), "s1"); err == nil {
		t.Fatalf("expected not found")
	}
}

func TestStoreHistoryTrim(t *testing.T) {
	store, _ := newTestStore(t, 1)

	if err := store.CreateSession(context.Background(), Meta{ID: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	entries := []llm.HistoryEntry{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
	}
	if err := store.AppendHistory(context.Background(), "s1", entries...); err != nil {
		t.Fatalf("append history: %v", err)
	}

	history, err := store.GetHistory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected trimmed history, got %d", len(history))
	}
	if history[0].Content != "two" || history[1].Content != "three" {
		t.Fatalf("unexpected history order")
	}
}

func TestStoreSessionCountAndPing(t *testing.T) {
	store, _ := newTestStore(t, 1)

	if err := store.CreateSession(context.Background(), Meta{ID: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := store.CreateSession(context.Background(), Meta{ID: "s2", CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	count, err := store.SessionCount(context.Background())
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 sessions, got %d", count)
	}

	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
