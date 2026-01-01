package alarm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// Repository: 알람 데이터의 영속 저장소 (PostgreSQL)
type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewRepository: 새로운 알람 Repository를 생성합니다.
func NewRepository(db *sql.DB, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Add: 알람을 DB에 추가한다. 이미 존재하면 무시한다. (upsert)
func (r *Repository) Add(ctx context.Context, alarm *domain.Alarm) error {
	query := `
		INSERT INTO alarms (room_id, user_id, channel_id, member_name, room_name, user_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (room_id, user_id, channel_id) DO UPDATE
		SET member_name = COALESCE(EXCLUDED.member_name, alarms.member_name),
		    room_name = COALESCE(EXCLUDED.room_name, alarms.room_name),
		    user_name = COALESCE(EXCLUDED.user_name, alarms.user_name)
	`

	_, err := r.db.ExecContext(ctx, query,
		alarm.RoomID, alarm.UserID, alarm.ChannelID,
		alarm.MemberName, alarm.RoomName, alarm.UserName,
	)
	if err != nil {
		return fmt.Errorf("add alarm: %w", err)
	}
	return nil
}

// Remove: 특정 알람을 DB에서 삭제합니다.
func (r *Repository) Remove(ctx context.Context, roomID, userID, channelID string) error {
	query := `DELETE FROM alarms WHERE room_id = $1 AND user_id = $2 AND channel_id = $3`
	_, err := r.db.ExecContext(ctx, query, roomID, userID, channelID)
	if err != nil {
		return fmt.Errorf("remove alarm: %w", err)
	}
	return nil
}

// ClearByUser: 특정 사용자의 모든 알람을 삭제합니다.
func (r *Repository) ClearByUser(ctx context.Context, roomID, userID string) (int64, error) {
	query := `DELETE FROM alarms WHERE room_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, roomID, userID)
	if err != nil {
		return 0, fmt.Errorf("clear alarms: %w", err)
	}
	affected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return 0, fmt.Errorf("get rows affected: %w", rowsErr)
	}
	return affected, nil
}

// FindByUser: 특정 사용자의 모든 알람을 조회합니다.
func (r *Repository) FindByUser(ctx context.Context, roomID, userID string) ([]*domain.Alarm, error) {
	query := `
		SELECT id, room_id, user_id, channel_id, member_name, room_name, user_name, created_at
		FROM alarms
		WHERE room_id = $1 AND user_id = $2
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("find alarms by user: %w", err)
	}
	defer rows.Close()

	return r.scanAlarms(rows)
}

// FindByChannel: 특정 채널의 모든 구독자 알람을 조회합니다.
func (r *Repository) FindByChannel(ctx context.Context, channelID string) ([]*domain.Alarm, error) {
	query := `
		SELECT id, room_id, user_id, channel_id, member_name, room_name, user_name, created_at
		FROM alarms
		WHERE channel_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, fmt.Errorf("find alarms by channel: %w", err)
	}
	defer rows.Close()

	return r.scanAlarms(rows)
}

// GetMemberName: 채널ID에 해당하는 멤버 이름을 조회합니다.
// 해당 채널에 알람을 설정한 적이 있는 레코드에서 member_name을 가져온다.
func (r *Repository) GetMemberName(ctx context.Context, channelID string) (string, error) {
	query := `
		SELECT member_name FROM alarms
		WHERE channel_id = $1 AND member_name IS NOT NULL AND member_name != ''
		LIMIT 1
	`

	var memberName string
	err := r.db.QueryRowContext(ctx, query, channelID).Scan(&memberName)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get member name: %w", err)
	}
	return memberName, nil
}

// LoadAll: 모든 알람을 조회한다. (앱 시작 시 캐시 워밍용)
func (r *Repository) LoadAll(ctx context.Context) ([]*domain.Alarm, error) {
	query := `
		SELECT id, room_id, user_id, channel_id, member_name, room_name, user_name, created_at
		FROM alarms
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load all alarms: %w", err)
	}
	defer rows.Close()

	return r.scanAlarms(rows)
}

// GetAllChannelIDs: 알람이 설정된 모든 채널 ID를 조회합니다.
func (r *Repository) GetAllChannelIDs(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT channel_id FROM alarms`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all channel ids: %w", err)
	}
	defer rows.Close()

	var channelIDs []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, fmt.Errorf("scan channel id: %w", err)
		}
		channelIDs = append(channelIDs, channelID)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate channel ids: %w", rowsErr)
	}
	return channelIDs, nil
}

// GetAllMemberNames: 모든 채널ID → 멤버이름 매핑을 조회합니다.
func (r *Repository) GetAllMemberNames(ctx context.Context) (map[string]string, error) {
	query := `
		SELECT DISTINCT ON (channel_id) channel_id, member_name
		FROM alarms
		WHERE member_name IS NOT NULL AND member_name != ''
		ORDER BY channel_id, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all member names: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var channelID, memberName string
		if err := rows.Scan(&channelID, &memberName); err != nil {
			return nil, fmt.Errorf("scan member name: %w", err)
		}
		result[channelID] = memberName
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate member names: %w", rowsErr)
	}
	return result, nil
}

func (r *Repository) scanAlarms(rows *sql.Rows) ([]*domain.Alarm, error) {
	var alarms []*domain.Alarm
	for rows.Next() {
		var a domain.Alarm
		var memberName, roomName, userName sql.NullString

		err := rows.Scan(
			&a.ID, &a.RoomID, &a.UserID, &a.ChannelID,
			&memberName, &roomName, &userName, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alarm: %w", err)
		}

		a.MemberName = memberName.String
		a.RoomName = roomName.String
		a.UserName = userName.String
		alarms = append(alarms, &a)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate alarms: %w", rowsErr)
	}
	return alarms, nil
}
