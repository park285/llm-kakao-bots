package gemini

import (
	"testing"

	"google.golang.org/genai"
)

func TestNormalizeThinkingLevel(t *testing.T) {
	level, ok := normalizeThinkingLevel("low")
	if !ok || level != genai.ThinkingLevelLow {
		t.Fatalf("unexpected thinking level")
	}

	if _, ok := normalizeThinkingLevel("none"); ok {
		t.Fatalf("expected none to be disabled")
	}

	if _, ok := normalizeThinkingLevel("unknown"); ok {
		t.Fatalf("expected unknown to be disabled")
	}
}

func TestIsGemini3(t *testing.T) {
	if !isGemini3("gemini-3-test") {
		t.Fatalf("expected gemini-3 match")
	}
	if isGemini3("gemini-2-test") {
		t.Fatalf("did not expect gemini-2 match")
	}
}
