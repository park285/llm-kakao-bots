package member

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
)

// Model: members 테이블과 매핑되는 GORM 모델입니다.
type Model struct {
	ID             int            `gorm:"primaryKey;column:id"`
	Slug           string         `gorm:"column:slug"`
	ChannelID      *string        `gorm:"column:channel_id"`
	EnglishName    string         `gorm:"column:english_name"`
	JapaneseName   *string        `gorm:"column:japanese_name"`
	KoreanName     *string        `gorm:"column:korean_name"`
	Status         string         `gorm:"column:status"`
	IsGraduated    bool           `gorm:"column:is_graduated"`
	Aliases        datatypes.JSON `gorm:"column:aliases;type:jsonb"`
	Photo          *string        `gorm:"column:photo"`            // YouTube 프로필 이미지 URL (고화질)
	PhotoUpdatedAt *time.Time     `gorm:"column:photo_updated_at"` // photo 마지막 동기화 시간
}

// TableName: GORM 모델이 매핑될 데이터베이스 테이블 이름을 반환한다. ("members")
func (Model) TableName() string {
	return "members"
}

// Repository: 멤버 정보에 대한 데이터베이스 접근을 담당하는 저장소
type Repository struct {
	db     *sql.DB
	gormDB *gorm.DB
	logger *slog.Logger
}

// NewMemberRepository: 새로운 MemberRepository 인스턴스를 생성합니다.
func NewMemberRepository(postgres *database.PostgresService, logger *slog.Logger) *Repository {
	return &Repository{
		db:     postgres.GetDB(),
		gormDB: postgres.GetGormDB(),
		logger: logger,
	}
}

// FindByChannelID: 채널 ID로 멤버를 조회합니다.
func (r *Repository) FindByChannelID(ctx context.Context, channelID string) (*domain.Member, error) {
	query := `
		SELECT id, slug, channel_id, english_name, japanese_name, korean_name,
		       status, is_graduated, aliases
		FROM members
		WHERE channel_id = $1
		LIMIT 1
	`

	var (
		id           int
		slug         string
		channelIDVal sql.NullString
		englishName  string
		japaneseName sql.NullString
		koreanName   sql.NullString
		status       string
		isGraduated  bool
		aliasesJSON  []byte
	)

	err := r.db.QueryRowContext(ctx, query, channelID).Scan(
		&id, &slug, &channelIDVal, &englishName, &japaneseName, &koreanName,
		&status, &isGraduated, &aliasesJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query member by channel_id: %w", err)
	}

	return r.scanMember(id, slug, channelIDVal, englishName, japaneseName, koreanName, status, isGraduated, aliasesJSON)
}

// FindByName: 이름으로 멤버를 조회합니다.
func (r *Repository) FindByName(ctx context.Context, name string) (*domain.Member, error) {
	query := `
		SELECT id, slug, channel_id, english_name, japanese_name, korean_name,
		       status, is_graduated, aliases
		FROM members
		WHERE english_name = $1
		LIMIT 1
	`

	var (
		id           int
		slug         string
		channelID    sql.NullString
		englishName  string
		japaneseName sql.NullString
		koreanName   sql.NullString
		status       string
		isGraduated  bool
		aliasesJSON  []byte
	)

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&id, &slug, &channelID, &englishName, &japaneseName, &koreanName,
		&status, &isGraduated, &aliasesJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query member by name: %w", err)
	}

	return r.scanMember(id, slug, channelID, englishName, japaneseName, koreanName, status, isGraduated, aliasesJSON)
}

// FindByAlias: 별칭으로 멤버를 검색합니다.
func (r *Repository) FindByAlias(ctx context.Context, alias string) (*domain.Member, error) {
	query := `
		SELECT m.id, m.slug, m.channel_id, m.english_name, m.japanese_name, m.korean_name,
		       m.status, m.is_graduated, m.aliases
		FROM members m
		WHERE m.aliases->'ko' ? $1
		   OR m.aliases->'ja' ? $1
		   OR m.english_name ILIKE $1
		   OR m.japanese_name ILIKE $1
		   OR m.korean_name ILIKE $1
		LIMIT 1
	`

	var (
		id           int
		slug         string
		channelID    sql.NullString
		englishName  string
		japaneseName sql.NullString
		koreanName   sql.NullString
		status       string
		isGraduated  bool
		aliasesJSON  []byte
	)

	err := r.db.QueryRowContext(ctx, query, alias).Scan(
		&id, &slug, &channelID, &englishName, &japaneseName, &koreanName,
		&status, &isGraduated, &aliasesJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query member by alias: %w", err)
	}

	return r.scanMember(id, slug, channelID, englishName, japaneseName, koreanName, status, isGraduated, aliasesJSON)
}

// GetAllChannelIDs: 모든 멤버의 채널 ID 목록을 반환합니다.
func (r *Repository) GetAllChannelIDs(ctx context.Context) ([]string, error) {
	query := `
		SELECT channel_id
		FROM members
		WHERE channel_id IS NOT NULL
		ORDER BY english_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channel ids: %w", err)
	}
	defer rows.Close()

	var channelIDs []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			r.logger.Warn("Failed to scan channel ID", slog.Any("error", err))
			continue
		}
		channelIDs = append(channelIDs, channelID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return channelIDs, nil
}

// GetAllMembers: 모든 멤버 목록을 조회합니다.
func (r *Repository) GetAllMembers(ctx context.Context) ([]*domain.Member, error) {
	query := `
		SELECT id, slug, channel_id, english_name, japanese_name, korean_name,
		       status, is_graduated, aliases
		FROM members
		ORDER BY english_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all members: %w", err)
	}
	defer rows.Close()

	var members []*domain.Member
	for rows.Next() {
		var (
			id           int
			slug         string
			channelID    sql.NullString
			englishName  string
			japaneseName sql.NullString
			koreanName   sql.NullString
			status       string
			isGraduated  bool
			aliasesJSON  []byte
		)

		if err := rows.Scan(&id, &slug, &channelID, &englishName, &japaneseName, &koreanName,
			&status, &isGraduated, &aliasesJSON); err != nil {
			r.logger.Warn("Failed to scan member row", slog.Any("error", err))
			continue
		}

		member, err := r.scanMember(id, slug, channelID, englishName, japaneseName, koreanName, status, isGraduated, aliasesJSON)
		if err != nil {
			r.logger.Warn("Failed to parse member", slog.String("name", englishName), slog.Any("error", err))
			continue
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return members, nil
}

// GetMembersWithPhoto: 프로필 이미지가 포함된 멤버 목록을 조회합니다. (API 응답용)
func (r *Repository) GetMembersWithPhoto(ctx context.Context, channelIDs []string) (map[string]*domain.Member, error) {
	if len(channelIDs) == 0 {
		return make(map[string]*domain.Member), nil
	}

	query := `
		SELECT id, channel_id, english_name, japanese_name, korean_name,
		       is_graduated, aliases, photo
		FROM members
		WHERE channel_id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(channelIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query members with photo: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*domain.Member, len(channelIDs))
	for rows.Next() {
		var (
			id           int
			channelID    sql.NullString
			englishName  string
			japaneseName sql.NullString
			koreanName   sql.NullString
			isGraduated  bool
			aliasesJSON  []byte
			photo        sql.NullString
		)

		if err := rows.Scan(&id, &channelID, &englishName, &japaneseName, &koreanName,
			&isGraduated, &aliasesJSON, &photo); err != nil {
			r.logger.Warn("Failed to scan member row", slog.Any("error", err))
			continue
		}

		member, err := r.scanMemberWithPhoto(id, channelID, englishName, japaneseName, koreanName, isGraduated, aliasesJSON, photo)
		if err != nil {
			r.logger.Warn("Failed to parse member", slog.String("name", englishName), slog.Any("error", err))
			continue
		}

		if channelID.Valid {
			result[channelID.String] = member
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

// GetMemberWithPhotoByChannelID: 채널 ID로 프로필 이미지가 포함된 멤버를 조회합니다.
func (r *Repository) GetMemberWithPhotoByChannelID(ctx context.Context, channelID string) (*domain.Member, error) {
	query := `
		SELECT id, channel_id, english_name, japanese_name, korean_name,
		       is_graduated, aliases, photo
		FROM members
		WHERE channel_id = $1
		LIMIT 1
	`

	var (
		id           int
		chID         sql.NullString
		englishName  string
		japaneseName sql.NullString
		koreanName   sql.NullString
		isGraduated  bool
		aliasesJSON  []byte
		photo        sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, channelID).Scan(
		&id, &chID, &englishName, &japaneseName, &koreanName,
		&isGraduated, &aliasesJSON, &photo,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query member by channel_id: %w", err)
	}

	return r.scanMemberWithPhoto(id, chID, englishName, japaneseName, koreanName, isGraduated, aliasesJSON, photo)
}

// scanMember: DB 조회 결과를 domain.Member로 변환함
func (r *Repository) scanMember(
	id int,
	_ string, // slug: domain.Member에서 미사용
	channelID sql.NullString,
	englishName string,
	japaneseName sql.NullString,
	koreanName sql.NullString,
	_ string, // status: domain.Member에서 미사용
	isGraduated bool,
	aliasesJSON []byte,
) (*domain.Member, error) {
	return r.scanMemberWithPhoto(id, channelID, englishName, japaneseName, koreanName, isGraduated, aliasesJSON, sql.NullString{})
}

// scanMemberWithPhoto: DB 조회 결과를 domain.Member로 변환 (photo 포함)
func (r *Repository) scanMemberWithPhoto(
	id int,
	channelID sql.NullString,
	englishName string,
	japaneseName sql.NullString,
	koreanName sql.NullString,
	isGraduated bool,
	aliasesJSON []byte,
	photo sql.NullString,
) (*domain.Member, error) {
	var aliases domain.Aliases
	if err := json.Unmarshal(aliasesJSON, &aliases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal aliases: %w", err)
	}

	member := &domain.Member{
		ID:          id,
		Name:        englishName,
		Aliases:     &aliases,
		IsGraduated: isGraduated,
	}

	if channelID.Valid {
		member.ChannelID = channelID.String
	}
	if japaneseName.Valid {
		member.NameJa = japaneseName.String
	}
	if koreanName.Valid {
		member.NameKo = koreanName.String
	}
	if photo.Valid {
		member.Photo = photo.String
	}

	return member, nil
}

// AddAlias: 멤버에게 별칭을 추가합니다.
func (r *Repository) AddAlias(ctx context.Context, memberID int, aliasType string, alias string) error {
	if aliasType != "ko" && aliasType != "ja" {
		return fmt.Errorf("invalid alias type: %s (must be 'ko' or 'ja')", aliasType)
	}

	var member Model
	if err := r.gormDB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		return fmt.Errorf("failed to find member: %w", err)
	}

	var aliases domain.Aliases
	if err := json.Unmarshal(member.Aliases, &aliases); err != nil {
		return fmt.Errorf("failed to unmarshal aliases: %w", err)
	}

	// 별칭 중복 여부 확인함
	existing := aliases.Ko
	if aliasType == "ja" {
		existing = aliases.Ja
	}

	for _, a := range existing {
		if a == alias {
			return nil
		}
	}

	// 새 별칭 추가함
	if aliasType == "ko" {
		aliases.Ko = append(aliases.Ko, alias)
	} else {
		aliases.Ja = append(aliases.Ja, alias)
	}

	updatedJSON, err := json.Marshal(aliases)
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if err := r.gormDB.WithContext(ctx).Model(&member).Update("aliases", updatedJSON).Error; err != nil {
		return fmt.Errorf("failed to update aliases: %w", err)
	}

	return nil
}

// RemoveAlias: 멤버의 별칭을 삭제합니다.
func (r *Repository) RemoveAlias(ctx context.Context, memberID int, aliasType string, alias string) error {
	if aliasType != "ko" && aliasType != "ja" {
		return fmt.Errorf("invalid alias type: %s (must be 'ko' or 'ja')", aliasType)
	}

	var member Model
	if err := r.gormDB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		return fmt.Errorf("failed to find member: %w", err)
	}

	var aliases domain.Aliases
	if err := json.Unmarshal(member.Aliases, &aliases); err != nil {
		return fmt.Errorf("failed to unmarshal aliases: %w", err)
	}

	// 별칭 제거함
	original := aliases.Ko
	if aliasType == "ja" {
		original = aliases.Ja
	}

	updated := make([]string, 0, len(original))
	for _, a := range original {
		if a != alias {
			updated = append(updated, a)
		}
	}

	if aliasType == "ko" {
		aliases.Ko = updated
	} else {
		aliases.Ja = updated
	}

	updatedJSON, err := json.Marshal(aliases)
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if err := r.gormDB.WithContext(ctx).Model(&member).Update("aliases", updatedJSON).Error; err != nil {
		return fmt.Errorf("failed to update aliases: %w", err)
	}

	return nil
}

// SetGraduation: 멤버의 졸업 여부를 설정합니다.
func (r *Repository) SetGraduation(ctx context.Context, memberID int, isGraduated bool) error {
	var member Model
	if err := r.gormDB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		return fmt.Errorf("failed to find member: %w", err)
	}

	if err := r.gormDB.WithContext(ctx).Model(&member).Update("is_graduated", isGraduated).Error; err != nil {
		return fmt.Errorf("failed to update graduation status: %w", err)
	}

	return nil
}

// UpdateChannelID: 멤버의 YouTube 채널 ID를 업데이트합니다.
func (r *Repository) UpdateChannelID(ctx context.Context, memberID int, channelID string) error {
	var member Model
	if err := r.gormDB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		return fmt.Errorf("failed to find member: %w", err)
	}

	if err := r.gormDB.WithContext(ctx).Model(&member).Update("channel_id", channelID).Error; err != nil {
		return fmt.Errorf("failed to update channel ID: %w", err)
	}

	return nil
}

// UpdateMemberName: 멤버의 영어 이름을 업데이트합니다.
func (r *Repository) UpdateMemberName(ctx context.Context, memberID int, name string) error {
	var member Model
	if err := r.gormDB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		return fmt.Errorf("failed to find member: %w", err)
	}

	if err := r.gormDB.WithContext(ctx).Model(&member).Update("english_name", name).Error; err != nil {
		return fmt.Errorf("failed to update member name: %w", err)
	}

	return nil
}

// CreateMember: 새로운 멤버를 데이터베이스에 생성합니다.
func (r *Repository) CreateMember(ctx context.Context, member *domain.Member) error {
	aliasesJSON, err := json.Marshal(member.Aliases)
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	// domain.Member가 Slug를 노출하지 않으므로 Name을 Slug로 사용함
	slug := member.Name

	chID := member.ChannelID
	var chIDPtr *string
	if chID != "" {
		chIDPtr = &chID
	}

	var nameJaPtr *string
	if member.NameJa != "" {
		val := member.NameJa
		nameJaPtr = &val
	}

	var nameKoPtr *string
	if member.NameKo != "" {
		val := member.NameKo
		nameKoPtr = &val
	}

	m := Model{
		Slug:         slug,
		ChannelID:    chIDPtr,
		EnglishName:  member.Name,
		JapaneseName: nameJaPtr,
		KoreanName:   nameKoPtr,
		Status:       "active",
		IsGraduated:  member.IsGraduated,
		Aliases:      aliasesJSON,
	}

	if err := r.gormDB.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}

	return nil
}

// UpdatePhoto: 채널 ID로 멤버의 프로필 이미지 URL을 업데이트합니다.
func (r *Repository) UpdatePhoto(ctx context.Context, channelID string, photoURL string) error {
	now := time.Now()
	result := r.gormDB.WithContext(ctx).
		Model(&Model{}).
		Where("channel_id = ?", channelID).
		Updates(map[string]interface{}{
			"photo":            photoURL,
			"photo_updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update photo: %w", result.Error)
	}

	return nil
}

// GetPhotoByChannelID: 채널 ID로 프로필 이미지 URL을 조회합니다.
func (r *Repository) GetPhotoByChannelID(ctx context.Context, channelID string) (string, error) {
	var member Model
	err := r.gormDB.WithContext(ctx).
		Select("photo").
		Where("channel_id = ?", channelID).
		First(&member).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get photo: %w", err)
	}

	if member.Photo == nil {
		return "", nil
	}

	return *member.Photo, nil
}

// GetMembersNeedingPhotoSync: photo가 없거나 오래된 멤버 목록을 조회합니다.
// staleThreshold: 이 기간보다 오래된 photo는 재동기화 대상
func (r *Repository) GetMembersNeedingPhotoSync(ctx context.Context, staleThreshold time.Duration) ([]string, error) {
	staleTime := time.Now().Add(-staleThreshold)

	var channelIDs []string
	err := r.gormDB.WithContext(ctx).
		Model(&Model{}).
		Select("channel_id").
		Where("channel_id IS NOT NULL").
		Where("photo IS NULL OR photo_updated_at IS NULL OR photo_updated_at < ?", staleTime).
		Pluck("channel_id", &channelIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get members needing photo sync: %w", err)
	}

	return channelIDs, nil
}

// UpgradePhotoResolution: YouTube 프로필 이미지 URL을 고화질(1024x1024)로 변환합니다.
// 입력 예: https://yt3.ggpht.com/xxx=s800-c-k-c0x00ffffff-no-rj
// 출력 예: https://yt3.ggpht.com/xxx=s1024-c-k-c0x00ffffff-no-rj
func UpgradePhotoResolution(photoURL string) string {
	if photoURL == "" {
		return ""
	}

	// =s숫자 패턴을 =s1024로 변환 (최대 고화질)
	// YouTube 프로필 이미지는 최대 1024x1024까지 지원
	for _, size := range []string{"=s88", "=s240", "=s800", "=s176", "=s68"} {
		if contains(photoURL, size) {
			return replaceFirst(photoURL, size, "=s1024")
		}
	}

	// 이미 s1024이거나 패턴이 없는 경우 그대로 반환
	return photoURL
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func replaceFirst(s, old, replacement string) string {
	idx := findSubstring(s, old)
	if idx < 0 {
		return s
	}
	return s[:idx] + replacement + s[idx+len(old):]
}
