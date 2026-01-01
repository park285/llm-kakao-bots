package service

import (
	"context"
	"fmt"

	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// GameSessionManager: 게임 세션의 저장, 조회, 삭제 및 락 관리를 담당합니다.
type GameSessionManager struct {
	sessionStore *tsredis.SessionStore
	lockManager  *tsredis.LockManager
}

// NewGameSessionManager: GameSessionManager 인스턴스를 생성합니다.
func NewGameSessionManager(sessionStore *tsredis.SessionStore, lockManager *tsredis.LockManager) *GameSessionManager {
	return &GameSessionManager{
		sessionStore: sessionStore,
		lockManager:  lockManager,
	}
}

// WithLock: 세션 락을 획득한 상태에서 콜백을 실행합니다.
func (m *GameSessionManager) WithLock(
	ctx context.Context,
	sessionID string,
	holderName *string,
	block func(ctx context.Context) error,
) error {
	if err := m.lockManager.WithLock(ctx, sessionID, holderName, block); err != nil {
		return fmt.Errorf("with lock failed: %w", err)
	}
	return nil
}

// WithOwnerLock: 세션 소유자 기준으로 락을 획득한 후 콜백을 실행합니다.
func (m *GameSessionManager) WithOwnerLock(
	ctx context.Context,
	sessionID string,
	block func(ctx context.Context) error,
) error {
	state, err := m.sessionStore.LoadGameState(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load game state for owner lock failed: %w", err)
	}

	var holderName *string
	if state != nil {
		holderName = &state.UserID
	}

	return m.WithLock(ctx, sessionID, holderName, block)
}

// Load: 세션 ID로 게임 상태를 조회합니다. 없으면 nil을 반환합니다.
func (m *GameSessionManager) Load(ctx context.Context, sessionID string) (*tsmodel.GameState, error) {
	state, err := m.sessionStore.LoadGameState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load game state failed: %w", err)
	}
	return state, nil
}

// LoadOrThrow: 세션 ID로 게임 상태를 조회하고, 없으면 에러를 반환합니다.
func (m *GameSessionManager) LoadOrThrow(ctx context.Context, sessionID string) (tsmodel.GameState, error) {
	state, err := m.Load(ctx, sessionID)
	if err != nil {
		return tsmodel.GameState{}, err
	}
	if state == nil {
		return tsmodel.GameState{}, tserrors.SessionNotFoundError{SessionID: sessionID}
	}
	return *state, nil
}

// Save: 게임 상태를 Redis에 저장합니다.
func (m *GameSessionManager) Save(ctx context.Context, state tsmodel.GameState) error {
	if err := m.sessionStore.SaveGameState(ctx, state); err != nil {
		return fmt.Errorf("save game state failed: %w", err)
	}
	return nil
}

// Refresh: 세션 TTL을 갱신합니다.
func (m *GameSessionManager) Refresh(ctx context.Context, sessionID string) error {
	_, err := m.sessionStore.RefreshTTL(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("refresh ttl failed: %w", err)
	}
	return nil
}

// Delete: 세션을 삭제합니다.
func (m *GameSessionManager) Delete(ctx context.Context, sessionID string) error {
	if err := m.sessionStore.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session failed: %w", err)
	}
	return nil
}

// EnsureSessionExists: 세션이 존재하는지 확인하고, 없으면 에러를 반환합니다.
func (m *GameSessionManager) EnsureSessionExists(ctx context.Context, sessionID string) error {
	exists, err := m.sessionStore.SessionExists(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session exists check failed: %w", err)
	}
	if !exists {
		return tserrors.SessionNotFoundError{SessionID: sessionID}
	}
	return nil
}
