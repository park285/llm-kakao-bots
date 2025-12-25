package handler

import (
	"strings"
	"testing"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

func TestResolveSessionID(t *testing.T) {
	id, derived := shared.ResolveSessionID("", "chat123", "", "twentyq")
	if !derived || id != "twentyq:chat123" {
		t.Fatalf("unexpected session id: %s", id)
	}

	id, derived = shared.ResolveSessionID("explicit", "chat123", "", "twentyq")
	if derived || id != "explicit" {
		t.Fatalf("unexpected session id: %s", id)
	}
}

func TestBuildRecentQAHistoryContext(t *testing.T) {
	history := []llm.HistoryEntry{
		{Role: "user", Content: "Q: first"},
		{Role: "assistant", Content: "A: yes"},
		{Role: "user", Content: "Q: second"},
		{Role: "assistant", Content: "A: no"},
	}
	context := shared.BuildRecentQAHistoryContext(history, "[header]", 1)
	if !strings.Contains(context, "Q: second") || !strings.Contains(context, "A: no") {
		t.Fatalf("unexpected history context: %s", context)
	}
	if strings.Contains(context, "Q: first") {
		t.Fatalf("expected older history to be trimmed")
	}
}

func TestSerializeDetails(t *testing.T) {
	payload, err := shared.SerializeDetails(map[string]any{"text": "<tag>"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(payload, "<tag>") {
		t.Fatalf("expected unescaped value: %s", payload)
	}
}

func TestParseStringHelpers(t *testing.T) {
	items, err := shared.ParseStringSlice(map[string]any{"hints": []any{"a", "b"}}, "hints")
	if err != nil || len(items) != 2 {
		t.Fatalf("unexpected parse result")
	}

	if _, err := shared.ParseStringSlice(map[string]any{"hints": []any{1}}, "hints"); err == nil {
		t.Fatalf("expected error")
	}

	value, err := shared.ParseStringField(map[string]any{"result": "ok"}, "result")
	if err != nil || value != "ok" {
		t.Fatalf("unexpected field value")
	}
}

func TestTrimRunes(t *testing.T) {
	if trimmed := shared.TrimRunes("abcdef", 3); trimmed != "abc" {
		t.Fatalf("unexpected trim result: %s", trimmed)
	}
}
