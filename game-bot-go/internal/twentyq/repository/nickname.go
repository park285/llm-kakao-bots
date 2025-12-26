package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

// NicknameEntry 배치 닉네임 UPSERT용 엔트리.
type NicknameEntry struct {
	UserID     string
	LastSender string
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
