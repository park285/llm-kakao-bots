package service

import (
	"context"
	"fmt"
	"time"

	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// SurrenderVoteService 는 타입이다.
type SurrenderVoteService struct {
	sessionManager *GameSessionManager
	voteStore      *tsredis.SurrenderVoteStore
}

// NewSurrenderVoteService 는 동작을 수행한다.
func NewSurrenderVoteService(sessionManager *GameSessionManager, voteStore *tsredis.SurrenderVoteStore) *SurrenderVoteService {
	return &SurrenderVoteService{
		sessionManager: sessionManager,
		voteStore:      voteStore,
	}
}

// RequireSession 는 동작을 수행한다.
func (s *SurrenderVoteService) RequireSession(ctx context.Context, chatID string) error {
	return s.sessionManager.EnsureSessionExists(ctx, chatID)
}

// ResolvePlayers 는 동작을 수행한다.
func (s *SurrenderVoteService) ResolvePlayers(ctx context.Context, chatID string) ([]string, error) {
	state, err := s.sessionManager.Load(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, tserrors.SessionNotFoundError{SessionID: chatID}
	}

	if len(state.Players) == 0 {
		return []string{state.UserID}, nil
	}
	return state.Players, nil
}

// ActiveVote 는 동작을 수행한다.
func (s *SurrenderVoteService) ActiveVote(ctx context.Context, chatID string) (*tsmodel.SurrenderVote, error) {
	vote, err := s.voteStore.Get(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get surrender vote failed: %w", err)
	}
	return vote, nil
}

// VoteStartResultType 는 타입이다.
type VoteStartResultType int

// VoteStartImmediate 는 투표 시작 결과 상수 목록이다.
const (
	VoteStartImmediate VoteStartResultType = iota
	VoteStartStarted
)

// VoteStartResult 는 타입이다.
type VoteStartResult struct {
	Type VoteStartResultType
	Vote tsmodel.SurrenderVote
}

// StartVote 는 동작을 수행한다.
func (s *SurrenderVoteService) StartVote(ctx context.Context, chatID string, initiator string, players []string) (VoteStartResult, error) {
	vote := tsmodel.SurrenderVote{
		Initiator:       initiator,
		EligiblePlayers: players,
		Approvals:       []string{initiator},
		CreatedAt:       time.Now().UnixMilli(),
	}

	if vote.IsApproved() {
		return VoteStartResult{Type: VoteStartImmediate, Vote: vote}, nil
	}

	if err := s.voteStore.Save(ctx, chatID, vote); err != nil {
		return VoteStartResult{}, fmt.Errorf("save surrender vote failed: %w", err)
	}
	return VoteStartResult{Type: VoteStartStarted, Vote: vote}, nil
}

// VoteApprovalResultType 는 타입이다.
type VoteApprovalResultType int

// VoteApprovalCompleted 는 투표 승인 결과 상수 목록이다.
const (
	VoteApprovalCompleted VoteApprovalResultType = iota
	VoteApprovalProgress
	VoteApprovalNotFound
	VoteApprovalNotEligible
	VoteApprovalAlreadyVoted
	VoteApprovalPersistenceFailure
)

// VoteApprovalResult 는 타입이다.
type VoteApprovalResult struct {
	Type VoteApprovalResultType
	Vote *tsmodel.SurrenderVote
}

// Approve 는 동작을 수행한다.
func (s *SurrenderVoteService) Approve(ctx context.Context, chatID string, userID string) (VoteApprovalResult, error) {
	vote, err := s.voteStore.Get(ctx, chatID)
	if err != nil {
		return VoteApprovalResult{}, fmt.Errorf("get surrender vote failed: %w", err)
	}
	if vote == nil {
		return VoteApprovalResult{Type: VoteApprovalNotFound}, nil
	}

	if !vote.CanVote(userID) {
		return VoteApprovalResult{Type: VoteApprovalNotEligible}, nil
	}
	if vote.HasVoted(userID) {
		return VoteApprovalResult{Type: VoteApprovalAlreadyVoted}, nil
	}

	updated, err := s.voteStore.Approve(ctx, chatID, userID)
	if err != nil {
		return VoteApprovalResult{}, fmt.Errorf("approve surrender vote failed: %w", err)
	}
	if updated == nil {
		return VoteApprovalResult{Type: VoteApprovalPersistenceFailure}, nil
	}

	if updated.IsApproved() {
		if err := s.voteStore.Clear(ctx, chatID); err != nil {
			return VoteApprovalResult{}, fmt.Errorf("clear surrender vote failed: %w", err)
		}
		return VoteApprovalResult{Type: VoteApprovalCompleted, Vote: updated}, nil
	}

	return VoteApprovalResult{Type: VoteApprovalProgress, Vote: updated}, nil
}

// Clear 는 동작을 수행한다.
func (s *SurrenderVoteService) Clear(ctx context.Context, chatID string) error {
	if err := s.voteStore.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("clear surrender vote failed: %w", err)
	}
	return nil
}
