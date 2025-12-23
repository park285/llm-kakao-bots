package redis

import (
	"context"
	"log/slog"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
)

// PuzzleDedupStore 는 타입이다.
type PuzzleDedupStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewPuzzleDedupStore 는 동작을 수행한다.
func NewPuzzleDedupStore(client valkey.Client, logger *slog.Logger) *PuzzleDedupStore {
	return &PuzzleDedupStore{
		client: client,
		logger: logger,
	}
}

// IsDuplicate 는 동작을 수행한다.
func (s *PuzzleDedupStore) IsDuplicate(ctx context.Context, signature string, chatID string) (bool, error) {
	globalKey := tsconfig.RedisKeyPuzzleGlobal
	chatKey := puzzleChatKey(chatID)

	globalCmd := s.client.B().Sismember().Key(globalKey).Member(signature).Build()
	globalExists, err := s.client.Do(ctx, globalCmd).AsBool()
	if err != nil && !valkeyx.IsNil(err) {
		return false, tserrors.RedisError{Operation: "puzzle_dedup_check_global", Err: err}
	}
	if globalExists {
		return true, nil
	}

	chatCmd := s.client.B().Sismember().Key(chatKey).Member(signature).Build()
	chatExists, err := s.client.Do(ctx, chatCmd).AsBool()
	if err != nil && !valkeyx.IsNil(err) {
		return false, tserrors.RedisError{Operation: "puzzle_dedup_check_chat", Err: err}
	}
	return chatExists, nil
}

// MarkUsed 는 동작을 수행한다.
func (s *PuzzleDedupStore) MarkUsed(ctx context.Context, signature string, chatID string) error {
	globalKey := tsconfig.RedisKeyPuzzleGlobal
	chatKey := puzzleChatKey(chatID)

	saddGlobalCmd := s.client.B().Sadd().Key(globalKey).Member(signature).Build()
	saddChatCmd := s.client.B().Sadd().Key(chatKey).Member(signature).Build()
	expireGlobalCmd := s.client.B().Expire().Key(globalKey).Seconds(int64(tsconfig.PuzzleDedupGlobalTTLSeconds)).Build()
	expireChatCmd := s.client.B().Expire().Key(chatKey).Seconds(int64(tsconfig.PuzzleDedupChatTTLSeconds)).Build()

	results := s.client.DoMulti(ctx, saddGlobalCmd, saddChatCmd, expireGlobalCmd, expireChatCmd)
	for _, r := range results {
		if err := r.Error(); err != nil && !valkeyx.IsNil(err) {
			return tserrors.RedisError{Operation: "puzzle_dedup_mark", Err: err}
		}
	}

	s.logger.Info("puzzle_dedup_marked", "chat_id", chatID)
	return nil
}
