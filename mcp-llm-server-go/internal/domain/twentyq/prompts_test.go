package twentyq

import "testing"

func TestPromptsLoad(t *testing.T) {
	prompts, err := NewPrompts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	system, err := prompts.HintsSystem("food")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if system == "" {
		t.Fatalf("expected system prompt")
	}
}
