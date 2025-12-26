package repository

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type categoryStat struct {
	GamesCompleted    int     `json:"gamesCompleted"`
	Surrenders        int     `json:"surrenders"`
	QuestionsAsked    int     `json:"questionsAsked"`
	HintsUsed         int     `json:"hintsUsed"`
	BestQuestionCount *int    `json:"bestQuestionCount,omitempty"`
	BestTarget        *string `json:"bestTarget,omitempty"`
}

func updateCategoryStatsJSON(tx *gorm.DB, userStatsID string, p GameCompletionParams) error {
	if tx == nil {
		return fmt.Errorf("db is nil")
	}

	categoryKey := strings.ToUpper(strings.TrimSpace(p.Category))
	if categoryKey == "" {
		return nil
	}

	dialector := tx.Dialector
	if dialector != nil && dialector.Name() == "postgres" {
		return updateCategoryStatsJSONPostgres(tx, userStatsID, categoryKey, p)
	}

	// Non-PostgreSQL 폴백: 기존 SELECT + UPDATE 방식
	return updateCategoryStatsJSONGeneric(tx, userStatsID, categoryKey, p)
}

// updateCategoryStatsJSONPostgres PostgreSQL JSONB 네이티브 함수로 단일 UPDATE 수행.
// 통계 누적과 베스트 기록 갱신을 하나의 UPDATE로 처리.
func updateCategoryStatsJSONPostgres(tx *gorm.DB, userStatsID, categoryKey string, p GameCompletionParams) error {
	surrenderInc := 0
	if p.Result == GameResultSurrender {
		surrenderInc = 1
	}

	shouldUpdateBest := p.Result == GameResultCorrect && p.Target != nil
	bestTarget := ""
	if shouldUpdateBest {
		target := strings.TrimSpace(*p.Target)
		if target == "" {
			target = *p.Target
		}
		bestTarget = target
	}

	// PostgreSQL JSONB 집계 쿼리
	// COALESCE로 기존 값이 없으면 0에서 시작
	// jsonb_set으로 특정 카테고리 키만 업데이트
	query := `
		UPDATE user_stats
		SET category_stats_json = jsonb_set(
			COALESCE(category_stats_json, '{}'::jsonb),
			$1::text[],
			jsonb_build_object(
				'gamesCompleted', COALESCE((category_stats_json->$2->>'gamesCompleted')::int, 0) + 1,
				'surrenders', COALESCE((category_stats_json->$2->>'surrenders')::int, 0) + $3,
				'questionsAsked', COALESCE((category_stats_json->$2->>'questionsAsked')::int, 0) + $4,
				'hintsUsed', COALESCE((category_stats_json->$2->>'hintsUsed')::int, 0) + $5,
				'bestQuestionCount', CASE
					WHEN $6 AND (
						category_stats_json->$2->>'bestQuestionCount' IS NULL
						OR (category_stats_json->$2->>'bestQuestionCount')::int > $7
					) THEN $7
					ELSE (category_stats_json->$2->>'bestQuestionCount')::int
				END,
				'bestTarget', CASE
					WHEN $6 AND (
						category_stats_json->$2->>'bestQuestionCount' IS NULL
						OR (category_stats_json->$2->>'bestQuestionCount')::int > $7
					) THEN $8::text
					ELSE category_stats_json->$2->>'bestTarget'
				END
			),
			true
		),
		updated_at = NOW(),
		version = version + 1
		WHERE id = $9
	`

	pathArray := []string{categoryKey}
	result := tx.Exec(
		query,
		pathArray,
		categoryKey,
		surrenderInc,
		p.QuestionCount,
		p.HintCount,
		shouldUpdateBest,
		p.TotalGameQuestionCount,
		bestTarget,
		userStatsID,
	)
	if result.Error != nil {
		return fmt.Errorf("update category stats postgres failed: %w", result.Error)
	}

	return nil
}

// updateCategoryStatsJSONGeneric Non-PostgreSQL용 폴백 (SQLite 등).
func updateCategoryStatsJSONGeneric(tx *gorm.DB, userStatsID, categoryKey string, p GameCompletionParams) error {
	var stats UserStats
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("category_stats_json").
		First(&stats, "id = ?", userStatsID).Error; err != nil {
		return fmt.Errorf("query category stats failed: %w", err)
	}

	parsed := make(map[string]categoryStat)
	if stats.CategoryStatsJSON != nil && strings.TrimSpace(*stats.CategoryStatsJSON) != "" {
		if err := json.Unmarshal([]byte(*stats.CategoryStatsJSON), &parsed); err != nil {
			return fmt.Errorf("unmarshal category stats failed: %w", err)
		}
	}

	stat := parsed[categoryKey]
	stat.GamesCompleted++
	if p.Result == GameResultSurrender {
		stat.Surrenders++
	}
	stat.QuestionsAsked += p.QuestionCount
	stat.HintsUsed += p.HintCount

	if p.Result == GameResultCorrect && p.Target != nil {
		candidate := p.TotalGameQuestionCount
		if stat.BestQuestionCount == nil || candidate < *stat.BestQuestionCount {
			bestCount := candidate
			target := strings.TrimSpace(*p.Target)
			if target == "" {
				target = *p.Target
			}
			stat.BestQuestionCount = &bestCount
			stat.BestTarget = &target
		}
	}

	parsed[categoryKey] = stat

	raw, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("marshal category stats failed: %w", err)
	}

	if err := tx.Model(&UserStats{}).
		Where("id = ?", userStatsID).
		Update("category_stats_json", string(raw)).Error; err != nil {
		return fmt.Errorf("update category stats failed: %w", err)
	}

	return nil
}
