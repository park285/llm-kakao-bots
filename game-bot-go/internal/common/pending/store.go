package pending

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

// Store: 대기 메시지 큐(Pending Message Queue)를 Redis에 저장하고 관리하는 공통 저장소 구현체
// Valkey(Redis) 클라이언트와 Lua 스크립트를 사용하여 원자적(Atomic) 연산을 수행한다.
type Store struct {
	client valkey.Client
	logger *slog.Logger
	config Config

	enqueueSHA string
	dequeueSHA string
}

// NewStore: 새로운 Store 인스턴스를 생성한다.
func NewStore(client valkey.Client, logger *slog.Logger, config Config) *Store {
	return &Store{
		client: client,
		logger: logger,
		config: config,
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

// loadScripts Lua 스크립트를 로드하고 SHA를 캐싱.
func (s *Store) loadScripts(ctx context.Context) error {
	if s.enqueueSHA == "" {
		cmd := s.client.B().ScriptLoad().Script(enqueueLua).Build()
		sha, err := s.client.Do(ctx, cmd).ToString()
		if err != nil {
			return fmt.Errorf("load enqueue script failed: %w", err)
		}
		s.enqueueSHA = sha
	}
	if s.dequeueSHA == "" {
		cmd := s.client.B().ScriptLoad().Script(dequeueLua).Build()
		sha, err := s.client.Do(ctx, cmd).ToString()
		if err != nil {
			return fmt.Errorf("load dequeue script failed: %w", err)
		}
		s.dequeueSHA = sha
	}
	return nil
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
	if err := s.loadScripts(ctx); err != nil {
		return EnqueueQueueFull, err
	}

	dataKey := s.dataKey(chatID)
	orderKey := s.orderKey(chatID)

	replaceArg := "0"
	if replaceOnDuplicate {
		replaceArg = "1"
	}

	cmd := s.client.B().Evalsha().Sha1(s.enqueueSHA).Numkeys(2).Key(dataKey, orderKey).Arg(
		userID,
		jsonValue,
		strconv.FormatInt(timestamp, 10),
		strconv.Itoa(s.config.MaxQueueSize),
		strconv.Itoa(s.config.QueueTTLSeconds),
		replaceArg,
	).Build()

	res, err := s.client.Do(ctx, cmd).ToAny()
	if err != nil {
		// NOSCRIPT 오류 시 스크립트 재로드
		if valkeyx.IsNoScript(err) {
			s.enqueueSHA = ""
			return s.enqueueInternal(ctx, chatID, userID, timestamp, jsonValue, replaceOnDuplicate)
		}
		return EnqueueQueueFull, fmt.Errorf("pending enqueue failed: %w", err)
	}

	switch normalizeLuaResult(res) {
	case "SUCCESS":
		s.logger.Debug("enqueue_success", "chat_id", chatID, "user_id", userID)
		return EnqueueSuccess, nil
	case "DUPLICATE":
		s.logger.Debug("enqueue_duplicate", "chat_id", chatID, "user_id", userID)
		return EnqueueDuplicate, nil
	case "QUEUE_FULL":
		s.logger.Warn("enqueue_queue_full", "chat_id", chatID)
		return EnqueueQueueFull, nil
	default:
		s.logger.Error("enqueue_unknown_result", "chat_id", chatID, "result", res)
		return EnqueueQueueFull, nil
	}
}

// Dequeue: 대기열에서 가장 오래된 메시지(stale 메시지 포함)를 꺼내어 반환한다.
// 성공 시 RawJSON 필드에 원본 데이터가 포함된다.
func (s *Store) Dequeue(ctx context.Context, chatID string) (DequeueResult, error) {
	if err := s.loadScripts(ctx); err != nil {
		return DequeueResult{}, err
	}

	dataKey := s.dataKey(chatID)
	orderKey := s.orderKey(chatID)

	cmd := s.client.B().Evalsha().Sha1(s.dequeueSHA).Numkeys(2).Key(dataKey, orderKey).Arg(
		strconv.FormatInt(s.config.StaleThresholdMS, 10),
	).Build()

	res, err := s.client.Do(ctx, cmd).ToAny()
	if err != nil {
		if valkeyx.IsNil(err) {
			s.logger.Debug("dequeue_empty", "chat_id", chatID)
			return DequeueResult{Status: DequeueEmpty}, nil
		}
		// NOSCRIPT 오류 시 스크립트 재로드
		if valkeyx.IsNoScript(err) {
			s.dequeueSHA = ""
			return s.Dequeue(ctx, chatID)
		}
		return DequeueResult{}, fmt.Errorf("pending dequeue failed: %w", err)
	}
	if res == nil {
		s.logger.Debug("dequeue_empty", "chat_id", chatID)
		return DequeueResult{Status: DequeueEmpty}, nil
	}

	switch typed := res.(type) {
	case []any:
		if len(typed) != 3 {
			return DequeueResult{}, fmt.Errorf("pending dequeue unexpected lua result: %T len=%d", res, len(typed))
		}
		userID := normalizeLuaResult(typed[0])
		timestamp, err := parseLuaScoreToInt64(typed[1])
		if err != nil {
			return DequeueResult{}, fmt.Errorf("pending dequeue parse score failed: %w", err)
		}
		raw := normalizeLuaResult(typed[2])
		s.logger.Debug("dequeue_success", "chat_id", chatID)
		return DequeueResult{Status: DequeueSuccess, UserID: userID, Timestamp: timestamp, RawJSON: raw}, nil
	default:
		// 과거 포맷(문자열만 반환) 호환.
		raw := normalizeLuaResult(res)
		if raw == "EXHAUSTED" {
			s.logger.Debug("dequeue_exhausted", "chat_id", chatID)
			return DequeueResult{Status: DequeueExhausted}, nil
		}

		s.logger.Debug("dequeue_success", "chat_id", chatID)
		return DequeueResult{Status: DequeueSuccess, RawJSON: raw}, nil
	}
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

func parseLuaScoreToInt64(v any) (int64, error) {
	score := strings.TrimSpace(normalizeLuaResult(v))
	if score == "" {
		return 0, errors.New("empty score")
	}
	f, err := strconv.ParseFloat(score, 64)
	if err != nil {
		return 0, fmt.Errorf("parse lua score failed: %w", err)
	}
	return int64(f), nil
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

func (s *Store) dataKey(chatID string) string {
	return fmt.Sprintf("%s:data:{%s}", s.config.KeyPrefix, chatID)
}

func (s *Store) orderKey(chatID string) string {
	return fmt.Sprintf("%s:order:{%s}", s.config.KeyPrefix, chatID)
}

func normalizeLuaResult(v any) string {
	switch typed := v.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
