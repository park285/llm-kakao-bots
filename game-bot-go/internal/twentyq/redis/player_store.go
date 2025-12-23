package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// PlayerStore 는 타입이다.
type PlayerStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewPlayerStore 는 동작을 수행한다.
func NewPlayerStore(client valkey.Client, logger *slog.Logger) *PlayerStore {
	return &PlayerStore{
		client: client,
		logger: logger,
	}
}

// Add 는 동작을 수행한다.
func (s *PlayerStore) Add(ctx context.Context, chatID string, userID string, sender string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, fmt.Errorf("invalid user id")
	}
	sender = strings.TrimSpace(sender)

	current, err := s.GetAll(ctx, chatID)
	if err != nil {
		return false, err
	}

	isNew := true
	updated := make([]qmodel.PlayerInfo, 0, len(current)+1)
	for _, p := range current {
		if p.UserID == userID {
			isNew = false
			if sender != "" {
				p.Sender = sender
			}
		}
		updated = append(updated, p)
	}

	if isNew {
		updated = append(updated, qmodel.PlayerInfo{UserID: userID, Sender: sender})
	}

	payload, err := json.Marshal(updated)
	if err != nil {
		return false, fmt.Errorf("marshal player list failed: %w", err)
	}

	key := playersKey(chatID)
	cmd := s.client.B().Set().Key(key).Value(string(payload)).Ex(time.Duration(qconfig.RedisSessionTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return false, qerrors.RedisError{Operation: "players_save", Err: err}
	}

	s.logger.Debug("player_saved", "chat_id", chatID, "user_id", userID, "is_new", isNew, "total", len(updated))
	return isNew, nil
}

// GetAll 는 동작을 수행한다.
func (s *PlayerStore) GetAll(ctx context.Context, chatID string) ([]qmodel.PlayerInfo, error) {
	key := playersKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, qerrors.RedisError{Operation: "players_get", Err: err}
	}

	var out []qmodel.PlayerInfo
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, qerrors.RedisError{Operation: "players_unmarshal", Err: err}
	}
	return out, nil
}

// Clear 는 동작을 수행한다.
func (s *PlayerStore) Clear(ctx context.Context, chatID string) error {
	key := playersKey(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "players_clear", Err: err}
	}
	return nil
}
