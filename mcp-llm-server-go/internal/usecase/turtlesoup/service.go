package turtlesoup

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	turtlesoupdomain "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
)

// Service: TurtleSoup 비즈니스 로직(HTTP/gRPC 공용) 구현체입니다.
type Service struct {
	cfg     *config.Config
	client  gemini.LLM
	guard   *guard.InjectionGuard
	store   *session.Store
	prompts *turtlesoupdomain.Prompts
	loader  *turtlesoupdomain.PuzzleLoader
	logger  *slog.Logger
}

// New: TurtleSoup Service 인스턴스를 생성합니다.
func New(
	cfg *config.Config,
	client gemini.LLM,
	injectionGuard *guard.InjectionGuard,
	store *session.Store,
	prompts *turtlesoupdomain.Prompts,
	loader *turtlesoupdomain.PuzzleLoader,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		cfg:     cfg,
		client:  client,
		guard:   injectionGuard,
		store:   store,
		prompts: prompts,
		loader:  loader,
		logger:  logger,
	}
}

type HistoryItem struct {
	Question string
	Answer   string
}

type AnswerRequest struct {
	SessionID *string
	ChatID    *string
	Namespace *string
	Scenario  string
	Solution  string
	Question  string
}

type AnswerResult struct {
	Answer        string
	RawText       string
	QuestionCount int
	History       []HistoryItem
}

func (s *Service) AnswerQuestion(ctx context.Context, requestID string, req AnswerRequest) (AnswerResult, error) {
	if s == nil || s.guard == nil || s.client == nil || s.store == nil || s.prompts == nil || s.cfg == nil {
		return AnswerResult{}, httperror.NewInternalError("service not configured")
	}

	scenario := strings.TrimSpace(req.Scenario)
	solution := strings.TrimSpace(req.Solution)
	if scenario == "" || solution == "" {
		return AnswerResult{}, httperror.NewInvalidInput("scenario, solution required")
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return AnswerResult{}, httperror.NewInvalidInput("question required")
	}

	if err := s.guard.EnsureSafe(question); err != nil {
		s.logError("turtlesoup_question_guard_failed", err)
		return AnswerResult{}, fmt.Errorf("guard question: %w", err)
	}

	sessionID, history, _, err := s.resolveHistory(ctx, req.SessionID, req.ChatID, req.Namespace, "turtle-soup")
	if err != nil {
		s.logError("session_create_failed", err)
		return AnswerResult{}, err
	}
	historyPairs := countQAPairs(history)

	puzzleToon := toon.EncodePuzzle(scenario, solution, "", nil)
	system, err := s.prompts.AnswerSystemWithPuzzle(puzzleToon)
	if err != nil {
		s.logError("turtlesoup_answer_system_prompt_failed", err)
		return AnswerResult{}, httperror.NewInternalError("load answer system prompt failed")
	}
	userContent, err := s.prompts.AnswerUser(question)
	if err != nil {
		s.logError("turtlesoup_answer_user_prompt_failed", err)
		return AnswerResult{}, httperror.NewInternalError("format answer user prompt failed")
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		History:      history,
		Task:         "answer",
	}, turtlesoupdomain.AnswerSchema())
	if err != nil {
		return AnswerResult{}, fmt.Errorf("answer structured: %w", err)
	}

	rawAnswer, _ := shared.ParseStringField(payload, "answer")
	isImportant, _ := payload["important"].(bool)

	base := turtlesoupdomain.AnswerType(rawAnswer)
	if rawAnswer == "" {
		base = turtlesoupdomain.AnswerCannotAnswer
	}
	answerText := turtlesoupdomain.FormatAnswerText(base, isImportant)
	if answerText == "" {
		answerText = string(turtlesoupdomain.AnswerCannotAnswer)
	}

	items := buildTurtleHistoryItems(history, question, answerText)

	if err := s.appendTurtleHistory(ctx, sessionID, question, answerText); err != nil {
		s.logError("turtlesoup_append_history_failed", err)
	}

	s.logInfo(
		"turtlesoup_answered",
		"request_id", requestID,
		"session_id", sessionID,
		"history_pairs", historyPairs,
	)

	return AnswerResult{
		Answer:        answerText,
		RawText:       rawAnswer,
		QuestionCount: historyPairs + 1,
		History:       items,
	}, nil
}

type ValidateRequest struct {
	Solution     string
	PlayerAnswer string
}

type ValidateResult struct {
	Result  string
	RawText string
}

func (s *Service) ValidateSolution(ctx context.Context, req ValidateRequest) (ValidateResult, error) {
	if s == nil || s.guard == nil || s.client == nil || s.prompts == nil {
		return ValidateResult{}, httperror.NewInternalError("service not configured")
	}

	solution := strings.TrimSpace(req.Solution)
	if solution == "" {
		return ValidateResult{}, httperror.NewInvalidInput("solution required")
	}

	playerAnswer := strings.TrimSpace(req.PlayerAnswer)
	if playerAnswer == "" {
		return ValidateResult{}, httperror.NewInvalidInput("player_answer required")
	}

	if err := s.guard.EnsureSafe(playerAnswer); err != nil {
		s.logError("turtlesoup_answer_guard_failed", err)
		return ValidateResult{}, fmt.Errorf("guard player answer: %w", err)
	}

	system, err := s.prompts.ValidateSystem()
	if err != nil {
		s.logError("turtlesoup_validate_system_prompt_failed", err)
		return ValidateResult{}, httperror.NewInternalError("load validate system prompt failed")
	}
	userContent, err := s.prompts.ValidateUser(solution, playerAnswer)
	if err != nil {
		s.logError("turtlesoup_validate_user_prompt_failed", err)
		return ValidateResult{}, httperror.NewInternalError("format validate user prompt failed")
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "verify",
	}, turtlesoupdomain.ValidateSchema())
	if err != nil {
		return ValidateResult{}, fmt.Errorf("validate structured: %w", err)
	}

	rawValue, parseErr := shared.ParseStringField(payload, "result")
	result := string(turtlesoupdomain.ValidationNo)
	if parseErr == nil && rawValue != "" {
		result = rawValue
	}

	return ValidateResult{
		Result:  result,
		RawText: rawValue,
	}, nil
}

type HintRequest struct {
	Scenario string
	Solution string
	Level    int
}

func (s *Service) GenerateHint(ctx context.Context, req HintRequest) (string, error) {
	if s == nil || s.prompts == nil || s.client == nil {
		return "", httperror.NewInternalError("service not configured")
	}

	scenario := strings.TrimSpace(req.Scenario)
	solution := strings.TrimSpace(req.Solution)
	if scenario == "" || solution == "" {
		return "", httperror.NewInvalidInput("scenario, solution required")
	}
	if req.Level <= 0 {
		return "", httperror.NewInvalidInput("level must be positive")
	}

	puzzleToon := toon.EncodePuzzle(scenario, solution, "", nil)
	system, err := s.prompts.HintSystem()
	if err != nil {
		s.logError("turtlesoup_hint_system_prompt_failed", err)
		return "", httperror.NewInternalError("load hint system prompt failed")
	}
	userContent, err := s.prompts.HintUser(puzzleToon, req.Level)
	if err != nil {
		s.logError("turtlesoup_hint_user_prompt_failed", err)
		return "", httperror.NewInternalError("format hint user prompt failed")
	}

	return s.generateHint(ctx, system, userContent)
}

type RevealRequest struct {
	Scenario string
	Solution string
}

func (s *Service) Reveal(ctx context.Context, req RevealRequest) (string, error) {
	if s == nil || s.prompts == nil || s.client == nil {
		return "", httperror.NewInternalError("service not configured")
	}

	scenario := strings.TrimSpace(req.Scenario)
	solution := strings.TrimSpace(req.Solution)
	if scenario == "" || solution == "" {
		return "", httperror.NewInvalidInput("scenario, solution required")
	}

	puzzleToon := toon.EncodePuzzle(scenario, solution, "", nil)
	system, err := s.prompts.RevealSystem()
	if err != nil {
		s.logError("turtlesoup_reveal_system_prompt_failed", err)
		return "", httperror.NewInternalError("load reveal system prompt failed")
	}
	userContent, err := s.prompts.RevealUser(puzzleToon)
	if err != nil {
		s.logError("turtlesoup_reveal_user_prompt_failed", err)
		return "", httperror.NewInternalError("format reveal user prompt failed")
	}

	narrative, _, err := s.client.Chat(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "reveal",
	})
	if err != nil {
		return "", fmt.Errorf("reveal chat: %w", err)
	}

	return strings.TrimSpace(narrative), nil
}

type RewriteRequest struct {
	Title      string
	Scenario   string
	Solution   string
	Difficulty int
}

type RewriteResult struct {
	Scenario string
	Solution string
}

func (s *Service) RewriteScenario(ctx context.Context, req RewriteRequest) (RewriteResult, error) {
	if s == nil || s.prompts == nil || s.client == nil {
		return RewriteResult{}, httperror.NewInternalError("service not configured")
	}

	title := strings.TrimSpace(req.Title)
	scenario := strings.TrimSpace(req.Scenario)
	solution := strings.TrimSpace(req.Solution)
	if title == "" || scenario == "" || solution == "" {
		return RewriteResult{}, httperror.NewInvalidInput("title, scenario, solution required")
	}

	system, err := s.prompts.RewriteSystem()
	if err != nil {
		s.logError("turtlesoup_rewrite_system_prompt_failed", err)
		return RewriteResult{}, httperror.NewInternalError("load rewrite system prompt failed")
	}
	userContent, err := s.prompts.RewriteUser(title, scenario, solution, req.Difficulty)
	if err != nil {
		s.logError("turtlesoup_rewrite_user_prompt_failed", err)
		return RewriteResult{}, httperror.NewInternalError("format rewrite user prompt failed")
	}

	newScenario, newSolution, err := s.rewritePuzzle(ctx, system, userContent)
	if err != nil {
		return RewriteResult{}, err
	}
	return RewriteResult{Scenario: newScenario, Solution: newSolution}, nil
}

// RandomPuzzleResult: GetRandomPuzzle 응답 구조체입니다.
type RandomPuzzleResult struct {
	ID         int
	Title      string
	Question   string
	Answer     string
	Difficulty int
}

// GetRandomPuzzle: 프리셋 퍼즐 중 하나를 랜덤으로 반환합니다.
func (s *Service) GetRandomPuzzle() (RandomPuzzleResult, error) {
	if s == nil || s.loader == nil {
		return RandomPuzzleResult{}, httperror.NewInternalError("service not configured")
	}

	puzzle, err := s.loader.GetRandomPuzzle()
	if err != nil {
		return RandomPuzzleResult{}, err
	}

	return RandomPuzzleResult{
		ID:         puzzle.ID,
		Title:      puzzle.Title,
		Question:   puzzle.Question,
		Answer:     puzzle.Answer,
		Difficulty: puzzle.Difficulty,
	}, nil
}

// GetRandomPuzzleByDifficulty: 지정된 난이도의 프리셋 퍼즐 중 하나를 랜덤으로 반환합니다.
func (s *Service) GetRandomPuzzleByDifficulty(difficulty int) (RandomPuzzleResult, error) {
	if s == nil || s.loader == nil {
		return RandomPuzzleResult{}, httperror.NewInternalError("service not configured")
	}

	puzzle, err := s.loader.GetRandomPuzzleByDifficulty(difficulty)
	if err != nil {
		return RandomPuzzleResult{}, err
	}

	return RandomPuzzleResult{
		ID:         puzzle.ID,
		Title:      puzzle.Title,
		Question:   puzzle.Question,
		Answer:     puzzle.Answer,
		Difficulty: puzzle.Difficulty,
	}, nil
}

type GeneratePuzzleRequest struct {
	Category   string
	Difficulty *int
	Theme      string
}

type GeneratePuzzleResult struct {
	Title      string
	Scenario   string
	Solution   string
	Category   string
	Difficulty int
	Hints      []string
}

func (s *Service) GeneratePuzzle(ctx context.Context, req GeneratePuzzleRequest) (GeneratePuzzleResult, error) {
	if s == nil || s.prompts == nil || s.loader == nil || s.guard == nil || s.client == nil {
		return GeneratePuzzleResult{}, httperror.NewInternalError("service not configured")
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = shared.DefaultCategory
	}

	difficulty := shared.DefaultDifficulty
	if req.Difficulty != nil {
		difficulty = *req.Difficulty
	}
	if difficulty < shared.MinDifficulty {
		difficulty = shared.MinDifficulty
	}
	if difficulty > shared.MaxDifficulty {
		difficulty = shared.MaxDifficulty
	}

	theme := strings.TrimSpace(req.Theme)
	if theme != "" {
		if err := s.guard.EnsureSafe(theme); err != nil {
			s.logError("turtlesoup_theme_guard_failed", err)
			return GeneratePuzzleResult{}, fmt.Errorf("guard theme: %w", err)
		}
	}

	preset, err := s.loader.GetRandomPuzzleByDifficulty(difficulty)
	if err == nil {
		return GeneratePuzzleResult{
			Title:      preset.Title,
			Scenario:   preset.Question,
			Solution:   preset.Answer,
			Category:   category,
			Difficulty: preset.Difficulty,
			Hints:      []string{},
		}, nil
	}

	return s.generatePuzzleLLM(ctx, category, difficulty, theme)
}

func (s *Service) generateHint(ctx context.Context, system string, userContent string) (string, error) {
	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoupdomain.HintSchema())
	if err == nil {
		hint, parseErr := shared.ParseStringField(payload, "hint")
		if parseErr == nil && strings.TrimSpace(hint) != "" {
			return strings.TrimSpace(hint), nil
		}
	}

	rawText, _, err := s.client.Chat(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	})
	if err != nil {
		return "", fmt.Errorf("hint chat: %w", err)
	}

	parsed := strings.TrimSpace(rawText)
	if strings.HasPrefix(parsed, "```") {
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```json"))
		parsed = strings.TrimSpace(strings.TrimPrefix(parsed, "```"))
		if idx := strings.Index(parsed, "```"); idx >= 0 {
			parsed = strings.TrimSpace(parsed[:idx])
		}
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(parsed), &decoded); err == nil {
		if hint, err := shared.ParseStringField(decoded, "hint"); err == nil && strings.TrimSpace(hint) != "" {
			return strings.TrimSpace(hint), nil
		}
	}

	return parsed, nil
}

func (s *Service) generatePuzzleLLM(ctx context.Context, category string, difficulty int, theme string) (GeneratePuzzleResult, error) {
	system, err := s.prompts.GenerateSystem()
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("load puzzle system prompt: %w", err)
	}

	examples := s.loader.GetExamples(difficulty, 3)
	exampleLines := make([]string, 0, len(examples))
	for _, p := range examples {
		exampleLines = append(exampleLines, strings.Join([]string{
			"- 제목: " + p.Title,
			"  시나리오: " + p.Question,
			"  정답: " + p.Answer,
			"  난이도: " + strconv.Itoa(p.Difficulty),
		}, "\n"))
	}
	examplesBlock := strings.Join(exampleLines, "\n\n")

	userContent, err := s.prompts.GenerateUser(category, difficulty, theme, examplesBlock)
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("format puzzle user prompt: %w", err)
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoupdomain.PuzzleSchema())
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("generate puzzle structured: %w", err)
	}

	title, err := shared.ParseStringField(payload, "title")
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("parse title: %w", err)
	}
	scenario, err := shared.ParseStringField(payload, "scenario")
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("parse scenario: %w", err)
	}
	solution, err := shared.ParseStringField(payload, "solution")
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("parse solution: %w", err)
	}

	respCategory := strings.TrimSpace(valueOrEmptyString(payload, "category"))
	if respCategory == "" {
		respCategory = category
	}

	respDifficulty := difficulty
	if value, ok := payload["difficulty"]; ok {
		switch number := value.(type) {
		case float64:
			respDifficulty = int(number)
		case int:
			respDifficulty = number
		}
	}

	hints, err := shared.ParseStringSlice(payload, "hints")
	if err != nil {
		return GeneratePuzzleResult{}, fmt.Errorf("parse hints: %w", err)
	}

	return GeneratePuzzleResult{
		Title:      strings.TrimSpace(title),
		Scenario:   strings.TrimSpace(scenario),
		Solution:   strings.TrimSpace(solution),
		Category:   respCategory,
		Difficulty: respDifficulty,
		Hints:      hints,
	}, nil
}

func (s *Service) rewritePuzzle(ctx context.Context, system string, userContent string) (string, string, error) {
	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	}, turtlesoupdomain.RewriteSchema())
	if err != nil {
		return "", "", fmt.Errorf("rewrite structured: %w", err)
	}

	scenario, sErr := shared.ParseStringField(payload, "scenario")
	solution, aErr := shared.ParseStringField(payload, "solution")
	if sErr != nil || aErr != nil || strings.TrimSpace(scenario) == "" || strings.TrimSpace(solution) == "" {
		return "", "", httperror.NewInternalError("rewrite response invalid")
	}

	return strings.TrimSpace(scenario), strings.TrimSpace(solution), nil
}

func valueOrEmptyString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func (s *Service) resolveHistory(
	ctx context.Context,
	sessionID *string,
	chatID *string,
	namespace *string,
	defaultNamespace string,
) (string, []llm.HistoryEntry, int, error) {
	effectiveSessionID, derived := shared.ResolveSessionID(
		shared.ValueOrEmpty(sessionID),
		shared.ValueOrEmpty(chatID),
		shared.ValueOrEmpty(namespace),
		defaultNamespace,
	)

	if effectiveSessionID != "" && derived && sessionID == nil && s.store != nil && s.cfg != nil {
		now := time.Now()
		meta := session.Meta{
			ID:           effectiveSessionID,
			SystemPrompt: "",
			Model:        s.cfg.Gemini.DefaultModel,
			CreatedAt:    now,
			UpdatedAt:    now,
			MessageCount: 0,
		}
		if err := s.store.CreateSession(ctx, meta); err != nil {
			return "", nil, 0, fmt.Errorf("create session: %w", err)
		}
	}

	if effectiveSessionID == "" || s.store == nil {
		return "", []llm.HistoryEntry{}, 0, nil
	}

	history, err := s.store.GetHistory(ctx, effectiveSessionID)
	if err != nil {
		s.logError("session_history_failed", err)
		return effectiveSessionID, []llm.HistoryEntry{}, 0, nil
	}
	return effectiveSessionID, history, len(history), nil
}

func (s *Service) appendTurtleHistory(ctx context.Context, sessionID string, question string, answer string) error {
	if sessionID == "" || s.store == nil {
		return nil
	}
	if err := s.store.AppendHistory(
		ctx,
		sessionID,
		llm.HistoryEntry{Role: "user", Content: "Q: " + question},
		llm.HistoryEntry{Role: "assistant", Content: "A: " + answer},
	); err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	return nil
}

func countQAPairs(history []llm.HistoryEntry) int {
	pairs := 0
	for i := 0; i+1 < len(history); i++ {
		q := strings.TrimSpace(history[i].Content)
		a := strings.TrimSpace(history[i+1].Content)
		if strings.HasPrefix(q, "Q:") && strings.HasPrefix(a, "A:") {
			pairs++
			i++
		}
	}
	return pairs
}

func buildTurtleHistoryItems(history []llm.HistoryEntry, currentQuestion string, currentAnswer string) []HistoryItem {
	items := make([]HistoryItem, 0)

	for i := 0; i+1 < len(history); i++ {
		q := strings.TrimSpace(history[i].Content)
		a := strings.TrimSpace(history[i+1].Content)
		if !strings.HasPrefix(q, "Q:") || !strings.HasPrefix(a, "A:") {
			continue
		}
		items = append(items, HistoryItem{
			Question: strings.TrimSpace(strings.TrimPrefix(q, "Q:")),
			Answer:   strings.TrimSpace(strings.TrimPrefix(a, "A:")),
		})
		i++
	}

	items = append(items, HistoryItem{
		Question: currentQuestion,
		Answer:   currentAnswer,
	})
	return items
}

func (s *Service) logError(event string, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(event, "err", err)
}

func (s *Service) logInfo(event string, fields ...any) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Info(event, fields...)
}
