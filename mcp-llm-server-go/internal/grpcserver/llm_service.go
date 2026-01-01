package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
	turtlesoupuc "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usecase/turtlesoup"
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

	guard *guard.InjectionGuard
	store *session.Store

	usageRepo *usage.Repository

	twentyqUsecase    *twentyquc.Service
	turtlesoupUsecase *turtlesoupuc.Service
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
		guard:             injectionGuard,
		store:             store,
		usageRepo:         usageRepo,
		twentyqUsecase:    twentyquc.New(cfg, client, injectionGuard, store, twentyqPrompts, topicLoader, logger),
		turtlesoupUsecase: turtlesoupuc.New(cfg, client, injectionGuard, store, turtlesoupPrompts, puzzleLoader, logger),
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
	if s.turtlesoupUsecase == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	var difficultyPtr *int
	if req.Difficulty != nil {
		d := int(req.GetDifficulty())
		difficultyPtr = &d
	}

	result, err := s.turtlesoupUsecase.GeneratePuzzle(ctx, turtlesoupuc.GeneratePuzzleRequest{
		Category:   req.GetCategory(),
		Difficulty: difficultyPtr,
		Theme:      req.GetTheme(),
	})
	if err != nil {
		return nil, fmt.Errorf("generate puzzle: %w", err)
	}

	return &llmv1.TurtleSoupGeneratePuzzleResponse{
		Title:      result.Title,
		Scenario:   result.Scenario,
		Solution:   result.Solution,
		Category:   result.Category,
		Difficulty: int32(result.Difficulty),
		Hints:      result.Hints,
	}, nil
}

func (s *LLMService) TurtleSoupGetRandomPuzzle(ctx context.Context, req *llmv1.TurtleSoupGetRandomPuzzleRequest) (*llmv1.TurtleSoupGetRandomPuzzleResponse, error) {
	if req == nil {
		req = &llmv1.TurtleSoupGetRandomPuzzleRequest{}
	}
	if s.turtlesoupUsecase == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	var puzzle turtlesoupuc.RandomPuzzleResult
	var err error

	if req.Difficulty == nil {
		puzzle, err = s.turtlesoupUsecase.GetRandomPuzzle()
	} else {
		puzzle, err = s.turtlesoupUsecase.GetRandomPuzzleByDifficulty(int(req.GetDifficulty()))
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
	if s.turtlesoupUsecase == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	result, err := s.turtlesoupUsecase.RewriteScenario(ctx, turtlesoupuc.RewriteRequest{
		Title:      req.Title,
		Scenario:   req.Scenario,
		Solution:   req.Solution,
		Difficulty: int(req.Difficulty),
	})
	if err != nil {
		return nil, fmt.Errorf("rewrite scenario: %w", err)
	}

	return &llmv1.TurtleSoupRewriteScenarioResponse{
		Scenario:         result.Scenario,
		Solution:         result.Solution,
		OriginalScenario: req.Scenario,
		OriginalSolution: req.Solution,
	}, nil
}

func (s *LLMService) TurtleSoupAnswerQuestion(ctx context.Context, req *llmv1.TurtleSoupAnswerQuestionRequest) (*llmv1.TurtleSoupAnswerQuestionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if s.turtlesoupUsecase == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	result, err := s.turtlesoupUsecase.AnswerQuestion(ctx, RequestIDFromContext(ctx), turtlesoupuc.AnswerRequest{
		SessionID: req.SessionId,
		ChatID:    req.ChatId,
		Namespace: req.Namespace,
		Scenario:  req.Scenario,
		Solution:  req.Solution,
		Question:  req.Question,
	})
	if err != nil {
		return nil, fmt.Errorf("answer question: %w", err)
	}

	history := make([]*llmv1.TurtleSoupHistoryItem, 0, len(result.History))
	for _, item := range result.History {
		history = append(history, &llmv1.TurtleSoupHistoryItem{
			Question: item.Question,
			Answer:   item.Answer,
		})
	}

	return &llmv1.TurtleSoupAnswerQuestionResponse{
		Answer:        result.Answer,
		RawText:       result.RawText,
		QuestionCount: int32(result.QuestionCount),
		History:       history,
	}, nil
}

func (s *LLMService) TurtleSoupValidateSolution(ctx context.Context, req *llmv1.TurtleSoupValidateSolutionRequest) (*llmv1.TurtleSoupValidateSolutionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if s.turtlesoupUsecase == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	result, err := s.turtlesoupUsecase.ValidateSolution(ctx, turtlesoupuc.ValidateRequest{
		Solution:     req.Solution,
		PlayerAnswer: req.PlayerAnswer,
	})
	if err != nil {
		return nil, fmt.Errorf("validate solution: %w", err)
	}

	return &llmv1.TurtleSoupValidateSolutionResponse{
		Result:  result.Result,
		RawText: result.RawText,
	}, nil
}

func (s *LLMService) TurtleSoupGenerateHint(ctx context.Context, req *llmv1.TurtleSoupGenerateHintRequest) (*llmv1.TurtleSoupGenerateHintResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if s.turtlesoupUsecase == nil {
		return nil, status.Error(codes.Internal, "service not configured")
	}

	hint, err := s.turtlesoupUsecase.GenerateHint(ctx, turtlesoupuc.HintRequest{
		Scenario: req.Scenario,
		Solution: req.Solution,
		Level:    int(req.Level),
	})
	if err != nil {
		return nil, fmt.Errorf("generate hint: %w", err)
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
