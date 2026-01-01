package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/gamesession"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// SessionStore: 바다거북스프 게임의 진행 상태(퍼즐 정보, 참여자, 이력 등)를 Redis에 JSON 형태로 저장하고 관리하는 저장소
// 공통 gamesession.Store를 내부적으로 사용하여 CRUD 로직을 위임합니다.
type SessionStore struct {
	base *gamesession.Store[tsmodel.GameState]
}

// NewSessionStore: 새로운 SessionStore 인스턴스를 생성합니다.
func NewSessionStore(client valkey.Client, logger *slog.Logger) *SessionStore {
	return &SessionStore{
		base: gamesession.NewStore[tsmodel.GameState](client, logger, gamesession.Config{
			KeyFunc: sessionKey,
			TTL:     time.Duration(tsconfig.RedisSessionTTLSeconds) * time.Second,
		}),
	}
}

// SaveGameState: 현재 게임 상태 객체(GameState)를 JSON으로 직렬화하여 Redis에 저장합니다. (TTL 갱신 포함)
func (s *SessionStore) SaveGameState(ctx context.Context, state tsmodel.GameState) error {
	if err := s.base.Save(ctx, state.SessionID, state); err != nil {
		return fmt.Errorf("save game state: %w", err)
	}
	return nil
}

// LoadGameState: Redis에 저장된 JSON 데이터를 조회하여 GameState 객체로 역직렬화합니다.
// 데이터가 없거나 만료된 경우 nil을 반환합니다.
func (s *SessionStore) LoadGameState(ctx context.Context, sessionID string) (*tsmodel.GameState, error) {
	state, err := s.base.Load(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load game state: %w", err)
	}
	return state, nil
}

// DeleteSession: 게임 종료 시 해당 게임 세션의 데이터를 Redis에서 영구 삭제합니다.
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	if err := s.base.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// SessionExists: 세션이 존재하는지 확인합니다.
func (s *SessionStore) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	exists, err := s.base.Exists(ctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("session exists: %w", err)
	}
	return exists, nil
}

// RefreshTTL: 세션의 TTL을 연장합니다.
func (s *SessionStore) RefreshTTL(ctx context.Context, sessionID string) (bool, error) {
	ok, err := s.base.RefreshTTL(ctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("refresh ttl: %w", err)
	}
	return ok, nil
}
