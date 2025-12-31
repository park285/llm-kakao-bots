package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	llmv1 "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/grpcserver/pb/llm/v1"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler/shared"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/httperror"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/toon"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
	twentyquc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/twentyq"
)

const (
	twentyqSafetyBlockMessage   = shared.MsgSafetyBlock
	twentyqInvalidQuestionScale = shared.MsgInvalidQuestion
)

// LLMService: game-bot-go 내부 통신용 gRPC 서비스입니다.
type LLMService struct {
	llmv1.UnimplementedLLMServiceServer

	cfg *config.Config

	logger *slog.Logger

	client *gemini.Client
	guard  *guard.InjectionGuard
	store  *session.Store

	usageRepo *usage.Repository

	twentyqUsecase *twentyquc.Service

	twentyqPrompts *twentyq.Prompts
	topicLoader    *twentyq.TopicLoader

	turtlesoupPrompts *turtlesoup.Prompts
	puzzleLoader      *turtlesoup.PuzzleLoader
}

// NewLLMService: gRPC LLMService를 생성합니다.
func NewLLMService(
	cfg *config.Config,
	logger *slog.Logger,
	client *gemini.Client,
	injectionGuard *guard.InjectionGuard,
	store *session.Store,
	usageRepo *usage.Repository,
	twentyqPrompts *twentyq.Prompts,
	topicLoader *twentyq.TopicLoader,
	turtlesoupPrompts *turtlesoup.Prompts,
	puzzleLoader *turtlesoup.PuzzleLoader,
) *LLMService {
	return &LLMService{
		cfg:               cfg,
		logger:            logger,
		client:            client,
		guard:             injectionGuard,
		store:             store,
		usageRepo:         usageRepo,
		twentyqUsecase:    twentyquc.New(cfg, client, injectionGuard, store, twentyqPrompts, topicLoader, logger),
		twentyqPrompts:    twentyqPrompts,
		topicLoader:       topicLoader,
		turtlesoupPrompts: turtlesoupPrompts,
		puzzleLoader:      puzzleLoader,
	}
}

func (s *LLMService) GetModelConfig(ctx context.Context, _ *emptypb.Empty) (*llmv1.ModelConfigResponse, error) {
	if s.cfg == nil {
		return nil, status.Error(codes.Internal, "config is nil")
	}

	defaultModel := s.cfg.Gemini.DefaultModel
	hintsModel := s.cfg.Gemini.HintsModel
	answerModel := s.cfg.Gemini.AnswerModel
	verifyModel := s.cfg.Gemini.VerifyModel

	if hintsModel == "" {
		hintsModel = defaultModel
	}
	if answerModel == "" {
		answerModel = defaultModel
	}
	if verifyModel == "" {
		verifyModel = defaultModel
	}

	transportMode := "h1"
	if s.cfg.HTTP.HTTP2Enabled {
		transportMode = "h2c"
	}

	temperature := s.cfg.Gemini.TemperatureForModel(defaultModel)
	configuredTemperature := s.cfg.Gemini.Temperature

	return &llmv1.ModelConfigResponse{
		ModelDefault:          defaultModel,
		ModelHints:            &hintsModel,
		ModelAnswer:           &answerModel,
		ModelVerify:           &verifyModel,
		Temperature:           temperature,
		ConfiguredTemperature: &configuredTemperature,
		TimeoutSeconds:        int32(s.cfg.Gemini.TimeoutSeconds),
		MaxRetries:            int32(s.cfg.Gemini.MaxRetries),
		Http2Enabled:          s.cfg.HTTP.HTTP2Enabled,
		TransportMode:         &transportMode,
	}, nil
}

func (s *LLMService) GuardIsMalicious(ctx context.Context, req *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
	if req == nil || strings.TrimSpace(req.InputText) == "" {
		return nil, status.Error(codes.InvalidArgument, "input_text required")
	}
	if s.guard == nil {
		return nil, status.Error(codes.Internal, "guard not configured")
	}

	return &llmv1.GuardIsMaliciousResponse{
		Malicious: s.guard.IsMalicious(req.InputText),
	}, nil
}

// EndSession: 내부 통신에서 세션 히스토리 누수를 방지하기 위해 세션을 종료합니다.
func (s *LLMService) EndSession(ctx context.Context, req *llmv1.EndSessionRequest) (*llmv1.EndSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	sessionID := strings.TrimSpace(req.SessionId)
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "session store not configured")
	}

	if err := s.store.DeleteSession(ctx, sessionID); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		if errors.Is(err, session.ErrStoreDisabled) {
			return nil, status.Error(codes.FailedPrecondition, "session store disabled")
		}

		s.logError("grpc_end_session_failed", err)
		return nil, status.Error(codes.Internal, "end session failed")
	}

	return &llmv1.EndSessionResponse{
		Message: "session deleted",
		Id:      sessionID,
	}, nil
}

func (s *LLMService) TwentyQSelectTopic(ctx context.Context, req *llmv1.TwentyQSelectTopicRequest) (*llmv1.TwentyQSelectTopicResponse, error) {
	if req == nil {
		return nil, httperror.NewInvalidInput("request required")
	}
	if s.twentyqUsecase == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	topic, err := s.twentyqUsecase.SelectTopic(ctx, RequestIDFromContext(ctx), req.Category, req.BannedTopics, req.ExcludedCategories)
	if err != nil {
		return nil, fmt.Errorf("select topic: %w", err)
	}

	var detailsStruct *structpb.Struct
	if len(topic.Details) > 0 {
		st, err := structpb.NewStruct(topic.Details)
		if err != nil {
			s.logError("twentyq_topic_details_invalid", err)
			return nil, httperror.NewInternalError("topic details invalid")
		}
		detailsStruct = st
	}

	return &llmv1.TwentyQSelectTopicResponse{
		Name:     topic.Name,
		Category: topic.Category,
		Details:  detailsStruct,
	}, nil
}

func (s *LLMService) TwentyQGetCategories(ctx context.Context, _ *emptypb.Empty) (*llmv1.TwentyQGetCategoriesResponse, error) {
	categories := twentyq.AllCategories
	if s.twentyqUsecase != nil {
		categories = s.twentyqUsecase.Categories()
	}
	return &llmv1.TwentyQGetCategoriesResponse{Categories: categories}, nil
}

func (s *LLMService) TwentyQGenerateHints(ctx context.Context, req *llmv1.TwentyQGenerateHintsRequest) (*llmv1.TwentyQGenerateHintsResponse, error) {
	if req == nil {
		return nil, httperror.NewInvalidInput("request required")
	}
	if s.twentyqUsecase == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	var details map[string]any
	if req.Details != nil {
		details = req.Details.AsMap()
	}
	hints, err := s.twentyqUsecase.GenerateHints(ctx, RequestIDFromContext(ctx), twentyquc.HintsRequest{
		Target:   req.Target,
		Category: req.Category,
		Details:  details,
	})
	if err != nil {
		return nil, fmt.Errorf("generate hints: %w", err)
	}

	return &llmv1.TwentyQGenerateHintsResponse{
		Hints:            hints,
		ThoughtSignature: nil,
	}, nil
}

func (s *LLMService) TwentyQAnswerQuestion(ctx context.Context, req *llmv1.TwentyQAnswerQuestionRequest) (*llmv1.TwentyQAnswerQuestionResponse, error) {
	if req == nil {
		return nil, httperror.NewInvalidInput("request required")
	}
	if s.twentyqUsecase == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	var details map[string]any
	if req.Details != nil {
		details = req.Details.AsMap()
	}

	result, err := s.twentyqUsecase.AnswerQuestion(ctx, RequestIDFromContext(ctx), twentyquc.AnswerRequest{
		SessionID: req.SessionId,
		ChatID:    req.ChatId,
		Namespace: req.Namespace,
		Target:    req.Target,
		Category:  req.Category,
		Question:  req.Question,
		Details:   details,
	})
	if err != nil {
		return nil, fmt.Errorf("answer question: %w", err)
	}

	if result.RawText == "" {
		scale := twentyqInvalidQuestionScale
		return &llmv1.TwentyQAnswerQuestionResponse{
			Scale:            &scale,
			RawText:          twentyqSafetyBlockMessage,
			ThoughtSignature: nil,
		}, nil
	}

	var scale *string
	if result.ScaleText != "" {
		scale = &result.ScaleText
	}

	return &llmv1.TwentyQAnswerQuestionResponse{
		Scale:            scale,
		RawText:          result.RawText,
		ThoughtSignature: nil,
	}, nil
}

func (s *LLMService) TwentyQVerifyGuess(ctx context.Context, req *llmv1.TwentyQVerifyGuessRequest) (*llmv1.TwentyQVerifyGuessResponse, error) {
	if req == nil {
		return nil, httperror.NewInvalidInput("request required")
	}
	if s.twentyqUsecase == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	result, err := s.twentyqUsecase.VerifyGuess(ctx, RequestIDFromContext(ctx), req.Target, req.Guess)
	if err != nil {
		return nil, fmt.Errorf("verify guess: %w", err)
	}

	return &llmv1.TwentyQVerifyGuessResponse{
		Result:  result.Result,
		RawText: result.RawText,
	}, nil
}

func (s *LLMService) TwentyQNormalizeQuestion(ctx context.Context, req *llmv1.TwentyQNormalizeQuestionRequest) (*llmv1.TwentyQNormalizeQuestionResponse, error) {
	if req == nil {
		return nil, httperror.NewInvalidInput("request required")
	}
	if s.twentyqUsecase == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	result, err := s.twentyqUsecase.NormalizeQuestion(ctx, RequestIDFromContext(ctx), req.Question)
	if err != nil {
		return nil, fmt.Errorf("normalize question: %w", err)
	}

	return &llmv1.TwentyQNormalizeQuestionResponse{
		Normalized: result.Normalized,
		Original:   result.Original,
	}, nil
}

func (s *LLMService) TwentyQCheckSynonym(ctx context.Context, req *llmv1.TwentyQCheckSynonymRequest) (*llmv1.TwentyQCheckSynonymResponse, error) {
	if req == nil {
		return nil, httperror.NewInvalidInput("request required")
	}
	if s.twentyqUsecase == nil {
		return nil, httperror.NewInternalError("service not configured")
	}

	result, err := s.twentyqUsecase.CheckSynonym(ctx, RequestIDFromContext(ctx), req.Target, req.Guess)
	if err != nil {
		return nil, fmt.Errorf("check synonym: %w", err)
	}

	return &llmv1.TwentyQCheckSynonymResponse{
		Result:  result.Result,
		RawText: result.RawText,
	}, nil
}

func (s *LLMService) TurtleSoupGeneratePuzzle(ctx context.Context, req *llmv1.TurtleSoupGeneratePuzzleRequest) (*llmv1.TurtleSoupGeneratePuzzleResponse, error) {
	if req == nil {
		req = &llmv1.TurtleSoupGeneratePuzzleRequest{}
	}
	if s.turtlesoupPrompts == nil || s.puzzleLoader == nil || s.guard == nil || s.client == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	category := strings.TrimSpace(req.GetCategory())
	if category == "" {
		category = shared.DefaultCategory
	}

	difficulty := int(req.GetDifficulty())
	if req.Difficulty == nil {
		difficulty = shared.DefaultDifficulty
	}
	if difficulty < shared.MinDifficulty {
		difficulty = shared.MinDifficulty
	}
	if difficulty > shared.MaxDifficulty {
		difficulty = shared.MaxDifficulty
	}

	theme := strings.TrimSpace(req.GetTheme())
	if theme != "" {
		if err := s.guard.EnsureSafe(theme); err != nil {
			s.logError("turtlesoup_theme_guard_failed", err)
			return nil, fmt.Errorf("guard theme: %w", err)
		}
	}

	preset, err := s.puzzleLoader.GetRandomPuzzleByDifficulty(difficulty)
	if err == nil {
		return &llmv1.TurtleSoupGeneratePuzzleResponse{
			Title:      preset.Title,
			Scenario:   preset.Question,
			Solution:   preset.Answer,
			Category:   category,
			Difficulty: int32(preset.Difficulty),
			Hints:      []string{},
		}, nil
	}

	return s.generatePuzzleLLM(ctx, category, difficulty, theme)
}

func (s *LLMService) TurtleSoupGetRandomPuzzle(ctx context.Context, req *llmv1.TurtleSoupGetRandomPuzzleRequest) (*llmv1.TurtleSoupGetRandomPuzzleResponse, error) {
	if req == nil {
		req = &llmv1.TurtleSoupGetRandomPuzzleRequest{}
	}
	if s.puzzleLoader == nil {
		return nil, status.Error(codes.Internal, "puzzle loader not configured")
	}

	var puzzle turtlesoup.PuzzlePreset
	var err error

	if req.Difficulty == nil {
		puzzle, err = s.puzzleLoader.GetRandomPuzzle()
	} else {
		puzzle, err = s.puzzleLoader.GetRandomPuzzleByDifficulty(int(req.GetDifficulty()))
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "no puzzle available")
	}

	id := int32(puzzle.ID)
	title := puzzle.Title
	question := puzzle.Question
	answer := puzzle.Answer
	difficulty := int32(puzzle.Difficulty)

	return &llmv1.TurtleSoupGetRandomPuzzleResponse{
		Id:         &id,
		Title:      &title,
		Question:   &question,
		Answer:     &answer,
		Difficulty: &difficulty,
	}, nil
}

func (s *LLMService) TurtleSoupRewriteScenario(ctx context.Context, req *llmv1.TurtleSoupRewriteScenarioRequest) (*llmv1.TurtleSoupRewriteScenarioResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Scenario) == "" || strings.TrimSpace(req.Solution) == "" {
		return nil, status.Error(codes.InvalidArgument, "title, scenario, solution required")
	}
	if s.turtlesoupPrompts == nil || s.client == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	system, err := s.turtlesoupPrompts.RewriteSystem()
	if err != nil {
		return nil, status.Error(codes.Internal, "load rewrite system prompt failed")
	}
	userContent, err := s.turtlesoupPrompts.RewriteUser(req.Title, req.Scenario, req.Solution, int(req.Difficulty))
	if err != nil {
		return nil, status.Error(codes.Internal, "format rewrite user prompt failed")
	}

	scenario, solution, err := s.rewritePuzzle(ctx, system, userContent)
	if err != nil {
		return nil, err
	}

	return &llmv1.TurtleSoupRewriteScenarioResponse{
		Scenario:         scenario,
		Solution:         solution,
		OriginalScenario: req.Scenario,
		OriginalSolution: req.Solution,
	}, nil
}

func (s *LLMService) TurtleSoupAnswerQuestion(ctx context.Context, req *llmv1.TurtleSoupAnswerQuestionRequest) (*llmv1.TurtleSoupAnswerQuestionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if strings.TrimSpace(req.Scenario) == "" || strings.TrimSpace(req.Solution) == "" {
		return nil, status.Error(codes.InvalidArgument, "scenario, solution required")
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return nil, status.Error(codes.InvalidArgument, "question required")
	}

	if s.guard == nil || s.client == nil || s.store == nil || s.turtlesoupPrompts == nil || s.cfg == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	if err := s.guard.EnsureSafe(question); err != nil {
		s.logError("turtlesoup_question_guard_failed", err)
		return nil, fmt.Errorf("guard question: %w", err)
	}

	sessionID, history, _, err := s.resolveHistory(ctx, req.SessionId, req.ChatId, req.Namespace, "turtle-soup")
	if err != nil {
		s.logError("session_create_failed", err)
		return nil, err
	}
	historyPairs := countQAPairs(history)

	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := s.turtlesoupPrompts.AnswerSystemWithPuzzle(puzzleToon)
	if err != nil {
		return nil, status.Error(codes.Internal, "load answer system prompt failed")
	}
	userContent, err := s.turtlesoupPrompts.AnswerUser(question)
	if err != nil {
		return nil, status.Error(codes.Internal, "format answer user prompt failed")
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		History:      history,
		Task:         "answer",
	}, turtlesoup.AnswerSchema())
	if err != nil {
		return nil, fmt.Errorf("answer structured: %w", err)
	}

	rawAnswer, _ := shared.ParseStringField(payload, "answer")
	isImportant, _ := payload["important"].(bool)

	base := turtlesoup.AnswerType(rawAnswer)
	if rawAnswer == "" {
		base = turtlesoup.AnswerCannotAnswer
	}
	answerText := turtlesoup.FormatAnswerText(base, isImportant)
	if answerText == "" {
		answerText = string(turtlesoup.AnswerCannotAnswer)
	}

	items := buildTurtleHistoryItems(history, question, answerText)

	if err := s.appendTurtleHistory(ctx, sessionID, question, answerText); err != nil {
		s.logError("turtlesoup_append_history_failed", err)
	}

	return &llmv1.TurtleSoupAnswerQuestionResponse{
		Answer:        answerText,
		RawText:       rawAnswer,
		QuestionCount: int32(historyPairs + 1),
		History:       items,
	}, nil
}

func (s *LLMService) TurtleSoupValidateSolution(ctx context.Context, req *llmv1.TurtleSoupValidateSolutionRequest) (*llmv1.TurtleSoupValidateSolutionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if strings.TrimSpace(req.Solution) == "" {
		return nil, status.Error(codes.InvalidArgument, "solution required")
	}

	playerAnswer := strings.TrimSpace(req.PlayerAnswer)
	if playerAnswer == "" {
		return nil, status.Error(codes.InvalidArgument, "player_answer required")
	}

	if s.guard == nil || s.client == nil || s.turtlesoupPrompts == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	if err := s.guard.EnsureSafe(playerAnswer); err != nil {
		s.logError("turtlesoup_answer_guard_failed", err)
		return nil, fmt.Errorf("guard player answer: %w", err)
	}

	system, err := s.turtlesoupPrompts.ValidateSystem()
	if err != nil {
		return nil, status.Error(codes.Internal, "load validate system prompt failed")
	}
	userContent, err := s.turtlesoupPrompts.ValidateUser(req.Solution, playerAnswer)
	if err != nil {
		return nil, status.Error(codes.Internal, "format validate user prompt failed")
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "verify",
	}, turtlesoup.ValidateSchema())
	if err != nil {
		return nil, fmt.Errorf("validate structured: %w", err)
	}

	rawValue, parseErr := shared.ParseStringField(payload, "result")
	result := string(turtlesoup.ValidationNo)
	if parseErr == nil && rawValue != "" {
		result = rawValue
	}

	return &llmv1.TurtleSoupValidateSolutionResponse{
		Result:  result,
		RawText: rawValue,
	}, nil
}

func (s *LLMService) TurtleSoupGenerateHint(ctx context.Context, req *llmv1.TurtleSoupGenerateHintRequest) (*llmv1.TurtleSoupGenerateHintResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if strings.TrimSpace(req.Scenario) == "" || strings.TrimSpace(req.Solution) == "" {
		return nil, status.Error(codes.InvalidArgument, "scenario, solution required")
	}
	if req.Level <= 0 {
		return nil, status.Error(codes.InvalidArgument, "level must be positive")
	}
	if s.turtlesoupPrompts == nil || s.client == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	puzzleToon := toon.EncodePuzzle(req.Scenario, req.Solution, "", nil)
	system, err := s.turtlesoupPrompts.HintSystem()
	if err != nil {
		return nil, status.Error(codes.Internal, "load hint system prompt failed")
	}
	userContent, err := s.turtlesoupPrompts.HintUser(puzzleToon, int(req.Level))
	if err != nil {
		return nil, status.Error(codes.Internal, "format hint user prompt failed")
	}

	hint, err := s.generateHint(ctx, system, userContent)
	if err != nil {
		return nil, err
	}

	return &llmv1.TurtleSoupGenerateHintResponse{
		Hint:  hint,
		Level: req.Level,
	}, nil
}

func (s *LLMService) GetDailyUsage(ctx context.Context, _ *emptypb.Empty) (*llmv1.DailyUsageResponse, error) {
	if s.usageRepo == nil {
		return nil, status.Error(codes.Internal, "usage repository not configured")
	}

	row, err := s.usageRepo.GetDailyUsage(ctx, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("get daily usage: %w", err)
	}

	model := ""
	if s.cfg != nil {
		model = s.cfg.Gemini.DefaultModel
	}

	if row == nil {
		now := time.Now()
		return &llmv1.DailyUsageResponse{
			UsageDate:       now.Format("2006-01-02"),
			InputTokens:     0,
			OutputTokens:    0,
			TotalTokens:     0,
			ReasoningTokens: 0,
			RequestCount:    0,
			Model:           model,
		}, nil
	}

	return &llmv1.DailyUsageResponse{
		UsageDate:       row.UsageDate.Format("2006-01-02"),
		InputTokens:     row.InputTokens,
		OutputTokens:    row.OutputTokens,
		TotalTokens:     row.TotalTokens(),
		ReasoningTokens: row.ReasoningTokens,
		RequestCount:    row.RequestCount,
		Model:           model,
	}, nil
}

func (s *LLMService) GetRecentUsage(ctx context.Context, req *llmv1.GetRecentUsageRequest) (*llmv1.UsageListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if s.usageRepo == nil {
		return nil, status.Error(codes.Internal, "usage repository not configured")
	}

	days := int(req.Days)
	if days <= 0 {
		days = 7
	}

	rows, err := s.usageRepo.GetRecentUsage(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("get recent usage: %w", err)
	}

	model := ""
	if s.cfg != nil {
		model = s.cfg.Gemini.DefaultModel
	}

	out := &llmv1.UsageListResponse{
		Usages:            make([]*llmv1.DailyUsageResponse, 0, len(rows)),
		TotalInputTokens:  0,
		TotalOutputTokens: 0,
		TotalTokens:       0,
		TotalRequestCount: 0,
		Model:             model,
	}

	for _, row := range rows {
		item := &llmv1.DailyUsageResponse{
			UsageDate:       row.UsageDate.Format("2006-01-02"),
			InputTokens:     row.InputTokens,
			OutputTokens:    row.OutputTokens,
			TotalTokens:     row.TotalTokens(),
			ReasoningTokens: row.ReasoningTokens,
			RequestCount:    row.RequestCount,
			Model:           model,
		}
		out.Usages = append(out.Usages, item)
		out.TotalInputTokens += row.InputTokens
		out.TotalOutputTokens += row.OutputTokens
		out.TotalTokens += row.TotalTokens()
		out.TotalRequestCount += row.RequestCount
	}

	return out, nil
}

func (s *LLMService) GetTotalUsage(ctx context.Context, req *llmv1.GetTotalUsageRequest) (*llmv1.UsageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if s.usageRepo == nil {
		return nil, status.Error(codes.Internal, "usage repository not configured")
	}

	days := int(req.Days)
	if days <= 0 {
		days = 30
	}

	totalUsage, err := s.usageRepo.GetTotalUsage(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("get total usage: %w", err)
	}

	model := ""
	if s.cfg != nil {
		model = s.cfg.Gemini.DefaultModel
	}

	return &llmv1.UsageResponse{
		InputTokens:     totalUsage.InputTokens,
		OutputTokens:    totalUsage.OutputTokens,
		TotalTokens:     totalUsage.TotalTokens(),
		ReasoningTokens: totalUsage.ReasoningTokens,
		Model:           model,
	}, nil
}

func (s *LLMService) resolveHistory(
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

func (s *LLMService) appendTurtleHistory(ctx context.Context, sessionID string, question string, answer string) error {
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

func buildTurtleHistoryItems(history []llm.HistoryEntry, currentQuestion string, currentAnswer string) []*llmv1.TurtleSoupHistoryItem {
	items := make([]*llmv1.TurtleSoupHistoryItem, 0)

	for i := 0; i+1 < len(history); i++ {
		q := strings.TrimSpace(history[i].Content)
		a := strings.TrimSpace(history[i+1].Content)
		if !strings.HasPrefix(q, "Q:") || !strings.HasPrefix(a, "A:") {
			continue
		}
		items = append(items, &llmv1.TurtleSoupHistoryItem{
			Question: strings.TrimSpace(strings.TrimPrefix(q, "Q:")),
			Answer:   strings.TrimSpace(strings.TrimPrefix(a, "A:")),
		})
		i++
	}

	items = append(items, &llmv1.TurtleSoupHistoryItem{
		Question: currentQuestion,
		Answer:   currentAnswer,
	})
	return items
}

func (s *LLMService) generateHint(ctx context.Context, system string, userContent string) (string, error) {
	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoup.HintSchema())
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

func (s *LLMService) generatePuzzleLLM(ctx context.Context, category string, difficulty int, theme string) (*llmv1.TurtleSoupGeneratePuzzleResponse, error) {
	system, err := s.turtlesoupPrompts.GenerateSystem()
	if err != nil {
		return nil, fmt.Errorf("load puzzle system prompt: %w", err)
	}

	examples := s.puzzleLoader.GetExamples(difficulty, 3)
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

	userContent, err := s.turtlesoupPrompts.GenerateUser(category, difficulty, theme, examplesBlock)
	if err != nil {
		return nil, fmt.Errorf("format puzzle user prompt: %w", err)
	}

	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "hints",
	}, turtlesoup.PuzzleSchema())
	if err != nil {
		return nil, fmt.Errorf("generate puzzle structured: %w", err)
	}

	title, err := shared.ParseStringField(payload, "title")
	if err != nil {
		return nil, fmt.Errorf("parse title: %w", err)
	}
	scenario, err := shared.ParseStringField(payload, "scenario")
	if err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	solution, err := shared.ParseStringField(payload, "solution")
	if err != nil {
		return nil, fmt.Errorf("parse solution: %w", err)
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
		return nil, fmt.Errorf("parse hints: %w", err)
	}

	return &llmv1.TurtleSoupGeneratePuzzleResponse{
		Title:      strings.TrimSpace(title),
		Scenario:   strings.TrimSpace(scenario),
		Solution:   strings.TrimSpace(solution),
		Category:   respCategory,
		Difficulty: int32(respDifficulty),
		Hints:      hints,
	}, nil
}

func (s *LLMService) rewritePuzzle(ctx context.Context, system string, userContent string) (string, string, error) {
	payload, _, err := s.client.Structured(ctx, gemini.Request{
		Prompt:       userContent,
		SystemPrompt: system,
		Task:         "answer",
	}, turtlesoup.RewriteSchema())
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

func statusFromError(err error) error {
	if err == nil {
		return nil
	}

	var blocked *guard.BlockedError
	if errors.As(err, &blocked) {
		return status.Error(codes.InvalidArgument, blocked.Error())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "llm request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request canceled")
	}

	if apiErr := httperror.FromError(err); apiErr != nil {
		switch apiErr.Status {
		case 400:
			return status.Error(codes.InvalidArgument, apiErr.Message)
		case 401:
			return status.Error(codes.Unauthenticated, apiErr.Message)
		case 404:
			return status.Error(codes.NotFound, apiErr.Message)
		case 422:
			return status.Error(codes.InvalidArgument, apiErr.Message)
		case 429:
			return status.Error(codes.ResourceExhausted, apiErr.Message)
		case 503:
			return status.Error(codes.Unavailable, apiErr.Message)
		case 504:
			return status.Error(codes.DeadlineExceeded, apiErr.Message)
		default:
			return status.Error(codes.Internal, apiErr.Message)
		}
	}

	return status.Error(codes.Internal, err.Error())
}

func (s *LLMService) logError(event string, err error) {
	if s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(event, "err", err)
}
