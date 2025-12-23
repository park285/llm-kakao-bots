package mq

import (
	"testing"

	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func TestCommandParser_ParseChainedQuestion_IFTRUE(t *testing.T) {
	// Given
	parser := NewCommandParser("/스자")
	input := "/스자 if 동물인가요, 척추동물인가요"

	// When
	cmd := parser.Parse(input)

	// Then
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandChainedQuestion {
		t.Fatalf("expected CommandChainedQuestion, got %v", cmd.Kind)
	}
	if cmd.ChainCondition != qmodel.ChainConditionIfTrue {
		t.Errorf("expected ChainConditionIfTrue, got %v", cmd.ChainCondition)
	}
	if len(cmd.ChainQuestions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(cmd.ChainQuestions))
	}
	if cmd.ChainQuestions[0] != "동물인가요" {
		t.Errorf("expected '동물인가요', got '%s'", cmd.ChainQuestions[0])
	}
	if cmd.ChainQuestions[1] != "척추동물인가요" {
		t.Errorf("expected '척추동물인가요', got '%s'", cmd.ChainQuestions[1])
	}
}

func TestCommandParser_ParseChainedQuestion_ALWAYS(t *testing.T) {
	// Given
	parser := NewCommandParser("/스자")
	input := "/스자 동물인가요, 척추동물인가요, 포유류인가요"

	// When
	cmd := parser.Parse(input)

	// Then
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandChainedQuestion {
		t.Fatalf("expected CommandChainedQuestion, got %v", cmd.Kind)
	}
	if cmd.ChainCondition != qmodel.ChainConditionAlways {
		t.Errorf("expected ChainConditionAlways, got %v", cmd.ChainCondition)
	}
	if len(cmd.ChainQuestions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(cmd.ChainQuestions))
	}
}

func TestCommandParser_ParseChainedQuestion_UppercaseIF(t *testing.T) {
	// Given
	parser := NewCommandParser("/스자")
	input := "/스자 IF 동물인가요, 척추동물인가요"

	// When
	cmd := parser.Parse(input)

	// Then
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandChainedQuestion {
		t.Fatalf("expected CommandChainedQuestion, got %v", cmd.Kind)
	}
	if cmd.ChainCondition != qmodel.ChainConditionIfTrue {
		t.Errorf("expected ChainConditionIfTrue, got %v", cmd.ChainCondition)
	}
}

func TestCommandParser_ParseChainedQuestion_SingleQuestionWithIfIsAsk(t *testing.T) {
	// Given
	parser := NewCommandParser("/스자")
	input := "/스자 if 동물인가요"

	// When
	cmd := parser.Parse(input)

	// Then
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	// 쉼표 없으면 일반 Ask 명령으로 파싱됨
	if cmd.Kind != CommandAsk {
		t.Fatalf("expected CommandAsk, got %v", cmd.Kind)
	}
	if cmd.Question != "if 동물인가요" {
		t.Errorf("expected 'if 동물인가요', got '%s'", cmd.Question)
	}
}

func TestCommandParser_ParseStart(t *testing.T) {
	parser := NewCommandParser("/스자")

	tests := []struct {
		name      string
		input     string
		wantKind  CommandKind
		wantCateg int
	}{
		{"시작 only", "/스자 시작", CommandStart, 0},
		{"시작 with category", "/스자 시작 동물", CommandStart, 1},
		{"시작 with multiple categories", "/스자 시작 동물 식물", CommandStart, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
			if len(cmd.Categories) != tt.wantCateg {
				t.Errorf("expected %d categories, got %d", tt.wantCateg, len(cmd.Categories))
			}
		})
	}
}

func TestCommandParser_ParseHints(t *testing.T) {
	parser := NewCommandParser("/스자")

	tests := []struct {
		name     string
		input    string
		wantKind CommandKind
	}{
		{"힌트", "/스자 힌트", CommandHints},
		{"ㅎㅌ", "/스자 ㅎㅌ", CommandHints},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	parser := NewCommandParser("/스자")

	tests := []struct {
		name     string
		input    string
		wantKind CommandKind
	}{
		{"하남자", "/스자 하남자", CommandSurrender},
		{"포기", "/스자 포기", CommandSurrender},
		{"동의", "/스자 동의", CommandAgree},
		{"거부", "/스자 거부", CommandReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestCommandParser_ParseAsk(t *testing.T) {
	parser := NewCommandParser("/스자")

	tests := []struct {
		name         string
		input        string
		wantQuestion string
	}{
		{"simple question", "/스자 동물인가요", "동물인가요"},
		{"question with spaces", "/스자 사람 인가요", "사람 인가요"},
		// 정답 prefix는 함께 포함되어 전달됨 (RiddleService에서 처리)
		{"정답 prefix", "/스자 정답 고양이", "정답 고양이"},
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
				t.Errorf("expected '%s', got '%s'", tt.wantQuestion, cmd.Question)
			}
		})
	}
}

func TestCommandParser_ParseHelp(t *testing.T) {
	parser := NewCommandParser("/스자")

	// /스자 (prefix만) -> Help
	cmd := parser.Parse("/스자")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandHelp {
		t.Errorf("expected CommandHelp, got %v", cmd.Kind)
	}
}

func TestCommandParser_ParseStatus(t *testing.T) {
	parser := NewCommandParser("/스자")

	tests := []struct {
		input    string
		wantKind CommandKind
	}{
		{"/스자 현황", CommandStatus},
		{"/스자 상황", CommandStatus},
		{"/스자 현재", CommandStatus},
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

func TestCommandParser_InvalidInput(t *testing.T) {
	parser := NewCommandParser("/스자")

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

func TestCommandParser_ParseModelInfo(t *testing.T) {
	parser := NewCommandParser("/스자")

	cmd := parser.Parse("/스자 모델")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandModelInfo {
		t.Errorf("expected CommandModelInfo, got %v", cmd.Kind)
	}
}

func TestCommandParser_ParseUserStats(t *testing.T) {
	parser := NewCommandParser("/스자")

	// /스자 전적 -> UserStats
	cmd := parser.Parse("/스자 전적")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Kind != CommandUserStats {
		t.Errorf("expected CommandUserStats, got %v", cmd.Kind)
	}

	// /스자 전적 닉네임 -> UserStats with target
	cmd2 := parser.Parse("/스자 전적 홍길동")
	if cmd2 == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd2.Kind != CommandUserStats {
		t.Errorf("expected CommandUserStats, got %v", cmd2.Kind)
	}
	if cmd2.TargetNickname == nil || *cmd2.TargetNickname != "홍길동" {
		t.Errorf("expected target nickname '홍길동', got %v", cmd2.TargetNickname)
	}
}

func TestCommandParser_ParseAdmin(t *testing.T) {
	parser := NewCommandParser("/스자")

		tests := []struct {
			name     string
			input    string
			wantKind CommandKind
		}{
			{"admin force-end EN", "/스자 admin force-end", CommandAdminForceEnd},
			{"admin force-end KR", "/스자 관리자 강제종료", CommandAdminForceEnd},
			{"admin clear-all EN", "/스자 admin clear-all", CommandAdminClearAll},
			{"admin clear-all KR", "/스자 관리자 전체삭제", CommandAdminClearAll},
			// 대소문자 무시 테스트
			{"admin force-end uppercase", "/스자 ADMIN FORCE-END", CommandAdminForceEnd},
		}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestCommandParser_ParseUsage(t *testing.T) {
	parser := NewCommandParser("/스자")

	tests := []struct {
		name       string
		input      string
		wantKind   CommandKind
		wantPeriod qmodel.UsagePeriod
		wantModel  *string
	}{
		{"사용량 only", "/스자 사용량", CommandAdminUsage, qmodel.UsagePeriodToday, nil},
		{"usage EN", "/스자 usage", CommandAdminUsage, qmodel.UsagePeriodToday, nil},
		{"사용량 오늘", "/스자 사용량 오늘", CommandAdminUsage, qmodel.UsagePeriodToday, nil},
		{"사용량 주간", "/스자 사용량 주간", CommandAdminUsage, qmodel.UsagePeriodWeekly, nil},
		{"사용량 월간", "/스자 사용량 월간", CommandAdminUsage, qmodel.UsagePeriodMonthly, nil},
		{"usage weekly", "/스자 usage weekly", CommandAdminUsage, qmodel.UsagePeriodWeekly, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
			if cmd.UsagePeriod != tt.wantPeriod {
				t.Errorf("expected period %v, got %v", tt.wantPeriod, cmd.UsagePeriod)
			}
			if tt.wantModel == nil && cmd.ModelOverride != nil {
				t.Errorf("expected nil model, got %v", *cmd.ModelOverride)
			}
			if tt.wantModel != nil && (cmd.ModelOverride == nil || *cmd.ModelOverride != *tt.wantModel) {
				t.Errorf("expected model %v, got %v", *tt.wantModel, cmd.ModelOverride)
			}
		})
	}
}

func TestCommandParser_ParseUsageWithModel(t *testing.T) {
	parser := NewCommandParser("/스자")

	flash25 := "flash-25"
	flash30 := "flash-30"
	pro25 := "pro-25"
	pro30 := "pro-30"

	tests := []struct {
		name      string
		input     string
		wantModel *string
	}{
		{"사용량 flash(미지정)", "/스자 사용량 오늘 flash", nil},
		{"사용량 2.5 flash", "/스자 사용량 오늘 2.5 flash", &flash25},
		{"사용량 2.5flash", "/스자 사용량 오늘 2.5flash", &flash25},
		{"사용량 3.0 flash", "/스자 사용량 오늘 3.0 flash", &flash30},
		{"사용량 3.0flash", "/스자 사용량 오늘 3.0flash", &flash30},
		{"사용량 2.5pro", "/스자 사용량 주간 2.5pro", &pro25},
		{"사용량 3.0pro", "/스자 사용량 월간 3.0pro", &pro30},
		{"사용량 pro", "/스자 사용량 오늘 pro", &pro30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != CommandAdminUsage {
				t.Errorf("expected CommandAdminUsage, got %v", cmd.Kind)
			}
			if tt.wantModel == nil && cmd.ModelOverride != nil {
				t.Errorf("expected nil model, got %v", *cmd.ModelOverride)
			}
			if tt.wantModel != nil {
				if cmd.ModelOverride == nil {
					t.Errorf("expected model %v, got nil", *tt.wantModel)
				} else if *cmd.ModelOverride != *tt.wantModel {
					t.Errorf("expected model %v, got %v", *tt.wantModel, *cmd.ModelOverride)
				}
			}
		})
	}
}

func TestCommandParser_ParseRoomStats(t *testing.T) {
	parser := NewCommandParser("/스자")

	tests := []struct {
		name       string
		input      string
		wantKind   CommandKind
		wantPeriod qmodel.StatsPeriod
	}{
		{"방 전적 기본", "/스자 전적 룸", CommandRoomStats, qmodel.StatsPeriodAll},
		{"방 전적 일간", "/스자 전적 룸 일간", CommandRoomStats, qmodel.StatsPeriodDaily},
		{"방 전적 주간", "/스자 전적 룸 주간", CommandRoomStats, qmodel.StatsPeriodWeekly},
		{"방 전적 월간", "/스자 전적 룸 월간", CommandRoomStats, qmodel.StatsPeriodMonthly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Kind != tt.wantKind {
				t.Errorf("expected %v, got %v", tt.wantKind, cmd.Kind)
			}
			if cmd.RoomPeriod != tt.wantPeriod {
				t.Errorf("expected period %v, got %v", tt.wantPeriod, cmd.RoomPeriod)
			}
		})
	}
}
