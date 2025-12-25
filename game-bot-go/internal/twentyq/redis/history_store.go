package redis

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// HistoryStore: 게임 진행 중 발생한 질문과 답변의 이력을 시간 순서대로 Redis List에 저장하고 관리하는 저장소
type HistoryStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewHistoryStore: 새로운 HistoryStore 인스턴스를 생성한다.
func NewHistoryStore(client valkey.Client, logger *slog.Logger) *HistoryStore {
	return &HistoryStore{
		client: client,
		logger: logger,
	}
}

// Get: Redis List(Lrange)를 사용하여 저장된 모든 질문/답변 이력을 시간 순으로 조회한다.
func (s *HistoryStore) Get(ctx context.Context, chatID string) ([]qmodel.QuestionHistory, error) {
	key := historyKey(chatID)

	cmd := s.client.B().Lrange().Key(key).Start(0).Stop(-1).Build()
	rawItems, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, cerrors.RedisError{Operation: "history_get", Err: err}
	}

	history := make([]qmodel.QuestionHistory, 0, len(rawItems))
	for _, raw := range rawItems {
		var item qmodel.QuestionHistory
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			s.logger.Warn("history_item_unmarshal_failed", "chat_id", chatID, "err", err)
			continue
		}
		history = append(history, item)
	}
	return history, nil
}

// Add: 새로운 질문/답변 이력을 Redis List의 끝(RPUSH)에 추가하고 TTL을 갱신한다.
// 원자성을 위해 파이프라인(DoMulti)을 사용한다.
func (s *HistoryStore) Add(ctx context.Context, chatID string, item qmodel.QuestionHistory) error {
	key := historyKey(chatID)

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal history item failed: %w", err)
	}

	// valkey-go는 DoMulti로 파이프라인 처리
	rpushCmd := s.client.B().Rpush().Key(key).Element(string(payload)).Build()
	expireCmd := s.client.B().Expire().Key(key).Seconds(int64(qconfig.RedisSessionTTLSeconds)).Build()

	results := s.client.DoMulti(ctx, rpushCmd, expireCmd)
	for _, r := range results {
		if err := r.Error(); err != nil {
			return cerrors.RedisError{Operation: "history_add", Err: err}
		}
	}
	return nil
}

// Clear: 게임 종료나 초기화 시 해당 채팅방의 모든 이력 정보를 삭제한다.
func (s *HistoryStore) Clear(ctx context.Context, chatID string) error {
	key := historyKey(chatID)

	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "history_clear", Err: err}
	}
	return nil
}
