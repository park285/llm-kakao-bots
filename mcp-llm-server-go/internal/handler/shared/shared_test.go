package shared_test

import (
	"strings"
	"testing"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

func TestResolveSessionID(t *testing.T) {
	// chatID로 파생되는 경우
	id, derived := shared.ResolveSessionID("", "chat123", "", "twentyq")
	if !derived || id != "twentyq:chat123" {
		t.Fatalf("expected derived session id 'twentyq:chat123', got: %s, derived: %v", id, derived)
	}

	// namespace 지정된 경우
	id, derived = shared.ResolveSessionID("", "chat123", "custom", "twentyq")
	if !derived || id != "custom:chat123" {
		t.Fatalf("expected derived session id 'custom:chat123', got: %s", id)
	}

	// 명시적 sessionID 사용
	id, derived = shared.ResolveSessionID("explicit", "chat123", "", "twentyq")
	if derived || id != "explicit" {
		t.Fatalf("expected explicit session id, got: %s, derived: %v", id, derived)
	}

	// chatID 없는 경우
	id, derived = shared.ResolveSessionID("", "", "", "twentyq")
	if derived || id != "" {
		t.Fatalf("expected empty session id, got: %s", id)
	}
}

func TestBuildRecentQAHistoryContext(t *testing.T) {
	history := []llm.HistoryEntry{
		{Role: "user", Content: "Q: first"},
		{Role: "assistant", Content: "A: yes"},
		{Role: "user", Content: "Q: second"},
		{Role: "assistant", Content: "A: no"},
		{Role: "user", Content: "Q: third"},
		{Role: "assistant", Content: "A: maybe"},
	}

	// maxPairs=1: 최근 1쌍만 포함
	context := shared.BuildRecentQAHistoryContext(history, "[header]", 1)
	if !strings.Contains(context, "Q: third") || !strings.Contains(context, "A: maybe") {
		t.Fatalf("expected recent pair in context: %s", context)
	}
	if strings.Contains(context, "Q: first") {
		t.Fatalf("expected older history to be trimmed")
	}

	// maxPairs=0: 빈 결과
	context = shared.BuildRecentQAHistoryContext(history, "[header]", 0)
	if context != "" {
		t.Fatalf("expected empty context for maxPairs=0, got: %s", context)
	}

	// Q/A 형식이 아닌 항목 필터링
	mixedHistory := []llm.HistoryEntry{
		{Role: "user", Content: "Q: valid"},
		{Role: "assistant", Content: "A: answer"},
		{Role: "system", Content: "System message"},
	}
	context = shared.BuildRecentQAHistoryContext(mixedHistory, "[header]", 10)
	if strings.Contains(context, "System message") {
		t.Fatalf("expected system message to be filtered out")
	}
}

func TestValueOrEmpty(t *testing.T) {
	str := "test"
	if shared.ValueOrEmpty(&str) != "test" {
		t.Fatalf("expected 'test'")
	}

	if shared.ValueOrEmpty(nil) != "" {
		t.Fatalf("expected empty string for nil")
	}
}

func TestParseStringField(t *testing.T) {
	payload := map[string]any{"name": "value", "number": 123}

	val, err := shared.ParseStringField(payload, "name")
	if err != nil || val != "value" {
		t.Fatalf("expected 'value', got: %s, err: %v", val, err)
	}

	_, err = shared.ParseStringField(payload, "missing")
	if err == nil {
		t.Fatalf("expected error for missing field")
	}

	_, err = shared.ParseStringField(payload, "number")
	if err == nil {
		t.Fatalf("expected error for wrong type")
	}
}

func TestParseStringSlice(t *testing.T) {
	payload := map[string]any{
		"items":   []any{"a", "b", "c"},
		"strings": []string{"x", "y"},
		"mixed":   []any{"ok", 123},
		"number":  42,
	}

	items, err := shared.ParseStringSlice(payload, "items")
	if err != nil || len(items) != 3 {
		t.Fatalf("expected 3 items, got: %d, err: %v", len(items), err)
	}

	items, err = shared.ParseStringSlice(payload, "strings")
	if err != nil || len(items) != 2 {
		t.Fatalf("expected 2 items for []string, got: %d, err: %v", len(items), err)
	}

	_, err = shared.ParseStringSlice(payload, "mixed")
	if err == nil {
		t.Fatalf("expected error for mixed types")
	}

	_, err = shared.ParseStringSlice(payload, "number")
	if err == nil {
		t.Fatalf("expected error for wrong type")
	}

	_, err = shared.ParseStringSlice(payload, "missing")
	if err == nil {
		t.Fatalf("expected error for missing field")
	}
}

func TestSerializeDetails(t *testing.T) {
	// 빈 맵
	result, err := shared.SerializeDetails(nil)
	if err != nil || result != "" {
		t.Fatalf("expected empty for nil map, got: %s", result)
	}

	result, err = shared.SerializeDetails(map[string]any{})
	if err != nil || result != "" {
		t.Fatalf("expected empty for empty map, got: %s", result)
	}

	// HTML 이스케이프 안 함
	result, err = shared.SerializeDetails(map[string]any{"text": "<tag>"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "<tag>") {
		t.Fatalf("expected unescaped HTML, got: %s", result)
	}
}

func TestTrimRunes(t *testing.T) {
	if shared.TrimRunes("abcdef", 3) != "abc" {
		t.Fatalf("expected 'abc'")
	}

	if shared.TrimRunes("abc", 5) != "abc" {
		t.Fatalf("expected 'abc' for shorter string")
	}

	if shared.TrimRunes("abc", 0) != "" {
		t.Fatalf("expected empty for maxRunes=0")
	}

	// 멀티바이트 문자
	korean := "가나다라마바"
	if shared.TrimRunes(korean, 3) != "가나다" {
		t.Fatalf("expected '가나다'")
	}
}
