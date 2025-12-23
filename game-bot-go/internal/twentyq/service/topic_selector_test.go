package service

import (
	"log/slog"
	"os"
	"testing"
)

func TestTopicSelector_Select(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewTopicSelector(logger)

	// 1. Specific Category
	t.Run("SpecificCategory", func(t *testing.T) {
		topic, err := selector.SelectTopic("organism", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if topic.Category != "organism" {
			t.Errorf("expected category organism, got %s", topic.Category)
		}
		if topic.Name == "" {
			t.Error("expected valid topic name")
		}
	})

	// 2. Random Category
	t.Run("RandomCategory", func(t *testing.T) {
		topic, err := selector.SelectTopic("", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if topic.Name == "" {
			t.Error("expected valid topic")
		}
	})

	// 3. Food category
	t.Run("FoodCategory", func(t *testing.T) {
		topic, err := selector.SelectTopic("food", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if topic.Category != "food" {
			t.Errorf("expected category food, got %s", topic.Category)
		}
		t.Logf("Selected food topic: %s", topic.Name)
	})

	// 4. 사자성어/속담 카테고리
	t.Run("IdiomProverbCategory", func(t *testing.T) {
		topic, err := selector.SelectTopic("idiom_proverb", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if topic.Category != "idiom_proverb" {
			t.Errorf("expected category idiom_proverb, got %s", topic.Category)
		}
		if topic.Name == "" {
			t.Error("expected valid topic name")
		}
	})
}

func TestTopicSelector_Fallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewTopicSelector(logger)

	// 존재하지 않는 카테고리로 이런 이름의 주제 금지
	banned := []string{"고양이", "참나무", "강아지", "사자", "호랑이"}

	topic, err := selector.SelectTopic("organism", banned, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 정상 선택 됐는지 확인
	if topic.Name == "" {
		t.Error("expected valid topic")
	}

	// 금지된 주제가 선택되지 않았는지 확인
	for _, b := range banned {
		if topic.Name == b {
			t.Errorf("banned topic selected: %s", b)
		}
	}
}

func TestTopicSelector_Categories(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewTopicSelector(logger)

	categories := selector.Categories()
	if len(categories) != 7 {
		t.Errorf("expected 7 categories, got %d: %v", len(categories), categories)
	}

	expectedCategories := map[string]bool{
		"object":        true,
		"food":          true,
		"place":         true,
		"concept":       true,
		"movie":         true,
		"organism":      true,
		"idiom_proverb": true,
	}

	for _, cat := range categories {
		if !expectedCategories[cat] {
			t.Errorf("unexpected category: %s", cat)
		}
	}
}

func TestTopicSelector_ExcludeCategoryOnRandom(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewTopicSelector(logger)
	selector.topics = map[string][]TopicEntry{
		"movie": {
			{Name: "Movie1", Category: "movie"},
		},
		"object": {
			{Name: "Object1", Category: "object"},
		},
	}

	topic, err := selector.SelectTopic("", nil, []string{"movie"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic.Category != "object" {
		t.Errorf("expected category object, got %s", topic.Category)
	}
}

func TestTopicSelector_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewTopicSelector(logger)
	// Force empty
	selector.topics = make(map[string][]TopicEntry)

	_, err := selector.SelectTopic("", nil, nil)
	if err == nil {
		t.Error("expected error for empty topics")
	}

	// Test empty random category
	if cat := selector.selectRandomCategory(); cat != "" {
		t.Errorf("expected empty string, got %s", cat)
	}
}
