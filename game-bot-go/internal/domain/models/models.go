package models

import (
	"fmt"
	"slices"
	"strings"
)

// PendingMessage 는 타입이다.
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

// DisplayName 는 동작을 수행한다.
func (m PendingMessage) DisplayName(chatID string, anonymous string) string {
	return DisplayName(chatID, m.UserID, m.Sender, anonymous)
}

// SurrenderVote 는 타입이다.
type SurrenderVote struct {
	Initiator       string   `json:"initiator"`
	EligiblePlayers []string `json:"eligiblePlayers"`
	Approvals       []string `json:"approvals,omitempty"`
	CreatedAt       int64    `json:"createdAt"`
}

// RequiredApprovals 는 동작을 수행한다.
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

// IsApproved 는 동작을 수행한다.
func (v SurrenderVote) IsApproved() bool { return len(v.Approvals) >= v.RequiredApprovals() }

// CanVote 는 동작을 수행한다.
func (v SurrenderVote) CanVote(userID string) bool { return slices.Contains(v.EligiblePlayers, userID) }

// HasVoted 는 동작을 수행한다.
func (v SurrenderVote) HasVoted(userID string) bool { return slices.Contains(v.Approvals, userID) }

// Approve 는 동작을 수행한다.
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
