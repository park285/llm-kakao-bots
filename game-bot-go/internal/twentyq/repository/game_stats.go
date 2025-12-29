package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GameResult: 게임 결과 상수
type GameResult string

// GameResultCorrect 는 게임 결과 상수 목록이다.
const (
	GameResultCorrect   GameResult = "CORRECT"
	GameResultSurrender GameResult = "SURRENDER"
)

// GameCompletionParams: 게임 완료 기록 파라미터 구조체
type GameCompletionParams struct {
	ChatID                 string
	UserID                 string
	Category               string
	Result                 GameResult
	QuestionCount          int
	HintCount              int
	WrongGuessCount        int
	Target                 *string
	TotalGameQuestionCount int
	CompletedAt            time.Time
	Now                    time.Time
}

// RecordGameStart: 게임 시작 정보를 기록합니다 (사용자 통계 초기화 등).
func (r *Repository) RecordGameStart(ctx context.Context, chatID string, userID string, now time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	chatID = strings.TrimSpace(chatID)
	userID = strings.TrimSpace(userID)
	if chatID == "" || userID == "" {
		return nil
	}

	id := CompositeUserStatsID(chatID, userID)

	entity := UserStats{
		ID:                id,
		ChatID:            chatID,
		UserID:            userID,
		TotalGamesStarted: 1,
		CreatedAt:         now,
		UpdatedAt:         now,
		Version:           0,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total_games_started": gorm.Expr("\"user_stats\".\"total_games_started\" + 1"),
			"updated_at":          now,
			"version":             gorm.Expr("\"user_stats\".\"version\" + 1"),
		}),
	}).Create(&entity).Error; err != nil {
		return fmt.Errorf("record game start failed: %w", err)
	}

	return nil
}

// RecordGameCompletion: 게임 완료 정보를 기록하고 관련 통계를 업데이트합니다.
func (r *Repository) RecordGameCompletion(ctx context.Context, p GameCompletionParams) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	p.ChatID = strings.TrimSpace(p.ChatID)
	p.UserID = strings.TrimSpace(p.UserID)
	p.Category = strings.TrimSpace(p.Category)
	if p.ChatID == "" || p.UserID == "" || p.Category == "" {
		return nil
	}

	surrenderInc := 0
	if p.Result == GameResultSurrender {
		surrenderInc = 1
	}

	id := CompositeUserStatsID(p.ChatID, p.UserID)
	entity := buildUserStatsEntity(p, id, surrenderInc)

	tx := r.db.WithContext(ctx).Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}

	if err := upsertUserStatsCompletion(tx, entity, p, surrenderInc); err != nil {
		tx.Rollback()
		return fmt.Errorf("record game completion failed: %w", err)
	}

	if err := updateCategoryStatsJSON(tx, id, p); err != nil {
		tx.Rollback()
		return fmt.Errorf("update category stats failed: %w", err)
	}

	if err := updateOverallBestScore(tx, id, p); err != nil {
		tx.Rollback()
		return fmt.Errorf("update best score failed: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("record game completion commit failed: %w", err)
	}

	return nil
}

func buildBestScoreFields(p GameCompletionParams) (
	bestQuestionCnt *int,
	bestWrongGuess *int,
	bestTarget *string,
	bestCategory *string,
	bestAchievedAt *time.Time,
) {
	if p.Result != GameResultCorrect || p.Target == nil {
		return nil, nil, nil, nil, nil
	}

	qc := p.TotalGameQuestionCount
	wg := p.WrongGuessCount
	return &qc, &wg, p.Target, &p.Category, &p.CompletedAt
}

func buildUserStatsEntity(p GameCompletionParams, id string, surrenderInc int) UserStats {
	bestQuestionCnt, bestWrongGuess, bestTarget, bestCategory, bestAchievedAt := buildBestScoreFields(p)

	return UserStats{
		ID:                   id,
		ChatID:               p.ChatID,
		UserID:               p.UserID,
		TotalGamesStarted:    1,
		TotalGamesCompleted:  1,
		TotalSurrenders:      surrenderInc,
		TotalQuestionsAsked:  p.QuestionCount,
		TotalHintsUsed:       p.HintCount,
		TotalWrongGuesses:    p.WrongGuessCount,
		BestScoreQuestionCnt: bestQuestionCnt,
		BestScoreWrongGuess:  bestWrongGuess,
		BestScoreTarget:      bestTarget,
		BestScoreCategory:    bestCategory,
		BestScoreAchievedAt:  bestAchievedAt,
		CreatedAt:            p.Now,
		UpdatedAt:            p.Now,
		Version:              0,
	}
}

func upsertUserStatsCompletion(
	tx *gorm.DB,
	entity UserStats,
	p GameCompletionParams,
	surrenderInc int,
) error {
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total_games_completed": gorm.Expr("\"user_stats\".\"total_games_completed\" + 1"),
			"total_surrenders":      gorm.Expr("\"user_stats\".\"total_surrenders\" + ?", surrenderInc),
			"total_questions_asked": gorm.Expr("\"user_stats\".\"total_questions_asked\" + ?", p.QuestionCount),
			"total_hints_used":      gorm.Expr("\"user_stats\".\"total_hints_used\" + ?", p.HintCount),
			"total_wrong_guesses":   gorm.Expr("\"user_stats\".\"total_wrong_guesses\" + ?", p.WrongGuessCount),
			"updated_at":            p.Now,
			"version":               gorm.Expr("\"user_stats\".\"version\" + 1"),
		}),
	}).Create(&entity).Error; err != nil {
		return err
	}

	return nil
}

func updateOverallBestScore(tx *gorm.DB, userStatsID string, p GameCompletionParams) error {
	if p.Result != GameResultCorrect || p.Target == nil {
		return nil
	}

	candidateQuestions := p.TotalGameQuestionCount
	maxQuestions := int(math.MaxInt32)

	updates := map[string]any{
		"best_score_question_count":    p.TotalGameQuestionCount,
		"best_score_wrong_guess_count": p.WrongGuessCount,
		"best_score_target":            p.Target,
		"best_score_category":          p.Category,
		"best_score_achieved_at":       p.CompletedAt,
	}

	if err := tx.
		Model(&UserStats{}).
		Where(
			"id = ? AND (? < COALESCE(best_score_question_count, ?))",
			userStatsID,
			candidateQuestions,
			maxQuestions,
		).
		Updates(updates).Error; err != nil {
		return err
	}

	return nil
}
