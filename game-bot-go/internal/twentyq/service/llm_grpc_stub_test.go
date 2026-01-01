package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

type twentyqLLMGRPCStub struct {
	llmv1.UnimplementedLLMServiceServer

	callCount *int
	hasError  func() bool

	guardMalicious func(req *llmv1.GuardIsMaliciousRequest) (bool, error)

	selectTopic    func(req *llmv1.TwentyQSelectTopicRequest) (*llmv1.TwentyQSelectTopicResponse, error)
	generateHints  func(req *llmv1.TwentyQGenerateHintsRequest) (*llmv1.TwentyQGenerateHintsResponse, error)
	answerQuestion func(req *llmv1.TwentyQAnswerQuestionRequest) (*llmv1.TwentyQAnswerQuestionResponse, error)
	verifyGuess    func(req *llmv1.TwentyQVerifyGuessRequest) (*llmv1.TwentyQVerifyGuessResponse, error)
	endSession     func(req *llmv1.EndSessionRequest) (*llmv1.EndSessionResponse, error)
	getDailyUsage  func() (*llmv1.DailyUsageResponse, error)
	getRecentUsage func(req *llmv1.GetRecentUsageRequest) (*llmv1.UsageListResponse, error)
	getTotalUsage  func(req *llmv1.GetTotalUsageRequest) (*llmv1.UsageResponse, error)
}

func (s *twentyqLLMGRPCStub) incCall() {
	if s != nil && s.callCount != nil {
		*s.callCount++
	}
}

func (s *twentyqLLMGRPCStub) isError() bool {
	if s == nil || s.hasError == nil {
		return false
	}
	return s.hasError()
}

func (s *twentyqLLMGRPCStub) GuardIsMalicious(ctx context.Context, req *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.guardMalicious != nil {
		malicious, err := s.guardMalicious(req)
		if err != nil {
			return nil, err
		}
		return &llmv1.GuardIsMaliciousResponse{Malicious: malicious}, nil
	}
	return &llmv1.GuardIsMaliciousResponse{Malicious: false}, nil
}

func (s *twentyqLLMGRPCStub) TwentyQSelectTopic(ctx context.Context, req *llmv1.TwentyQSelectTopicRequest) (*llmv1.TwentyQSelectTopicResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.selectTopic != nil {
		return s.selectTopic(req)
	}

	details, err := structpb.NewStruct(map[string]any{"type": "test"})
	if err != nil {
		return nil, status.Error(codes.Internal, "mock details error")
	}

	category := ""
	if req != nil {
		category = req.Category
	}
	return &llmv1.TwentyQSelectTopicResponse{Name: "테스트토픽", Category: category, Details: details}, nil
}

func (s *twentyqLLMGRPCStub) TwentyQGenerateHints(ctx context.Context, req *llmv1.TwentyQGenerateHintsRequest) (*llmv1.TwentyQGenerateHintsResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.generateHints != nil {
		return s.generateHints(req)
	}
	return &llmv1.TwentyQGenerateHintsResponse{Hints: []string{"It has fur"}}, nil
}

func (s *twentyqLLMGRPCStub) TwentyQAnswerQuestion(ctx context.Context, req *llmv1.TwentyQAnswerQuestionRequest) (*llmv1.TwentyQAnswerQuestionResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.answerQuestion != nil {
		return s.answerQuestion(req)
	}

	scale := "아니오"
	return &llmv1.TwentyQAnswerQuestionResponse{Scale: &scale, RawText: "아니오"}, nil
}

func (s *twentyqLLMGRPCStub) TwentyQVerifyGuess(ctx context.Context, req *llmv1.TwentyQVerifyGuessRequest) (*llmv1.TwentyQVerifyGuessResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.verifyGuess != nil {
		return s.verifyGuess(req)
	}
	return &llmv1.TwentyQVerifyGuessResponse{Result: nil, RawText: ""}, nil
}

func (s *twentyqLLMGRPCStub) EndSession(ctx context.Context, req *llmv1.EndSessionRequest) (*llmv1.EndSessionResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.endSession != nil {
		return s.endSession(req)
	}

	id := ""
	if req != nil {
		id = req.SessionId
	}
	return &llmv1.EndSessionResponse{Message: "ended", Id: id}, nil
}

func (s *twentyqLLMGRPCStub) GetDailyUsage(ctx context.Context, _ *emptypb.Empty) (*llmv1.DailyUsageResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.getDailyUsage != nil {
		return s.getDailyUsage()
	}
	return &llmv1.DailyUsageResponse{
		UsageDate:       "2023-01-01",
		InputTokens:     1_000_000,
		OutputTokens:    1_000_000,
		TotalTokens:     2_000_000,
		ReasoningTokens: 0,
		RequestCount:    10,
		Model:           "gemini-3-flash-preview",
	}, nil
}

func (s *twentyqLLMGRPCStub) GetRecentUsage(ctx context.Context, req *llmv1.GetRecentUsageRequest) (*llmv1.UsageListResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.getRecentUsage != nil {
		return s.getRecentUsage(req)
	}
	return &llmv1.UsageListResponse{
		Usages:            nil,
		TotalInputTokens:  7_000_000,
		TotalOutputTokens: 14_000_000,
		TotalTokens:       21_000_000,
		TotalRequestCount: 0,
		Model:             "gemini-3-flash-preview",
	}, nil
}

func (s *twentyqLLMGRPCStub) GetTotalUsage(ctx context.Context, req *llmv1.GetTotalUsageRequest) (*llmv1.UsageResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	if s != nil && s.getTotalUsage != nil {
		return s.getTotalUsage(req)
	}
	return &llmv1.UsageResponse{
		InputTokens:     1_000_000,
		OutputTokens:    2_000_000,
		TotalTokens:     3_000_000,
		ReasoningTokens: 0,
		Model:           "gemini-3-flash-preview",
	}, nil
}
