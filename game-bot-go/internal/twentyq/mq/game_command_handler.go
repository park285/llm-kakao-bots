package mq

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qsvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/service"
)

// GameCommandHandler: 게임 관련 명령어를 적절한 서비스 메서드로 라우팅하고 응답을 생성하는 핸들러
type GameCommandHandler struct {
	gameService            *qsvc.RiddleService
	statsService           *qsvc.StatsService
	adminHandler           *qsvc.AdminHandler
	usageHandler           *qsvc.UsageHandler
	chainedQuestionHandler *ChainedQuestionHandler
	msgProvider            *messageprovider.Provider
	logger                 *slog.Logger
	handlers               map[CommandKind]commandHandlerFunc
}

type commandHandlerFunc func(context.Context, mqmsg.InboundMessage, Command) ([]string, error)

// NewGameCommandHandler: 새로운 GameCommandHandler 인스턴스를 생성하고 명령어별 핸들러를 등록한다.
func NewGameCommandHandler(
	gameService *qsvc.RiddleService,
	statsService *qsvc.StatsService,
	adminHandler *qsvc.AdminHandler,
	usageHandler *qsvc.UsageHandler,
	chainedQuestionHandler *ChainedQuestionHandler,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) *GameCommandHandler {
	h := &GameCommandHandler{
		gameService:            gameService,
		statsService:           statsService,
		adminHandler:           adminHandler,
		usageHandler:           usageHandler,
		chainedQuestionHandler: chainedQuestionHandler,
		msgProvider:            msgProvider,
		logger:                 logger,
	}

	h.handlers = map[CommandKind]commandHandlerFunc{
		CommandStart:           h.handleStart,
		CommandAsk:             h.handleAsk,
		CommandChainedQuestion: h.handleChainedQuestion,
		CommandHints:           h.handleHints,
		CommandStatus:          h.handleStatus,
		CommandSurrender:       h.handleSurrender,
		CommandAgree:           h.handleAgree,
		CommandReject:          h.handleReject,
		CommandUserStats:       h.handleUserStats,
		CommandRoomStats:       h.handleRoomStats,
		CommandAdminForceEnd:   h.handleAdminForceEnd,
		CommandAdminClearAll:   h.handleAdminClearAll,
		CommandAdminUsage:      h.handleAdminUsage,
		CommandHelp:            h.handleHelp,
		CommandUnknown:         h.handleUnknown,
	}

	return h
}

// ProcessCommand: 인입된 메시지와 파싱된 명령어를 바탕으로 적절한 핸들러를 실행하여 응답을 반환한다.
func (h *GameCommandHandler) ProcessCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	handler, ok := h.handlers[command.Kind]
	if !ok {
		h.logger.Debug("command_kind_unhandled", "kind", command.Kind)
		return []string{h.msgProvider.Get(qmessages.ErrorUnknownCommand)}, nil
	}

	return handler(ctx, message, command)
}

func (h *GameCommandHandler) handleStart(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	h.logger.Info("handle_start", "chat_id", message.ChatID, "categories", command.Categories)
	text, err := h.gameService.Start(ctx, message.ChatID, message.UserID, command.Categories)
	if err != nil {
		return nil, fmt.Errorf("start failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleAsk(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.gameService.Answer(ctx, message.ChatID, message.UserID, message.Sender, command.Question)
	if err != nil {
		return nil, fmt.Errorf("answer failed: %w", err)
	}
	if isAnswerCommand(command.Question) {
		return []string{text}, nil
	}

	main, hint, questionCount, statusErr := h.gameService.StatusSeparatedWithCount(ctx, message.ChatID)
	if statusErr != nil {
		return []string{text}, nil
	}
	messages := []string{main}
	if shouldShowHint(hint, questionCount) {
		messages = append(messages, hint)
	}
	return messages, nil
}

func (h *GameCommandHandler) handleChainedQuestion(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	// 첫 번째 질문 처리
	response, err := h.chainedQuestionHandler.Handle(
		ctx, message.ChatID, message.UserID, message.Sender,
		command.ChainQuestions, command.ChainCondition)
	if err != nil {
		return nil, fmt.Errorf("chained question failed: %w", err)
	}

	main, hint, questionCount, statusErr := h.gameService.StatusSeparatedWithCount(ctx, message.ChatID)
	if statusErr != nil {
		return []string{response}, nil
	}

	// 스킵 알림이 포함된 경우 (조건 불만족으로 체인 드랍 시) 상태 메시지와 함께 반환
	parts := strings.SplitN(response, "\n\n", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		main = main + "\n\n" + parts[1]
	}

	messages := []string{main}
	if shouldShowHint(hint, questionCount) {
		messages = append(messages, hint)
	}
	return messages, nil
}

func (h *GameCommandHandler) handleHints(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.gameService.GenerateHint(ctx, message.ChatID)
	if err != nil {
		return nil, fmt.Errorf("generate hint failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleStatus(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	main, hint, err := h.gameService.StatusSeparated(ctx, message.ChatID)
	if err != nil {
		return nil, fmt.Errorf("status failed: %w", err)
	}

	messages := []string{main}
	if hint != "" {
		messages = append(messages, hint)
	}
	return messages, nil
}

func (h *GameCommandHandler) handleSurrender(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.gameService.HandleSurrenderConsensus(ctx, message.ChatID, message.UserID)
	if err != nil {
		return nil, fmt.Errorf("surrender consensus failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleAgree(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.gameService.HandleSurrenderAgree(ctx, message.ChatID, message.UserID)
	if err != nil {
		return nil, fmt.Errorf("surrender agree failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleReject(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.gameService.HandleSurrenderReject(ctx, message.ChatID, message.UserID)
	if err != nil {
		return nil, fmt.Errorf("surrender reject failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleUserStats(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.statsService.GetUserStats(ctx, message.ChatID, message.UserID, message.Sender, command.TargetNickname)
	if err != nil {
		return nil, fmt.Errorf("get user stats failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleRoomStats(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.statsService.GetRoomStats(ctx, message.ChatID, command.RoomPeriod)
	if err != nil {
		return nil, fmt.Errorf("get room stats failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleAdminForceEnd(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.adminHandler.ForceEnd(ctx, message.ChatID, message.UserID)
	if err != nil {
		return nil, fmt.Errorf("admin force end failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleAdminClearAll(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.adminHandler.ClearAll(ctx, message.ChatID, message.UserID)
	if err != nil {
		return nil, fmt.Errorf("admin clear all failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleAdminUsage(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	text, err := h.usageHandler.Handle(ctx, message.ChatID, message.UserID, command.UsagePeriod, command.ModelOverride)
	if err != nil {
		return nil, fmt.Errorf("admin usage failed: %w", err)
	}
	return []string{text}, nil
}

func (h *GameCommandHandler) handleHelp(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	return []string{h.msgProvider.Get(qmessages.HelpMessage)}, nil
}

func (h *GameCommandHandler) handleUnknown(ctx context.Context, message mqmsg.InboundMessage, command Command) ([]string, error) {
	return []string{h.msgProvider.Get(qmessages.ErrorUnknownCommand)}, nil
}

func isAnswerCommand(question string) bool {
	question = strings.TrimSpace(question)
	return strings.HasPrefix(strings.ToLower(question), "정답")
}

// shouldShowHint: 힌트 라인을 표시할지 결정한다.
// HintDisplayInterval이 0이면 항상 표시, 양수면 해당 질문 횟수마다 표시.
func shouldShowHint(hint string, questionCount int) bool {
	if hint == "" {
		return false
	}
	if qconfig.HintDisplayInterval <= 0 {
		return true
	}
	return questionCount > 0 && questionCount%qconfig.HintDisplayInterval == 0
}
