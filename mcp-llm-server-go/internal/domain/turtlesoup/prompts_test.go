package turtlesoup

import "testing"

func TestPromptsLoad(t *testing.T) {
	prompts, err := NewPrompts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	system, err := prompts.AnswerSystem()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if system == "" {
		t.Fatalf("expected system prompt")
	}

	user, err := prompts.AnswerUser("puzzle", "question", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == "" {
		t.Fatalf("expected user prompt")
	}
}
