package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tssecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/security"
)

// GameService: TurtleSoup 게임의 핵심 비즈니스 로직을 담당하는 서비스입니다.
// 게임 시작, 질문 처리, 정답 제출, 힌트, 항복 등의 기능을 제공합니다.
type GameService struct {
	restClient     *llmrest.Client
	sessionManager *GameSessionManager
	setupService   *GameSetupService
	injectionGuard tssecurity.InjectionGuard
	logger         *slog.Logger
}

// NewGameService: GameService 인스턴스를 생성합니다.
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

// StartGame: 새 게임을 시작하고 퍼즐을 생성합니다.
// 난이도, 카테고리, 테마를 선택적으로 지정할 수 있습니다.
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

// RegisterPlayer: 진행 중인 게임에 플레이어를 등록합니다.
// 이미 등록된 플레이어는 무시됩니다.
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

// AnswerQuestionResult: 질문에 대한 응답 결과를 담는 구조체입니다.
type AnswerQuestionResult struct {
	Answer        string
	QuestionCount int
	History       []tsmodel.HistoryEntry
}

// AskQuestion: LLM에 질문을 전달하고 예/아니오 답변을 받습니다.
// 질문 유효성 검증 및 Injection Guard를 거친 후 처리합니다.
func (s *GameService) AskQuestion(ctx context.Context, sessionID string, question string) (tsmodel.GameState, AnswerQuestionResult, error) {
	if !isValidQuestion(question) {
		return tsmodel.GameState{}, AnswerQuestionResult{}, cerrors.InvalidQuestionError{Message: "invalid question format"}
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

		chatID := loaded.ChatID
		if chatID == "" {
			chatID = sessionID
		}

		result, restErr := s.restClient.TurtleSoupAnswerQuestion(
			ctx,
			chatID,
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

// SubmitSolution: 플레이어의 정답 제출을 검증합니다.
// 정답이면 게임을 종료하고, 오답이면 계속 진행합니다.
func (s *GameService) SubmitSolution(ctx context.Context, sessionID string, playerAnswer string) (tsmodel.GameState, tsmodel.ValidationResult, error) {
	if !isValidAnswer(playerAnswer) {
		return tsmodel.GameState{}, "", cerrors.InvalidAnswerError{Message: "invalid answer format"}
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

		chatID := loaded.ChatID
		if chatID == "" {
			chatID = sessionID
		}

		res, validateErr := s.restClient.TurtleSoupValidateSolution(ctx, chatID, tsconfig.LlmNamespace, loaded.Puzzle.Solution, sanitizedAnswer)
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

			_, _ = s.restClient.EndSessionByChat(ctx, tsconfig.LlmNamespace, chatID)

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

// SubmitAnswer: SubmitSolution의 래퍼로, AnswerResult 형태로 결과를 반환합니다.
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

// RequestHint: LLM에 힌트를 요청합니다.
// 최대 힌트 횟수를 초과하면 에러를 반환합니다.
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

		chatID := loaded.ChatID
		if chatID == "" {
			chatID = sessionID
		}

		res, hintErr := s.restClient.TurtleSoupGenerateHint(ctx, chatID, tsconfig.LlmNamespace, loaded.Puzzle.Scenario, loaded.Puzzle.Solution, loaded.HintsUsed+1)
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

// Surrender: 게임을 포기하고 정답을 공개합니다.
// 세션을 삭제하고 LLM 세션도 종료합니다.
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

		chatID := state.ChatID
		if chatID == "" {
			chatID = sessionID
		}

		if err := s.sessionManager.Delete(ctx, sessionID); err != nil {
			return err
		}
		_, _ = s.restClient.EndSessionByChat(ctx, tsconfig.LlmNamespace, chatID)

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

// GetGameState: 현재 게임 상태를 조회합니다.
func (s *GameService) GetGameState(ctx context.Context, sessionID string) (tsmodel.GameState, error) {
	return s.sessionManager.LoadOrThrow(ctx, sessionID)
}

// EndGame: 게임을 강제 종료하고 모든 리소스를 정리합니다.
func (s *GameService) EndGame(ctx context.Context, sessionID string) error {
	err := s.sessionManager.WithOwnerLock(ctx, sessionID, func(ctx context.Context) error {
		loaded, loadErr := s.sessionManager.Load(ctx, sessionID)
		if loadErr != nil {
			s.logger.Warn("load_session_failed", "session_id", sessionID, "err", loadErr)
		}

		chatID := sessionID
		if loaded != nil && loaded.ChatID != "" {
			chatID = loaded.ChatID
		}

		_ = s.sessionManager.Delete(ctx, sessionID)
		_, _ = s.restClient.EndSessionByChat(ctx, tsconfig.LlmNamespace, chatID)
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
