package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"

	commonvote "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/vote"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// SurrenderVoteStore: 게임 항복(Surrender) 투표의 진행 상황(찬성 수, 만료 시간 등)을 Redis에 저장하고 관리하는 저장소
type SurrenderVoteStore struct {
	base   *commonvote.SurrenderVoteStore
	logger *slog.Logger
}

// NewSurrenderVoteStore: 새로운 SurrenderVoteStore 인스턴스를 생성합니다.
func NewSurrenderVoteStore(client valkey.Client, logger *slog.Logger) *SurrenderVoteStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &SurrenderVoteStore{
		base: commonvote.NewSurrenderVoteStore(
			client,
			voteKey,
			time.Duration(tsconfig.RedisVoteTTLSeconds)*time.Second,
		),
		logger: logger,
	}
}

// Get: 현재 활성화된 투표 상태를 조회합니다. 투표가 없으면 nil을 반환합니다.
func (s *SurrenderVoteStore) Get(ctx context.Context, chatID string) (*tsmodel.SurrenderVote, error) {
	vote, err := s.base.Get(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("vote get failed: %w", err)
	}
	return vote, nil
}

// Save: 변경된 투표 상태 객체를 Redis에 저장하고 TTL을 갱신합니다.
func (s *SurrenderVoteStore) Save(ctx context.Context, chatID string, vote tsmodel.SurrenderVote) error {
	if err := s.base.Save(ctx, chatID, vote); err != nil {
		return fmt.Errorf("vote save failed: %w", err)
	}
	s.logger.Debug("vote_saved", "chat_id", chatID, "approvals", len(vote.Approvals))
	return nil
}

// Approve: 특정 사용자의 '찬성' 의사를 투표에 반영하고, 갱신된 투표 상태를 반환합니다.
func (s *SurrenderVoteStore) Approve(ctx context.Context, chatID string, userID string) (*tsmodel.SurrenderVote, error) {
	vote, err := s.base.Get(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("vote get failed: %w", err)
	}
	if vote == nil {
		return nil, nil
	}

	updated, err := vote.Approve(userID)
	if err != nil {
		return nil, fmt.Errorf("vote approve failed: %w", err)
	}
	if err := s.base.Save(ctx, chatID, updated); err != nil {
		return nil, fmt.Errorf("vote save failed: %w", err)
	}
	s.logger.Debug("vote_approved", "chat_id", chatID, "user_id", userID, "approvals", len(updated.Approvals), "required", updated.RequiredApprovals())
	return &updated, nil
}

// Clear: 투표 상태를 Redis에서 삭제합니다. (투표 완료 또는 취소 시)
func (s *SurrenderVoteStore) Clear(ctx context.Context, chatID string) error {
	if err := s.base.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("vote clear failed: %w", err)
	}
	s.logger.Debug("vote_cleared", "chat_id", chatID)
	return nil
}
