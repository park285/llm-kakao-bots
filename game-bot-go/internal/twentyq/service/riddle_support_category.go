package service

import (
	"math/rand/v2"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
)

const (
	categoryOrganism     = "organism"
	categoryFood         = "food"
	categoryObject       = "object"
	categoryPlace        = "place"
	categoryConcept      = "concept"
	categoryMovie        = "movie"
	categoryIdiomProverb = "idiom_proverb"
)

func categoryToKorean(category string) *string {
	category = strings.ToLower(strings.TrimSpace(category))
	switch category {
	case categoryOrganism:
		return ptr.String("생물")
	case categoryFood:
		return ptr.String("음식")
	case categoryObject:
		return ptr.String("사물")
	case categoryPlace:
		return ptr.String("장소")
	case categoryConcept:
		return ptr.String("개념")
	case categoryMovie:
		return ptr.String("영화")
	case categoryIdiomProverb:
		return ptr.String("사자성어/속담")
	default:
		return nil
	}
}

func selectCategory(inputs []string) (selectedKey string, invalidInput bool) {
	valid := make([]string, 0, len(inputs))
	for _, raw := range inputs {
		key := normalizeCategoryInput(raw)
		if key == "" {
			continue
		}
		valid = append(valid, key)
	}

	switch len(valid) {
	case 0:
		if len(inputs) > 0 {
			return "", true
		}
		return "", false
	case 1:
		key := valid[0]
		return key, false
	default:
		key := valid[randInt(len(valid))]
		return key, false
	}
}

func normalizeCategoryInput(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	switch input {
	case "생물":
		return categoryOrganism
	case "음식":
		return categoryFood
	case "사물":
		return categoryObject
	case "장소":
		return categoryPlace
	case "개념":
		return categoryConcept
	case "영화":
		return categoryMovie
	case "사자성어", "속담", "사자성어/속담":
		return categoryIdiomProverb
	default:
	}

	lower := strings.ToLower(input)
	switch lower {
	case categoryOrganism, categoryFood, categoryObject, categoryPlace, categoryConcept, categoryMovie, categoryIdiomProverb:
		return lower
	default:
		return ""
	}
}

func randInt(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.IntN(n)
}
