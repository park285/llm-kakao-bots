package redis

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// HistoryStore 는 타입이다.
type HistoryStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewHistoryStore 는 동작을 수행한다.
func NewHistoryStore(client valkey.Client, logger *slog.Logger) *HistoryStore {
	return &HistoryStore{
		client: client,
		logger: logger,
	}
}

// Get 는 동작을 수행한다.
func (s *HistoryStore) Get(ctx context.Context, chatID string) ([]qmodel.QuestionHistory, error) {
	key := historyKey(chatID)

	cmd := s.client.B().Lrange().Key(key).Start(0).Stop(-1).Build()
	rawItems, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, qerrors.RedisError{Operation: "history_get", Err: err}
	}

	history := make([]qmodel.QuestionHistory, 0, len(rawItems))
	for _, raw := range rawItems {
		var item qmodel.QuestionHistory
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			s.logger.Warn("history_item_unmarshal_failed", "chat_id", chatID, "err", err)
			continue
		}
		history = append(history, item)
	}
	return history, nil
}

// Add 는 동작을 수행한다.
func (s *HistoryStore) Add(ctx context.Context, chatID string, item qmodel.QuestionHistory) error {
	key := historyKey(chatID)

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal history item failed: %w", err)
	}

	// valkey-go는 DoMulti로 파이프라인 처리
	rpushCmd := s.client.B().Rpush().Key(key).Element(string(payload)).Build()
	expireCmd := s.client.B().Expire().Key(key).Seconds(int64(qconfig.RedisSessionTTLSeconds)).Build()

	results := s.client.DoMulti(ctx, rpushCmd, expireCmd)
	for _, r := range results {
		if err := r.Error(); err != nil {
			return qerrors.RedisError{Operation: "history_add", Err: err}
		}
	}
	return nil
}

// Clear 는 동작을 수행한다.
func (s *HistoryStore) Clear(ctx context.Context, chatID string) error {
	key := historyKey(chatID)

	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "history_clear", Err: err}
	}
	return nil
}
