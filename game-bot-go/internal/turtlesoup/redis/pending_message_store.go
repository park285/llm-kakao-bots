package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/pending"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// PendingMessageStore 는 타입이다.
type PendingMessageStore struct {
	store *pending.Store
}

type pendingMessagePayload struct {
	Content  string  `json:"content"`
	ThreadID *string `json:"threadId,omitempty"`
	Sender   *string `json:"sender,omitempty"`
}

// NewPendingMessageStore 는 동작을 수행한다.
func NewPendingMessageStore(client valkey.Client, logger *slog.Logger) *PendingMessageStore {
	config := pending.DefaultConfig("pending:turtlesoup")
	return &PendingMessageStore{
		store: pending.NewStore(client, logger, config),
	}
}

// ... EnqueueResult ...

// Size 는 대기 메시지 수를 반환한다.
func (s *PendingMessageStore) Size(ctx context.Context, chatID string) (int, error) {
	size, err := s.store.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("pending size failed: %w", err)
	}
	return size, nil
}

// EnqueueResult uses generic result type
type EnqueueResult = pending.EnqueueResult

// EnqueueSuccess 는 대기 메시지 enqueue 결과 상수 목록이다.
const (
	EnqueueSuccess   = pending.EnqueueSuccess
	EnqueueQueueFull = pending.EnqueueQueueFull
	EnqueueDuplicate = pending.EnqueueDuplicate
)

// Enqueue 는 동작을 수행한다.
func (s *PendingMessageStore) Enqueue(ctx context.Context, chatID string, message tsmodel.PendingMessage) (EnqueueResult, error) {
	payload := pendingMessagePayload{
		Content:  message.Content,
		ThreadID: message.ThreadID,
		Sender:   message.Sender,
	}
	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return EnqueueQueueFull, fmt.Errorf("marshal pending message failed: %w", err)
	}

	// Assuming message has UserID and Timestamp
	result, err := s.store.Enqueue(ctx, chatID, message.UserID, message.Timestamp, string(jsonValue))
	if err != nil {
		return result, fmt.Errorf("pending enqueue failed: %w", err)
	}
	return result, nil
}

// DequeueResult 는 타입이다.
type DequeueResult struct {
	Status  pending.DequeueStatus
	Message *tsmodel.PendingMessage
}

// DequeueSuccess 는 대기 메시지 dequeue 결과 상수 목록이다.
const (
	DequeueSuccess   = pending.DequeueSuccess
	DequeueEmpty     = pending.DequeueEmpty
	DequeueExhausted = pending.DequeueExhausted
)

// Dequeue 는 동작을 수행한다.
func (s *PendingMessageStore) Dequeue(ctx context.Context, chatID string) (DequeueResult, error) {
	result, err := s.store.Dequeue(ctx, chatID)
	if err != nil {
		return DequeueResult{}, fmt.Errorf("pending dequeue failed: %w", err)
	}

	switch result.Status {
	case pending.DequeueEmpty:
		return DequeueResult{Status: pending.DequeueEmpty}, nil
	case pending.DequeueExhausted:
		return DequeueResult{Status: pending.DequeueExhausted}, nil
	case pending.DequeueSuccess:
		var payload pendingMessagePayload
		if err := json.Unmarshal([]byte(result.RawJSON), &payload); err != nil {
			return DequeueResult{}, fmt.Errorf("unmarshal dequeued pending message failed: %w", err)
		}
		message := tsmodel.PendingMessage{
			UserID:    result.UserID,
			Content:   payload.Content,
			ThreadID:  payload.ThreadID,
			Sender:    payload.Sender,
			Timestamp: result.Timestamp,
		}
		return DequeueResult{Status: pending.DequeueSuccess, Message: &message}, nil
	default:
		return DequeueResult{}, fmt.Errorf("unknown dequeue status: %s", result.Status)
	}
}

// HasPending 는 동작을 수행한다.
func (s *PendingMessageStore) HasPending(ctx context.Context, chatID string) (bool, error) {
	has, err := s.store.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("check pending failed: %w", err)
	}
	return has, nil
}

// GetQueueDetails 는 동작을 수행한다.
func (s *PendingMessageStore) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	entries, err := s.store.GetRawEntries(ctx, chatID)
	if err != nil {
		return "", tserrors.RedisError{Operation: "pending_queue_details", Err: err}
	}
	if len(entries) == 0 {
		return "", nil
	}

	lines := make([]string, 0, len(entries))
	for idx, entry := range entries {
		entryUserID, jsonPart, ok := pending.ExtractUserIDAndJSON(entry)
		if !ok {
			continue
		}

		var payload pendingMessagePayload
		if err := json.Unmarshal([]byte(jsonPart), &payload); err != nil {
			continue
		}

		displayName := domainmodels.DisplayNameFromUser(entryUserID, payload.Sender)

		lines = append(lines, fmt.Sprintf("%d. %s - %s", idx+1, displayName, payload.Content))
	}

	return strings.Join(lines, "\n"), nil
}

// Clear 는 동작을 수행한다.
func (s *PendingMessageStore) Clear(ctx context.Context, chatID string) error {
	if err := s.store.Clear(ctx, chatID); err != nil {
		return tserrors.RedisError{Operation: "pending_clear", Err: err}
	}
	return nil
}
