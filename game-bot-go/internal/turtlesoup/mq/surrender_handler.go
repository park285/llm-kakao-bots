package mq

import (
	"context"
	"fmt"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tssvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/service"
)

// SurrenderHandler 는 타입이다.
type SurrenderHandler struct {
	gameService *tssvc.GameService
	voteService *tssvc.SurrenderVoteService
	msgProvider *messageprovider.Provider
}

// NewSurrenderHandler 는 동작을 수행한다.
func NewSurrenderHandler(gameService *tssvc.GameService, voteService *tssvc.SurrenderVoteService, msgProvider *messageprovider.Provider) *SurrenderHandler {
	return &SurrenderHandler{
		gameService: gameService,
		voteService: voteService,
		msgProvider: msgProvider,
	}
}

// HandleConsensus 는 동작을 수행한다.
func (h *SurrenderHandler) HandleConsensus(ctx context.Context, chatID string, userID string) (string, error) {
	if err := h.voteService.RequireSession(ctx, chatID); err != nil {
		return "", fmt.Errorf("require session failed: %w", err)
	}

	players, err := h.voteService.ResolvePlayers(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("resolve players failed: %w", err)
	}

	active, err := h.voteService.ActiveVote(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("get active vote failed: %w", err)
	}

	if active != nil {
		if len(players) == 1 {
			_ = h.voteService.Clear(ctx, chatID)
			return h.executeSurrender(ctx, chatID)
		}
		return h.inProgressMessage(*active), nil
	}

	startResult, err := h.voteService.StartVote(ctx, chatID, userID, players)
	if err != nil {
		return "", fmt.Errorf("start vote failed: %w", err)
	}

	switch startResult.Type {
	case tssvc.VoteStartImmediate:
		return h.executeSurrender(ctx, chatID)
	case tssvc.VoteStartStarted:
		return h.msgProvider.Get(
			tsmessages.VoteStart,
			messageprovider.P("required", startResult.Vote.RequiredApprovals()),
			messageprovider.P("current", len(startResult.Vote.Approvals)),
		), nil
	default:
		return "", fmt.Errorf("unknown vote start result")
	}
}

// HandleAgree 는 동작을 수행한다.
func (h *SurrenderHandler) HandleAgree(ctx context.Context, chatID string, userID string) (string, error) {
	if err := h.voteService.RequireSession(ctx, chatID); err != nil {
		return "", fmt.Errorf("require session failed: %w", err)
	}

	result, err := h.voteService.Approve(ctx, chatID, userID)
	if err != nil {
		return "", fmt.Errorf("approve vote failed: %w", err)
	}

	switch result.Type {
	case tssvc.VoteApprovalNotFound:
		return h.msgProvider.Get(tsmessages.VoteNotFound), nil
	case tssvc.VoteApprovalNotEligible:
		return h.msgProvider.Get(tsmessages.VoteNotFound), nil
	case tssvc.VoteApprovalAlreadyVoted:
		return h.msgProvider.Get(tsmessages.VoteAlreadyVoted), nil
	case tssvc.VoteApprovalPersistenceFailure:
		return h.msgProvider.Get(tsmessages.ErrorInternal), nil
	case tssvc.VoteApprovalProgress:
		if result.Vote == nil {
			return h.msgProvider.Get(tsmessages.ErrorInternal), nil
		}
		return h.inProgressMessage(*result.Vote), nil
	case tssvc.VoteApprovalCompleted:
		if result.Vote == nil {
			return h.msgProvider.Get(tsmessages.ErrorInternal), nil
		}
		body, err := h.executeSurrender(ctx, chatID)
		if err != nil {
			return "", fmt.Errorf("execute surrender failed: %w", err)
		}
		return h.msgProvider.Get(tsmessages.VotePassed) + "\n\n" + body, nil
	default:
		return h.msgProvider.Get(tsmessages.ErrorInternal), nil
	}
}

func (h *SurrenderHandler) inProgressMessage(vote tsmodel.SurrenderVote) string {
	remain := vote.RequiredApprovals() - len(vote.Approvals)
	return h.msgProvider.Get(
		tsmessages.VoteInProgress,
		messageprovider.P("current", len(vote.Approvals)),
		messageprovider.P("required", vote.RequiredApprovals()),
		messageprovider.P("remain", remain),
	)
}

func (h *SurrenderHandler) executeSurrender(ctx context.Context, chatID string) (string, error) {
	result, err := h.gameService.Surrender(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("surrender failed: %w", err)
	}

	hintBlock := ""
	if len(result.HintsUsed) > 0 {
		header := h.msgProvider.Get(tsmessages.SurrenderHintBlockHeader, messageprovider.P("hintCount", len(result.HintsUsed)))
		items := make([]string, 0, len(result.HintsUsed))
		for i, hint := range result.HintsUsed {
			items = append(items, h.msgProvider.Get(
				tsmessages.SurrenderHintItem,
				messageprovider.P("hintNumber", i+1),
				messageprovider.P("content", hint),
			))
		}
		hintBlock = header + strings.Join(items, "\n")
	}

	return h.msgProvider.Get(
		tsmessages.SurrenderResult,
		messageprovider.P("solution", result.Solution),
		messageprovider.P("hintBlock", hintBlock),
	), nil
}
