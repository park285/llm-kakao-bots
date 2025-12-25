package pending

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
)

func newTestStore(t *testing.T) (*Store, valkey.Client) {
	t.Helper()
	client := testhelper.NewTestValkeyClient(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prefix := testhelper.UniqueTestPrefix(t)
	config := DefaultConfig(prefix + "pending")
	// 테스트를 위해 제한 설정 조정
	config.MaxQueueSize = 3
	config.QueueTTLSeconds = 60
	config.StaleThresholdMS = 1000 // 1초

	testhelper.CleanupTestKeys(t, client, config.KeyPrefix+":")

	store := NewStore(client, logger, config)
	return store, client
}

func TestStore_Enqueue(t *testing.T) {
	store, client := newTestStore(t)
	defer client.Close()

	ctx := context.Background()
	chatID := "room1"

	// 1. 첫 번째 메시지 Enqueue
	res, err := store.Enqueue(ctx, chatID, "user1", time.Now().UnixMilli(), `{"msg":"1"}`)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if res != EnqueueSuccess {
		t.Errorf("expected EnqueueSuccess, got %v", res)
	}

	// 2. 중복 UserID Enqueue (같은 방, 다른 메시지여도 UserID 기준 중복 체크)
	res, err = store.Enqueue(ctx, chatID, "user1", time.Now().UnixMilli(), `{"msg":"2"}`)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if res != EnqueueDuplicate {
		t.Errorf("expected EnqueueDuplicate, got %v", res)
	}

	// 3. 다른 UserID Enqueue
	res, err = store.Enqueue(ctx, chatID, "user2", time.Now().UnixMilli(), `{"msg":"3"}`)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if res != EnqueueSuccess {
		t.Errorf("expected EnqueueSuccess, got %v", res)
	}

	// 4. 큐 사이즈 확인
	size, err := store.Size(ctx, chatID)
	if err != nil {
		t.Fatalf("size failed: %v", err)
	}
	if size != 2 {
		t.Errorf("expected size 2, got %d", size)
	}
}

func TestStore_EnqueueReplacingDuplicate(t *testing.T) {
	store, client := newTestStore(t)
	defer client.Close()

	ctx := context.Background()
	chatID := "room1"

	ts := time.Now().UnixMilli()

	res, err := store.Enqueue(ctx, chatID, "user1", ts, `{"msg":"1"}`)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if res != EnqueueSuccess {
		t.Errorf("expected EnqueueSuccess, got %v", res)
	}

	// 동일 유저의 기존 대기 메시지를 교체
	res, err = store.EnqueueReplacingDuplicate(ctx, chatID, "user1", ts+1, `{"msg":"2"}`)
	if err != nil {
		t.Fatalf("enqueue replace failed: %v", err)
	}
	if res != EnqueueSuccess {
		t.Errorf("expected EnqueueSuccess, got %v", res)
	}

	size, err := store.Size(ctx, chatID)
	if err != nil {
		t.Fatalf("size failed: %v", err)
	}
	if size != 1 {
		t.Errorf("expected size 1, got %d", size)
	}

	dequeued, err := store.Dequeue(ctx, chatID)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if dequeued.Status != DequeueSuccess {
		t.Errorf("expected DequeueSuccess, got %v", dequeued.Status)
	}
	if dequeued.RawJSON != `{"msg":"2"}` {
		t.Errorf("expected json '{\"msg\":\"2\"}', got '%s'", dequeued.RawJSON)
	}
}

func TestStore_Enqueue_QueueFull(t *testing.T) {
	store, client := newTestStore(t)
	defer client.Close()

	// Config에서 MaxQueueSize = 3

	ctx := context.Background()
	chatID := "room1"

	// 3명 채우기
	store.Enqueue(ctx, chatID, "user1", time.Now().UnixMilli(), "{}")
	store.Enqueue(ctx, chatID, "user2", time.Now().UnixMilli(), "{}")
	store.Enqueue(ctx, chatID, "user3", time.Now().UnixMilli(), "{}")

	// 4번째 시도
	res, err := store.Enqueue(ctx, chatID, "user4", time.Now().UnixMilli(), "{}")
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if res != EnqueueQueueFull {
		t.Errorf("expected EnqueueQueueFull, got %v", res)
	}
}

func TestStore_Dequeue(t *testing.T) {
	store, client := newTestStore(t)
	defer client.Close()

	ctx := context.Background()
	chatID := "room1"

	// 빈 큐 Dequeue
	res, err := store.Dequeue(ctx, chatID)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if res.Status != DequeueEmpty {
		t.Errorf("expected DequeueEmpty, got %v", res.Status)
	}

	// 데이터 넣기
	ts := time.Now().UnixMilli()
	store.Enqueue(ctx, chatID, "user1", ts, `{"id":1}`)
	store.Enqueue(ctx, chatID, "user2", ts+100, `{"id":2}`)

	// 첫 번째 Dequeue (user1)
	res, err = store.Dequeue(ctx, chatID)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if res.Status != DequeueSuccess {
		t.Errorf("expected DequeueSuccess, got %v", res.Status)
	}
	if res.RawJSON != `{"id":1}` {
		t.Errorf("expected json '{\"id\":1}', got '%s'", res.RawJSON)
	}

	// 두 번째 Dequeue (user2)
	res, err = store.Dequeue(ctx, chatID)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if res.Status != DequeueSuccess {
		t.Errorf("expected DequeueSuccess, got %v", res.Status)
	}
	if res.RawJSON != `{"id":2}` {
		t.Errorf("expected json '{\"id\":2}', got '%s'", res.RawJSON)
	}

	// 다시 빈 큐
	res, err = store.Dequeue(ctx, chatID)
	if res.Status != DequeueEmpty {
		t.Errorf("expected DequeueEmpty, got %v", res.Status)
	}
}

func TestStore_Clear(t *testing.T) {
	store, client := newTestStore(t)
	defer client.Close()

	ctx := context.Background()
	chatID := "room1"

	store.Enqueue(ctx, chatID, "user1", time.Now().UnixMilli(), "{}")

	exists, _ := store.HasPending(ctx, chatID)
	if !exists {
		t.Fatal("expected pending messages")
	}

	if err := store.Clear(ctx, chatID); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	exists, _ = store.HasPending(ctx, chatID)
	if exists {
		t.Error("expected no pending messages after clear")
	}

	size, _ := store.Size(ctx, chatID)
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}
}

func TestStore_GetRawEntries(t *testing.T) {
	store, client := newTestStore(t)
	defer client.Close()

	ctx := context.Background()
	chatID := "room1"
	ts := int64(1234567890000)

	store.Enqueue(ctx, chatID, "user1", ts, `{"msg":1}`)
	store.Enqueue(ctx, chatID, "user2", ts+1, `{"msg":2}`)

	entries, err := store.GetRawEntries(ctx, chatID)
	if err != nil {
		t.Fatalf("get entries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}
