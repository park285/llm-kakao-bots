package models

import (
	"fmt"
	"slices"
	"strings"
)

// PendingMessage: 큐에 대기 중인 사용자 메시지 구조체
type PendingMessage struct {
	UserID    string  `json:"userId"`
	Content   string  `json:"content"`
	ThreadID  *string `json:"threadId,omitempty"`
	Sender    *string `json:"sender,omitempty"`
	Timestamp int64   `json:"timestamp"`
	// 체인 질문 배치 처리용
	IsChainBatch   bool     `json:"isChainBatch,omitempty"`
	BatchQuestions []string `json:"batchQuestions,omitempty"`
}

// DisplayName: 사용자의 표시 이름을 반환합니다. (익명 설정 시 대체 이름 사용)
func (m PendingMessage) DisplayName(chatID string, anonymous string) string {
	return DisplayName(chatID, m.UserID, m.Sender, anonymous)
}

// SurrenderVote: 항복 투표 상태 관리 구조체
type SurrenderVote struct {
	Initiator       string   `json:"initiator"`
	EligiblePlayers []string `json:"eligiblePlayers"`
	Approvals       []string `json:"approvals,omitempty"`
	CreatedAt       int64    `json:"createdAt"`
}

// RequiredApprovals: 항복 승인에 필요한 최소 득표 수를 반환합니다.
func (v SurrenderVote) RequiredApprovals() int {
	playerCount := len(v.EligiblePlayers)
	switch {
	case playerCount <= 1:
		return 1
	case playerCount == 2:
		return 2
	default:
		return 3
	}
}

// IsApproved: 투표가 가결되었는지(필요 득표 수 충족) 확인합니다.
func (v SurrenderVote) IsApproved() bool { return len(v.Approvals) >= v.RequiredApprovals() }

// CanVote: 해당 사용자가 투표 자격이 있는지 확인합니다.
func (v SurrenderVote) CanVote(userID string) bool { return slices.Contains(v.EligiblePlayers, userID) }

// HasVoted: 해당 사용자가 이미 투표했는지 확인합니다.
func (v SurrenderVote) HasVoted(userID string) bool { return slices.Contains(v.Approvals, userID) }

// Approve: 사용자의 찬성 투표를 기록하고 갱신된 투표 상태를 반환합니다.
func (v SurrenderVote) Approve(userID string) (SurrenderVote, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return SurrenderVote{}, fmt.Errorf("invalid user id")
	}
	if !v.CanVote(userID) {
		return SurrenderVote{}, fmt.Errorf("user %s is not eligible to vote", userID)
	}
	if v.HasVoted(userID) {
		return v, nil
	}

	next := v
	next.Approvals = append(slices.Clone(v.Approvals), userID)
	return next, nil
}
