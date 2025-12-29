package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

// GameSessionParams: 게임 세션 기록 파라미터 구조체
type GameSessionParams struct {
	SessionID        string
	ChatID           string
	Category         string
	Result           GameResult
	ParticipantCount int
	QuestionCount    int
	HintCount        int
	CompletedAt      time.Time
	Now              time.Time
}

// RecordGameSession: 게임 세션 메타데이터를 기록합니다.
func (r *Repository) RecordGameSession(ctx context.Context, p GameSessionParams) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	p.SessionID = strings.TrimSpace(p.SessionID)
	p.ChatID = strings.TrimSpace(p.ChatID)
	p.Category = strings.TrimSpace(p.Category)
	p.Result = GameResult(strings.TrimSpace(string(p.Result)))
	if p.SessionID == "" {
		p.SessionID = GenerateFallbackSessionID(p.ChatID)
	}

	if p.ChatID == "" || p.Category == "" || p.Result == "" {
		return nil
	}

	entity := GameSession{
		SessionID:        p.SessionID,
		ChatID:           p.ChatID,
		Category:         p.Category,
		Result:           string(p.Result),
		ParticipantCount: p.ParticipantCount,
		QuestionCount:    p.QuestionCount,
		HintCount:        p.HintCount,
		CompletedAt:      p.CompletedAt,
		CreatedAt:        p.Now,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoNothing: true,
	}).Create(&entity).Error; err != nil {
		return fmt.Errorf("record game session failed: %w", err)
	}

	return nil
}

// GameLogParams: 게임 로그 파라미터 구조체
type GameLogParams struct {
	ChatID          string
	UserID          string
	Sender          string
	Category        string
	QuestionCount   int
	HintCount       int
	WrongGuessCount int
	Result          GameResult
	Target          *string
	CompletedAt     time.Time
	Now             time.Time
}

// RecordGameLog: 플레이어별 게임 활동 로그를 기록합니다.
func (r *Repository) RecordGameLog(ctx context.Context, p GameLogParams) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	p.ChatID = strings.TrimSpace(p.ChatID)
	p.UserID = strings.TrimSpace(p.UserID)
	p.Sender = strings.TrimSpace(p.Sender)
	p.Category = strings.TrimSpace(p.Category)
	p.Result = GameResult(strings.TrimSpace(string(p.Result)))

	if p.ChatID == "" || p.UserID == "" || p.Category == "" || p.Result == "" {
		return nil
	}

	entity := GameLog{
		ChatID:          p.ChatID,
		UserID:          p.UserID,
		Sender:          p.Sender,
		Category:        p.Category,
		QuestionCount:   p.QuestionCount,
		HintCount:       p.HintCount,
		WrongGuessCount: p.WrongGuessCount,
		Result:          string(p.Result),
		Target:          p.Target,
		CompletedAt:     p.CompletedAt,
		CreatedAt:       p.Now,
	}

	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return fmt.Errorf("record game log failed: %w", err)
	}

	return nil
}
