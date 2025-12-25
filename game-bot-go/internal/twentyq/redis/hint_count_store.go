package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// HintCountStore: 게임별 힌트 사용 횟수를 Redis에 저장하고 관리하는 저장소
type HintCountStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewHintCountStore: 새로운 HintCountStore 인스턴스를 생성한다.
func NewHintCountStore(client valkey.Client, logger *slog.Logger) *HintCountStore {
	return &HintCountStore{
		client: client,
		logger: logger,
	}
}

// Get: 현재까지 사용된 힌트 횟수를 조회한다.
func (s *HintCountStore) Get(ctx context.Context, chatID string) (int, error) {
	key := hintCountKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	value, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		if valkeyx.IsNil(err) {
			return 0, nil
		}
		return 0, cerrors.RedisError{Operation: "hint_count_get", Err: err}
	}
	if value < 0 {
		return 0, nil
	}
	return int(value), nil
}

// Increment: 힌트 사용 횟수를 1 증가시킨다.
func (s *HintCountStore) Increment(ctx context.Context, chatID string) (int, error) {
	key := hintCountKey(chatID)

	incrCmd := s.client.B().Incr().Key(key).Build()
	value, err := s.client.Do(ctx, incrCmd).AsInt64()
	if err != nil {
		return 0, cerrors.RedisError{Operation: "hint_count_incr", Err: err}
	}

	expireCmd := s.client.B().Expire().Key(key).Seconds(int64(qconfig.RedisSessionTTLSeconds)).Build()
	if err := s.client.Do(ctx, expireCmd).Error(); err != nil {
		return int(value), cerrors.RedisError{Operation: "hint_count_expire", Err: err}
	}
	return int(value), nil
}

// Delete: 힌트 카운트 정보를 삭제한다.
func (s *HintCountStore) Delete(ctx context.Context, chatID string) error {
	key := hintCountKey(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "hint_count_delete", Err: err}
	}
	return nil
}

// compile-time assertion to ensure time package is used (for TTL calculations)
var _ = time.Second
