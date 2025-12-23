package mq

import (
	"testing"
)

func TestCommandParser_ParseHelp(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		name  string
		input string
	}{
		{"prefix only", "/스프"},
		{"help KR", "/스프 도움"},
		{"help EN", "/스프 help"},
		{"with spaces", "/스프  도움"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != CommandHelp {
				t.Errorf("expected CommandHelp, got %v", cmd.Kind)
			}
		})
	}
}

func TestCommandParser_ParseStart(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		name           string
		input          string
		wantDifficulty *int
		wantInvalid    bool
	}{
		{"시작 only", "/스프 시작", nil, false},
		{"start EN", "/스프 start", nil, false},
		{"시작 with difficulty", "/스프 시작 3", intPtr(3), false},
		{"시작 with invalid input", "/스프 시작 abc", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != CommandStart {
				t.Errorf("expected CommandStart, got %v", cmd.Kind)
			}
			if tt.wantDifficulty == nil && cmd.Difficulty != nil {
				t.Errorf("expected nil difficulty, got %d", *cmd.Difficulty)
			}
			if tt.wantDifficulty != nil {
				if cmd.Difficulty == nil {
					t.Errorf("expected difficulty %d, got nil", *tt.wantDifficulty)
				} else if *cmd.Difficulty != *tt.wantDifficulty {
					t.Errorf("expected difficulty %d, got %d", *tt.wantDifficulty, *cmd.Difficulty)
				}
			}
			if cmd.HasInvalidInput != tt.wantInvalid {
				t.Errorf("expected HasInvalidInput=%v, got %v", tt.wantInvalid, cmd.HasInvalidInput)
			}
		})
	}
}

func TestCommandParser_ParseHint(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		input    string
		wantKind CommandKind
	}{
		{"/스프 힌트", CommandHint},
		{"/스프 hint", CommandHint},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
		})
	}
}

func TestCommandParser_ParseProblem(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		input    string
		wantKind CommandKind
	}{
		{"/스프 문제", CommandProblem},
		{"/스프 제시문", CommandProblem},
		{"/스프 problem", CommandProblem},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
		})
	}
}

func TestCommandParser_ParseSurrender(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		input    string
		wantKind CommandKind
	}{
		{"/스프 포기", CommandSurrender},
		{"/스프 surrender", CommandSurrender},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
		})
	}
}

func TestCommandParser_ParseAgree(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		input    string
		wantKind CommandKind
	}{
		{"/스프 동의", CommandAgree},
		{"/스프 agree", CommandAgree},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
		})
	}
}

func TestCommandParser_ParseSummary(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		input    string
		wantKind CommandKind
	}{
		{"/스프 정리", CommandSummary},
		{"/스프 summary", CommandSummary},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
		})
	}
}

func TestCommandParser_ParseAnswer(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		name       string
		input      string
		wantAnswer string
	}{
		{"정답 KR", "/스프 정답 고양이", "고양이"},
		{"answer EN", "/스프 answer the cat", "the cat"},
		{"정답 with spaces", "/스프 정답  여러  단어  정답", "여러  단어  정답"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != CommandAnswer {
				t.Errorf("expected CommandAnswer, got %v", cmd.Kind)
			}
			if cmd.Answer != tt.wantAnswer {
				t.Errorf("expected answer '%s', got '%s'", tt.wantAnswer, cmd.Answer)
			}
		})
	}
}

func TestCommandParser_ParseAnswerEmpty(t *testing.T) {
	parser := NewCommandParser("/스프")

	// "/스프 정답 " (빈 답변)은 Ask로 처리됨
	cmd := parser.Parse("/스프 정답 ")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	// 빈 정답은 parseAnswer에서 nil 반환하므로 parseAsk로 처리됨
	if cmd.Kind != CommandAsk {
		t.Errorf("expected CommandAsk for empty answer, got %v", cmd.Kind)
	}
}

func TestCommandParser_ParseAsk(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		name         string
		input        string
		wantQuestion string
	}{
		{"simple question", "/스프 사람인가요", "사람인가요"},
		{"question with spaces", "/스프 남자 인가요", "남자 인가요"},
		{"long question", "/스프 이것은 매우 긴 질문입니다", "이것은 매우 긴 질문입니다"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != CommandAsk {
				t.Errorf("expected CommandAsk, got %v", cmd.Kind)
			}
			if cmd.Question != tt.wantQuestion {
				t.Errorf("expected question '%s', got '%s'", tt.wantQuestion, cmd.Question)
			}
		})
	}
}

func TestCommandParser_InvalidInput(t *testing.T) {
	parser := NewCommandParser("/스프")

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"wrong prefix", "/wrong 시작"},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd != nil {
				t.Errorf("expected nil for invalid input '%s', got %v", tt.input, cmd.Kind)
			}
		})
	}
}

func TestCommandParser_CustomPrefix(t *testing.T) {
	parser := NewCommandParser("/turtle")

	cmd := parser.Parse("/turtle start")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandStart {
		t.Errorf("expected CommandStart, got %v", cmd.Kind)
	}
}

func TestCommandParser_EmptyPrefix(t *testing.T) {
	parser := NewCommandParser("")

	// 빈 prefix → 기본값 "/스프" 사용
	cmd := parser.Parse("/스프 시작")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandStart {
		t.Errorf("expected CommandStart, got %v", cmd.Kind)
	}
}

func intPtr(v int) *int {
	return &v
}
