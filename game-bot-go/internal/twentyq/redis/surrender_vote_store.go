package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
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
func (s *SurrenderVoteStore) Get(ctx context.Context, chatID string) (*qmodel.SurrenderVote, error) {
	key := voteKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, qerrors.RedisError{Operation: "vote_get", Err: err}
	}

	var vote qmodel.SurrenderVote
	if err := json.Unmarshal(raw, &vote); err != nil {
		return nil, qerrors.RedisError{Operation: "vote_unmarshal", Err: err}
	}
	return &vote, nil
}

// Save 는 동작을 수행한다.
func (s *SurrenderVoteStore) Save(ctx context.Context, chatID string, vote qmodel.SurrenderVote) error {
	key := voteKey(chatID)

	payload, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("marshal surrender vote failed: %w", err)
	}

	cmd := s.client.B().Set().Key(key).Value(string(payload)).Ex(time.Duration(qconfig.RedisVoteTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "vote_save", Err: err}
	}
	return nil
}

// Clear 는 동작을 수행한다.
func (s *SurrenderVoteStore) Clear(ctx context.Context, chatID string) error {
	key := voteKey(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "vote_clear", Err: err}
	}
	return nil
}

// Exists 는 동작을 수행한다.
func (s *SurrenderVoteStore) Exists(ctx context.Context, chatID string) (bool, error) {
	key := voteKey(chatID)
	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, qerrors.RedisError{Operation: "vote_exists", Err: err}
	}
	return n > 0, nil
}

// Approve 는 동작을 수행한다.
func (s *SurrenderVoteStore) Approve(ctx context.Context, chatID string, userID string) (*qmodel.SurrenderVote, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("invalid user id")
	}

	vote, err := s.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if vote == nil {
		return nil, nil
	}

	next, approveErr := vote.Approve(userID)
	if approveErr != nil {
		return nil, fmt.Errorf("vote approve failed: %w", approveErr)
	}

	if err := s.Save(ctx, chatID, next); err != nil {
		s.logger.Warn("vote_save_failed", "chat_id", chatID, "err", err)
		return nil, err
	}
	return &next, nil
}
