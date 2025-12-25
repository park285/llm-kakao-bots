package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
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

// loadUserStats 사용자 통계 로드.
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

// resolveTargetUserByNickname 닉네임으로 대상 사용자 조회.
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

// getPeriodStartTime 기간 시작 시간 반환.
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
