package service

import (
	"context"
	"fmt"
	"time"

	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
)

// SurrenderVoteService: 항복 투표 기능을 관리하는 서비스입니다.
// 투표 시작, 찬성, 결과 확인 등을 담당합니다.
type SurrenderVoteService struct {
	sessionManager *GameSessionManager
	voteStore      *tsredis.SurrenderVoteStore
}

// NewSurrenderVoteService: SurrenderVoteService 인스턴스를 생성합니다.
func NewSurrenderVoteService(sessionManager *GameSessionManager, voteStore *tsredis.SurrenderVoteStore) *SurrenderVoteService {
	return &SurrenderVoteService{
		sessionManager: sessionManager,
		voteStore:      voteStore,
	}
}

// RequireSession: 세션이 존재하는지 확인합니다.
func (s *SurrenderVoteService) RequireSession(ctx context.Context, chatID string) error {
	return s.sessionManager.EnsureSessionExists(ctx, chatID)
}

// ResolvePlayers: 현재 게임에 참여 중인 플레이어 목록을 반환합니다.
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

// ActiveVote: 현재 진행 중인 항복 투표를 조회합니다.
func (s *SurrenderVoteService) ActiveVote(ctx context.Context, chatID string) (*tsmodel.SurrenderVote, error) {
	vote, err := s.voteStore.Get(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get surrender vote failed: %w", err)
	}
	return vote, nil
}

// VoteStartResultType: 투표 시작 결과 유형을 나타냅니다.
type VoteStartResultType int

// VoteStartImmediate: 투표 시작 결과 상수 목록입니다.
const (
	VoteStartImmediate VoteStartResultType = iota
	VoteStartStarted
)

// VoteStartResult: 투표 시작 결과를 담는 구조체입니다.
type VoteStartResult struct {
	Type VoteStartResultType
	Vote tsmodel.SurrenderVote
}

// StartVote: 새 항복 투표를 시작합니다.
// 1인 게임이면 즉시 승인되고, 다중 플레이어면 과반수 동의가 필요합니다.
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

// VoteApprovalResultType: 투표 찬성 결과 유형을 나타냅니다.
type VoteApprovalResultType int

// VoteApprovalCompleted: 투표 승인 결과 상수 목록입니다.
const (
	VoteApprovalCompleted VoteApprovalResultType = iota
	VoteApprovalProgress
	VoteApprovalNotFound
	VoteApprovalNotEligible
	VoteApprovalAlreadyVoted
	VoteApprovalPersistenceFailure
)

// VoteApprovalResult: 투표 찬성 처리 결과를 담는 구조체입니다.
type VoteApprovalResult struct {
	Type VoteApprovalResultType
	Vote *tsmodel.SurrenderVote
}

// Approve: 사용자의 항복 투표 찬성을 처리합니다.
// 과반수가 찬성하면 투표가 완료됩니다.
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

// Clear: 항복 투표 데이터를 삭제합니다.
func (s *SurrenderVoteService) Clear(ctx context.Context, chatID string) error {
	if err := s.voteStore.Clear(ctx, chatID); err != nil {
		return fmt.Errorf("clear surrender vote failed: %w", err)
	}
	return nil
}
