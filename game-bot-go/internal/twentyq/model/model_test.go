package model

import (
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
)

func TestPendingMessage_DisplayName(t *testing.T) {
	tests := []struct {
		name      string
		msg       PendingMessage
		chatID    string
		anonymous string
		want      string
	}{
		{
			name: "With Sender",
			msg: PendingMessage{
				UserID: "user1",
				Sender: ptr.String("Nickname"),
			},
			chatID:    "chat1",
			anonymous: "Anon",
			want:      "Nickname",
		},
		{
			name: "No Sender, Different UserID",
			msg: PendingMessage{
				UserID: "user1",
			},
			chatID:    "chat1",
			anonymous: "Anon",
			want:      "user1",
		},
		{
			name: "No Sender, Same UserID (Anonymous)",
			msg: PendingMessage{
				UserID: "chat1", // Assuming user is the channel itself
			},
			chatID:    "chat1",
			anonymous: "Anon",
			want:      "Anon",
		},
		{
			name: "Empty UserID (Anonymous)",
			msg: PendingMessage{
				UserID: "",
			},
			chatID:    "chat1",
			anonymous: "Anon",
			want:      "Anon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.DisplayName(tt.chatID, tt.anonymous)
			if got != tt.want {
				t.Errorf("DisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSurrenderVote_RequiredApprovals(t *testing.T) {
	tests := []struct {
		players int
		want    int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 3},
		{10, 3},
	}

	for _, tt := range tests {
		v := SurrenderVote{EligiblePlayers: make([]string, tt.players)}
		if got := v.RequiredApprovals(); got != tt.want {
			t.Errorf("players=%d RequiredApprovals() = %d, want %d", tt.players, got, tt.want)
		}
	}
}

func TestSurrenderVote_Approve(t *testing.T) {
	v := SurrenderVote{
		EligiblePlayers: []string{"u1", "u2"},
	}

	// 1. Success
	next, err := v.Approve("u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !next.HasVoted("u1") {
		t.Error("u1 should have voted")
	}

	// 2. Duplicate
	next2, err := next.Approve("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(next2.Approvals) != 1 {
		t.Error("approvals count should not increase on duplicate")
	}

	// 3. Not Eligible
	_, err = next.Approve("u3")
	if err == nil {
		t.Error("expected error for u3")
	}
}

func TestFiveScaleKo(t *testing.T) {
	tests := []struct {
		input string
		want  FiveScaleKo
		ok    bool
	}{
		{"예", FiveScaleAlwaysYes, true},
		{"아마도 예", FiveScaleMostlyYes, true},
		{"아마도 아니오", FiveScaleMostlyNo, true},
		{"아니오", FiveScaleAlwaysNo, true},
		{"이해할 수 없는 질문입니다", FiveScaleInvalid, true},
		{" 예 ", FiveScaleAlwaysYes, true}, // trim
		{"예.", FiveScaleAlwaysYes, true},  // punctuation
		{"예!", FiveScaleAlwaysYes, true},  // punctuation
		{"몰라", FiveScaleInvalid, false},   // unknown
		{"", FiveScaleInvalid, false},     // empty
	}

	for _, tt := range tests {
		got, ok := ParseFiveScaleKo(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseFiveScaleKo(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && *got != tt.want {
			t.Errorf("ParseFiveScaleKo(%q) = %v, want %v", tt.input, *got, tt.want)
		}

		if ok {
			// Reverse check
			token := FiveScaleToken(*got)
			if token == "" {
				t.Errorf("FiveScaleToken(%v) returned empty", *got)
			}
		}
	}
}

func TestChainCondition_ShouldContinue(t *testing.T) {
	tests := []struct {
		condition ChainCondition
		scale     FiveScaleKo
		want      bool
	}{
		{ChainConditionAlways, FiveScaleAlwaysYes, true},
		{ChainConditionAlways, FiveScaleAlwaysNo, true},
		{ChainConditionIfTrue, FiveScaleAlwaysYes, true},
		{ChainConditionIfTrue, FiveScaleMostlyYes, true},
		{ChainConditionIfTrue, FiveScaleMostlyNo, false},
		{ChainConditionIfTrue, FiveScaleAlwaysNo, false},
		{ChainConditionIfTrue, FiveScaleInvalid, false},
	}

	for _, tt := range tests {
		if got := tt.condition.ShouldContinue(tt.scale); got != tt.want {
			t.Errorf("Condition(%v).ShouldContinue(%v) = %v, want %v", tt.condition, tt.scale, got, tt.want)
		}
	}
}
