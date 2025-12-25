package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository: DB 접근을 위한 GORM 기반 리포지토리
type Repository struct {
	db *gorm.DB
}

// New: 새로운 Repository 인스턴스를 생성한다.
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// AutoMigrate: 자동으로 DB 테이블 스키마를 마이그레이션한다.
func (r *Repository) AutoMigrate(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := r.db.WithContext(ctx).AutoMigrate(
		&GameSession{},
		&GameLog{},
		&UserStats{},
		&UserNicknameMap{},
	); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	return nil
}

// CompositeUserStatsID: 사용자 통계 ID (ChatID:UserID) 생성 함수
func CompositeUserStatsID(chatID string, userID string) string {
	return strings.TrimSpace(chatID) + ":" + strings.TrimSpace(userID)
}

// GenerateFallbackSessionID: 세션 ID가 없을 경우 대체 ID 생성 함수
func GenerateFallbackSessionID(chatID string) string {
	chatID = strings.TrimSpace(chatID)

	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return chatID + ":" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return chatID + ":" + hex.EncodeToString(b[:])
}

// UpsertNickname: 사용자 닉네임 정보를 삽입하거나 업데이트한다.
func (r *Repository) UpsertNickname(
	ctx context.Context,
	chatID string,
	userID string,
	lastSender string,
	lastSeenAt time.Time,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	chatID = strings.TrimSpace(chatID)
	userID = strings.TrimSpace(userID)
	lastSender = strings.TrimSpace(lastSender)
	if chatID == "" || userID == "" || lastSender == "" {
		return nil
	}

	entry := UserNicknameMap{
		ChatID:     chatID,
		UserID:     userID,
		LastSender: lastSender,
		LastSeenAt: lastSeenAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chat_id"},
			{Name: "user_id"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"last_sender":  lastSender,
			"last_seen_at": lastSeenAt,
		}),
	}).Create(&entry).Error; err != nil {
		return fmt.Errorf("upsert nickname failed: %w", err)
	}

	return nil
}

// NicknameEntry 배치 닉네임 UPSERT용 엔트리.
type NicknameEntry struct {
	UserID     string
	LastSender string
}

// BatchUpsertNicknames 여러 플레이어의 닉네임을 한 번의 DB 호출로 UPSERT.
// N개의 개별 DB 호출을 1개로 줄여 성능 향상.
func (r *Repository) BatchUpsertNicknames(
	ctx context.Context,
	chatID string,
	entries []NicknameEntry,
	lastSeenAt time.Time,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("db is nil")
	}

	chatID = strings.TrimSpace(chatID)
	if chatID == "" || len(entries) == 0 {
		return nil
	}

	// 유효한 엔트리만 필터링
	records := make([]UserNicknameMap, 0, len(entries))
	for _, e := range entries {
		userID := strings.TrimSpace(e.UserID)
		sender := strings.TrimSpace(e.LastSender)
		if userID == "" || sender == "" {
			continue
		}
		records = append(records, UserNicknameMap{
			ChatID:     chatID,
			UserID:     userID,
			LastSender: sender,
			LastSeenAt: lastSeenAt,
		})
	}

	if len(records) == 0 {
		return nil
	}

	// 단일 배치 UPSERT
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chat_id"},
			{Name: "user_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"last_sender", "last_seen_at"}),
	}).CreateInBatches(records, 100).Error; err != nil {
		return fmt.Errorf("batch upsert nicknames failed: %w", err)
	}

	return nil
}

// RecordGameStart: 게임 시작 정보를 기록한다 (사용자 통계 초기화 등).
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

type categoryStat struct {
	GamesCompleted    int     `json:"gamesCompleted"`
	Surrenders        int     `json:"surrenders"`
	QuestionsAsked    int     `json:"questionsAsked"`
	HintsUsed         int     `json:"hintsUsed"`
	BestQuestionCount *int    `json:"bestQuestionCount,omitempty"`
	BestTarget        *string `json:"bestTarget,omitempty"`
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

// RecordGameCompletion: 게임 완료 정보를 기록하고 관련 통계를 업데이트한다.
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
// SELECT FOR UPDATE + UPDATE 2회 쿼리를 1회로 최적화하여 락 보유 시간 및 라운드트립 감소.
func updateCategoryStatsJSONPostgres(tx *gorm.DB, userStatsID, categoryKey string, p GameCompletionParams) error {
	surrenderInc := 0
	if p.Result == GameResultSurrender {
		surrenderInc = 1
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
				'hintsUsed', COALESCE((category_stats_json->$2->>'hintsUsed')::int, 0) + $5
			) || COALESCE(
				jsonb_build_object(
					'bestQuestionCount', category_stats_json->$2->'bestQuestionCount',
					'bestTarget', category_stats_json->$2->'bestTarget'
				),
				'{}'::jsonb
			),
			true
		),
		updated_at = NOW(),
		version = version + 1
		WHERE id = $6
	`

	pathArray := "{" + categoryKey + "}"
	result := tx.Exec(query, pathArray, categoryKey, surrenderInc, p.QuestionCount, p.HintCount, userStatsID)
	if result.Error != nil {
		return fmt.Errorf("update category stats postgres failed: %w", result.Error)
	}

	// 베스트 스코어 업데이트 (정답인 경우만)
	if p.Result == GameResultCorrect && p.Target != nil {
		if err := updateCategoryBestScorePostgres(tx, userStatsID, categoryKey, p); err != nil {
			return err
		}
	}

	return nil
}

// updateCategoryBestScorePostgres 카테고리별 베스트 스코어를 원자적으로 업데이트.
func updateCategoryBestScorePostgres(tx *gorm.DB, userStatsID, categoryKey string, p GameCompletionParams) error {
	target := strings.TrimSpace(*p.Target)
	if target == "" {
		target = *p.Target
	}

	// 기존 베스트보다 좋은 경우에만 업데이트 (CASE WHEN 사용)
	query := `
		UPDATE user_stats
		SET category_stats_json = jsonb_set(
			category_stats_json,
			$1::text[],
			(category_stats_json->$2) || jsonb_build_object(
				'bestQuestionCount', $3,
				'bestTarget', $4::text
			),
			true
		)
		WHERE id = $5
		AND (
			category_stats_json->$2->>'bestQuestionCount' IS NULL
			OR (category_stats_json->$2->>'bestQuestionCount')::int > $3
		)
	`

	pathArray := "{" + categoryKey + "}"
	result := tx.Exec(query, pathArray, categoryKey, p.TotalGameQuestionCount, target, userStatsID)
	if result.Error != nil {
		return fmt.Errorf("update category best score postgres failed: %w", result.Error)
	}

	return nil
}

// updateCategoryStatsJSONGeneric Non-PostgreSQL용 폴백 (SQLite 등).
func updateCategoryStatsJSONGeneric(tx *gorm.DB, userStatsID, categoryKey string, p GameCompletionParams) error {
	var stats UserStats
	if err := tx.Select("category_stats_json").
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

// RecordGameSession: 게임 세션 메타데이터를 기록한다.
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

// RecordGameLog: 플레이어별 게임 활동 로그를 기록한다.
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
