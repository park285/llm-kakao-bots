package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	"github.com/valkey-io/valkey-go"
)

type MockRiddleService struct {
	SurrenderFunc func(ctx context.Context, chatID string) (string, error)
}

func (m *MockRiddleService) Surrender(ctx context.Context, chatID string) (string, error) {
	if m.SurrenderFunc != nil {
		return m.SurrenderFunc(ctx, chatID)
	}
	return "surrender_called", nil
}

func newTestAdminHandler(t *testing.T) (*AdminHandler, *MockRiddleService, *redis.SessionStore, valkey.Client) {
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sessionStore := redis.NewSessionStore(client, logger)
	msgProvider, _ := messageprovider.NewFromYAML("")

	mockService := &MockRiddleService{}
	admins := []string{"admin1", "admin2"}

	handler := NewAdminHandler(admins, mockService, sessionStore, msgProvider, logger)
	return handler, mockService, sessionStore, client
}

func TestAdminHandler_IsAdmin(t *testing.T) {
	handler, _, _, client := newTestAdminHandler(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	if !handler.IsAdmin("admin1") {
		t.Error("admin1 should be admin")
	}
	if handler.IsAdmin("user1") {
		t.Error("user1 should not be admin")
	}
}

func TestAdminHandler_ForceEnd(t *testing.T) {
	handler, mockService, store, client := newTestAdminHandler(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "room1"

	// 1. Not admin
	res, err := handler.ForceEnd(ctx, chatID, "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != qmessages.ErrorNoPermission {
		t.Errorf("expected no permission, got %s", res)
	}

	// 2. No session
	res, err = handler.ForceEnd(ctx, chatID, "admin1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != qmessages.ErrorNoSessionShort {
		t.Errorf("expected no session, got %s", res)
	}

	// 3. Success
	store.SaveSecret(ctx, chatID, qmodel.RiddleSecret{Target: "T"})
	mockService.SurrenderFunc = func(ctx context.Context, chatID string) (string, error) {
		return "Surrender OK", nil
	}

	res, err = handler.ForceEnd(ctx, chatID, "admin1")
	if err != nil {
		t.Fatalf("force end failed: %v", err)
	}
	if res != qmessages.AdminForceEndPrefix+"Surrender OK" {
		t.Errorf("unexpected result: %s", res)
	}

	// 4. Surrender Error
	mockService.SurrenderFunc = func(ctx context.Context, chatID string) (string, error) {
		return "", errors.New("boom")
	}
	_, err = handler.ForceEnd(ctx, chatID, "admin1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestAdminHandler_ClearAll(t *testing.T) {
	handler, _, _, client := newTestAdminHandler(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)
	chatID := prefix + "room1"

	// 1. Not admin
	res, err := handler.ClearAll(ctx, chatID, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if res != qmessages.ErrorNoPermission {
		t.Errorf("expected no permission, got %s", res)
	}

	// 2. Success
	res, err = handler.ClearAll(ctx, chatID, "admin1")
	if err != nil {
		t.Fatal(err)
	}
	if res != qmessages.AdminClearAllSuccess {
		t.Errorf("expected success msg, got %s", res)
	}
}
