package service

import (
	"testing"

	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

func TestMatchExplicitAnswer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		found    bool
	}{
		{"정답 사과", "사과", true},
		{"정답   사과  ", "사과", true},
		{"정답 사과인가요", "사과", true},
		{"정답 사과입니까", "사과", true},
		{"정답사과", "", false}, // pattern requires space after 정답
		{"사과", "", false},
		{"정답 ", "", false},
	}

	for _, tt := range tests {
		got, found := matchExplicitAnswer(tt.input)
		if found != tt.found {
			t.Errorf("matchExplicitAnswer(%q) found=%v, want %v", tt.input, found, tt.found)
		}
		if got != tt.expected {
			t.Errorf("matchExplicitAnswer(%q) got=%q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeForEquality(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  사과  ", "사과"},
		{"사과입니까?", "사과"},
		{"사과인가요", "사과"},
		{"Apple", "apple"},
		{"a p p l e", "apple"},
		{"사과!", "사과"},
	}

	for _, tt := range tests {
		got := normalizeForEquality(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeForEquality(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDisplayName(t *testing.T) {
	sender := "Sender"
	anon := "Anonymous"

	tests := []struct {
		chatID string
		userID string
		sender *string
		anon   string
		want   string
	}{
		{"c", "u", &sender, anon, "Sender"},
		{"c", "u", nil, anon, "u"},
		{"c", "", nil, anon, anon},
		{"", "u", nil, anon, anon},
		{"c", "c", nil, anon, anon}, // chatID == userID
	}

	for _, tt := range tests {
		got := domainmodels.DisplayName(tt.chatID, tt.userID, tt.sender, tt.anon)
		if got != tt.want {
			t.Errorf("displayName(%q, %q, %v, %q) = %q, want %q", tt.chatID, tt.userID, tt.sender, tt.anon, got, tt.want)
		}
	}
}

func TestCategoryToKorean(t *testing.T) {
	tests := []struct {
		input string
		want  string // empty if nil
	}{
		{"organism", "생물"},
		{"FOOD", "음식"},
		{"object", "사물"},
		{"place", "장소"},
		{"concept", "개념"},
		{"movie", "영화"},
		{"idiom_proverb", "사자성어/속담"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		got := categoryToKorean(tt.input)
		if tt.want == "" {
			if got != nil {
				t.Errorf("categoryToKorean(%q) = %v, want nil", tt.input, *got)
			}
		} else {
			if got == nil || *got != tt.want {
				t.Errorf("categoryToKorean(%q) = %v, want %q", tt.input, got, tt.want)
			}
		}
	}
}

func TestNormalizeCategoryInput(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"생물", "organism"},
		{"음식", "food"},
		{"사물", "object"},
		{"장소", "place"},
		{"개념", "concept"},
		{"영화", "movie"},
		{"사자성어", "idiom_proverb"},
		{"속담", "idiom_proverb"},
		{"사자성어/속담", "idiom_proverb"},
		{"idiom_proverb", "idiom_proverb"},
		{"Organism", "organism"},
		{"FOOD", "food"},
		{" unknown ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeCategoryInput(tt.input)
		if got != tt.want {
			t.Errorf("normalizeCategoryInput(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSelectCategory(t *testing.T) {
	// 1. Empty -> false
	key, invalid := selectCategory(nil)
	if key != "" || invalid {
		t.Errorf("expected empty/valid for nil inputs")
	}

	// 2. Single Valid
	key, invalid = selectCategory([]string{"생물"})
	if key != "organism" || invalid {
		t.Errorf("expected organism, got %q invalid=%v", key, invalid)
	}

	// 3. Invalid Only -> invalid=true
	key, invalid = selectCategory([]string{"invalid"})
	if key != "" || !invalid {
		t.Errorf("expected invalid=true for invalid input")
	}

	// 4. Multiple Valid (Random pick)
	key, invalid = selectCategory([]string{"생물", "음식"})
	if invalid {
		t.Error("expected valid")
	}
	if key != "organism" && key != "food" {
		t.Errorf("expected organism or food, got %q", key)
	}
}

func TestRandInt(t *testing.T) {
	if v := randInt(0); v != 0 {
		t.Errorf("randInt(0) = %d, want 0", v)
	}
	if v := randInt(-5); v != 0 {
		t.Errorf("randInt(-5) = %d, want 0", v)
	}
	if v := randInt(10); v < 0 || v >= 10 {
		t.Errorf("randInt(10) = %d out of range", v)
	}
}
