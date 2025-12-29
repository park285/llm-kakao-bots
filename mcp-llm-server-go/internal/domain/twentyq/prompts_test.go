package twentyq

import (
	"strings"
	"testing"
)

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

func TestAnswerPrompts(t *testing.T) {
	prompts, err := NewPrompts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AnswerSystem 기본 테스트
	system, err := prompts.AnswerSystem()
	if err != nil {
		t.Fatalf("AnswerSystem error: %v", err)
	}
	if system == "" {
		t.Fatalf("expected system prompt")
	}

	// AnswerSystemWithSecret: Static Prefix 테스트
	secret := "target: 사과\ncategory: 음식"
	systemWithSecret, err := prompts.AnswerSystemWithSecret(secret)
	if err != nil {
		t.Fatalf("AnswerSystemWithSecret error: %v", err)
	}
	if !strings.Contains(systemWithSecret, "[이번 게임의 정답]") {
		t.Fatalf("expected secret header in system prompt")
	}
	if !strings.Contains(systemWithSecret, secret) {
		t.Fatalf("expected secret content in system prompt")
	}

	// AnswerUser: 질문만 포함 테스트
	question := "이것은 먹을 수 있나요?"
	user, err := prompts.AnswerUser(question)
	if err != nil {
		t.Fatalf("AnswerUser error: %v", err)
	}
	if !strings.Contains(user, question) {
		t.Fatalf("expected question in user prompt")
	}
}
