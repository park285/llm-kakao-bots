package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repository: DB 접근을 위한 GORM 기반 리포지토리
// 메서드들은 도메인별 파일로 분리됨:
//   - nickname.go: 닉네임 관리
//   - game_stats.go: 게임 시작/완료 통계
//   - category_stats.go: 카테고리별 통계 JSON
//   - session_log.go: 세션/로그 기록
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
