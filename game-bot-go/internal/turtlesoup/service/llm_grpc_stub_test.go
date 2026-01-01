package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

type turtlesoupLLMGRPCStub struct {
	llmv1.UnimplementedLLMServiceServer

	callCount *int
	hasError  func() bool

	guardMalicious func() bool

	generatePuzzle  func() *llmrest.TurtleSoupPuzzleGenerationResponse
	getRandomPuzzle func() *llmrest.TurtleSoupPuzzlePresetResponse
	rewriteScenario func() *llmrest.TurtleSoupRewriteResponse

	answerQuestion   func() *llmrest.TurtleSoupAnswerResponse
	validateSolution func() *llmrest.TurtleSoupValidateResponse
	generateHint     func() *llmrest.TurtleSoupHintResponse
}

func (s *turtlesoupLLMGRPCStub) incCall() {
	if s != nil && s.callCount != nil {
		*s.callCount++
	}
}

func (s *turtlesoupLLMGRPCStub) isError() bool {
	if s == nil || s.hasError == nil {
		return false
	}
	return s.hasError()
}

func (s *turtlesoupLLMGRPCStub) GuardIsMalicious(ctx context.Context, _ *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}
	malicious := false
	if s != nil && s.guardMalicious != nil {
		malicious = s.guardMalicious()
	}
	return &llmv1.GuardIsMaliciousResponse{Malicious: malicious}, nil
}

func (s *turtlesoupLLMGRPCStub) TurtleSoupGeneratePuzzle(ctx context.Context, _ *llmv1.TurtleSoupGeneratePuzzleRequest) (*llmv1.TurtleSoupGeneratePuzzleResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	resp := (*llmrest.TurtleSoupPuzzleGenerationResponse)(nil)
	if s != nil && s.generatePuzzle != nil {
		resp = s.generatePuzzle()
	}
	if resp == nil {
		resp = &llmrest.TurtleSoupPuzzleGenerationResponse{
			Title:      "Test Puzzle",
			Scenario:   "A man walks into a bar...",
			Solution:   "He was thirsty.",
			Category:   "mystery",
			Difficulty: 1,
		}
	}

	return &llmv1.TurtleSoupGeneratePuzzleResponse{
		Title:      resp.Title,
		Scenario:   resp.Scenario,
		Solution:   resp.Solution,
		Category:   resp.Category,
		Difficulty: int32(resp.Difficulty),
		Hints:      resp.Hints,
	}, nil
}

func (s *turtlesoupLLMGRPCStub) TurtleSoupGetRandomPuzzle(ctx context.Context, _ *llmv1.TurtleSoupGetRandomPuzzleRequest) (*llmv1.TurtleSoupGetRandomPuzzleResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	resp := (*llmrest.TurtleSoupPuzzlePresetResponse)(nil)
	if s != nil && s.getRandomPuzzle != nil {
		resp = s.getRandomPuzzle()
	}
	if resp == nil {
		title := "Preset Title"
		question := "Preset Question"
		answer := "Preset Answer"
		diff := 1
		resp = &llmrest.TurtleSoupPuzzlePresetResponse{
			Title:      &title,
			Question:   &question,
			Answer:     &answer,
			Difficulty: &diff,
		}
	}

	var id *int32
	if resp.ID != nil {
		v := int32(*resp.ID)
		id = &v
	}
	var difficulty *int32
	if resp.Difficulty != nil {
		v := int32(*resp.Difficulty)
		difficulty = &v
	}

	return &llmv1.TurtleSoupGetRandomPuzzleResponse{
		Id:         id,
		Title:      resp.Title,
		Question:   resp.Question,
		Answer:     resp.Answer,
		Difficulty: difficulty,
	}, nil
}

func (s *turtlesoupLLMGRPCStub) TurtleSoupRewriteScenario(ctx context.Context, req *llmv1.TurtleSoupRewriteScenarioRequest) (*llmv1.TurtleSoupRewriteScenarioResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	resp := (*llmrest.TurtleSoupRewriteResponse)(nil)
	if s != nil && s.rewriteScenario != nil {
		resp = s.rewriteScenario()
	}
	if resp == nil {
		resp = &llmrest.TurtleSoupRewriteResponse{Scenario: "Rewritten Scenario", Solution: "Rewritten Solution"}
	}

	originalScenario := ""
	originalSolution := ""
	if req != nil {
		originalScenario = req.Scenario
		originalSolution = req.Solution
	}
	if resp.OriginalScenario != "" {
		originalScenario = resp.OriginalScenario
	}
	if resp.OriginalSolution != "" {
		originalSolution = resp.OriginalSolution
	}

	return &llmv1.TurtleSoupRewriteScenarioResponse{
		Scenario:         resp.Scenario,
		Solution:         resp.Solution,
		OriginalScenario: originalScenario,
		OriginalSolution: originalSolution,
	}, nil
}

func (s *turtlesoupLLMGRPCStub) TurtleSoupAnswerQuestion(ctx context.Context, _ *llmv1.TurtleSoupAnswerQuestionRequest) (*llmv1.TurtleSoupAnswerQuestionResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	resp := (*llmrest.TurtleSoupAnswerResponse)(nil)
	if s != nil && s.answerQuestion != nil {
		resp = s.answerQuestion()
	}
	if resp == nil {
		resp = &llmrest.TurtleSoupAnswerResponse{
			Answer:        "No",
			History:       []llmrest.TurtleSoupHistoryItem{{Question: "Is it food?", Answer: "No"}},
			QuestionCount: 1,
		}
	}

	history := make([]*llmv1.TurtleSoupHistoryItem, 0, len(resp.History))
	for _, item := range resp.History {
		history = append(history, &llmv1.TurtleSoupHistoryItem{Question: item.Question, Answer: item.Answer})
	}

	return &llmv1.TurtleSoupAnswerQuestionResponse{
		Answer:        resp.Answer,
		RawText:       resp.RawText,
		QuestionCount: int32(resp.QuestionCount),
		History:       history,
	}, nil
}

func (s *turtlesoupLLMGRPCStub) TurtleSoupValidateSolution(ctx context.Context, _ *llmv1.TurtleSoupValidateSolutionRequest) (*llmv1.TurtleSoupValidateSolutionResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	resp := (*llmrest.TurtleSoupValidateResponse)(nil)
	if s != nil && s.validateSolution != nil {
		resp = s.validateSolution()
	}
	if resp == nil {
		resp = &llmrest.TurtleSoupValidateResponse{Result: "NO"}
	}

	return &llmv1.TurtleSoupValidateSolutionResponse{Result: resp.Result, RawText: resp.RawText}, nil
}

func (s *turtlesoupLLMGRPCStub) TurtleSoupGenerateHint(ctx context.Context, _ *llmv1.TurtleSoupGenerateHintRequest) (*llmv1.TurtleSoupGenerateHintResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	resp := (*llmrest.TurtleSoupHintResponse)(nil)
	if s != nil && s.generateHint != nil {
		resp = s.generateHint()
	}
	if resp == nil {
		resp = &llmrest.TurtleSoupHintResponse{Hint: "This is a hint", Level: 1}
	}

	return &llmv1.TurtleSoupGenerateHintResponse{Hint: resp.Hint, Level: int32(resp.Level)}, nil
}

func (s *turtlesoupLLMGRPCStub) EndSession(ctx context.Context, req *llmv1.EndSessionRequest) (*llmv1.EndSessionResponse, error) {
	s.incCall()
	if s.isError() {
		return nil, status.Error(codes.Internal, "mock error")
	}

	id := ""
	if req != nil {
		id = req.SessionId
	}
	return &llmv1.EndSessionResponse{Message: "ended", Id: id}, nil
}
