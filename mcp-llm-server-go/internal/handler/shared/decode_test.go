package shared

import (
	"testing"
)

func TestDecode(t *testing.T) {
	type Puzzle struct {
		Title      string   `json:"title"`
		Scenario   string   `json:"scenario"`
		Solution   string   `json:"solution"`
		Difficulty int      `json:"difficulty"`
		Hints      []string `json:"hints"`
	}

	tests := []struct {
		name    string
		input   map[string]any
		want    Puzzle
		wantErr bool
	}{
		{
			name: "valid map",
			input: map[string]any{
				"title":      "Test Title",
				"scenario":   "Test Scenario",
				"solution":   "Test Solution",
				"difficulty": 3,
				"hints":      []any{"hint1", "hint2"},
			},
			want: Puzzle{
				Title:      "Test Title",
				Scenario:   "Test Scenario",
				Solution:   "Test Solution",
				Difficulty: 3,
				Hints:      []string{"hint1", "hint2"},
			},
		},
		{
			name: "float difficulty",
			input: map[string]any{
				"title":      "Test",
				"scenario":   "Scenario",
				"solution":   "Solution",
				"difficulty": 4.0,
				"hints":      []any{},
			},
			want: Puzzle{
				Title:      "Test",
				Scenario:   "Scenario",
				Solution:   "Solution",
				Difficulty: 4,
				Hints:      []string{},
			},
		},
		{
			name:  "empty map",
			input: map[string]any{},
			want:  Puzzle{Hints: nil},
		},
		{
			name: "missing fields",
			input: map[string]any{
				"title": "Only Title",
			},
			want: Puzzle{Title: "Only Title", Hints: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Puzzle
			err := Decode(tt.input, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Title != tt.want.Title {
					t.Errorf("Title = %v, want %v", got.Title, tt.want.Title)
				}
				if got.Scenario != tt.want.Scenario {
					t.Errorf("Scenario = %v, want %v", got.Scenario, tt.want.Scenario)
				}
				if got.Solution != tt.want.Solution {
					t.Errorf("Solution = %v, want %v", got.Solution, tt.want.Solution)
				}
				if got.Difficulty != tt.want.Difficulty {
					t.Errorf("Difficulty = %v, want %v", got.Difficulty, tt.want.Difficulty)
				}
				if len(got.Hints) != len(tt.want.Hints) {
					t.Errorf("Hints len = %v, want %v", len(got.Hints), len(tt.want.Hints))
				}
			}
		})
	}
}

func TestDecodeStrict(t *testing.T) {
	type Simple struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name:    "valid",
			input:   map[string]any{"name": "test"},
			wantErr: false,
		},
		{
			name:    "unknown field",
			input:   map[string]any{"name": "test", "unknown": "value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Simple
			err := DecodeStrict(tt.input, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
