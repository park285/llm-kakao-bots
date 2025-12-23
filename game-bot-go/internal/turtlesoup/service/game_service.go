package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tssecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/security"
)

// GameService 는 타입이다.
type GameService struct {
	restClient     *llmrest.Client
	sessionManager *GameSessionManager
	setupService   *GameSetupService
	injectionGuard tssecurity.InjectionGuard
	logger         *slog.Logger
}

// NewGameService 는 동작을 수행한다.
func NewGameService(
	restClient *llmrest.Client,
	sessionManager *GameSessionManager,
	setupService *GameSetupService,
	injectionGuard tssecurity.InjectionGuard,
	logger *slog.Logger,
) *GameService {
	return &GameService{
		restClient:     restClient,
		sessionManager: sessionManager,
		setupService:   setupService,
		injectionGuard: injectionGuard,
		logger:         logger,
	}
}

// StartGame 는 동작을 수행한다.
func (s *GameService) StartGame(
	ctx context.Context,
	sessionID string,
	userID string,
	chatID string,
	difficulty *int,
	category *tsmodel.PuzzleCategory,
	theme *string,
) (tsmodel.GameState, error) {
	var state tsmodel.GameState
	err := s.sessionManager.WithLock(ctx, sessionID, &userID, func(ctx context.Context) error {
		setup, err := s.setupService.PrepareNewGame(ctx, sessionID, userID, chatID, difficulty, category, theme)
		if err != nil {
			return err
		}
		s.logGameStarted(setup.State.SessionID, userID, setup.Puzzle)
		state = setup.State
		return nil
	})
	if err != nil {
		return tsmodel.GameState{}, err
	}
	return state, nil
}

// RegisterPlayer 는 동작을 수행한다.
func (s *GameService) RegisterPlayer(ctx context.Context, sessionID string, userID string) error {
	err := s.sessionManager.WithLock(ctx, sessionID, &userID, func(ctx context.Context) error {
		state, err := s.sessionManager.Load(ctx, sessionID)
		if err != nil {
			return err
		}
		if state == nil {
			return nil
		}

		base := *state
		if len(base.Players) == 0 && base.UserID != "" {
			base.Players = []string{base.UserID}
		}
		if slices.Contains(base.Players, userID) {
			return nil
		}

		updated := base.AddPlayer(userID)
		if err := s.sessionManager.Save(ctx, updated); err != nil {
			return err
		}
		return nil
	})
	return err
}

// AnswerQuestionResult 는 타입이다.
type AnswerQuestionResult struct {
	Answer        string
	QuestionCount int
	History       []tsmodel.HistoryEntry
}

// AskQuestion 는 동작을 수행한다.
func (s *GameService) AskQuestion(ctx context.Context, sessionID string, question string) (tsmodel.GameState, AnswerQuestionResult, error) {
	if !isValidQuestion(question) {
		return tsmodel.GameState{}, AnswerQuestionResult{}, tserrors.InvalidQuestionError{Message: "invalid question format"}
	}

	sanitizedQuestion, err := s.injectionGuard.ValidateOrThrow(ctx, question)
	if err != nil {
		return tsmodel.GameState{}, AnswerQuestionResult{}, fmt.Errorf("validate question failed: %w", err)
	}

	var answerResult AnswerQuestionResult
	var state tsmodel.GameState
	err = s.sessionManager.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
		loaded, loadErr := s.sessionManager.LoadOrThrow(ctx, sessionID)
		if loadErr != nil {
			return loadErr
		}
		if loaded.IsSolved {
			return tserrors.GameAlreadySolvedError{SessionID: sessionID}
		}
		if loaded.Puzzle == nil {
			return tserrors.GameNotStartedError{SessionID: sessionID}
		}

		result, restErr := s.restClient.TurtleSoupAnswerQuestion(
			ctx,
			sessionID,
			tsconfig.LlmNamespace,
			loaded.Puzzle.Scenario,
			loaded.Puzzle.Solution,
			sanitizedQuestion,
		)
		if restErr != nil {
			return fmt.Errorf("llm answer question failed: %w", restErr)
		}

		resolvedHistory := make([]tsmodel.HistoryEntry, 0, len(result.History))
		for _, item := range result.History {
			resolvedHistory = append(resolvedHistory, tsmodel.HistoryEntry{
				Question: item.Question,
				Answer:   item.Answer,
			})
		}

		mergedHistory, mergedQuestionCount := mergeHistory(loaded, resolvedHistory, result.QuestionCount)

		now := time.Now()
		loaded.QuestionCount = mergedQuestionCount
		loaded.History = mergedHistory
		loaded.LastActivityAt = now

		if saveErr := s.sessionManager.Save(ctx, loaded); saveErr != nil {
			return saveErr
		}
		_ = s.sessionManager.Refresh(ctx, sessionID)

		s.logger.Info("question_answered", "session_id", sessionID, "question_count", loaded.QuestionCount)

		answerResult = AnswerQuestionResult{
			Answer:        result.Answer,
			QuestionCount: loaded.QuestionCount,
			History:       slices.Clone(loaded.History),
		}
		state = loaded
		return nil
	})
	if err != nil {
		return tsmodel.GameState{}, AnswerQuestionResult{}, err
	}

	return state, answerResult, nil
}

// SubmitSolution 는 동작을 수행한다.
func (s *GameService) SubmitSolution(ctx context.Context, sessionID string, playerAnswer string) (tsmodel.GameState, tsmodel.ValidationResult, error) {
	if !isValidAnswer(playerAnswer) {
		return tsmodel.GameState{}, "", tserrors.InvalidAnswerError{Message: "invalid answer format"}
	}

	sanitizedAnswer, err := s.injectionGuard.ValidateOrThrow(ctx, playerAnswer)
	if err != nil {
		return tsmodel.GameState{}, "", fmt.Errorf("validate answer failed: %w", err)
	}

	var validation tsmodel.ValidationResult

	var state tsmodel.GameState
	err = s.sessionManager.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
		loaded, loadErr := s.sessionManager.LoadOrThrow(ctx, sessionID)
		if loadErr != nil {
			return loadErr
		}
		if loaded.IsSolved {
			return tserrors.GameAlreadySolvedError{SessionID: sessionID}
		}
		if loaded.Puzzle == nil {
			return tserrors.GameNotStartedError{SessionID: sessionID}
		}

		res, validateErr := s.restClient.TurtleSoupValidateSolution(ctx, sessionID, tsconfig.LlmNamespace, loaded.Puzzle.Solution, sanitizedAnswer)
		if validateErr != nil {
			return fmt.Errorf("llm validate solution failed: %w", validateErr)
		}

		parsed, parseErr := tsmodel.ParseValidationResult(res.Result)
		if parseErr != nil {
			return fmt.Errorf("parse validation result failed: %w", parseErr)
		}
		validation = parsed

		if validation == tsmodel.ValidationYes {
			loaded = loaded.MarkSolved()
		} else {
			loaded = loaded.UpdateActivity()
		}

		if saveErr := s.sessionManager.Save(ctx, loaded); saveErr != nil {
			return saveErr
		}

		s.logger.Info("solution_submitted", "session_id", sessionID, "result", validation)

		if validation == tsmodel.ValidationYes {
			if deleteErr := s.sessionManager.Delete(ctx, sessionID); deleteErr != nil {
				return deleteErr
			}

			_, _ = s.restClient.EndSessionByChat(ctx, tsconfig.LlmNamespace, sessionID)

			s.logger.Info("game_ended", "session_id", sessionID, "reason", "solved", "question_count", loaded.QuestionCount, "hints_used", loaded.HintsUsed)
		}

		state = loaded
		return nil
	})
	if err != nil {
		return tsmodel.GameState{}, "", err
	}
	return state, validation, nil
}

// SubmitAnswer 는 동작을 수행한다.
func (s *GameService) SubmitAnswer(ctx context.Context, sessionID string, answer string) (tsmodel.AnswerResult, error) {
	state, result, err := s.SubmitSolution(ctx, sessionID, answer)
	if err != nil {
		return tsmodel.AnswerResult{}, err
	}

	explanation := ""
	if result == tsmodel.ValidationYes && state.Puzzle != nil {
		explanation = state.Puzzle.Solution
	}

	return tsmodel.AnswerResult{
		Result:        result,
		QuestionCount: state.QuestionCount,
		HintCount:     state.HintsUsed,
		MaxHints:      tsconfig.GameMaxHints,
		HintsUsed:     slices.Clone(state.HintContents),
		Explanation:   explanation,
	}, nil
}

// RequestHint 는 동작을 수행한다.
func (s *GameService) RequestHint(ctx context.Context, sessionID string) (tsmodel.GameState, string, error) {
	var state tsmodel.GameState
	err := s.sessionManager.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
		loaded, err := s.sessionManager.LoadOrThrow(ctx, sessionID)
		if err != nil {
			return err
		}
		if loaded.IsSolved {
			return tserrors.GameAlreadySolvedError{SessionID: sessionID}
		}
		if loaded.HintsUsed >= tsconfig.GameMaxHints {
			return tserrors.MaxHintsReachedError{MaxHints: tsconfig.GameMaxHints}
		}
		if loaded.Puzzle == nil {
			return tserrors.GameNotStartedError{SessionID: sessionID}
		}

		res, hintErr := s.restClient.TurtleSoupGenerateHint(ctx, sessionID, tsconfig.LlmNamespace, loaded.Puzzle.Scenario, loaded.Puzzle.Solution, loaded.HintsUsed+1)
		if hintErr != nil {
			return fmt.Errorf("llm generate hint failed: %w", hintErr)
		}

		loaded = loaded.UseHint(res.Hint)
		if err := s.sessionManager.Save(ctx, loaded); err != nil {
			return err
		}

		s.logger.Info("hint_requested", "session_id", sessionID, "hints_used", loaded.HintsUsed)
		state = loaded
		return nil
	})
	if err != nil {
		return tsmodel.GameState{}, "", err
	}

	latestHint := ""
	if len(state.HintContents) > 0 {
		latestHint = state.HintContents[len(state.HintContents)-1]
	}
	return state, latestHint, nil
}

// Surrender 는 동작을 수행한다.
func (s *GameService) Surrender(ctx context.Context, sessionID string) (tsmodel.SurrenderResult, error) {
	var out tsmodel.SurrenderResult
	err := s.sessionManager.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
		state, err := s.sessionManager.LoadOrThrow(ctx, sessionID)
		if err != nil {
			return err
		}
		if state.Puzzle == nil {
			return tserrors.GameNotStartedError{SessionID: sessionID}
		}

		if err := s.sessionManager.Delete(ctx, sessionID); err != nil {
			return err
		}
		_, _ = s.restClient.EndSessionByChat(ctx, tsconfig.LlmNamespace, sessionID)

		s.logger.Info("game_surrendered", "session_id", sessionID, "question_count", state.QuestionCount, "hints_used", state.HintsUsed)

		out = tsmodel.SurrenderResult{
			Solution:  state.Puzzle.Solution,
			HintsUsed: slices.Clone(state.HintContents),
		}
		return nil
	})
	if err != nil {
		return tsmodel.SurrenderResult{}, err
	}
	return out, nil
}

// GetGameState 는 동작을 수행한다.
func (s *GameService) GetGameState(ctx context.Context, sessionID string) (tsmodel.GameState, error) {
	return s.sessionManager.LoadOrThrow(ctx, sessionID)
}

// EndGame 는 동작을 수행한다.
func (s *GameService) EndGame(ctx context.Context, sessionID string) error {
	err := s.sessionManager.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
		_ = s.sessionManager.Delete(ctx, sessionID)
		_, _ = s.restClient.EndSessionByChat(ctx, tsconfig.LlmNamespace, sessionID)
		s.logger.Info("game_ended", "session_id", sessionID)
		return nil
	})
	return err
}

func (s *GameService) logGameStarted(sessionID string, userID string, puzzle tsmodel.Puzzle) {
	s.logger.Info("game_started",
		"session_id", sessionID,
		"user_id", userID,
		"puzzle_title", puzzle.Title,
		"difficulty", puzzle.Difficulty,
		"solution", puzzle.Solution,
	)
}

func mergeHistory(state tsmodel.GameState, resolvedHistory []tsmodel.HistoryEntry, resolvedQuestionCount int) ([]tsmodel.HistoryEntry, int) {
	var lastEntry *tsmodel.HistoryEntry
	if len(resolvedHistory) > 0 {
		last := resolvedHistory[len(resolvedHistory)-1]
		lastEntry = &last
	}

	shouldAppend := false
	if lastEntry != nil {
		if len(state.History) == 0 {
			shouldAppend = true
		} else {
			shouldAppend = state.History[len(state.History)-1] != *lastEntry
		}
	}

	mergedHistory := state.History
	switch {
	case len(resolvedHistory) >= len(state.History):
		mergedHistory = resolvedHistory
	case shouldAppend && lastEntry != nil:
		mergedHistory = append(slices.Clone(state.History), *lastEntry)
	}

	mergedQuestionCount := max(
		resolvedQuestionCount,
		max(state.QuestionCount+boolToInt(shouldAppend), len(mergedHistory)),
	)

	return mergedHistory, mergedQuestionCount
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func isValidQuestion(question string) bool {
	q := question
	if len(q) < tsconfig.ValidationMinQuestionLength || len(q) > tsconfig.ValidationMaxQuestionLength {
		return false
	}
	for _, r := range q {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

func isValidAnswer(answer string) bool {
	a := answer
	if len(a) < tsconfig.ValidationMinAnswerLength || len(a) > tsconfig.ValidationMaxAnswerLength {
		return false
	}
	for _, r := range a {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}
