package redis

import (
	"context"
	"log/slog"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// SessionStore: 바다거북스프 게임의 진행 상태(퍼즐 정보, 참여자, 이력 등)를 Redis에 JSON 형태로 저장하고 관리하는 저장소
type SessionStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewSessionStore: 새로운 SessionStore 인스턴스를 생성합니다.
func NewSessionStore(client valkey.Client, logger *slog.Logger) *SessionStore {
	return &SessionStore{
		client: client,
		logger: logger,
	}
}

// SaveGameState: 현재 게임 상태 객체(GameState)를 JSON으로 직렬화하여 Redis에 저장합니다. (TTL 갱신 포함)
func (s *SessionStore) SaveGameState(ctx context.Context, state tsmodel.GameState) error {
	key := sessionKey(state.SessionID)

	payload, err := json.Marshal(state)
	if err != nil {
		return cerrors.RedisError{Operation: "marshal_game_state", Err: err}
	}

	ttl := time.Duration(tsconfig.RedisSessionTTLSeconds) * time.Second
	if err := valkeyx.SetStringEX(ctx, s.client, key, string(payload), ttl); err != nil {
		return cerrors.RedisError{Operation: "save_game_state", Err: err}
	}

	s.logger.Debug("game_state_saved", "session_id", state.SessionID)
	return nil
}

// LoadGameState: Redis에 저장된 JSON 데이터를 조회하여 GameState 객체로 역직렬화합니다.
// 데이터가 없거나 만료된 경우 nil을 반환합니다.
func (s *SessionStore) LoadGameState(ctx context.Context, sessionID string) (*tsmodel.GameState, error) {
	key := sessionKey(sessionID)

	raw, ok, err := valkeyx.GetBytes(ctx, s.client, key)
	if err != nil {
		return nil, cerrors.RedisError{Operation: "load_game_state", Err: err}
	}
	if !ok {
		return nil, nil
	}

	var state tsmodel.GameState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, cerrors.RedisError{Operation: "unmarshal_game_state", Err: err}
	}
	return &state, nil
}

// DeleteSession: 게임 종료 시 해당 게임 세션의 데이터를 Redis에서 영구 삭제합니다.
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionKey(sessionID)

	if err := valkeyx.DeleteKeys(ctx, s.client, key); err != nil {
		return cerrors.RedisError{Operation: "delete_session", Err: err}
	}
	s.logger.Debug("session_deleted", "session_id", sessionID)
	return nil
}

// SessionExists: 세션이 존재하는지 확인합니다.
func (s *SessionStore) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	key := sessionKey(sessionID)

	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, cerrors.RedisError{Operation: "session_exists", Err: err}
	}
	return n > 0, nil
}

// RefreshTTL: 세션의 TTL을 연장합니다.
func (s *SessionStore) RefreshTTL(ctx context.Context, sessionID string) (bool, error) {
	key := sessionKey(sessionID)

	cmd := s.client.B().Expire().Key(key).Seconds(int64(tsconfig.RedisSessionTTLSeconds)).Build()
	ok, err := s.client.Do(ctx, cmd).AsBool()
	if err != nil {
		return false, cerrors.RedisError{Operation: "refresh_ttl", Err: err}
	}
	return ok, nil
}
