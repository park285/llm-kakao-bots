package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// SurrenderVoteStore: 게임 항복(Surrender) 투표의 진행 상황(찬성 수, 만료 시간 등)을 Redis에 저장하고 관리하는 저장소
type SurrenderVoteStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewSurrenderVoteStore: 새로운 SurrenderVoteStore 인스턴스를 생성한다.
func NewSurrenderVoteStore(client valkey.Client, logger *slog.Logger) *SurrenderVoteStore {
	return &SurrenderVoteStore{
		client: client,
		logger: logger,
	}
}

// Get: 현재 활성화된 투표 상태를 조회한다. 투표가 없으면 nil을 반환한다.
func (s *SurrenderVoteStore) Get(ctx context.Context, chatID string) (*tsmodel.SurrenderVote, error) {
	key := voteKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, cerrors.RedisError{Operation: "vote_get", Err: err}
	}

	var vote tsmodel.SurrenderVote
	if err := json.Unmarshal(raw, &vote); err != nil {
		return nil, cerrors.RedisError{Operation: "vote_unmarshal", Err: err}
	}
	return &vote, nil
}

// Save: 변경된 투표 상태 객체를 Redis에 저장하고 TTL을 갱신한다.
func (s *SurrenderVoteStore) Save(ctx context.Context, chatID string, vote tsmodel.SurrenderVote) error {
	key := voteKey(chatID)

	raw, err := json.Marshal(vote)
	if err != nil {
		return cerrors.RedisError{Operation: "vote_marshal", Err: err}
	}

	cmd := s.client.B().Set().Key(key).Value(string(raw)).Ex(time.Duration(tsconfig.RedisVoteTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "vote_save", Err: err}
	}
	s.logger.Debug("vote_saved", "chat_id", chatID, "approvals", len(vote.Approvals))
	return nil
}

// Approve: 특정 사용자의 '찬성' 의사를 투표에 반영하고, 갱신된 투표 상태를 반환한다.
func (s *SurrenderVoteStore) Approve(ctx context.Context, chatID string, userID string) (*tsmodel.SurrenderVote, error) {
	vote, err := s.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if vote == nil {
		return nil, nil
	}

	updated, err := vote.Approve(userID)
	if err != nil {
		return nil, fmt.Errorf("vote approve failed: %w", err)
	}
	if err := s.Save(ctx, chatID, updated); err != nil {
		return nil, err
	}
	s.logger.Debug("vote_approved", "chat_id", chatID, "user_id", userID, "approvals", len(updated.Approvals), "required", updated.RequiredApprovals())
	return &updated, nil
}

// Clear: 투표 상태를 Redis에서 삭제한다. (투표 완료 또는 취소 시)
func (s *SurrenderVoteStore) Clear(ctx context.Context, chatID string) error {
	key := voteKey(chatID)

	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "vote_clear", Err: err}
	}
	s.logger.Debug("vote_cleared", "chat_id", chatID)
	return nil
}
