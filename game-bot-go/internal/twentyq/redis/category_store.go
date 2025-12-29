package redis

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// CategoryStore: 현재 게임의 카테고리 정보를 관리하는 저장소
type CategoryStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewCategoryStore: 새로운 CategoryStore 인스턴스를 생성합니다.
func NewCategoryStore(client valkey.Client, logger *slog.Logger) *CategoryStore {
	return &CategoryStore{
		client: client,
		logger: logger,
	}
}

// Save: 현재 게임의 카테고리를 저장합니다.
func (s *CategoryStore) Save(ctx context.Context, chatID string, category *string) error {
	key := categoryKey(chatID)

	if category == nil || strings.TrimSpace(*category) == "" {
		cmd := s.client.B().Del().Key(key).Build()
		if err := s.client.Do(ctx, cmd).Error(); err != nil {
			return cerrors.RedisError{Operation: "category_delete", Err: err}
		}
		return nil
	}

	value := strings.TrimSpace(*category)
	cmd := s.client.B().Set().Key(key).Value(value).Ex(time.Duration(qconfig.RedisSessionTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "category_save", Err: err}
	}
	s.logger.Debug("category_saved", "chat_id", chatID, "category", value)
	return nil
}

// Get: 현재 설정된 카테고리를 조회합니다.
func (s *CategoryStore) Get(ctx context.Context, chatID string) (*string, error) {
	key := categoryKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	value, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, cerrors.RedisError{Operation: "category_get", Err: err}
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	return &value, nil
}
