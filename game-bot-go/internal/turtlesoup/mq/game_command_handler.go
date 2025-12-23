package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tssvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/service"
)

// GameCommandHandler 는 타입이다.
type GameCommandHandler struct {
	gameService      *tssvc.GameService
	surrenderHandler *SurrenderHandler
	msgProvider      *messageprovider.Provider
	messageBuilder   *MessageBuilder
	logger           *slog.Logger
}

// NewGameCommandHandler 는 동작을 수행한다.
func NewGameCommandHandler(
	gameService *tssvc.GameService,
	surrenderHandler *SurrenderHandler,
	msgProvider *messageprovider.Provider,
	messageBuilder *MessageBuilder,
	logger *slog.Logger,
) *GameCommandHandler {
	return &GameCommandHandler{
		gameService:      gameService,
		surrenderHandler: surrenderHandler,
		msgProvider:      msgProvider,
		messageBuilder:   messageBuilder,
		logger:           logger,
	}
}

// ProcessCommand 는 동작을 수행한다.
func (h *GameCommandHandler) ProcessCommand(ctx context.Context, message mqmsg.InboundMessage, command Command) (string, error) {
	if h.shouldRegisterPlayer(command) {
		_ = h.gameService.RegisterPlayer(ctx, message.ChatID, message.UserID)
	}

	switch command.Kind {
	case CommandStart:
		return h.handleStart(ctx, message, command)
	case CommandAsk:
		return h.handleAsk(ctx, message, command.Question)
	case CommandAnswer:
		return h.handleAnswer(ctx, message, command.Answer)
	case CommandHint:
		return h.handleHint(ctx, message)
	case CommandProblem:
		return h.handleProblem(ctx, message)
	case CommandSurrender:
		return h.surrenderHandler.HandleConsensus(ctx, message.ChatID, message.UserID)
	case CommandAgree:
		return h.surrenderHandler.HandleAgree(ctx, message.ChatID, message.UserID)
	case CommandSummary:
		return h.handleSummary(ctx, message)
	case CommandHelp:
		return h.msgProvider.Get(tsmessages.HelpMessage), nil
	case CommandUnknown:
		return h.msgProvider.Get(tsmessages.ErrorUnknownCommand), nil
	default:
		return h.msgProvider.Get(tsmessages.ErrorUnknownCommand), nil
	}
}

func (h *GameCommandHandler) handleStart(ctx context.Context, message mqmsg.InboundMessage, command Command) (string, error) {
	selection := h.resolveDifficulty(command)
	startState, err := h.startOrResumeGame(ctx, message, selection.Value)
	if err != nil {
		return "", fmt.Errorf("start or resume game failed: %w", err)
	}

	if startState.IsResuming {
		return h.composeStartReply(selection, startState.State, true), nil
	}

	scenario := h.buildScenarioMessage(startState.State, false)
	instruction := h.buildInstructionMessage(startState.State, false)
	if selection.Warning != "" {
		return selection.Warning + "\n\n" + scenario + "\n\n" + instruction, nil
	}
	return scenario + "\n\n" + instruction, nil
}

func (h *GameCommandHandler) handleAsk(ctx context.Context, message mqmsg.InboundMessage, question string) (string, error) {
	h.logger.Debug("handleAsk_start", "session_id", message.ChatID)
	_, result, err := h.gameService.AskQuestion(ctx, message.ChatID, question)
	if err != nil {
		return "", fmt.Errorf("ask question failed: %w", err)
	}
	h.logger.Debug("handleAsk_complete", "session_id", message.ChatID)

	return h.msgProvider.Get(tsmessages.AnswerResponseSingle, messageprovider.P("answer", result.Answer)), nil
}

func (h *GameCommandHandler) handleAnswer(ctx context.Context, message mqmsg.InboundMessage, answer string) (string, error) {
	result, err := h.gameService.SubmitAnswer(ctx, message.ChatID, answer)
	if err != nil {
		return "", fmt.Errorf("submit answer failed: %w", err)
	}

	switch {
	case result.IsCorrect():
		return h.msgProvider.Get(
			tsmessages.AnswerCorrect,
			messageprovider.P("explanation", result.Explanation),
			messageprovider.P("questionCount", result.QuestionCount),
			messageprovider.P("hintCount", result.HintCount),
			messageprovider.P("maxHints", result.MaxHints),
			messageprovider.P("hintBlock", h.messageBuilder.BuildHintBlock(result.HintsUsed)),
		), nil
	case result.IsClose():
		return h.msgProvider.Get(tsmessages.AnswerCloseCall), nil
	default:
		return h.msgProvider.Get(tsmessages.AnswerIncorrect), nil
	}
}

func (h *GameCommandHandler) handleHint(ctx context.Context, message mqmsg.InboundMessage) (string, error) {
	state, hint, err := h.gameService.RequestHint(ctx, message.ChatID)
	if err != nil {
		return "", fmt.Errorf("request hint failed: %w", err)
	}
	return h.msgProvider.Get(
		tsmessages.HintGenerated,
		messageprovider.P("hintNumber", state.HintsUsed),
		messageprovider.P("content", hint),
	), nil
}

func (h *GameCommandHandler) handleProblem(ctx context.Context, message mqmsg.InboundMessage) (string, error) {
	state, err := h.gameService.GetGameState(ctx, message.ChatID)
	if err != nil {
		return "", fmt.Errorf("get game state failed: %w", err)
	}

	scenario := h.msgProvider.Get(tsmessages.FallbackPuzzleNotFound)
	if state.Puzzle != nil && strings.TrimSpace(state.Puzzle.Scenario) != "" {
		scenario = state.Puzzle.Scenario
	}

	return h.msgProvider.Get(
		tsmessages.ProblemDisplay,
		messageprovider.P("scenario", scenario),
		messageprovider.P("questionCount", state.QuestionCount),
		messageprovider.P("hintCount", state.HintsUsed),
		messageprovider.P("maxHints", tsconfig.GameMaxHints),
	), nil
}

func (h *GameCommandHandler) handleSummary(ctx context.Context, message mqmsg.InboundMessage) (string, error) {
	state, err := h.gameService.GetGameState(ctx, message.ChatID)
	if err != nil {
		return "", fmt.Errorf("get game state failed: %w", err)
	}
	return h.messageBuilder.BuildSummary(state.History), nil
}

type difficultySelection struct {
	Value   *int
	Warning string
}

func (h *GameCommandHandler) resolveDifficulty(command Command) difficultySelection {
	if command.HasInvalidInput {
		return difficultySelection{
			Value: nil,
			Warning: h.msgProvider.Get(
				tsmessages.StartInvalidDifficulty,
				messageprovider.P("min", tsconfig.PuzzleMinDifficulty),
				messageprovider.P("max", tsconfig.PuzzleMaxDifficulty),
			),
		}
	}

	if command.Difficulty == nil {
		return difficultySelection{Value: nil}
	}

	desired := *command.Difficulty
	if desired >= tsconfig.PuzzleMinDifficulty && desired <= tsconfig.PuzzleMaxDifficulty {
		return difficultySelection{Value: &desired}
	}

	return difficultySelection{
		Value: nil,
		Warning: h.msgProvider.Get(
			tsmessages.StartInvalidDifficulty,
			messageprovider.P("min", tsconfig.PuzzleMinDifficulty),
			messageprovider.P("max", tsconfig.PuzzleMaxDifficulty),
		),
	}
}

type startState struct {
	State      tsmodel.GameState
	IsResuming bool
}

func (h *GameCommandHandler) startOrResumeGame(ctx context.Context, message mqmsg.InboundMessage, difficulty *int) (startState, error) {
	state, err := h.gameService.StartGame(ctx, message.ChatID, message.UserID, message.ChatID, difficulty, nil, nil)
	if err == nil {
		return startState{State: state, IsResuming: false}, nil
	}

	var alreadyStarted *tserrors.GameAlreadyStartedError
	if errors.As(err, &alreadyStarted) {
		resumed, errLoad := h.gameService.GetGameState(ctx, message.ChatID)
		if errLoad != nil {
			return startState{}, fmt.Errorf("resume game load failed: %w", errLoad)
		}
		h.logger.Debug("game_already_started_resuming", "session_id", message.ChatID)
		return startState{State: resumed, IsResuming: true}, nil
	}

	return startState{}, fmt.Errorf("start game failed: %w", err)
}

func (h *GameCommandHandler) buildScenarioMessage(state tsmodel.GameState, isResuming bool) string {
	scenario := h.msgProvider.Get(tsmessages.FallbackPuzzleNotFound)
	difficulty := tsconfig.PuzzleDefaultDifficulty
	if state.Puzzle != nil {
		if strings.TrimSpace(state.Puzzle.Scenario) != "" {
			scenario = state.Puzzle.Scenario
		}
		difficulty = state.Puzzle.Difficulty
	}

	if isResuming {
		return h.msgProvider.Get(tsmessages.StartResume, messageprovider.P("scenario", scenario))
	}

	return h.msgProvider.Get(
		tsmessages.StartScenario,
		messageprovider.P("scenario", scenario),
		messageprovider.P("difficulty", buildDifficultyStars(difficulty)),
	)
}

func (h *GameCommandHandler) composeStartReply(selection difficultySelection, state tsmodel.GameState, isResuming bool) string {
	scenarioMessage := h.buildScenarioMessage(state, isResuming)
	instruction := h.buildInstructionMessage(state, isResuming)

	parts := make([]string, 0, 3)
	if !isResuming && selection.Warning != "" {
		parts = append(parts, selection.Warning)
	}
	parts = append(parts, scenarioMessage, instruction)
	return strings.Join(parts, "\n\n")
}

func (h *GameCommandHandler) buildInstructionMessage(state tsmodel.GameState, isResuming bool) string {
	if isResuming {
		return h.msgProvider.Get(
			tsmessages.StartResumeStatus,
			messageprovider.P("questionCount", state.QuestionCount),
			messageprovider.P("hintCount", state.HintsUsed),
		)
	}
	return h.msgProvider.Get(tsmessages.StartInstruction)
}

func buildDifficultyStars(difficulty int) string {
	clamped := difficulty
	if clamped < tsconfig.PuzzleMinDifficulty {
		clamped = tsconfig.PuzzleMinDifficulty
	}
	if clamped > tsconfig.PuzzleMaxDifficulty {
		clamped = tsconfig.PuzzleMaxDifficulty
	}

	return strings.Repeat("★", clamped) + strings.Repeat("☆", tsconfig.PuzzleMaxDifficulty-clamped)
}

func (h *GameCommandHandler) shouldRegisterPlayer(command Command) bool {
	switch command.Kind {
	case CommandHelp, CommandUnknown:
		return false
	default:
		return true
	}
}
