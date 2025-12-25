package service

import (
	"strings"

	json "github.com/goccy/go-json"

	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// CategoryStat 카테고리별 통계 (JSON에서 파싱).
type CategoryStat struct {
	GamesCompleted    int     `json:"gamesCompleted"`
	Surrenders        int     `json:"surrenders"`
	QuestionsAsked    int     `json:"questionsAsked"`
	HintsUsed         int     `json:"hintsUsed"`
	BestQuestionCount *int    `json:"bestQuestionCount,omitempty"`
	BestTarget        *string `json:"bestTarget,omitempty"`
}

// parseCategoryStats JSON 문자열을 카테고리 통계 맵으로 파싱.
func (s *StatsService) parseCategoryStats(jsonStr string) map[string]CategoryStat {
	result := make(map[string]CategoryStat)
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}

// normalizeCategoryStats 카테고리 키를 정규화.
func normalizeCategoryStats(categoryStats map[string]CategoryStat) map[string]CategoryStat {
	if len(categoryStats) == 0 {
		return categoryStats
	}

	normalized := make(map[string]CategoryStat, len(categoryStats))
	for key, stat := range categoryStats {
		normalizedKey := normalizeCategoryKey(key)
		if normalizedKey == "" {
			normalizedKey = strings.TrimSpace(key)
		}
		if normalizedKey == "" {
			continue
		}
		normalized[normalizedKey] = mergeCategoryStat(normalized[normalizedKey], stat)
	}

	return normalized
}

// normalizeCategoryKey 카테고리 키 정규화.
func normalizeCategoryKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return normalizeCategoryInput(key)
}

// mergeCategoryStat 두 카테고리 통계 병합.
func mergeCategoryStat(base CategoryStat, add CategoryStat) CategoryStat {
	base.GamesCompleted += add.GamesCompleted
	base.Surrenders += add.Surrenders
	base.QuestionsAsked += add.QuestionsAsked
	base.HintsUsed += add.HintsUsed
	base.BestQuestionCount, base.BestTarget = chooseBestScore(
		base.BestQuestionCount,
		base.BestTarget,
		add.BestQuestionCount,
		add.BestTarget,
	)
	return base
}

// chooseBestScore 더 나은 베스트 스코어 선택.
func chooseBestScore(
	baseCount *int,
	baseTarget *string,
	addCount *int,
	addTarget *string,
) (*int, *string) {
	if baseCount == nil {
		return addCount, addTarget
	}
	if addCount == nil {
		return baseCount, baseTarget
	}
	if *addCount < *baseCount {
		return addCount, addTarget
	}
	if *addCount > *baseCount {
		return baseCount, baseTarget
	}
	if baseTarget == nil && addTarget != nil {
		return baseCount, addTarget
	}
	return baseCount, baseTarget
}

// ensureAllCategoryStats 모든 카테고리가 포함되도록 보장.
func ensureAllCategoryStats(categoryStats map[string]CategoryStat) map[string]CategoryStat {
	if categoryStats == nil {
		categoryStats = make(map[string]CategoryStat, len(qconfig.AllCategories))
	}
	for _, category := range qconfig.AllCategories {
		if _, ok := categoryStats[category]; !ok {
			categoryStats[category] = CategoryStat{}
		}
	}
	return categoryStats
}
