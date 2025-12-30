package youtube

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
)

// StatsRepository: YouTube 채널 통계 데이터(구독자 수 등)를 관리하는 저장소 (TimescaleDB)
type StatsRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewYouTubeStatsRepository: 새로운 StatsRepository 인스턴스를 생성한다.
func NewYouTubeStatsRepository(postgres *database.PostgresService, logger *slog.Logger) *StatsRepository {
	return &StatsRepository{
		db:     postgres.GetDB(),
		logger: logger,
	}
}

// SaveStats: 채널 통계 데이터를 저장한다.
func (r *StatsRepository) SaveStats(ctx context.Context, stats *domain.TimestampedStats) error {
	query := `
		INSERT INTO youtube_stats_history (time, channel_id, member_name, subscribers, videos, views)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (time, channel_id) DO UPDATE
		SET subscribers = EXCLUDED.subscribers,
		    videos = EXCLUDED.videos,
		    views = EXCLUDED.views
	`

	_, err := r.db.ExecContext(ctx, query,
		stats.Timestamp,
		stats.ChannelID,
		stats.MemberName,
		stats.SubscriberCount,
		stats.VideoCount,
		stats.ViewCount,
	)

	if err != nil {
		return fmt.Errorf("failed to save stats: %w", err)
	}

	r.logger.Debug("Stats saved to TimescaleDB",
		slog.String("channel", stats.ChannelID),
		slog.Any("subscribers", stats.SubscriberCount),
	)

	return nil
}

// GetLatestStats: 각 채널의 최신 통계 데이터를 조회한다.
func (r *StatsRepository) GetLatestStats(ctx context.Context, channelID string) (*domain.TimestampedStats, error) {
	query := `
		SELECT time, channel_id, member_name, subscribers, videos, views
		FROM youtube_stats_history
		WHERE channel_id = $1
		ORDER BY time DESC
		LIMIT 1
	`

	var stats domain.TimestampedStats
	var memberName sql.NullString

	err := r.db.QueryRowContext(ctx, query, channelID).Scan(
		&stats.Timestamp,
		&stats.ChannelID,
		&memberName,
		&stats.SubscriberCount,
		&stats.VideoCount,
		&stats.ViewCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest stats: %w", err)
	}

	if memberName.Valid {
		stats.MemberName = memberName.String
	}

	return &stats, nil
}

// GetLatestStatsForChannels: 여러 채널의 최신 통계를 한 번에 조회한다. (N+1 쿼리 방지)
func (r *StatsRepository) GetLatestStatsForChannels(ctx context.Context, channelIDs []string) (map[string]*domain.TimestampedStats, error) {
	if len(channelIDs) == 0 {
		return make(map[string]*domain.TimestampedStats), nil
	}

	// PostgreSQL의 DISTINCT ON을 사용한 배치 조회
	query := `
		SELECT DISTINCT ON (channel_id)
			time, channel_id, member_name, subscribers, videos, views
		FROM youtube_stats_history
		WHERE channel_id = ANY($1)
		ORDER BY channel_id, time DESC
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(channelIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to batch query stats: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*domain.TimestampedStats, len(channelIDs))
	for rows.Next() {
		var stats domain.TimestampedStats
		var memberName sql.NullString

		if err := rows.Scan(
			&stats.Timestamp,
			&stats.ChannelID,
			&memberName,
			&stats.SubscriberCount,
			&stats.VideoCount,
			&stats.ViewCount,
		); err != nil {
			r.logger.Warn("Failed to scan batch stats row", slog.Any("error", err))
			continue
		}

		if memberName.Valid {
			stats.MemberName = memberName.String
		}
		result[stats.ChannelID] = &stats
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

// GetAchievedMilestones: 여러 채널의 달성된 마일스톤을 배치 조회한다. (N+1 쿼리 방지)
// 반환: map[channelID][]uint64 (채널별 달성된 마일스톤 값 목록)
func (r *StatsRepository) GetAchievedMilestones(ctx context.Context, channelIDs []string, milestoneType domain.MilestoneType) (map[string][]uint64, error) {
	if len(channelIDs) == 0 {
		return make(map[string][]uint64), nil
	}

	query := `
		SELECT channel_id, value
		FROM youtube_milestones
		WHERE channel_id = ANY($1) AND type = $2
		ORDER BY channel_id, value
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(channelIDs), string(milestoneType))
	if err != nil {
		return nil, fmt.Errorf("failed to batch query milestones: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]uint64, len(channelIDs))
	for rows.Next() {
		var channelID string
		var value uint64

		if err := rows.Scan(&channelID, &value); err != nil {
			r.logger.Warn("Failed to scan milestone row", slog.Any("error", err))
			continue
		}

		result[channelID] = append(result[channelID], value)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

// RecordChange: 구독자 수 등의 변화를 기록한다.
func (r *StatsRepository) RecordChange(ctx context.Context, change *domain.StatsChange) error {
	query := `
		INSERT INTO youtube_stats_changes
		(channel_id, member_name, subscriber_change, video_change, view_change,
		 previous_subs, current_subs, previous_videos, current_videos, detected_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var prevSubs, currSubs, prevVideos, currVideos sql.NullInt64

	if change.PreviousStats != nil {
		prevSubs = sql.NullInt64{Int64: int64(change.PreviousStats.SubscriberCount), Valid: true}
		prevVideos = sql.NullInt64{Int64: int64(change.PreviousStats.VideoCount), Valid: true}
	}

	if change.CurrentStats != nil {
		currSubs = sql.NullInt64{Int64: int64(change.CurrentStats.SubscriberCount), Valid: true}
		currVideos = sql.NullInt64{Int64: int64(change.CurrentStats.VideoCount), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		change.ChannelID,
		change.MemberName,
		change.SubscriberChange,
		change.VideoChange,
		change.ViewChange,
		prevSubs,
		currSubs,
		prevVideos,
		currVideos,
		change.DetectedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record change: %w", err)
	}

	r.logger.Info("Change recorded",
		slog.String("member", change.MemberName),
		slog.Int64("sub_change", change.SubscriberChange),
		slog.Int64("vid_change", change.VideoChange),
	)

	return nil
}

// RecordMilestone: 구독자 수 달성 등 마일스톤 이벤트를 기록한다.
func (r *StatsRepository) SaveMilestone(ctx context.Context, milestone *domain.Milestone) error {
	query := `
		INSERT INTO youtube_milestones (channel_id, member_name, type, value, achieved_at, notified)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		milestone.ChannelID,
		milestone.MemberName,
		string(milestone.Type),
		milestone.Value,
		milestone.AchievedAt,
		milestone.Notified,
	)

	if err != nil {
		return fmt.Errorf("failed to save milestone: %w", err)
	}

	r.logger.Info("Milestone saved",
		slog.String("member", milestone.MemberName),
		slog.String("type", string(milestone.Type)),
		slog.Any("value", milestone.Value),
	)

	return nil
}

// HasAchievedMilestone: 특정 채널이 특정 마일스톤을 이미 달성했는지 확인한다.
// 구독자가 감소 후 다시 증가해도 중복 달성으로 처리되지 않도록 방지한다.
func (r *StatsRepository) HasAchievedMilestone(ctx context.Context, channelID string, milestoneType domain.MilestoneType, value uint64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM youtube_milestones
			WHERE channel_id = $1 AND type = $2 AND value = $3
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, channelID, string(milestoneType), value).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check milestone: %w", err)
	}

	return exists, nil
}

// GetUnnotifiedChanges: 아직 알림이 발송되지 않은 통계 변화 내역을 최신순으로 조회한다.
// PreviousStats와 CurrentStats를 복원하여 마일스톤 검출이 가능하도록 한다.
func (r *StatsRepository) GetUnnotifiedChanges(ctx context.Context, limit int) ([]*domain.StatsChange, error) {
	query := `
		SELECT channel_id, member_name, subscriber_change, video_change, view_change, 
		       previous_subs, current_subs, detected_at
		FROM youtube_stats_changes
		WHERE notified = false
		ORDER BY detected_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unnotified changes: %w", err)
	}
	defer rows.Close()

	var changes []*domain.StatsChange
	for rows.Next() {
		var change domain.StatsChange
		var prevSubs, currSubs sql.NullInt64

		if err := rows.Scan(
			&change.ChannelID,
			&change.MemberName,
			&change.SubscriberChange,
			&change.VideoChange,
			&change.ViewChange,
			&prevSubs,
			&currSubs,
			&change.DetectedAt,
		); err != nil {
			r.logger.Warn("Failed to scan change row", slog.Any("error", err))
			continue
		}

		// PreviousStats/CurrentStats 복원 (마일스톤 검출에 필요)
		if prevSubs.Valid {
			change.PreviousStats = &domain.TimestampedStats{
				ChannelID:       change.ChannelID,
				MemberName:      change.MemberName,
				SubscriberCount: uint64(prevSubs.Int64),
			}
		}
		if currSubs.Valid {
			change.CurrentStats = &domain.TimestampedStats{
				ChannelID:       change.ChannelID,
				MemberName:      change.MemberName,
				SubscriberCount: uint64(currSubs.Int64),
			}
		}

		changes = append(changes, &change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return changes, nil
}

// MarkChangeNotified: 특정 통계 변화 내역을 알림 발송 완료 상태로 처리한다.
func (r *StatsRepository) MarkChangeNotified(ctx context.Context, channelID string, detectedAt time.Time) error {
	query := `
		UPDATE youtube_stats_changes
		SET notified = true
		WHERE channel_id = $1 AND detected_at = $2
	`

	_, err := r.db.ExecContext(ctx, query, channelID, detectedAt)
	if err != nil {
		return fmt.Errorf("failed to mark change notified: %w", err)
	}

	return nil
}

// GetTopGainers: 특정 시점 이후 구독자 증가량이 가장 높은 채널 상위 목록을 조회한다.
func (r *StatsRepository) GetTopGainers(ctx context.Context, since time.Time, limit int) ([]domain.RankEntry, error) {
	query := `
		WITH latest AS (
			SELECT DISTINCT ON (channel_id)
				channel_id, member_name, subscribers
			FROM youtube_stats_history
			WHERE time >= $1
			ORDER BY channel_id, time DESC
		),
		earliest AS (
			SELECT DISTINCT ON (channel_id)
				channel_id, subscribers
			FROM youtube_stats_history
			WHERE time >= $1
			ORDER BY channel_id, time ASC
		)
		SELECT
			latest.channel_id,
			latest.member_name,
			(latest.subscribers - earliest.subscribers) AS gain,
			latest.subscribers AS current_subscribers
		FROM latest
		JOIN earliest ON latest.channel_id = earliest.channel_id
		WHERE (latest.subscribers - earliest.subscribers) > 0
		ORDER BY gain DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top gainers: %w", err)
	}
	defer rows.Close()

	var entries []domain.RankEntry
	rank := 1
	for rows.Next() {
		var entry domain.RankEntry
		var currentSubs int64
		if err := rows.Scan(&entry.ChannelID, &entry.MemberName, &entry.Value, &currentSubs); err != nil {
			r.logger.Warn("Failed to scan rank entry", slog.Any("error", err))
			continue
		}
		if currentSubs > 0 {
			entry.CurrentSubscribers = uint64(currentSubs)
		}
		entry.Rank = rank
		entries = append(entries, entry)
		rank++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

// MilestoneEntry: API 응답용 마일스톤 정보
type MilestoneEntry struct {
	ChannelID  string    `json:"channelId"`
	MemberName string    `json:"memberName"`
	Type       string    `json:"type"`
	Value      uint64    `json:"value"`
	AchievedAt time.Time `json:"achievedAt"`
	Notified   bool      `json:"notified"`
}

// MilestoneFilter: 마일스톤 조회 필터
type MilestoneFilter struct {
	Limit      int
	Offset     int
	ChannelID  string
	MemberName string
}

// MilestoneResult: 마일스톤 조회 결과 (페이지네이션 정보 포함)
type MilestoneResult struct {
	Milestones []MilestoneEntry `json:"milestones"`
	Total      int              `json:"total"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
}

// GetAllMilestones: 달성된 마일스톤 목록을 조회한다 (페이지네이션/필터링 지원)
func (r *StatsRepository) GetAllMilestones(ctx context.Context, filter MilestoneFilter) (*MilestoneResult, error) {
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filter.ChannelID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("channel_id = $%d", argIdx))
		args = append(args, filter.ChannelID)
		argIdx++
	}
	if filter.MemberName != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("member_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.MemberName+"%")
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Count Total
	countQuery := "SELECT COUNT(*) FROM youtube_milestones " + whereSQL
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count milestones: %w", err)
	}

	// 2. Select Data
	// nolint:gosec // G201: 동적 WHERE 절은 파라미터화된 값만 사용하므로 안전
	query := fmt.Sprintf(`
		SELECT channel_id, member_name, type, value, achieved_at, notified
		FROM youtube_milestones
		%s
		ORDER BY achieved_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query milestones: %w", err)
	}
	defer rows.Close()

	var entries []MilestoneEntry
	for rows.Next() {
		var e MilestoneEntry
		if err := rows.Scan(&e.ChannelID, &e.MemberName, &e.Type, &e.Value, &e.AchievedAt, &e.Notified); err != nil {
			r.logger.Warn("Failed to scan milestone entry", slog.Any("error", err))
			continue
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return &MilestoneResult{
		Milestones: entries,
		Total:      totalCount,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}, nil
}

// NearMilestoneEntry: API 응답용 마일스톤 직전 멤버 정보
type NearMilestoneEntry struct {
	ChannelID     string  `json:"channelId"`
	MemberName    string  `json:"memberName"`
	CurrentSubs   uint64  `json:"currentSubs"`
	NextMilestone uint64  `json:"nextMilestone"`
	Remaining     int64   `json:"remaining"`
	ProgressPct   float64 `json:"progressPct"`
}

// GetNearMilestoneMembers: 마일스톤 직전(threshold% 이상) 멤버를 조회한다. 졸업 멤버 제외, Limit 지원.
func (r *StatsRepository) GetNearMilestoneMembers(ctx context.Context, thresholdPct float64, milestones []uint64, limit int) ([]NearMilestoneEntry, error) {
	if len(milestones) == 0 {
		return nil, nil
	}

	// CTE를 사용한 효율적인 쿼리
	query := `
		WITH latest_stats AS (
			SELECT DISTINCT ON (h.channel_id)
				h.channel_id, h.member_name, h.subscribers
			FROM youtube_stats_history h
			JOIN members m ON h.channel_id = m.channel_id
			WHERE m.is_graduated = false
			ORDER BY h.channel_id, h.time DESC
		),
		milestones AS (
			SELECT unnest($1::bigint[]) as milestone
		),
		achieved AS (
			SELECT channel_id, value
			FROM youtube_milestones
			WHERE type = 'subscribers'
		)
		SELECT 
			ls.channel_id,
			ls.member_name,
			ls.subscribers as current_subs,
			m.milestone as next_milestone,
			m.milestone - ls.subscribers as remaining,
			ROUND(100.0 * ls.subscribers / m.milestone, 2) as progress_pct
		FROM latest_stats ls
		CROSS JOIN milestones m
		LEFT JOIN achieved a ON ls.channel_id = a.channel_id AND m.milestone = a.value
		WHERE ls.subscribers < m.milestone 
			AND ls.subscribers >= m.milestone::float8 * $2::float8
			AND a.channel_id IS NULL
			AND ls.member_name IS NOT NULL
		ORDER BY progress_pct DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(milestones), thresholdPct, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query near milestone members: %w", err)
	}
	defer rows.Close()

	var entries []NearMilestoneEntry
	for rows.Next() {
		var e NearMilestoneEntry
		if err := rows.Scan(&e.ChannelID, &e.MemberName, &e.CurrentSubs, &e.NextMilestone, &e.Remaining, &e.ProgressPct); err != nil {
			r.logger.Warn("Failed to scan near milestone entry", slog.Any("error", err))
			continue
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

// GetClosestMilestoneMembers: 마일스톤 달성률이 높은 순서대로 상위 멤버를 조회한다 (threshold 없음, 졸업 멤버 자동 제외)
// allowedChannelIDs는 더 이상 사용하지 않고 DB JOIN으로 처리함
func (r *StatsRepository) GetClosestMilestoneMembers(ctx context.Context, limit int, milestones []uint64) ([]NearMilestoneEntry, error) {
	if len(milestones) == 0 {
		return nil, nil
	}

	query := `
		WITH latest_stats AS (
			SELECT DISTINCT ON (h.channel_id)
				h.channel_id, h.member_name, h.subscribers
			FROM youtube_stats_history h
			JOIN members m ON h.channel_id = m.channel_id
			WHERE m.is_graduated = false
			ORDER BY h.channel_id, h.time DESC
		),
		milestones AS (
			SELECT unnest($1::bigint[]) as milestone
		),
		next_milestones AS (
            SELECT 
                ls.channel_id,
                ls.member_name,
                ls.subscribers,
                MIN(m.milestone) as next_milestone
            FROM latest_stats ls
            CROSS JOIN milestones m
            WHERE ls.subscribers < m.milestone
            GROUP BY ls.channel_id, ls.member_name, ls.subscribers
        )
		SELECT 
			nm.channel_id,
			nm.member_name,
			nm.subscribers as current_subs,
			nm.next_milestone,
			nm.next_milestone - nm.subscribers as remaining,
			ROUND(100.0 * nm.subscribers / nm.next_milestone, 2) as progress_pct
		FROM next_milestones nm
        LEFT JOIN youtube_milestones ym ON nm.channel_id = ym.channel_id AND nm.next_milestone = ym.value
        WHERE ym.channel_id IS NULL
		ORDER BY progress_pct DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(milestones), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query closest milestone members: %w", err)
	}
	defer rows.Close()

	var entries []NearMilestoneEntry
	for rows.Next() {
		var e NearMilestoneEntry
		if err := rows.Scan(&e.ChannelID, &e.MemberName, &e.CurrentSubs, &e.NextMilestone, &e.Remaining, &e.ProgressPct); err != nil {
			r.logger.Warn("Failed to scan closest milestone entry", slog.Any("error", err))
			continue
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

// MilestoneStats: 마일스톤 관련 통계 요약
type MilestoneStats struct {
	TotalAchieved      int `json:"totalAchieved"`
	TotalNearMilestone int `json:"totalNearMilestone"`
	RecentAchievements int `json:"recentAchievements"` // 최근 30일
	NotNotifiedCount   int `json:"notNotifiedCount"`
}

// GetMilestoneStats: 마일스톤 통계 요약을 조회한다
func (r *StatsRepository) GetMilestoneStats(ctx context.Context) (*MilestoneStats, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM youtube_milestones) as total_achieved,
			(SELECT COUNT(*) FROM youtube_milestones WHERE achieved_at > NOW() - INTERVAL '30 days') as recent,
			(SELECT COUNT(*) FROM youtube_milestones WHERE notified = false) as not_notified
	`

	var stats MilestoneStats
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalAchieved,
		&stats.RecentAchievements,
		&stats.NotNotifiedCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get milestone stats: %w", err)
	}

	return &stats, nil
}
