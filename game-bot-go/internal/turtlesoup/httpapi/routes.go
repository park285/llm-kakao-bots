package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	commonhttputil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httputil"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tssvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/service"
)

const (
	apiErrorGameAlreadyStarted = "GAME_ALREADY_STARTED"
	apiErrorSessionNotFound    = "SESSION_NOT_FOUND"
	apiErrorGameError          = "GAME_ERROR"
	apiErrorMaxHintsReached    = "MAX_HINTS_REACHED"
	apiErrorInvalidRequest     = "INVALID_REQUEST"
	apiErrorInternalError      = "INTERNAL_ERROR"
)

const maxBodyBytes = 1 << 20

// Register 는 동작을 수행한다.
func Register(mux *http.ServeMux, llmCfg tsconfig.LlmConfig, restClient *llmrest.Client, gameService *tssvc.GameService, logger *slog.Logger) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"goroutines": runtime.NumGoroutine(),
		})
	})

	mux.HandleFunc("GET /debug/models", func(w http.ResponseWriter, r *http.Request) {
		handleDebugModels(w, r, llmCfg, restClient, logger)
	})

	mux.HandleFunc("POST /api/game/start", func(w http.ResponseWriter, r *http.Request) {
		handleStartGame(w, r, gameService, logger)
	})
	mux.HandleFunc("POST /api/game/question", func(w http.ResponseWriter, r *http.Request) {
		handleAskQuestion(w, r, gameService, logger)
	})
	mux.HandleFunc("POST /api/game/solution", func(w http.ResponseWriter, r *http.Request) {
		handleSubmitSolution(w, r, gameService, logger)
	})
	mux.HandleFunc("POST /api/game/hint", func(w http.ResponseWriter, r *http.Request) {
		handleRequestHint(w, r, gameService, logger)
	})
	mux.HandleFunc("GET /api/game/status/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		handleGetStatus(w, r, gameService, logger)
	})
	mux.HandleFunc("DELETE /api/game/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		handleEndGame(w, r, gameService, logger)
	})
}

func handleDebugModels(w http.ResponseWriter, r *http.Request, llmCfg tsconfig.LlmConfig, restClient *llmrest.Client, logger *slog.Logger) {
	transport := LlmDebugTransport{
		BaseURL:               llmCfg.BaseURL,
		HTTP2Enabled:          llmCfg.HTTP2Enabled,
		TimeoutSeconds:        int64(llmCfg.Timeout.Seconds()),
		ConnectTimeoutSeconds: int64(llmCfg.ConnectTimeout.Seconds()),
	}

	modelConfig, err := restClient.GetModelConfig(r.Context())
	status := "ok"
	if err != nil {
		status = "unavailable"
		modelConfig = nil
		logger.Warn("debug_models_fetch_failed", "err", err)
	}

	resp := LlmDebugResponse{
		LlmRest:           transport,
		ModelConfig:       modelConfig,
		ModelConfigStatus: status,
	}

	if err := commonhttputil.WriteJSON(w, http.StatusOK, resp); err != nil {
		logger.Warn("debug_models_response_write_failed", "err", err)
	}
}

func handleStartGame(w http.ResponseWriter, r *http.Request, gameService *tssvc.GameService, logger *slog.Logger) {
	var req StartGameRequest
	if err := commonhttputil.ReadJSON(r, &req, maxBodyBytes); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, err.Error())
		return
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	req.UserID = strings.TrimSpace(req.UserID)
	req.ChatID = strings.TrimSpace(req.ChatID)
	if req.SessionID == "" || req.UserID == "" || req.ChatID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, "missing required fields")
		return
	}

	var categoryPtr *tsmodel.PuzzleCategory
	if req.Category != nil {
		parsed := tsmodel.ParsePuzzleCategory(*req.Category)
		categoryPtr = &parsed
	}

	state, err := gameService.StartGame(r.Context(), req.SessionID, req.UserID, req.ChatID, req.Difficulty, categoryPtr, req.Theme)
	if err != nil {
		respondGameError(w, err, "start_game_failed", logger)
		return
	}

	resp, err := toGameStateResponse(state)
	if err != nil {
		respondGameError(w, err, "start_game_response_build_failed", logger)
		return
	}
	_ = commonhttputil.WriteJSON(w, http.StatusOK, resp)
}

func handleAskQuestion(w http.ResponseWriter, r *http.Request, gameService *tssvc.GameService, logger *slog.Logger) {
	var req AskQuestionRequest
	if err := commonhttputil.ReadJSON(r, &req, maxBodyBytes); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, err.Error())
		return
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, "Session ID required")
		return
	}

	state, result, err := gameService.AskQuestion(r.Context(), req.SessionID, req.Question)
	if err != nil {
		respondGameError(w, err, "ask_question_failed", logger)
		return
	}

	_ = commonhttputil.WriteJSON(w, http.StatusOK, QuestionResponse{
		Answer:        result.Answer,
		QuestionCount: state.QuestionCount,
	})
}

func handleSubmitSolution(w http.ResponseWriter, r *http.Request, gameService *tssvc.GameService, logger *slog.Logger) {
	var req SubmitSolutionRequest
	if err := commonhttputil.ReadJSON(r, &req, maxBodyBytes); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, err.Error())
		return
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, "Session ID required")
		return
	}

	state, result, err := gameService.SubmitSolution(r.Context(), req.SessionID, req.Answer)
	if err != nil {
		respondGameError(w, err, "submit_solution_failed", logger)
		return
	}

	var solutionPtr *string
	if result == tsmodel.ValidationYes && state.Puzzle != nil && strings.TrimSpace(state.Puzzle.Solution) != "" {
		s := strings.TrimSpace(state.Puzzle.Solution)
		solutionPtr = &s
	}

	_ = commonhttputil.WriteJSON(w, http.StatusOK, SolutionResponse{
		Result:   string(result),
		Solution: solutionPtr,
	})
}

func handleRequestHint(w http.ResponseWriter, r *http.Request, gameService *tssvc.GameService, logger *slog.Logger) {
	var req HintRequest
	if err := commonhttputil.ReadJSON(r, &req, maxBodyBytes); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, err.Error())
		return
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, "Session ID required")
		return
	}

	state, hint, err := gameService.RequestHint(r.Context(), req.SessionID)
	if err != nil {
		respondGameError(w, err, "request_hint_failed", logger)
		return
	}

	remaining := tsconfig.GameMaxHints - state.HintsUsed
	if remaining < 0 {
		remaining = 0
	}

	_ = commonhttputil.WriteJSON(w, http.StatusOK, HintResponse{
		Hint:           hint,
		HintsUsed:      state.HintsUsed,
		HintsRemaining: remaining,
	})
}

func handleGetStatus(w http.ResponseWriter, r *http.Request, gameService *tssvc.GameService, logger *slog.Logger) {
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, "Session ID required")
		return
	}

	state, err := gameService.GetGameState(r.Context(), sessionID)
	if err != nil {
		respondGameError(w, err, "get_status_failed", logger)
		return
	}

	resp, err := toGameStateResponse(state)
	if err != nil {
		respondGameError(w, err, "get_status_response_build_failed", logger)
		return
	}
	_ = commonhttputil.WriteJSON(w, http.StatusOK, resp)
}

func handleEndGame(w http.ResponseWriter, r *http.Request, gameService *tssvc.GameService, logger *slog.Logger) {
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, apiErrorInvalidRequest, "Session ID required")
		return
	}

	if err := gameService.EndGame(r.Context(), sessionID); err != nil {
		respondGameError(w, err, "end_game_failed", logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toGameStateResponse(state tsmodel.GameState) (GameStateResponse, error) {
	if state.Puzzle == nil {
		return GameStateResponse{}, tserrors.GameNotStartedError{SessionID: state.SessionID}
	}

	elapsed := int64(0)
	if !state.StartedAt.IsZero() {
		elapsed = int64(time.Since(state.StartedAt).Seconds())
		if elapsed < 0 {
			elapsed = 0
		}
	}

	return GameStateResponse{
		SessionID:      state.SessionID,
		UserID:         state.UserID,
		ChatID:         state.ChatID,
		ScenarioTitle:  state.Puzzle.Title,
		Scenario:       state.Puzzle.Scenario,
		QuestionCount:  state.QuestionCount,
		HintsUsed:      state.HintsUsed,
		IsSolved:       state.IsSolved,
		ElapsedSeconds: elapsed,
	}, nil
}

func respondGameError(w http.ResponseWriter, err error, logEvent string, logger *slog.Logger) {
	if !tserrors.IsExpectedUserBehavior(err) {
		logger.Error(logEvent, "err", err)
	}

	status := http.StatusInternalServerError
	code := apiErrorInternalError

	var sessionNotFound tserrors.SessionNotFoundError
	var alreadyStarted tserrors.GameAlreadyStartedError
	var maxHintsReached tserrors.MaxHintsReachedError
	var invalidQuestion cerrors.InvalidQuestionError
	var invalidAnswer cerrors.InvalidAnswerError
	var notStarted tserrors.GameNotStartedError
	var alreadySolved tserrors.GameAlreadySolvedError
	var puzzleErr tserrors.PuzzleGenerationError
	var redisErr cerrors.RedisError
	var lockErr cerrors.LockError
	var accessDenied cerrors.AccessDeniedError
	var userBlocked cerrors.UserBlockedError
	var chatBlocked cerrors.ChatBlockedError
	var injectionErr cerrors.InputInjectionError
	var malformedInput cerrors.MalformedInputError

	switch {
	case errors.As(err, &sessionNotFound):
		status = http.StatusNotFound
		code = apiErrorSessionNotFound
	case errors.As(err, &alreadyStarted):
		status = http.StatusConflict
		code = apiErrorGameAlreadyStarted
	case errors.As(err, &maxHintsReached):
		status = http.StatusBadRequest
		code = apiErrorMaxHintsReached
	case errors.As(err, &invalidQuestion),
		errors.As(err, &invalidAnswer),
		errors.As(err, &notStarted),
		errors.As(err, &alreadySolved),
		errors.As(err, &puzzleErr),
		errors.As(err, &redisErr),
		errors.As(err, &lockErr),
		errors.As(err, &accessDenied),
		errors.As(err, &userBlocked),
		errors.As(err, &chatBlocked),
		errors.As(err, &injectionErr),
		errors.As(err, &malformedInput):
		status = http.StatusBadRequest
		code = apiErrorGameError
	}

	message := err.Error()
	_ = commonhttputil.WriteErrorJSON(w, status, code, message)
}
