package redis

import (
	"context"
	"log/slog"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// SessionStore 는 타입이다.
type SessionStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewSessionStore 는 동작을 수행한다.
func NewSessionStore(client valkey.Client, logger *slog.Logger) *SessionStore {
	return &SessionStore{
		client: client,
		logger: logger,
	}
}

// SaveGameState 는 동작을 수행한다.
func (s *SessionStore) SaveGameState(ctx context.Context, state tsmodel.GameState) error {
	key := sessionKey(state.SessionID)

	payload, err := json.Marshal(state)
	if err != nil {
		return tserrors.RedisError{Operation: "marshal_game_state", Err: err}
	}

	cmd := s.client.B().Set().Key(key).Value(string(payload)).Ex(time.Duration(tsconfig.RedisSessionTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return tserrors.RedisError{Operation: "save_game_state", Err: err}
	}

	s.logger.Debug("game_state_saved", "session_id", state.SessionID)
	return nil
}

// LoadGameState 는 동작을 수행한다.
func (s *SessionStore) LoadGameState(ctx context.Context, sessionID string) (*tsmodel.GameState, error) {
	key := sessionKey(sessionID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, tserrors.RedisError{Operation: "load_game_state", Err: err}
	}

	var state tsmodel.GameState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, tserrors.RedisError{Operation: "unmarshal_game_state", Err: err}
	}
	return &state, nil
}

// DeleteSession 는 동작을 수행한다.
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionKey(sessionID)

	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return tserrors.RedisError{Operation: "delete_session", Err: err}
	}
	s.logger.Debug("session_deleted", "session_id", sessionID)
	return nil
}

// SessionExists 는 동작을 수행한다.
func (s *SessionStore) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	key := sessionKey(sessionID)

	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, tserrors.RedisError{Operation: "session_exists", Err: err}
	}
	return n > 0, nil
}

// RefreshTTL 는 동작을 수행한다.
func (s *SessionStore) RefreshTTL(ctx context.Context, sessionID string) (bool, error) {
	key := sessionKey(sessionID)

	cmd := s.client.B().Expire().Key(key).Seconds(int64(tsconfig.RedisSessionTTLSeconds)).Build()
	ok, err := s.client.Do(ctx, cmd).AsBool()
	if err != nil {
		return false, tserrors.RedisError{Operation: "refresh_ttl", Err: err}
	}
	return ok, nil
}
