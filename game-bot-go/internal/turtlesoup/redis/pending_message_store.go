package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/pending"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// PendingMessageStore: 바다거북스프 게임의 대기 메시지(처리 전 명령어)를 Redis 큐에 저장하는 래퍼(Wrapper) 저장소
// 공통 모듈(common/pending)을 사용하여 실제 저장 로직을 위임합니다.
type PendingMessageStore struct {
	store *pending.Store
}

type pendingMessagePayload = pending.BaseMessagePayload

// NewPendingMessageStore: 새로운 PendingMessageStore 인스턴스를 생성합니다.
func NewPendingMessageStore(client valkey.Client, logger *slog.Logger) *PendingMessageStore {
	config := pending.DefaultConfig(pendingKeyPrefix())
	return &PendingMessageStore{
		store: pending.NewStore(client, logger, config),
	}
}

// ... EnqueueResult ...

// Size: 현재 대기열에 쌓여 있는 메시지의 개수를 반환합니다.
func (s *PendingMessageStore) Size(ctx context.Context, chatID string) (int, error) {
	size, err := s.store.Size(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("pending size failed: %w", err)
	}
	return size, nil
}

// EnqueueResult uses generic result type
type EnqueueResult = pending.EnqueueResult

// EnqueueSuccess: 대기열 등록 성공
const (
	EnqueueSuccess   = pending.EnqueueSuccess
	EnqueueQueueFull = pending.EnqueueQueueFull
	EnqueueDuplicate = pending.EnqueueDuplicate
)

// Enqueue: 메시지 내용을 JSON으로 직렬화하고, 공통 Store를 통해 Redis 대기열에 추가합니다.
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

	// message에서 `UserID`와 `Timestamp` 사용함
	result, err := s.store.Enqueue(ctx, chatID, message.UserID, message.Timestamp, string(jsonValue))
	if err != nil {
		return result, fmt.Errorf("pending enqueue failed: %w", err)
	}
	return result, nil
}

// DequeueResult: 대기열 조회 결과
type DequeueResult struct {
	Status  pending.DequeueStatus
	Message *tsmodel.PendingMessage
}

// DequeueSuccess: 대기열 조회 성공
const (
	DequeueSuccess   = pending.DequeueSuccess
	DequeueEmpty     = pending.DequeueEmpty
	DequeueExhausted = pending.DequeueExhausted
)

// Dequeue: 대기열에서 가장 오래된 메시지를 꺼내고(FIFO), Game PendingMessage 구조체로 변환하여 반환합니다.
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

// HasPending: 대기 중인 메시지가 있는지 확인합니다.
func (s *PendingMessageStore) HasPending(ctx context.Context, chatID string) (bool, error) {
	has, err := s.store.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("check pending failed: %w", err)
	}
	return has, nil
}

// GetQueueDetails: 대기열에 있는 모든 메시지의 요약 정보(순번, 사용자, 내용 등)를 문자열로 포맷팅하여 반환합니다.
func (s *PendingMessageStore) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	entries, err := s.store.GetRawEntries(ctx, chatID)
	if err != nil {
		return "", cerrors.RedisError{Operation: "pending_queue_details", Err: err}
	}
	if len(entries) == 0 {
		return "", nil
	}

	details := pending.FormatQueueDetails(entries, domainmodels.DisplayNameFromUser, func(jsonPart string) (pending.QueueDetailsItem, bool) {
		var payload pendingMessagePayload
		if err := json.Unmarshal([]byte(jsonPart), &payload); err != nil {
			return pending.QueueDetailsItem{}, false
		}
		return pending.QueueDetailsItem{Sender: payload.Sender, Content: payload.Content}, true
	})
	return details, nil
}

// Clear: 대기열의 모든 메시지를 삭제합니다.
func (s *PendingMessageStore) Clear(ctx context.Context, chatID string) error {
	if err := s.store.Clear(ctx, chatID); err != nil {
		return cerrors.RedisError{Operation: "pending_clear", Err: err}
	}
	return nil
}
