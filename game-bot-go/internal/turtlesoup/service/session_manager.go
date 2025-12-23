package service

import (
	"context"
	"fmt"

	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// GameSessionManager 는 타입이다.
type GameSessionManager struct {
	sessionStore *tsredis.SessionStore
	lockManager  *tsredis.LockManager
}

// NewGameSessionManager 는 동작을 수행한다.
func NewGameSessionManager(sessionStore *tsredis.SessionStore, lockManager *tsredis.LockManager) *GameSessionManager {
	return &GameSessionManager{
		sessionStore: sessionStore,
		lockManager:  lockManager,
	}
}

// WithLock 는 동작을 수행한다.
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

// WithOwnerLock 는 동작을 수행한다.
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

// Load 는 동작을 수행한다.
func (m *GameSessionManager) Load(ctx context.Context, sessionID string) (*tsmodel.GameState, error) {
	state, err := m.sessionStore.LoadGameState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load game state failed: %w", err)
	}
	return state, nil
}

// LoadOrThrow 는 동작을 수행한다.
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

// Save 는 동작을 수행한다.
func (m *GameSessionManager) Save(ctx context.Context, state tsmodel.GameState) error {
	if err := m.sessionStore.SaveGameState(ctx, state); err != nil {
		return fmt.Errorf("save game state failed: %w", err)
	}
	return nil
}

// Refresh 는 동작을 수행한다.
func (m *GameSessionManager) Refresh(ctx context.Context, sessionID string) error {
	_, err := m.sessionStore.RefreshTTL(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("refresh ttl failed: %w", err)
	}
	return nil
}

// Delete 는 동작을 수행한다.
func (m *GameSessionManager) Delete(ctx context.Context, sessionID string) error {
	if err := m.sessionStore.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session failed: %w", err)
	}
	return nil
}

// EnsureSessionExists 는 동작을 수행한다.
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
