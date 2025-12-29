package turtlesoup

import (
	"strings"
	"testing"
)

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

	// 암시적 캐싱 최적화: question만 전달
	user, err := prompts.AnswerUser("question")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == "" {
		t.Fatalf("expected user prompt")
	}

	// Static Prefix 테스트: 퍼즐 정보 포함
	puzzle := "scenario: \"테스트 시나리오\"\nsolution: \"테스트 정답\""
	systemWithPuzzle, err := prompts.AnswerSystemWithPuzzle(puzzle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if systemWithPuzzle == "" {
		t.Fatalf("expected system prompt with puzzle")
	}
	// 퍼즐 헤더 포함 확인
	if !strings.Contains(systemWithPuzzle, "[이번 게임의 시나리오]") {
		t.Fatalf("expected puzzle header in system prompt")
	}
	// 퍼즐 내용 포함 확인
	if !strings.Contains(systemWithPuzzle, puzzle) {
		t.Fatalf("expected puzzle content in system prompt")
	}
	// 경고 문구 포함 확인
	if !strings.Contains(systemWithPuzzle, "절대 직접 노출하지 마시오") {
		t.Fatalf("expected warning in system prompt")
	}
}

func TestAnswerUserMinimal(t *testing.T) {
	prompts, err := NewPrompts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 현재 질문만 포함되어야 함
	question := "이것은 사람인가요?"
	user, err := prompts.AnswerUser(question)
	if err != nil {
		t.Fatalf("AnswerUser error: %v", err)
	}
	if !strings.Contains(user, question) {
		t.Fatalf("expected question in user prompt")
	}
	// 이전 포맷([Puzzle Info], [이전 질문/답변 기록])이 없어야 함
	if strings.Contains(user, "[Puzzle Info]") {
		t.Fatalf("user prompt should not contain [Puzzle Info]")
	}
	if strings.Contains(user, "[이전 질문/답변 기록]") {
		t.Fatalf("user prompt should not contain history header")
	}
}
