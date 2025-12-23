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
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// SurrenderVoteStore 는 타입이다.
type SurrenderVoteStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewSurrenderVoteStore 는 동작을 수행한다.
func NewSurrenderVoteStore(client valkey.Client, logger *slog.Logger) *SurrenderVoteStore {
	return &SurrenderVoteStore{
		client: client,
		logger: logger,
	}
}

// Get 는 동작을 수행한다.
func (s *SurrenderVoteStore) Get(ctx context.Context, chatID string) (*tsmodel.SurrenderVote, error) {
	key := voteKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, tserrors.RedisError{Operation: "vote_get", Err: err}
	}

	var vote tsmodel.SurrenderVote
	if err := json.Unmarshal(raw, &vote); err != nil {
		return nil, tserrors.RedisError{Operation: "vote_unmarshal", Err: err}
	}
	return &vote, nil
}

// Save 는 동작을 수행한다.
func (s *SurrenderVoteStore) Save(ctx context.Context, chatID string, vote tsmodel.SurrenderVote) error {
	key := voteKey(chatID)

	raw, err := json.Marshal(vote)
	if err != nil {
		return tserrors.RedisError{Operation: "vote_marshal", Err: err}
	}

	cmd := s.client.B().Set().Key(key).Value(string(raw)).Ex(time.Duration(tsconfig.RedisVoteTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return tserrors.RedisError{Operation: "vote_save", Err: err}
	}
	s.logger.Debug("vote_saved", "chat_id", chatID, "approvals", len(vote.Approvals))
	return nil
}

// Approve 는 동작을 수행한다.
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

// Clear 는 동작을 수행한다.
func (s *SurrenderVoteStore) Clear(ctx context.Context, chatID string) error {
	key := voteKey(chatID)

	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return tserrors.RedisError{Operation: "vote_clear", Err: err}
	}
	s.logger.Debug("vote_cleared", "chat_id", chatID)
	return nil
}
