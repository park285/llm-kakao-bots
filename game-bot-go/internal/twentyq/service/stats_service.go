package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"gorm.io/gorm"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
)

const (
	percentageMultiplier   = 100
	maxActivityDisplaySize = 10
)

// StatsService 전적 조회 서비스.
type StatsService struct {
	db           *gorm.DB
	sessionStore *redis.SessionStore
	msgProvider  *messageprovider.Provider
	logger       *slog.Logger
}

// NewStatsService 생성자.
func NewStatsService(
	db *gorm.DB,
	sessionStore *redis.SessionStore,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) *StatsService {
	return &StatsService{
		db:           db,
		sessionStore: sessionStore,
		msgProvider:  msgProvider,
		logger:       logger,
	}
}

// CategoryStat 카테고리별 통계 (JSON에서 파싱).
type CategoryStat struct {
	GamesCompleted    int     `json:"gamesCompleted"`
	Surrenders        int     `json:"surrenders"`
	QuestionsAsked    int     `json:"questionsAsked"`
	HintsUsed         int     `json:"hintsUsed"`
	BestQuestionCount *int    `json:"bestQuestionCount,omitempty"`
	BestTarget        *string `json:"bestTarget,omitempty"`
}

// ParticipantActivity 참여자별 활동.
type ParticipantActivity struct {
	Sender      string
	GamesPlayed int
}

// GetUserStats 개인 전적 조회.
func (s *StatsService) GetUserStats(
	ctx context.Context,
	chatID string,
	userID string,
	sender *string,
	targetNickname *string,
) (string, error) {
	// 다른 사용자 전적 조회
	if targetNickname != nil {
		nickname := strings.TrimSpace(*targetNickname)
		targetUserID, resolvedSender, ok, err := s.resolveTargetUserByNickname(ctx, chatID, nickname)
		if err != nil {
			s.logger.Warn("resolve_target_nickname_failed", "error", err, "chatID", chatID, "nickname", nickname)
			return s.msgProvider.Get(
				qmessages.StatsUserNotFound,
				messageprovider.P("nickname", nickname),
			), nil
		}
		if !ok {
			return s.msgProvider.Get(
				qmessages.StatsUserNotFound,
				messageprovider.P("nickname", nickname),
			), nil
		}

		stats, err := s.loadUserStats(ctx, chatID, targetUserID)
		if err != nil {
			s.logger.Warn("loadUserStats_failed", "error", err, "chatID", chatID, "userID", targetUserID)
			return s.msgProvider.Get(
				qmessages.StatsNoStats,
				messageprovider.P("nickname", resolvedSender),
			), nil
		}
		if stats == nil {
			return s.msgProvider.Get(
				qmessages.StatsNoStats,
				messageprovider.P("nickname", resolvedSender),
			), nil
		}
		return s.formatStats(stats, resolvedSender), nil
	}

	// 본인 전적 조회
	stats, err := s.loadUserStats(ctx, chatID, userID)
	if err != nil {
		s.logger.Warn("loadUserStats_failed", "error", err, "chatID", chatID, "userID", userID)
		return s.msgProvider.Get(qmessages.StatsNotFound), nil
	}
	if stats == nil {
		return s.msgProvider.Get(qmessages.StatsNotFound), nil
	}
	displayName := "누군가"
	if sender != nil && *sender != "" {
		displayName = *sender
	}
	return s.formatStats(stats, displayName), nil
}

// roomStatsAggregate DB 집계 결과를 담는 구조체.
type roomStatsAggregate struct {
	TotalGames   int `gorm:"column:total_games"`
	CorrectCount int `gorm:"column:correct_count"`
}

// GetRoomStats 방 전적 조회 (DB 수준 집계 최적화).
func (s *StatsService) GetRoomStats(
	ctx context.Context,
	chatID string,
	period qmodel.StatsPeriod,
) (string, error) {
	startTime := s.getPeriodStartTime(period)
	now := time.Now()

	// DB 수준에서 총 판수 및 정답 수 집계 (Over-fetching 방지)
	var stats roomStatsAggregate
	query := s.db.WithContext(ctx).
		Model(&repository.GameSession{}).
		Select("count(*) as total_games, sum(case when result = ? then 1 else 0 end) as correct_count",
			string(repository.GameResultCorrect)).
		Where("chat_id = ?", chatID)
	if startTime != nil {
		query = query.Where("completed_at >= ? AND completed_at <= ?", *startTime, now)
	}
	if err := query.Scan(&stats).Error; err != nil {
		return "", fmt.Errorf("aggregate game_sessions: %w", err)
	}

	if stats.TotalGames == 0 {
		return s.msgProvider.Get(
			qmessages.StatsRoomNoGames,
			messageprovider.P("period", s.getPeriodName(period)),
		), nil
	}

	completionRate := (stats.CorrectCount * percentageMultiplier) / stats.TotalGames

	// DB 수준에서 참여자 활동 집계 (GROUP BY, ORDER BY, LIMIT)
	var activities []ParticipantActivity
	activityQuery := s.db.WithContext(ctx).
		Model(&repository.GameLog{}).
		Select("sender, count(*) as games_played").
		Where("chat_id = ?", chatID)
	if startTime != nil {
		activityQuery = activityQuery.Where("completed_at >= ? AND completed_at <= ?", *startTime, now)
	}
	if err := activityQuery.
		Group("sender").
		Order("games_played DESC").
		Limit(maxActivityDisplaySize).
		Scan(&activities).Error; err != nil {
		return "", fmt.Errorf("aggregate game_logs: %w", err)
	}

	// 총 참여자 수 조회 (별도 쿼리로 정확한 카운트)
	var totalParticipants int64
	participantQuery := s.db.WithContext(ctx).
		Model(&repository.GameLog{}).
		Where("chat_id = ?", chatID)
	if startTime != nil {
		participantQuery = participantQuery.Where("completed_at >= ? AND completed_at <= ?", *startTime, now)
	}
	if err := participantQuery.Distinct("sender").Count(&totalParticipants).Error; err != nil {
		return "", fmt.Errorf("count participants: %w", err)
	}

	return s.formatRoomStats(period, stats.TotalGames, int(totalParticipants), completionRate, activities), nil
}

func (s *StatsService) loadUserStats(ctx context.Context, chatID string, userID string) (*repository.UserStats, error) {
	compositeID := repository.CompositeUserStatsID(chatID, userID)
	var stats repository.UserStats
	if err := s.db.WithContext(ctx).First(&stats, "id = ?", compositeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user_stats: %w", err)
	}
	return &stats, nil
}

func (s *StatsService) resolveTargetUserByNickname(
	ctx context.Context,
	chatID string,
	nickname string,
) (string, string, bool, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return "", "", false, nil
	}

	if s.sessionStore != nil {
		if targetPlayer := s.sessionStore.GetPlayerByNickname(ctx, chatID, nickname); targetPlayer != nil {
			return targetPlayer.UserID, targetPlayer.Sender, true, nil
		}
	}

	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", "", false, nil
	}

	var mapping repository.UserNicknameMap
	if err := s.db.WithContext(ctx).
		Where("chat_id = ? AND last_sender = ?", chatID, nickname).
		Order("last_seen_at DESC").
		First(&mapping).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", false, fmt.Errorf("query user_nickname_map: %w", err)
		}
		if err := s.db.WithContext(ctx).
			Where("chat_id = ? AND lower(last_sender) = lower(?)", chatID, nickname).
			Order("last_seen_at DESC").
			First(&mapping).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", "", false, nil
			}
			return "", "", false, fmt.Errorf("query user_nickname_map_ci: %w", err)
		}
	}

	return mapping.UserID, mapping.LastSender, true, nil
}

func (s *StatsService) formatStats(stats *repository.UserStats, senderName string) string {
	var parts []string

	// 카테고리별 통계 파싱
	totalGames := stats.TotalGamesCompleted
	categoryStats := make(map[string]CategoryStat)
	if stats.CategoryStatsJSON != nil && strings.TrimSpace(*stats.CategoryStatsJSON) != "" {
		categoryStats = s.parseCategoryStats(*stats.CategoryStatsJSON)
	}
	categoryStats = normalizeCategoryStats(categoryStats)
	if len(categoryStats) > 0 {
		sumCategoryGames := 0
		for _, stat := range categoryStats {
			sumCategoryGames += stat.GamesCompleted
		}
		if sumCategoryGames > 0 {
			totalGames = sumCategoryGames
		}
	}
	categoryStats = ensureAllCategoryStats(categoryStats)

	// 헤더
	parts = append(parts, s.msgProvider.Get(
		qmessages.StatsHeader,
		messageprovider.P("nickname", senderName),
		messageprovider.P("totalGames", totalGames),
	))

	if len(categoryStats) > 0 {
		s.addCategoryStats(&parts, categoryStats)
	}

	return strings.Join(parts, "\n")
}

func (s *StatsService) parseCategoryStats(jsonStr string) map[string]CategoryStat {
	result := make(map[string]CategoryStat)
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}

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

func normalizeCategoryKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return normalizeCategoryInput(key)
}

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

func (s *StatsService) addCategoryStats(parts *[]string, categoryStats map[string]CategoryStat) {
	// 카테고리 정렬
	type catEntry struct {
		name string
		stat CategoryStat
	}
	order := make(map[string]int, len(qconfig.AllCategories))
	for i, name := range qconfig.AllCategories {
		order[name] = i
	}
	entries := make([]catEntry, 0, len(categoryStats))
	for name, stat := range categoryStats {
		entries = append(entries, catEntry{name: name, stat: stat})
	}
	slices.SortFunc(entries, func(a, b catEntry) int {
		// GamesCompleted 내림차순 정렬
		if c := cmp.Compare(b.stat.GamesCompleted, a.stat.GamesCompleted); c != 0 {
			return c
		}
		// order map 기준 정렬
		orderA, okA := order[a.name]
		orderB, okB := order[b.name]
		if okA && okB {
			return cmp.Compare(orderA, orderB)
		}
		if okA != okB {
			if okA {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.name, b.name)
	})

	for _, entry := range entries {
		s.addSingleCategoryStats(parts, entry.name, entry.stat)
	}
}

func (s *StatsService) addSingleCategoryStats(parts *[]string, category string, stat CategoryStat) {
	displayCategory := category
	if korean := categoryToKorean(category); korean != nil {
		displayCategory = *korean
	}

	// 카테고리 헤더
	*parts = append(*parts, s.msgProvider.Get(
		qmessages.StatsCategoryHdr,
		messageprovider.P("category", displayCategory),
		messageprovider.P("games", stat.GamesCompleted),
	))

	// 완주율
	completed := stat.GamesCompleted - stat.Surrenders
	if completed < 0 {
		completed = 0
	}
	completionRate := 0
	if stat.GamesCompleted > 0 {
		completionRate = (completed * percentageMultiplier) / stat.GamesCompleted
	}
	*parts = append(*parts, s.msgProvider.Get(
		qmessages.StatsCategoryResults,
		messageprovider.P("completed", completed),
		messageprovider.P("surrender", stat.Surrenders),
		messageprovider.P("completionRate", completionRate),
	))

	// 평균 질문/힌트
	avgQuestions := 0.0
	avgHints := 0.0
	if stat.GamesCompleted > 0 {
		avgQuestions = float64(stat.QuestionsAsked) / float64(stat.GamesCompleted)
		avgHints = float64(stat.HintsUsed) / float64(stat.GamesCompleted)
	}
	*parts = append(*parts, s.msgProvider.Get(
		qmessages.StatsCategoryAverages,
		messageprovider.P("avgQuestions", fmt.Sprintf("%.1f", avgQuestions)),
		messageprovider.P("avgHints", fmt.Sprintf("%.1f", avgHints)),
	))

	// 베스트 스코어 출력 제거 (모바일 메시지 길이 축소)
}

func (s *StatsService) formatRoomStats(
	period qmodel.StatsPeriod,
	totalGames int,
	totalParticipants int,
	completionRate int,
	activities []ParticipantActivity,
) string {
	var parts []string

	// 헤더
	parts = append(parts,
		s.msgProvider.Get(
			qmessages.StatsRoomHeader,
			messageprovider.P("period", s.getPeriodName(period)),
		),
		"",
		s.msgProvider.Get(
			qmessages.StatsRoomSummary,
			messageprovider.P("totalGames", totalGames),
			messageprovider.P("totalParticipants", totalParticipants),
			messageprovider.P("completionRate", completionRate),
		),
	)

	// 참여 활동
	if len(activities) > 0 {
		parts = append(parts, "", s.msgProvider.Get(qmessages.StatsRoomActivityHdr))
		for _, activity := range activities {
			parts = append(parts, s.msgProvider.Get(
				qmessages.StatsRoomActivityItem,
				messageprovider.P("sender", activity.Sender),
				messageprovider.P("games", activity.GamesPlayed),
			))
		}
	}

	return strings.Join(parts, "\n")
}

func (s *StatsService) getPeriodStartTime(period qmodel.StatsPeriod) *time.Time {
	now := time.Now()
	var start time.Time
	switch period {
	case qmodel.StatsPeriodDaily:
		start = now.Truncate(24 * time.Hour)
	case qmodel.StatsPeriodWeekly:
		start = now.AddDate(0, 0, -7)
	case qmodel.StatsPeriodMonthly:
		start = now.AddDate(0, -1, 0)
	case qmodel.StatsPeriodAll:
		return nil
	default:
		return nil
	}
	return &start
}

func (s *StatsService) getPeriodName(period qmodel.StatsPeriod) string {
	switch period {
	case qmodel.StatsPeriodDaily:
		return s.msgProvider.Get(qmessages.StatsPeriodDaily)
	case qmodel.StatsPeriodWeekly:
		return s.msgProvider.Get(qmessages.StatsPeriodWeekly)
	case qmodel.StatsPeriodMonthly:
		return s.msgProvider.Get(qmessages.StatsPeriodMonthly)
	case qmodel.StatsPeriodAll:
		return s.msgProvider.Get(qmessages.StatsPeriodAll)
	default:
		return s.msgProvider.Get(qmessages.StatsPeriodAll)
	}
}
