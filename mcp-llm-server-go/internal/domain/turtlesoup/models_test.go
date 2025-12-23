package turtlesoup

import "testing"

func TestParseBaseAnswer(t *testing.T) {
	base, ok := ParseBaseAnswer("예, 중요한 질문입니다!")
	if !ok || base != AnswerYes {
		t.Fatalf("unexpected base answer: %v %v", base, ok)
	}

	base, ok = ParseBaseAnswer("아니오 하지만 중요한 질문입니다!")
	if !ok || base != AnswerNo {
		t.Fatalf("unexpected base answer: %v %v", base, ok)
	}
}

func TestImportantAnswerDetection(t *testing.T) {
	if !IsImportantAnswer("중요한 질문입니다!") {
		t.Fatalf("expected important to be detected")
	}
	if IsImportantAnswer("관계없습니다.") {
		t.Fatalf("did not expect important")
	}
}

func TestParseValidationResult(t *testing.T) {
	result, ok := ParseValidationResult("CLOSE")
	if !ok || result != ValidationClose {
		t.Fatalf("unexpected validation parse: %v %v", result, ok)
	}
}
