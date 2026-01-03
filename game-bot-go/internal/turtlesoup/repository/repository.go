package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Repository: TurtleSoup DB 리포지토리
type Repository struct {
	db *gorm.DB
}

// New: 새로운 Repository 인스턴스 생성
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// AutoMigrate: DB 테이블 스키마 자동 마이그레이션
func (r *Repository) AutoMigrate(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := r.db.WithContext(ctx).AutoMigrate(
		&GameArchive{},
		&Puzzle{},
	); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	return nil
}

// GameArchive: 게임 아카이브 (세션 종료 시 저장)
type GameArchive struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SessionID     string    `gorm:"column:session_id;not null;uniqueIndex" json:"sessionId"`
	ChatID        string    `gorm:"column:chat_id;not null;index" json:"chatId"`
	PuzzleID      *uint64   `gorm:"column:puzzle_id;index" json:"puzzleId,omitempty"`
	QuestionCount int       `gorm:"column:question_count;not null;default:0" json:"questionCount"`
	HintsUsed     int       `gorm:"column:hints_used;not null;default:0" json:"hintsUsed"`
	Result        string    `gorm:"column:result;not null;index" json:"result"` // solved, surrendered, timeout
	HistoryJSON   string    `gorm:"column:history_json;type:jsonb" json:"historyJson"`
	StartedAt     time.Time `gorm:"column:started_at;not null" json:"startedAt"`
	CompletedAt   time.Time `gorm:"column:completed_at;not null;index" json:"completedAt"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
}

func (GameArchive) TableName() string { return "turtle_game_archives" }

// Puzzle: 퍼즐 시나리오 CMS
type Puzzle struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"column:title;not null" json:"title"`
	Scenario    string    `gorm:"column:scenario;not null" json:"scenario"`
	Solution    string    `gorm:"column:solution;not null" json:"solution"`
	Category    string    `gorm:"column:category;not null;index;default:'MYSTERY'" json:"category"`
	Difficulty  int       `gorm:"column:difficulty;not null;default:3" json:"difficulty"`
	HintsJSON   string    `gorm:"column:hints_json;type:jsonb;default:'[]'" json:"hintsJson"`
	Status      string    `gorm:"column:status;not null;index;default:'draft'" json:"status"` // draft, test, published
	AuthorID    string    `gorm:"column:author_id" json:"authorId"`
	PlayCount   int       `gorm:"column:play_count;not null;default:0" json:"playCount"`
	SolveCount  int       `gorm:"column:solve_count;not null;default:0" json:"solveCount"`
	AvgQuestion float64   `gorm:"column:avg_question;not null;default:0" json:"avgQuestion"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (Puzzle) TableName() string { return "turtle_puzzles" }

// ArchiveGameParams: 게임 아카이브 파라미터
type ArchiveGameParams struct {
	SessionID     string
	ChatID        string
	PuzzleID      *uint64
	QuestionCount int
	HintsUsed     int
	Result        string
	HistoryJSON   string
	StartedAt     time.Time
	CompletedAt   time.Time
}

// ArchiveGame: 게임 결과를 PostgreSQL에 아카이브
func (r *Repository) ArchiveGame(ctx context.Context, p ArchiveGameParams) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	archive := GameArchive{
		SessionID:     p.SessionID,
		ChatID:        p.ChatID,
		PuzzleID:      p.PuzzleID,
		QuestionCount: p.QuestionCount,
		HintsUsed:     p.HintsUsed,
		Result:        p.Result,
		HistoryJSON:   p.HistoryJSON,
		StartedAt:     p.StartedAt,
		CompletedAt:   p.CompletedAt,
	}

	if err := r.db.WithContext(ctx).Create(&archive).Error; err != nil {
		return fmt.Errorf("archive game failed: %w", err)
	}
	return nil
}

// CreatePuzzle: 새 퍼즐 생성
func (r *Repository) CreatePuzzle(ctx context.Context, puzzle *Puzzle) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := r.db.WithContext(ctx).Create(puzzle).Error; err != nil {
		return fmt.Errorf("create puzzle failed: %w", err)
	}
	return nil
}

// GetPuzzle: 퍼즐 조회
func (r *Repository) GetPuzzle(ctx context.Context, id uint64) (*Puzzle, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	var puzzle Puzzle
	if err := r.db.WithContext(ctx).First(&puzzle, id).Error; err != nil {
		return nil, err
	}
	return &puzzle, nil
}

// ListPuzzles: 퍼즐 목록 조회
func (r *Repository) ListPuzzles(ctx context.Context, status string, limit, offset int) ([]Puzzle, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("db is nil")
	}

	query := r.db.WithContext(ctx).Model(&Puzzle{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count puzzles failed: %w", err)
	}

	var puzzles []Puzzle
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&puzzles).Error; err != nil {
		return nil, 0, fmt.Errorf("list puzzles failed: %w", err)
	}

	return puzzles, total, nil
}

// UpdatePuzzle: 퍼즐 업데이트
func (r *Repository) UpdatePuzzle(ctx context.Context, puzzle *Puzzle) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := r.db.WithContext(ctx).Save(puzzle).Error; err != nil {
		return fmt.Errorf("update puzzle failed: %w", err)
	}
	return nil
}

// DeletePuzzle: 퍼즐 삭제
func (r *Repository) DeletePuzzle(ctx context.Context, id uint64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := r.db.WithContext(ctx).Delete(&Puzzle{}, id).Error; err != nil {
		return fmt.Errorf("delete puzzle failed: %w", err)
	}
	return nil
}

// GetRandomPublishedPuzzle: 랜덤 공개 퍼즐 조회
func (r *Repository) GetRandomPublishedPuzzle(ctx context.Context, category string) (*Puzzle, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	query := r.db.WithContext(ctx).Model(&Puzzle{}).Where("status = ?", "published")
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var puzzle Puzzle
	if err := query.Order("random()").First(&puzzle).Error; err != nil {
		return nil, err
	}
	return &puzzle, nil
}

// PuzzleStatsResult: 퍼즐 통계 결과
type PuzzleStatsResult struct {
	TotalPuzzles     int64   `json:"totalPuzzles"`
	PublishedCount   int64   `json:"publishedCount"`
	DraftCount       int64   `json:"draftCount"`
	TotalPlays       int64   `json:"totalPlays"`
	TotalSolves      int64   `json:"totalSolves"`
	OverallSolveRate float64 `json:"overallSolveRate"`
}

// GetPuzzleStats: 퍼즐 전체 통계
func (r *Repository) GetPuzzleStats(ctx context.Context) (*PuzzleStatsResult, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	result := &PuzzleStatsResult{}

	// 퍼즐 수 통계
	r.db.WithContext(ctx).Model(&Puzzle{}).Count(&result.TotalPuzzles)
	r.db.WithContext(ctx).Model(&Puzzle{}).Where("status = ?", "published").Count(&result.PublishedCount)
	r.db.WithContext(ctx).Model(&Puzzle{}).Where("status = ?", "draft").Count(&result.DraftCount)

	// 아카이브 통계
	var archiveStats struct {
		TotalPlays  int64 `gorm:"column:total_plays"`
		TotalSolves int64 `gorm:"column:total_solves"`
	}
	r.db.WithContext(ctx).Model(&GameArchive{}).
		Select("count(*) as total_plays, sum(case when result = 'solved' then 1 else 0 end) as total_solves").
		Scan(&archiveStats)

	result.TotalPlays = archiveStats.TotalPlays
	result.TotalSolves = archiveStats.TotalSolves

	if result.TotalPlays > 0 {
		result.OverallSolveRate = float64(result.TotalSolves) / float64(result.TotalPlays) * 100
	}

	return result, nil
}

// CategoryStats: 카테고리별 통계
type CategoryStats struct {
	Category   string  `json:"category"`
	TotalGames int     `json:"totalGames"`
	SolveCount int     `json:"solveCount"`
	SolveRate  float64 `json:"solveRate"`
}

// GetCategoryStats: 카테고리별 통계 조회
func (r *Repository) GetCategoryStats(ctx context.Context) ([]CategoryStats, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	type catAgg struct {
		Category   string `gorm:"column:category"`
		TotalGames int    `gorm:"column:total_games"`
		SolveCount int    `gorm:"column:solve_count"`
	}

	var results []catAgg
	if err := r.db.WithContext(ctx).Model(&Puzzle{}).
		Select("category, sum(play_count) as total_games, sum(solve_count) as solve_count").
		Group("category").
		Order("total_games DESC").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("get category stats failed: %w", err)
	}

	stats := make([]CategoryStats, 0, len(results))
	for _, r := range results {
		solveRate := float64(0)
		if r.TotalGames > 0 {
			solveRate = float64(r.SolveCount) / float64(r.TotalGames) * 100
		}
		stats = append(stats, CategoryStats{
			Category:   r.Category,
			TotalGames: r.TotalGames,
			SolveCount: r.SolveCount,
			SolveRate:  solveRate,
		})
	}

	return stats, nil
}
