package health

import (
	"context"
	"testing"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestCollectStatus(t *testing.T) {
	cfg := &config.Config{
		Gemini: config.GeminiConfig{
			APIKeys:        nil,
			DefaultModel:   "gemini-3-test",
			TimeoutSeconds: 10,
			MaxRetries:     2,
		},
		SessionStore: config.SessionStoreConfig{Enabled: false},
		Session: config.SessionConfig{
			SessionTTLMinutes: 30,
		},
	}

	resp := Collect(context.Background(), cfg, false)
	if resp.Status != "degraded" {
		t.Fatalf("expected degraded status, got %s", resp.Status)
	}
	if resp.Components["session_store"].Status != "ok" {
		t.Fatalf("expected session_store ok, got %s", resp.Components["session_store"].Status)
	}
}

func TestStoreAddress(t *testing.T) {
	addr, err := storeAddress("redis://localhost:6379")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "localhost:6379" {
		t.Fatalf("unexpected addr: %s", addr)
	}

	if _, err := storeAddress("redis://"); err == nil {
		t.Fatalf("expected error")
	}
}
