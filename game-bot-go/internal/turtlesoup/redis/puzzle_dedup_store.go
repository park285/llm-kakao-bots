package redis

import (
	"context"
	"log/slog"

	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

// PuzzleDedupStore: 생성된 퍼즐의 내용(Signature)을 기반으로 중복 생성을 감지하고 방지하는 저장소
// 전역 범위(Global)와 채팅방 범위(Chat) 두 가지 레벨에서 중복을 체크한다.
type PuzzleDedupStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewPuzzleDedupStore: 새로운 PuzzleDedupStore 인스턴스를 생성한다.
func NewPuzzleDedupStore(client valkey.Client, logger *slog.Logger) *PuzzleDedupStore {
	return &PuzzleDedupStore{
		client: client,
		logger: logger,
	}
}

// IsDuplicate: 주어진 퍼즐 서명(Signature)이 전역적으로 또는 현재 채팅방 내에서 이미 사용된 적이 있는지 확인한다.
func (s *PuzzleDedupStore) IsDuplicate(ctx context.Context, signature string, chatID string) (bool, error) {
	globalKey := tsconfig.RedisKeyPuzzleGlobal
	chatKey := puzzleChatKey(chatID)

	globalCmd := s.client.B().Sismember().Key(globalKey).Member(signature).Build()
	globalExists, err := s.client.Do(ctx, globalCmd).AsBool()
	if err != nil && !valkeyx.IsNil(err) {
		return false, cerrors.RedisError{Operation: "puzzle_dedup_check_global", Err: err}
	}
	if globalExists {
		return true, nil
	}

	chatCmd := s.client.B().Sismember().Key(chatKey).Member(signature).Build()
	chatExists, err := s.client.Do(ctx, chatCmd).AsBool()
	if err != nil && !valkeyx.IsNil(err) {
		return false, cerrors.RedisError{Operation: "puzzle_dedup_check_chat", Err: err}
	}
	return chatExists, nil
}

// MarkUsed: 생성된 퍼즐을 '사용됨' 상태로 Redis Set에 등록하여 이후 중복 생성을 방지한다.
// 전역 관리 셋과 채팅방 관리 셋에 각각 TTL을 적용하여 저장한다.
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
			return cerrors.RedisError{Operation: "puzzle_dedup_mark", Err: err}
		}
	}

	s.logger.Info("puzzle_dedup_marked", "chat_id", chatID)
	return nil
}
