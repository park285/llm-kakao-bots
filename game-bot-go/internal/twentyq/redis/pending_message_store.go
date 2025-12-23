package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/pending"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// EnqueueResult 는 공통 타입 re-export다.
type EnqueueResult = pending.EnqueueResult

// DequeueStatus 는 공통 타입 re-export다.
type DequeueStatus = pending.DequeueStatus

// EnqueueSuccess 는 대기 메시지 처리 상수 목록이다.
const (
	EnqueueSuccess   = pending.EnqueueSuccess
	EnqueueQueueFull = pending.EnqueueQueueFull
	EnqueueDuplicate = pending.EnqueueDuplicate

	DequeueEmpty     = pending.DequeueEmpty
	DequeueExhausted = pending.DequeueExhausted
	DequeueSuccess   = pending.DequeueSuccess
)

// DequeueResult 는 타입이다.
type DequeueResult struct {
	Status  DequeueStatus
	Message *qmodel.PendingMessage
}

type pendingMessagePayload struct {
	Content  string  `json:"content"`
	ThreadID *string `json:"threadId,omitempty"`
	Sender   *string `json:"sender,omitempty"`
	// 체인 질문 배치 처리용
	IsChainBatch   bool     `json:"isChainBatch,omitempty"`
	BatchQuestions []string `json:"batchQuestions,omitempty"`
}

// PendingMessageStore 는 타입이다.
type PendingMessageStore struct {
	client valkey.Client
	logger *slog.Logger
	store  *pending.Store
}

// NewPendingMessageStore 는 동작을 수행한다.
func NewPendingMessageStore(client valkey.Client, logger *slog.Logger) *PendingMessageStore {
	config := pending.Config{
		KeyPrefix:            qconfig.RedisKeyPendingPrefix,
		MaxQueueSize:         qconfig.RedisMaxQueueSize,
		QueueTTLSeconds:      qconfig.RedisQueueTTLSeconds,
		StaleThresholdMS:     qconfig.RedisStaleThresholdMS,
		MaxDequeueIterations: qconfig.QueueMaxDequeueIterations,
	}

	return &PendingMessageStore{
		client: client,
		logger: logger,
		store:  pending.NewStore(client, logger.With("component", "pending_store"), config),
	}
}

// Enqueue 는 동작을 수행한다.
func (s *PendingMessageStore) Enqueue(ctx context.Context, chatID string, message qmodel.PendingMessage) (EnqueueResult, error) {
	payload := pendingMessagePayload{
		Content:        message.Content,
		ThreadID:       message.ThreadID,
		Sender:         message.Sender,
		IsChainBatch:   message.IsChainBatch,
		BatchQuestions: message.BatchQuestions,
	}
	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return EnqueueQueueFull, fmt.Errorf("marshal pending message failed: %w", err)
	}

	result, err := s.store.Enqueue(ctx, chatID, message.UserID, message.Timestamp, string(jsonValue))
	if err != nil {
		return EnqueueQueueFull, qerrors.RedisError{Operation: "pending_enqueue", Err: err}
	}
	return result, nil
}

// EnqueueReplacingDuplicate 는 동작을 수행한다.
func (s *PendingMessageStore) EnqueueReplacingDuplicate(ctx context.Context, chatID string, message qmodel.PendingMessage) (EnqueueResult, error) {
	payload := pendingMessagePayload{
		Content:        message.Content,
		ThreadID:       message.ThreadID,
		Sender:         message.Sender,
		IsChainBatch:   message.IsChainBatch,
		BatchQuestions: message.BatchQuestions,
	}
	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return EnqueueQueueFull, fmt.Errorf("marshal pending message failed: %w", err)
	}

	result, err := s.store.EnqueueReplacingDuplicate(ctx, chatID, message.UserID, message.Timestamp, string(jsonValue))
	if err != nil {
		return EnqueueQueueFull, qerrors.RedisError{Operation: "pending_enqueue", Err: err}
	}
	return result, nil
}

// Dequeue 는 동작을 수행한다.
func (s *PendingMessageStore) Dequeue(ctx context.Context, chatID string) (DequeueResult, error) {
	result, err := s.store.Dequeue(ctx, chatID)
	if err != nil {
		return DequeueResult{}, qerrors.RedisError{Operation: "pending_dequeue", Err: err}
	}

	switch result.Status {
	case DequeueEmpty:
		return DequeueResult{Status: DequeueEmpty}, nil
	case DequeueExhausted:
		return DequeueResult{Status: DequeueExhausted}, nil
	case DequeueSuccess:
		var payload pendingMessagePayload
		if err := json.Unmarshal([]byte(result.RawJSON), &payload); err != nil {
			return DequeueResult{}, fmt.Errorf("unmarshal dequeued pending message failed: %w", err)
		}
		message := qmodel.PendingMessage{
			UserID:         result.UserID,
			Content:        payload.Content,
			ThreadID:       payload.ThreadID,
			Sender:         payload.Sender,
			Timestamp:      result.Timestamp,
			IsChainBatch:   payload.IsChainBatch,
			BatchQuestions: payload.BatchQuestions,
		}
		return DequeueResult{Status: DequeueSuccess, Message: &message}, nil
	default:
		return DequeueResult{}, fmt.Errorf("unknown dequeue status: %d", result.Status)
	}
}

// Size 는 동작을 수행한다.
func (s *PendingMessageStore) Size(ctx context.Context, chatID string) (int, error) {
	n, err := s.store.Size(ctx, chatID)
	if err != nil {
		return 0, qerrors.RedisError{Operation: "pending_size", Err: err}
	}
	return n, nil
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
		return "", qerrors.RedisError{Operation: "pending_queue_details", Err: err}
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

		// 체인 메시지 처리: batchQuestions 사용
		content := payload.Content
		if payload.IsChainBatch && len(payload.BatchQuestions) > 0 {
			content = strings.Join(payload.BatchQuestions, ", ")
		}

		lines = append(lines, fmt.Sprintf("%d. %s - %s", idx+1, displayName, content))
	}

	return strings.Join(lines, "\n"), nil
}

// Clear 는 동작을 수행한다.
func (s *PendingMessageStore) Clear(ctx context.Context, chatID string) error {
	if err := s.store.Clear(ctx, chatID); err != nil {
		return qerrors.RedisError{Operation: "pending_clear", Err: err}
	}
	return nil
}

// SetChainSkipFlag 체인 질문 스킵 플래그 설정.
func (s *PendingMessageStore) SetChainSkipFlag(ctx context.Context, chatID string, userID string) error {
	key := chainSkipFlagKey(chatID, userID)
	cmd := s.client.B().Set().Key(key).Value("1").Ex(time.Duration(qconfig.RedisQueueTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "set_chain_skip_flag", Err: err}
	}
	s.logger.Debug("chain_skip_flag_set", "chat_id", chatID, "user_id", userID)
	return nil
}

// CheckAndClearChainSkipFlag 체인 질문 스킵 플래그 확인 및 삭제.
func (s *PendingMessageStore) CheckAndClearChainSkipFlag(ctx context.Context, chatID string, userID string) (bool, error) {
	key := chainSkipFlagKey(chatID, userID)
	cmd := s.client.B().Getdel().Key(key).Build()
	res, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if valkeyx.IsNil(err) {
			return false, nil
		}
		return false, qerrors.RedisError{Operation: "check_chain_skip_flag", Err: err}
	}
	s.logger.Debug("chain_skip_flag_cleared", "chat_id", chatID, "user_id", userID)
	return res == "1", nil
}
