package member

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
)

// Model represents the GORM model for members table
type Model struct {
	ID           int            `gorm:"primaryKey;column:id"`
	Slug         string         `gorm:"column:slug"`
	ChannelID    *string        `gorm:"column:channel_id"`
	EnglishName  string         `gorm:"column:english_name"`
	JapaneseName *string        `gorm:"column:japanese_name"`
	KoreanName   *string        `gorm:"column:korean_name"`
	Status       string         `gorm:"column:status"`
	IsGraduated  bool           `gorm:"column:is_graduated"`
	Aliases      datatypes.JSON `gorm:"column:aliases;type:jsonb"`
}

// TableName 는 동작을 수행한다.
func (Model) TableName() string {
	return "members"
}

// Repository 는 타입이다.
type Repository struct {
	db     *sql.DB
	gormDB *gorm.DB
	logger *zap.Logger
}

// NewMemberRepository 는 동작을 수행한다.
func NewMemberRepository(postgres *database.PostgresService, logger *zap.Logger) *Repository {
	return &Repository{
		db:     postgres.GetDB(),
		gormDB: postgres.GetGormDB(),
		logger: logger,
	}
}

// FindByChannelID retrieves member by YouTube channel ID
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

// FindByName retrieves member by English name
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

// FindByAlias searches member by any alias (Korean or Japanese)
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

// GetAllChannelIDs returns all channel IDs
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
			r.logger.Warn("Failed to scan channel ID", zap.Error(err))
			continue
		}
		channelIDs = append(channelIDs, channelID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return channelIDs, nil
}

// GetAllMembers returns all members (for initial cache warming)
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
			r.logger.Warn("Failed to scan member row", zap.Error(err))
			continue
		}

		member, err := r.scanMember(id, slug, channelID, englishName, japaneseName, koreanName, status, isGraduated, aliasesJSON)
		if err != nil {
			r.logger.Warn("Failed to parse member", zap.String("name", englishName), zap.Error(err))
			continue
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return members, nil
}

// scanMember converts DB row to domain.Member
func (r *Repository) scanMember(
	id int,
	slug string,
	channelID sql.NullString,
	englishName string,
	japaneseName sql.NullString,
	koreanName sql.NullString,
	status string,
	isGraduated bool,
	aliasesJSON []byte,
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

	return member, nil
}

// AddAlias adds an alias to a member's alias list using GORM
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

	// Check if alias already exists
	existing := aliases.Ko
	if aliasType == "ja" {
		existing = aliases.Ja
	}

	for _, a := range existing {
		if a == alias {
			return nil
		}
	}

	// Add new alias
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

// RemoveAlias removes an alias from a member's alias list using GORM
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

	// Remove alias
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

// SetGraduation updates the graduation status of a member using GORM
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

// UpdateChannelID updates the YouTube channel ID of a member using GORM
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

// CreateMember creates a new member in the database
func (r *Repository) CreateMember(ctx context.Context, member *domain.Member) error {
	aliasesJSON, err := json.Marshal(member.Aliases)
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	// Use Name as Slug since domain.Member doesn't expose Slug yet
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
