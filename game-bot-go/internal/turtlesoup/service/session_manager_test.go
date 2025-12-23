package service

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

func TestGameSessionManager_Lock(t *testing.T) {
	client := testhelper.NewTestValkeyClient(t)
	defer client.Close()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	store := tsredis.NewSessionStore(client, logger)
	lockMgr := tsredis.NewLockManager(client, logger)
	mgr := NewGameSessionManager(store, lockMgr)

	ctx := context.Background()
	sessionID := prefix + "sess_lock"

	// Mock session for WithOwnerLock
	err := store.SaveGameState(ctx, tsmodel.GameState{SessionID: sessionID, UserID: "owner"})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Basic Lock
	t.Run("BasicLock", func(t *testing.T) {
		err := mgr.WithLock(ctx, sessionID, nil, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf(" WithLock failed: %v", err)
		}
	})

	// 2. Owner Lock
	t.Run("OwnerLock", func(t *testing.T) {
		err := mgr.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
			// Simulate work
			return nil
		})
		if err != nil {
			t.Errorf("WithOwnerLock failed: %v", err)
		}
	})

	// 3. Concurrent Lock (Verification that it doesn't crash, logic depends on redis lock impl)
	t.Run("ConcurrentLock", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			_ = mgr.WithLock(ctx, sessionID, nil, func(c context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}()

		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			_ = mgr.WithLock(ctx, sessionID, nil, func(c context.Context) error {
				return nil
			})
		}()

		wg.Wait()
	})
}

func TestGameSessionManager_CRUD(t *testing.T) {
	client := testhelper.NewTestValkeyClient(t)
	defer client.Close()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, tsconfig.RedisKeyPrefix+":")

	store := tsredis.NewSessionStore(client, logger)
	lockMgr := tsredis.NewLockManager(client, logger)
	mgr := NewGameSessionManager(store, lockMgr)

	ctx := context.Background()
	sessionID := prefix + "sess_crud"

	// Save
	state := tsmodel.GameState{SessionID: sessionID, UserID: "u1"}
	if err := mgr.Save(ctx, state); err != nil {
		t.Fatal(err)
	}

	// Load
	loaded, err := mgr.Load(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.UserID != "u1" {
		t.Error("Load failed")
	}

	// EnsureSessionExists
	if err := mgr.EnsureSessionExists(ctx, sessionID); err != nil {
		t.Error("EnsureSessionExists failed for existing session")
	}

	nonExistentID := prefix + "non_existent"
	if err := mgr.EnsureSessionExists(ctx, nonExistentID); err == nil {
		t.Error("EnsureSessionExists should fail for non-existent session")
	}

	// Delete
	if err := mgr.Delete(ctx, sessionID); err != nil {
		t.Fatal(err)
	}
	loaded, _ = mgr.Load(ctx, sessionID)
	if loaded != nil {
		t.Error("Delete failed")
	}

	// LoadOrThrow
	_, err = mgr.LoadOrThrow(ctx, sessionID) // sessionID was deleted above, should fail
	if err == nil {
		t.Error("LoadOrThrow should fail for deleted session")
	}

	// Restore session for Refresh
	_ = mgr.Save(ctx, state)

	// Refresh
	if err := mgr.Refresh(ctx, sessionID); err != nil {
		t.Errorf("Refresh failed: %v", err)
	}

	// LoadOrThrow Success
	l, err := mgr.LoadOrThrow(ctx, sessionID)
	if err != nil {
		t.Errorf("LoadOrThrow failed: %v", err)
	}
	if l.UserID != "u1" {
		t.Error("LoadOrThrow data mismatch")
	}
}
