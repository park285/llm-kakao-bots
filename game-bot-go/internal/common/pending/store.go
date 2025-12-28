package pending

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/valkey-io/valkey-go"

	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

// Store: 대기 메시지 큐(Pending Message Queue)를 Redis에 저장하고 관리하는 공통 저장소 구현체
// Valkey(Redis) 클라이언트와 Lua 스크립트를 사용하여 원자적(Atomic) 연산을 수행한다.
type Store struct {
	client   valkey.Client
	logger   *slog.Logger
	config   Config
	registry *luautil.Registry
}

// NewStore: 새로운 Store 인스턴스를 생성한다.
func NewStore(client valkey.Client, logger *slog.Logger, config Config) *Store {
	registry := luautil.NewRegistry([]luautil.Script{
		{Name: luautil.ScriptPendingEnqueue, Source: enqueueLua},
		{Name: luautil.ScriptPendingDequeue, Source: dequeueLua},
	})
	if err := registry.Preload(context.Background(), client); err != nil && logger != nil {
		logger.Warn("lua_preload_failed", "component", "pending_store", "err", err)
	}
	return &Store{
		client:   client,
		logger:   logger,
		config:   config,
		registry: registry,
	}
}

// DequeueResult: Dequeue 연산의 결과를 담는 구조체
type DequeueResult struct {
	Status DequeueStatus
	// UserID: HASH 필드 또는 ZSET 멤버로 저장된 사용자 ID
	UserID string
	// Timestamp: ZSET 점수(score)로 저장된 타임스탬프 (Unix ms)
	Timestamp int64
	// RawJSON: 저장된 원본 JSON 데이터 (호출자가 구조체로 언마샬링 필요)
	RawJSON string
}

// Enqueue: 메시지를 JSON 형태로 대기열에 추가한다.
// Lua 스크립트를 사용하여 중복 체크(UserID 기준)와 용량 제한을 원자적으로 처리한다.
func (s *Store) Enqueue(ctx context.Context, chatID string, userID string, timestamp int64, jsonValue string) (EnqueueResult, error) {
	return s.enqueueInternal(ctx, chatID, userID, timestamp, jsonValue, false)
}

// EnqueueReplacingDuplicate: 메시지를 대기열에 추가하되, 동일 UserID의 중복 메시지가 있다면 최신 메시지로 교체한다.
func (s *Store) EnqueueReplacingDuplicate(ctx context.Context, chatID string, userID string, timestamp int64, jsonValue string) (EnqueueResult, error) {
	return s.enqueueInternal(ctx, chatID, userID, timestamp, jsonValue, true)
}

func (s *Store) enqueueInternal(
	ctx context.Context,
	chatID string,
	userID string,
	timestamp int64,
	jsonValue string,
	replaceOnDuplicate bool,
) (EnqueueResult, error) {
	dataKey := s.dataKey(chatID)
	orderKey := s.orderKey(chatID)

	replaceArg := "0"
	if replaceOnDuplicate {
		replaceArg = "1"
	}

	resp, err := s.registry.Exec(ctx, s.client, luautil.ScriptPendingEnqueue, []string{dataKey, orderKey}, []string{
		userID,
		jsonValue,
		strconv.FormatInt(timestamp, 10),
		strconv.Itoa(s.config.MaxQueueSize),
		strconv.Itoa(s.config.QueueTTLSeconds),
		replaceArg,
	})
	if err != nil {
		return EnqueueQueueFull, wrapRedisError("pending_enqueue_exec", err)
	}

	result, err := valkeyx.ParseLuaString(resp)
	if err != nil {
		return EnqueueQueueFull, wrapRedisError("pending_enqueue_parse", err)
	}

	switch strings.TrimSpace(result) {
	case luaStatusSuccess:
		s.logger.Debug("enqueue_success", "chat_id", chatID, "user_id", userID)
		return EnqueueSuccess, nil
	case luaStatusDuplicate:
		s.logger.Debug("enqueue_duplicate", "chat_id", chatID, "user_id", userID)
		return EnqueueDuplicate, nil
	case luaStatusQueueFull:
		s.logger.Warn("enqueue_queue_full", "chat_id", chatID)
		return EnqueueQueueFull, nil
	default:
		s.logger.Error("enqueue_unknown_result", "chat_id", chatID, "result", result)
		return EnqueueQueueFull, nil
	}
}

// Dequeue: 대기열에서 가장 오래된 메시지(stale 메시지 포함)를 꺼내어 반환한다.
// 성공 시 RawJSON 필드에 원본 데이터가 포함된다.
func (s *Store) Dequeue(ctx context.Context, chatID string) (DequeueResult, error) {
	dataKey := s.dataKey(chatID)
	orderKey := s.orderKey(chatID)

	resp, err := s.registry.Exec(ctx, s.client, luautil.ScriptPendingDequeue, []string{dataKey, orderKey}, []string{
		strconv.FormatInt(s.config.StaleThresholdMS, 10),
	})
	if err != nil {
		return DequeueResult{}, wrapRedisError("pending_dequeue_exec", err)
	}

	if respErr := resp.Error(); respErr != nil {
		if valkeyx.IsNil(respErr) {
			s.logger.Debug("dequeue_empty", "chat_id", chatID)
			return DequeueResult{Status: DequeueEmpty}, nil
		}
		return DequeueResult{}, wrapRedisError("pending_dequeue", respErr)
	}

	// Lua 반환값 파싱: 3개(Success) 또는 2개(Inconsistent)
	rawValues, err := resp.ToArray()
	if err != nil {
		return DequeueResult{}, wrapRedisError("pending_dequeue_parse_array", err)
	}

	// [Stability] INCONSISTENT 상태 처리: {"INCONSISTENT", userId}
	if len(rawValues) == 2 {
		firstVal, _ := rawValues[0].ToString()
		if firstVal == luaStatusInconsistent {
			userID, _ := rawValues[1].ToString()
			s.logger.Warn("dequeue_inconsistent", "chat_id", chatID, "user_id", userID)
			return DequeueResult{Status: DequeueInconsistent, UserID: userID}, nil
		}
	}

	if len(rawValues) != 3 {
		return DequeueResult{}, fmt.Errorf("pending dequeue unexpected array length: %d", len(rawValues))
	}

	userID, err := rawValues[0].ToString()
	if err != nil {
		return DequeueResult{}, fmt.Errorf("pending dequeue parse user id failed: %w", err)
	}
	timestamp, err := valkeyx.ParseLuaScoreToInt64(rawValues[1])
	if err != nil {
		return DequeueResult{}, fmt.Errorf("pending dequeue parse score failed: %w", err)
	}
	raw, err := rawValues[2].ToString()
	if err != nil {
		return DequeueResult{}, fmt.Errorf("pending dequeue parse json failed: %w", err)
	}

	s.logger.Debug("dequeue_success", "chat_id", chatID)
	return DequeueResult{Status: DequeueSuccess, UserID: strings.TrimSpace(userID), Timestamp: timestamp, RawJSON: raw}, nil
}

// Size: 현재 대기열에 쌓여 있는 메시지의 개수를 반환한다. (ZCard 사용)
func (s *Store) Size(ctx context.Context, chatID string) (int, error) {
	orderKey := s.orderKey(chatID)
	cmd := s.client.B().Zcard().Key(orderKey).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		if valkeyx.IsNil(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("pending size failed: %w", err)
	}
	return int(n), nil
}

// HasPending 대기 메시지 존재 여부.
func (s *Store) HasPending(ctx context.Context, chatID string) (bool, error) {
	n, err := s.Size(ctx, chatID)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetRawEntries 큐의 모든 원본 메시지 조회 (순서대로).
// 호출자가 직접 파싱하여 사용.
func (s *Store) GetRawEntries(ctx context.Context, chatID string) ([]string, error) {
	dataKey := s.dataKey(chatID)
	orderKey := s.orderKey(chatID)

	// 1. ZSET에서 순서(UserIDs) 조회
	zrangeCmd := s.client.B().Zrange().Key(orderKey).Min("0").Max("-1").Build()
	userIDs, err := s.client.Do(ctx, zrangeCmd).AsStrSlice()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("pending get order failed: %w", err)
	}
	if len(userIDs) == 0 {
		return nil, nil
	}

	// 2. HASH에서 데이터(Messages) 조회
	hmgetCmd := s.client.B().Hmget().Key(dataKey).Field(userIDs...).Build()
	rawMessages, err := s.client.Do(ctx, hmgetCmd).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("pending get data failed: %w", err)
	}

	result := make([]string, 0, len(rawMessages))
	for i, val := range rawMessages {
		if val == "" {
			continue
		}
		if i < len(userIDs) {
			result = append(result, fmt.Sprintf("0|%s|%s", userIDs[i], val))
		}
	}

	return result, nil
}

// Clear: 대기열의 데이터와 순서 정보를 모두 삭제하여 초기화한다.
func (s *Store) Clear(ctx context.Context, chatID string) error {
	dataKey := s.dataKey(chatID)
	orderKey := s.orderKey(chatID)

	cmd := s.client.B().Del().Key(dataKey, orderKey).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("pending clear failed: %w", err)
	}

	s.logger.Debug("queue_cleared", "chat_id", chatID)
	return nil
}

// ExtractJSON timestamp|userId|JSON 포맷에서 JSON 부분만 추출.
func ExtractJSON(entry string) (string, bool) {
	_, jsonPart, ok := ExtractUserIDAndJSON(entry)
	return jsonPart, ok
}

// ExtractUserIDAndJSON timestamp|userId|JSON 포맷에서 userId/JSON 추출.
func ExtractUserIDAndJSON(entry string) (string, string, bool) {
	entry = strings.TrimSpace(entry)

	// 1. timestamp 분리 (timestamp|userId|JSON)
	// timestamp가 비어있으면 유효하지 않은 포맷으로 간주
	timestamp, afterTimestamp, found := strings.Cut(entry, "|")
	if !found || timestamp == "" {
		return "", "", false
	}

	// 2. userId 분리 (userId|JSON)
	userID, jsonPart, found := strings.Cut(afterTimestamp, "|")
	if !found {
		return "", "", false
	}

	return userID, jsonPart, true
}

func wrapRedisError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", valkeyx.WrapRedisError(operation, err))
}

func (s *Store) dataKey(chatID string) string {
	return fmt.Sprintf("%s:data:{%s}", s.config.KeyPrefix, chatID)
}

func (s *Store) orderKey(chatID string) string {
	return fmt.Sprintf("%s:order:{%s}", s.config.KeyPrefix, chatID)
}
