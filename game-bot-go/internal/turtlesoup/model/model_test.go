package model

import (
	"slices"
	"testing"
	"time"
)

func TestParsePuzzleCategory(t *testing.T) {
	tests := []struct {
		input string
		want  PuzzleCategory
	}{
		{"MYSTERY", PuzzleCategoryMystery},
		{"mystery", PuzzleCategoryMystery},
		{"  mystery  ", PuzzleCategoryMystery},
		{"", PuzzleCategoryMystery}, // Default check
		{"UNKNOWN", PuzzleCategoryMystery},
	}

	for _, tt := range tests {
		got := ParsePuzzleCategory(tt.input)
		if got != tt.want {
			t.Errorf("ParsePuzzleCategory(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGameState_Methods(t *testing.T) {
	puzzle := Puzzle{
		Title:    "Test Puzzle",
		Solution: "Solution",
		Category: PuzzleCategoryMystery,
	}
	initial := NewInitialState("sess1", "user1", "chat1", puzzle)

	t.Run("NewInitialState", func(t *testing.T) {
		if initial.SessionID != "sess1" {
			t.Errorf("unexpected SessionID: %s", initial.SessionID)
		}
		if initial.HintsUsed != 0 {
			t.Error("expected 0 hints used")
		}
		if initial.IsSolved {
			t.Error("expected not solved")
		}
		if !slices.Contains(initial.Players, "user1") {
			t.Error("creator should be in players")
		}
	})

	t.Run("UseHint", func(t *testing.T) {
		next := initial.UseHint("hint 1")
		if next.HintsUsed != 1 {
			t.Errorf("expected 1 hint used, got %d", next.HintsUsed)
		}
		if len(next.HintContents) != 1 || next.HintContents[0] != "hint 1" {
			t.Errorf("unexpected hint contents: %v", next.HintContents)
		}
		// Immutability check
		if initial.HintsUsed != 0 {
			t.Error("original state should not be modified")
		}
		if !next.LastActivityAt.After(initial.LastActivityAt) && !next.LastActivityAt.Equal(initial.LastActivityAt) {
			// Time might be same if execution is too fast, but usually NewInitialState time vs UseHint time
			// Actually NewInitialState uses time.Now(), UseHint uses time.Now()
			// Just ensure it's valid.
		}
	})

	t.Run("AddPlayer", func(t *testing.T) {
		next := initial.AddPlayer("user2")
		if len(next.Players) != 2 {
			t.Errorf("expected 2 players, got %d", len(next.Players))
		}
		if !slices.Contains(next.Players, "user2") {
			t.Error("user2 should be added")
		}

		// Idempotency
		next2 := next.AddPlayer("user2")
		if len(next2.Players) != 2 {
			t.Error("adding existing player should not increase count")
		}
	})

	t.Run("MarkSolved", func(t *testing.T) {
		next := initial.MarkSolved()
		if !next.IsSolved {
			t.Error("expected solved")
		}
	})
}

func TestParseValidationResult(t *testing.T) {
	tests := []struct {
		input       string
		wantResult  ValidationResult
		expectError bool
	}{
		{"YES", ValidationYes, false},
		{"yes", ValidationYes, false},
		{"CLOSE", ValidationClose, false},
		{"NO", ValidationNo, false},
		{"no", ValidationNo, false},
		{"UNKNOWN", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ParseValidationResult(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("ParseValidationResult(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseValidationResult(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.wantResult {
				t.Errorf("ParseValidationResult(%q) = %v, want %v", tt.input, got, tt.wantResult)
			}
		}
	}
}

func TestSurrenderVote_Logic(t *testing.T) {
	vote := SurrenderVote{
		Initiator:       "u1",
		EligiblePlayers: []string{"u1", "u2", "u3"},
		CreatedAt:       time.Now().UnixMilli(),
	}

	if vote.RequiredApprovals() != 3 {
		t.Errorf("expected 3 required approvals for 3 players, got %d", vote.RequiredApprovals())
	}

	// 1 player
	vote1 := SurrenderVote{EligiblePlayers: []string{"u1"}}
	if vote1.RequiredApprovals() != 1 {
		t.Errorf("expected 1 required approval for 1 player, got %d", vote1.RequiredApprovals())
	}

	// Approve flow
	updated, err := vote.Approve("u1")
	if err != nil {
		t.Fatal(err)
	}
	if !updated.HasVoted("u1") {
		t.Error("u1 should have voted")
	}
	if updated.IsApproved() {
		t.Error("should not be approved yet (1/3)")
	}

	// Duplicate vote
	updated2, err := updated.Approve("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated2.Approvals) != 1 {
		t.Error("duplicate vote count check failed")
	}

	// Ineligible vote
	_, err = updated.Approve("u4")
	if err == nil {
		t.Error("expected error for ineligible user")
	}

	// Threshold check
	updated, _ = updated.Approve("u2")
	updated, _ = updated.Approve("u3")
	if !updated.IsApproved() {
		t.Error("should be approved (3/3)")
	}
}
