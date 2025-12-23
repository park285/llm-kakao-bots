package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// HandleSurrenderConsensus 는 동작을 수행한다.
func (s *RiddleService) HandleSurrenderConsensus(ctx context.Context, chatID string, userID string) (string, error) {
	holderName := userID
	out := ""

	err := s.lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		secret, err := s.sessionStore.GetSecret(ctx, chatID)
		if err != nil {
			return fmt.Errorf("secret get failed: %w", err)
		}
		if secret == nil {
			return qerrors.SessionNotFoundError{ChatID: chatID}
		}

		players, err := s.playerStore.GetAll(ctx, chatID)
		if err != nil {
			return fmt.Errorf("player getAll failed: %w", err)
		}
		eligible := make([]string, 0, len(players))
		for _, p := range players {
			if strings.TrimSpace(p.UserID) == "" {
				continue
			}
			eligible = append(eligible, p.UserID)
		}
		if len(eligible) == 0 {
			eligible = []string{userID}
		}

		hasVote, err := s.voteStore.Exists(ctx, chatID)
		if err != nil {
			return fmt.Errorf("vote exists failed: %w", err)
		}

		if hasVote {
			vote, err := s.voteStore.Get(ctx, chatID)
			if err != nil {
				return fmt.Errorf("vote get failed: %w", err)
			}
			if vote == nil {
				out = s.msgProvider.Get(qmessages.VoteInProgress,
					messageprovider.P("current", 0),
					messageprovider.P("required", 1),
					messageprovider.P("remain", 1),
					messageprovider.P("prefix", s.commandPrefix),
				)
				return nil
			}

			if len(eligible) <= 1 {
				result, err := s.Surrender(ctx, chatID)
				if err != nil {
					return err
				}
				_ = s.voteStore.Clear(ctx, chatID)
				out = result
				return nil
			}

			_ = s.voteStore.Save(ctx, chatID, *vote)
			remain := vote.RequiredApprovals() - len(vote.Approvals)
			out = s.msgProvider.Get(
				qmessages.VoteInProgress,
				messageprovider.P("current", len(vote.Approvals)),
				messageprovider.P("required", vote.RequiredApprovals()),
				messageprovider.P("remain", remain),
				messageprovider.P("prefix", s.commandPrefix),
			)
			return nil
		}

		vote := qmodel.SurrenderVote{
			Initiator:       userID,
			EligiblePlayers: eligible,
			Approvals:       []string{userID},
			CreatedAt:       time.Now().UnixMilli(),
		}
		if vote.IsApproved() {
			result, err := s.Surrender(ctx, chatID)
			if err != nil {
				return err
			}
			out = result
			return nil
		}

		if err := s.voteStore.Save(ctx, chatID, vote); err != nil {
			return fmt.Errorf("vote save failed: %w", err)
		}

		out = s.msgProvider.Get(
			qmessages.VoteStart,
			messageprovider.P("required", vote.RequiredApprovals()),
			messageprovider.P("current", len(vote.Approvals)),
			messageprovider.P("prefix", s.commandPrefix),
		)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("surrender consensus failed: %w", err)
	}
	return out, nil
}

// HandleSurrenderAgree 는 동작을 수행한다.
func (s *RiddleService) HandleSurrenderAgree(ctx context.Context, chatID string, userID string) (string, error) {
	holderName := userID
	out := ""

	err := s.lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		secret, err := s.sessionStore.GetSecret(ctx, chatID)
		if err != nil {
			return fmt.Errorf("secret get failed: %w", err)
		}
		if secret == nil {
			return qerrors.SessionNotFoundError{ChatID: chatID}
		}

		vote, err := s.voteStore.Get(ctx, chatID)
		if err != nil {
			return fmt.Errorf("vote get failed: %w", err)
		}
		if vote == nil {
			out = s.msgProvider.Get(qmessages.VoteNotFound, messageprovider.P("prefix", s.commandPrefix))
			return nil
		}

		if !vote.CanVote(userID) {
			out = s.msgProvider.Get(qmessages.VoteCannotVote)
			return nil
		}
		if vote.HasVoted(userID) {
			out = s.msgProvider.Get(qmessages.VoteAlreadyVoted)
			return nil
		}

		updated, err := s.voteStore.Approve(ctx, chatID, userID)
		if err != nil {
			return fmt.Errorf("vote approve failed: %w", err)
		}
		if updated == nil {
			out = s.msgProvider.Get(qmessages.VoteProcessingFailed)
			return nil
		}

		if updated.IsApproved() {
			result, err := s.Surrender(ctx, chatID)
			if err != nil {
				return err
			}
			_ = s.voteStore.Clear(ctx, chatID)
			out = result
			return nil
		}

		remain := updated.RequiredApprovals() - len(updated.Approvals)
		out = s.msgProvider.Get(
			qmessages.VoteAgreeProgress,
			messageprovider.P("current", len(updated.Approvals)),
			messageprovider.P("required", updated.RequiredApprovals()),
			messageprovider.P("remain", remain),
		)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("surrender agree failed: %w", err)
	}

	return out, nil
}

// HandleSurrenderReject 는 동작을 수행한다.
func (s *RiddleService) HandleSurrenderReject(ctx context.Context, chatID string, userID string) (string, error) {
	if _, err := s.sessionStore.GetSecret(ctx, chatID); err != nil {
		return "", fmt.Errorf("secret get failed: %w", err)
	}
	return s.msgProvider.Get(qmessages.VoteRejectNotSupported), nil
}
