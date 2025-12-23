package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"

	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
)

// TopicHistoryStore 는 타입이다.
type TopicHistoryStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewTopicHistoryStore 는 동작을 수행한다.
func NewTopicHistoryStore(client valkey.Client, logger *slog.Logger) *TopicHistoryStore {
	return &TopicHistoryStore{
		client: client,
		logger: logger,
	}
}

// GetRecent 는 동작을 수행한다.
func (s *TopicHistoryStore) GetRecent(ctx context.Context, chatID string, limit int) ([]string, error) {
	key := topicsGlobalKey(chatID)
	return s.getRecentByKey(ctx, key, limit, "topics_get_recent_global")
}

// GetRecentByCategory 는 동작을 수행한다.
func (s *TopicHistoryStore) GetRecentByCategory(ctx context.Context, chatID string, category string, limit int) ([]string, error) {
	key := topicsCategoryKey(chatID, category)
	return s.getRecentByKey(ctx, key, limit, "topics_get_recent_category")
}

// GetBannedTopics 는 동작을 수행한다.
func (s *TopicHistoryStore) GetBannedTopics(ctx context.Context, chatID string, category *string, limit int, allCategories []string) ([]string, error) {
	globalRecent, err := s.GetRecent(ctx, chatID, limit)
	if err != nil {
		return nil, err
	}

	merged := make([]string, 0, len(globalRecent))
	seen := make(map[string]struct{}, len(globalRecent))
	for _, topic := range globalRecent {
		normalized := strings.ToLower(strings.TrimSpace(topic))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		merged = append(merged, topic)
	}

	var categories []string
	if category != nil && strings.TrimSpace(*category) != "" {
		categories = []string{strings.TrimSpace(*category)}
	} else {
		categories = allCategories
	}

	for _, cat := range categories {
		cat = strings.TrimSpace(cat)
		if cat == "" {
			continue
		}
		recents, err := s.GetRecentByCategory(ctx, chatID, cat, limit)
		if err != nil {
			return nil, err
		}
		for _, topic := range recents {
			normalized := strings.ToLower(strings.TrimSpace(topic))
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			merged = append(merged, topic)
		}
	}

	return merged, nil
}

// AddCompletedTopic 는 동작을 수행한다.
func (s *TopicHistoryStore) AddCompletedTopic(ctx context.Context, chatID string, category string, topic string, limit int) error {
	category = strings.TrimSpace(category)
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil
	}

	globalKey := topicsGlobalKey(chatID)
	categoryKey := topicsCategoryKey(chatID, category)

	if err := s.addWithLimit(ctx, globalKey, topic, limit); err != nil {
		return err
	}
	if category != "" {
		if err := s.addWithLimit(ctx, categoryKey, topic, limit); err != nil {
			return err
		}
	}

	s.logger.Debug("topic_added", "chat_id", chatID, "category", category, "topic", topic)
	return nil
}

func (s *TopicHistoryStore) getRecentByKey(ctx context.Context, key string, limit int, op string) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	cmd := s.client.B().Lrange().Key(key).Start(0).Stop(int64(limit - 1)).Build()
	values, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, qerrors.RedisError{Operation: op, Err: err}
	}
	return values, nil
}

func (s *TopicHistoryStore) addWithLimit(ctx context.Context, key string, topic string, limit int) error {
	if limit <= 0 {
		limit = 20
	}

	lpushCmd := s.client.B().Lpush().Key(key).Element(topic).Build()
	ltrimCmd := s.client.B().Ltrim().Key(key).Start(0).Stop(int64(limit - 1)).Build()
	expireCmd := s.client.B().Expire().Key(key).Seconds(int64(12 * time.Hour / time.Second)).Build()

	results := s.client.DoMulti(ctx, lpushCmd, ltrimCmd, expireCmd)
	for _, r := range results {
		if err := r.Error(); err != nil {
			return fmt.Errorf("topic add pipeline failed: %w", qerrors.RedisError{Operation: "topics_add", Err: err})
		}
	}
	return nil
}
