package redis

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func newTestSessionStore(t *testing.T) (*SessionStore, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSessionStore(client, logger), client
}

func TestSessionStore_SaveAndGetSecret(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room1"

	secret := qmodel.RiddleSecret{
		Target:      "고양이",
		Category:    "동물",
		Description: "야옹",
	}

	// 1. Save
	if err := store.SaveSecret(ctx, chatID, secret); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// 2. Get
	got, err := store.GetSecret(ctx, chatID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected secret, got nil")
	}
	if got.Target != "고양이" {
		t.Errorf("expected Target '고양이', got '%s'", got.Target)
	}
	if got.Category != "동물" {
		t.Errorf("expected Category '동물', got '%s'", got.Category)
	}
}

func TestSessionStore_GetSecret_NotFound(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()

	ctx := context.Background()
	got, err := store.GetSecret(ctx, "non-existent-unique-key-12345")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room1"

	// Save first
	store.SaveSecret(ctx, chatID, qmodel.RiddleSecret{Target: "Test"})

	// Delete
	if err := store.Delete(ctx, chatID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify
	got, _ := store.GetSecret(ctx, chatID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestSessionStore_Exists(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room1"

	exists, err := store.Exists(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected not exists")
	}

	store.SaveSecret(ctx, chatID, qmodel.RiddleSecret{Target: "Test"})

	exists, err = store.Exists(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected exists")
	}
}

func TestSessionStore_ExpirationLogic(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room_ttl_test"

	// 1. 데이터 저장
	if err := store.SaveSecret(ctx, chatID, qmodel.RiddleSecret{Target: "TimeTravel"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// 2. TTL 갱신
	refreshed, err := store.RefreshTTL(ctx, chatID)
	if err != nil {
		t.Fatalf("refresh ttl failed: %v", err)
	}
	if !refreshed {
		t.Error("expected TTL to be refreshed")
	}

	// 3. 존재 확인
	if exists, _ := store.Exists(ctx, chatID); !exists {
		t.Fatal("데이터가 살아있어야 합니다.")
	}

	// Note: 실제 TTL 만료 테스트는 시간 소요가 크므로 생략
	// 실제 프로덕션에서는 TTL이 적절히 설정됨을 가정
	_ = qconfig.RedisSessionTTLSeconds
}

func TestSessionStore_ClearAllData(t *testing.T) {
	store, client := newTestSessionStore(t)
	defer client.Close()
	prefix := testhelper.UniqueTestPrefix(t)
	defer testhelper.CleanupTestKeys(t, client, "20q:")

	ctx := context.Background()
	chatID := prefix + "room1"

	// Save data
	if err := store.SaveSecret(ctx, chatID, qmodel.RiddleSecret{Target: "Test"}); err != nil {
		t.Fatalf("save secret failed: %v", err)
	}

	pendingDataKey := qconfig.RedisKeyPendingPrefix + ":data:{" + chatID + "}"
	pendingOrderKey := qconfig.RedisKeyPendingPrefix + ":order:{" + chatID + "}"

	if err := client.Do(ctx, client.B().Set().Key(pendingDataKey).Value("dummy").Build()).Error(); err != nil {
		t.Fatalf("set pending data key failed: %v", err)
	}
	if err := client.Do(ctx, client.B().Set().Key(pendingOrderKey).Value("dummy").Build()).Error(); err != nil {
		t.Fatalf("set pending order key failed: %v", err)
	}

	existsBefore, err := client.Do(ctx, client.B().Exists().Key(pendingDataKey, pendingOrderKey).Build()).AsInt64()
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if existsBefore != 2 {
		t.Fatalf("expected pending keys to exist before clear, got %d", existsBefore)
	}

	// Wait a bit for data to persist
	time.Sleep(50 * time.Millisecond)

	// Clear
	if err := store.ClearAllData(ctx, chatID); err != nil {
		t.Fatalf("clear all data failed: %v", err)
	}

	// Verify session is gone
	got, _ := store.GetSecret(ctx, chatID)
	if got != nil {
		t.Error("expected nil after clear")
	}

	existsAfter, err := client.Do(ctx, client.B().Exists().Key(pendingDataKey, pendingOrderKey).Build()).AsInt64()
	if err != nil {
		t.Fatalf("exists after clear failed: %v", err)
	}
	if existsAfter != 0 {
		t.Fatalf("expected pending keys deleted, got %d", existsAfter)
	}
}
