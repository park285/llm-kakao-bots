package mq

import (
	"strings"
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	tsassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/assets"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

func TestGameCommandHandler_composeStartReply_ResumingOrdersMessages(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML(tsassets.GameMessagesYAML)
	if err != nil {
		t.Fatalf("load messages failed: %v", err)
	}

	handler := &GameCommandHandler{msgProvider: msgProvider}
	state := tsmodel.GameState{
		Puzzle: &tsmodel.Puzzle{
			Scenario:   "SCENARIO",
			Difficulty: 3,
		},
		QuestionCount: 7,
		HintsUsed:     2,
	}

	selection := difficultySelection{Warning: "WARN"}
	out := handler.composeStartReply(selection, state, true)

	if strings.Contains(out, selection.Warning) {
		t.Fatalf("expected warning to be omitted on resume, got: %q", out)
	}

	scenarioMessage := handler.buildScenarioMessage(state, true)
	statusMessage := handler.buildInstructionMessage(state, true)

	idxScenario := strings.Index(out, scenarioMessage)
	idxStatus := strings.Index(out, statusMessage)
	if idxScenario < 0 || idxStatus < 0 {
		t.Fatalf("expected both scenario and status messages to be present, got: %q", out)
	}
	if idxScenario > idxStatus {
		t.Fatalf("expected scenario before status, got: %q", out)
	}
}

func TestGameCommandHandler_composeStartReply_NewStartOrdersMessages(t *testing.T) {
	msgProvider, err := messageprovider.NewFromYAML(tsassets.GameMessagesYAML)
	if err != nil {
		t.Fatalf("load messages failed: %v", err)
	}

	handler := &GameCommandHandler{msgProvider: msgProvider}
	state := tsmodel.GameState{
		Puzzle: &tsmodel.Puzzle{
			Scenario:   "SCENARIO",
			Difficulty: 3,
		},
	}

	selection := difficultySelection{Warning: "WARN"}
	out := handler.composeStartReply(selection, state, false)

	if !strings.HasPrefix(out, selection.Warning) {
		t.Fatalf("expected warning first, got: %q", out)
	}

	scenarioMessage := handler.buildScenarioMessage(state, false)
	instruction := handler.buildInstructionMessage(state, false)

	idxWarning := strings.Index(out, selection.Warning)
	idxScenario := strings.Index(out, scenarioMessage)
	idxInstruction := strings.Index(out, instruction)
	if idxWarning < 0 || idxScenario < 0 || idxInstruction < 0 {
		t.Fatalf("expected all parts present, got: %q", out)
	}
	if !(idxWarning < idxScenario && idxScenario < idxInstruction) {
		t.Fatalf("expected warning -> scenario -> instruction order, got: %q", out)
	}
}

