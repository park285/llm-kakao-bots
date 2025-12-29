package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/pending"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// EnqueueResult: pending 패키지의 EnqueueResult 재정의
type EnqueueResult = pending.EnqueueResult

// DequeueStatus: pending 패키지의 DequeueStatus 재정의
type DequeueStatus = pending.DequeueStatus

// EnqueueSuccess: 대기 메시지 등록 성공 상태 상수
const (
	EnqueueSuccess   = pending.EnqueueSuccess
	EnqueueQueueFull = pending.EnqueueQueueFull
	EnqueueDuplicate = pending.EnqueueDuplicate

	DequeueEmpty     = pending.DequeueEmpty
	DequeueExhausted = pending.DequeueExhausted
	DequeueSuccess   = pending.DequeueSuccess
)

// DequeueResult: 대기열에서 메시지를 꺼낸 결과를 담는 구조체
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

// PendingMessageStore: 스무고개 게임 대기 메시지를 Redis에 저장하는 저장소 (common/pending 래퍼)
type PendingMessageStore struct {
	client valkey.Client
	logger *slog.Logger
	store  *pending.Store
}

// NewPendingMessageStore: 새로운 PendingMessageStore 인스턴스를 생성합니다.
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

// Enqueue: 메시지를 JSON으로 변환하여 대기열에 추가하고 결과를 반환합니다.
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
		return EnqueueQueueFull, cerrors.RedisError{Operation: "pending_enqueue", Err: err}
	}
	return result, nil
}

// EnqueueReplacingDuplicate: 중복 메시지가 있을 경우 기존 메시지를 삭제하고 최신 메시지로 대체하여 대기열에 추가합니다.
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
		return EnqueueQueueFull, cerrors.RedisError{Operation: "pending_enqueue", Err: err}
	}
	return result, nil
}

// Dequeue: 대기열에서 가장 오래된 메시지를 꺼내어 반환합니다. (FIFO)
func (s *PendingMessageStore) Dequeue(ctx context.Context, chatID string) (DequeueResult, error) {
	result, err := s.store.Dequeue(ctx, chatID)
	if err != nil {
		return DequeueResult{}, cerrors.RedisError{Operation: "pending_dequeue", Err: err}
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

// DequeueBatch: 대기열에서 여러 메시지를 꺼내어 반환합니다. (FIFO)
func (s *PendingMessageStore) DequeueBatch(ctx context.Context, chatID string, limit int) ([]qmodel.PendingMessage, error) {
	results, err := s.store.DequeueBatch(ctx, chatID, limit)
	if err != nil {
		return nil, cerrors.RedisError{Operation: "pending_dequeue_batch", Err: err}
	}
	if len(results) == 0 {
		return nil, nil
	}

	messages := make([]qmodel.PendingMessage, 0, len(results))
	var parseErr error
	for _, result := range results {
		var payload pendingMessagePayload
		if err := json.Unmarshal([]byte(result.RawJSON), &payload); err != nil {
			if parseErr == nil {
				parseErr = fmt.Errorf("unmarshal dequeued pending message failed: %w", err)
			}
			s.logger.Warn("pending_message_unmarshal_failed", "chat_id", chatID, "user_id", result.UserID, "err", err)
			continue
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
		messages = append(messages, message)
	}

	if parseErr != nil && len(messages) == 0 {
		return nil, parseErr
	}
	if parseErr != nil {
		return messages, parseErr
	}
	return messages, nil
}

// Size: 현재 대기열에 쌓여있는 메시지의 개수를 반환합니다.
func (s *PendingMessageStore) Size(ctx context.Context, chatID string) (int, error) {
	n, err := s.store.Size(ctx, chatID)
	if err != nil {
		return 0, cerrors.RedisError{Operation: "pending_size", Err: err}
	}
	return n, nil
}

// HasPending: 대기 중인 메시지가 하나라도 있는지 확인합니다.
func (s *PendingMessageStore) HasPending(ctx context.Context, chatID string) (bool, error) {
	has, err := s.store.HasPending(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("check pending failed: %w", err)
	}
	return has, nil
}

// GetQueueDetails: 대기열의 메시지 목록 요약 정보를 문자열로 반환합니다. (디버깅용)
func (s *PendingMessageStore) GetQueueDetails(ctx context.Context, chatID string) (string, error) {
	entries, err := s.store.GetRawEntries(ctx, chatID)
	if err != nil {
		return "", cerrors.RedisError{Operation: "pending_queue_details", Err: err}
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

// Clear: 대기열의 모든 메시지를 삭제합니다.
func (s *PendingMessageStore) Clear(ctx context.Context, chatID string) error {
	if err := s.store.Clear(ctx, chatID); err != nil {
		return cerrors.RedisError{Operation: "pending_clear", Err: err}
	}
	return nil
}

// SetChainSkipFlag: 특정 사용자에 대해 체인 질문 스킵 플래그를 설정합니다. (체인 질문 중단 시 사용)
func (s *PendingMessageStore) SetChainSkipFlag(ctx context.Context, chatID string, userID string) error {
	key := chainSkipFlagKey(chatID, userID)
	cmd := s.client.B().Set().Key(key).Value("1").Ex(time.Duration(qconfig.RedisQueueTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "set_chain_skip_flag", Err: err}
	}
	s.logger.Debug("chain_skip_flag_set", "chat_id", chatID, "user_id", userID)
	return nil
}

// CheckAndClearChainSkipFlag: 스킵 플래그를 확인하고 만약 설정되어 있다면 true를 반환하며 플래그를 삭제합니다. (GetDel)
func (s *PendingMessageStore) CheckAndClearChainSkipFlag(ctx context.Context, chatID string, userID string) (bool, error) {
	key := chainSkipFlagKey(chatID, userID)
	cmd := s.client.B().Getdel().Key(key).Build()
	res, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if valkeyx.IsNil(err) {
			return false, nil
		}
		return false, cerrors.RedisError{Operation: "check_chain_skip_flag", Err: err}
	}
	s.logger.Debug("chain_skip_flag_cleared", "chat_id", chatID, "user_id", userID)
	return res == "1", nil
}
