package llmrest

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

type TwentyQHintsRequest struct {
	Target   string         `json:"target"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details,omitempty"`
}

// TwentyQHintsResponse: 스무고개 힌트 생성 응답
type TwentyQHintsResponse struct {
	Hints            []string `json:"hints"`
	ThoughtSignature *string  `json:"thought_signature,omitempty"`
}

// TwentyQAnswerRequest: 스무고개 질문 답변 요청 파라미터
type TwentyQAnswerRequest struct {
	SessionID *string        `json:"session_id,omitempty"`
	ChatID    *string        `json:"chat_id,omitempty"`
	Namespace *string        `json:"namespace,omitempty"`
	Target    string         `json:"target"`
	Category  string         `json:"category"`
	Question  string         `json:"question"`
	Details   map[string]any `json:"details,omitempty"`
}

// TwentyQAnswerResponse: 스무고개 질문 답변 응답
type TwentyQAnswerResponse struct {
	Scale            *string `json:"scale,omitempty"`
	RawText          string  `json:"raw_text"`
	ThoughtSignature *string `json:"thought_signature,omitempty"`
}

// TwentyQVerifyRequest: 정답 추측 검증 요청 파라미터
type TwentyQVerifyRequest struct {
	Target string `json:"target"`
	Guess  string `json:"guess"`
}

// TwentyQVerifyResponse: 정답 추측 검증 응답
type TwentyQVerifyResponse struct {
	Result  *string `json:"result,omitempty"`
	RawText string  `json:"raw_text"`
}

// TwentyQNormalizeRequest: 질문 정규화 요청 파라미터
type TwentyQNormalizeRequest struct {
	Question string `json:"question"`
}

// TwentyQNormalizeResponse: 질문 정규화 응답
type TwentyQNormalizeResponse struct {
	Normalized string `json:"normalized"`
	Original   string `json:"original"`
}

// TwentyQSynonymRequest: 동의어 확인 요청 파라미터
type TwentyQSynonymRequest struct {
	Target string `json:"target"`
	Guess  string `json:"guess"`
}

// TwentyQSynonymResponse: 동의어 확인 응답
type TwentyQSynonymResponse struct {
	Result  *string `json:"result,omitempty"`
	RawText string  `json:"raw_text"`
}

// TwentyQGenerateHints: 힌트를 생성 요청을 전송합니다.
func (c *Client) TwentyQGenerateHints(ctx context.Context, target string, category string, details map[string]any) (*TwentyQHintsResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	var detailsStruct *structpb.Struct
	if len(details) > 0 {
		st, err := structpb.NewStruct(details)
		if err != nil {
			return nil, fmt.Errorf("convert details failed: %w", err)
		}
		detailsStruct = st
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.TwentyQGenerateHints(callCtx, &llmv1.TwentyQGenerateHintsRequest{
		Target:   target,
		Category: category,
		Details:  detailsStruct,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq generate hints failed: %w", err)
	}
	return &TwentyQHintsResponse{
		Hints:            resp.Hints,
		ThoughtSignature: resp.ThoughtSignature,
	}, nil
}

// TwentyQAnswerQuestion: 질문에 대한 답변 요청을 전송합니다.
func (c *Client) TwentyQAnswerQuestion(
	ctx context.Context,
	chatID string,
	namespace string,
	target string,
	category string,
	question string,
	details map[string]any,
) (*TwentyQAnswerResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	var detailsStruct *structpb.Struct
	if len(details) > 0 {
		st, err := structpb.NewStruct(details)
		if err != nil {
			return nil, fmt.Errorf("convert details failed: %w", err)
		}
		detailsStruct = st
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	req := &llmv1.TwentyQAnswerQuestionRequest{
		ChatId:    &chatID,
		Namespace: &namespace,
		Target:    target,
		Category:  category,
		Question:  question,
		Details:   detailsStruct,
	}
	resp, err := c.grpcClient.TwentyQAnswerQuestion(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq answer failed: %w", err)
	}
	return &TwentyQAnswerResponse{
		Scale:            resp.Scale,
		RawText:          resp.RawText,
		ThoughtSignature: resp.ThoughtSignature,
	}, nil
}

// TwentyQVerifyGuess: 정답 추측 검증 요청을 전송합니다.
func (c *Client) TwentyQVerifyGuess(ctx context.Context, target string, guess string) (*TwentyQVerifyResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.TwentyQVerifyGuess(callCtx, &llmv1.TwentyQVerifyGuessRequest{Target: target, Guess: guess})
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq verify failed: %w", err)
	}
	return &TwentyQVerifyResponse{Result: resp.Result, RawText: resp.RawText}, nil
}

// TwentyQNormalizeQuestion: 질문 정규화 요청을 전송합니다.
func (c *Client) TwentyQNormalizeQuestion(ctx context.Context, question string) (*TwentyQNormalizeResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.TwentyQNormalizeQuestion(callCtx, &llmv1.TwentyQNormalizeQuestionRequest{Question: question})
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq normalize failed: %w", err)
	}
	return &TwentyQNormalizeResponse{Normalized: resp.Normalized, Original: resp.Original}, nil
}

// TwentyQCheckSynonym: 동의어 여부 확인 요청을 전송합니다.
func (c *Client) TwentyQCheckSynonym(ctx context.Context, target string, guess string) (*TwentyQSynonymResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.TwentyQCheckSynonym(callCtx, &llmv1.TwentyQCheckSynonymRequest{Target: target, Guess: guess})
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq synonym check failed: %w", err)
	}
	return &TwentyQSynonymResponse{Result: resp.Result, RawText: resp.RawText}, nil
}

// TwentyQSelectTopicRequest: 토픽 선택 요청 파라미터
type TwentyQSelectTopicRequest struct {
	Category           string   `json:"category"`
	BannedTopics       []string `json:"bannedTopics"`
	ExcludedCategories []string `json:"excludedCategories"`
}

// TwentyQSelectTopicResponse: 토픽 선택 응답
type TwentyQSelectTopicResponse struct {
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details"`
}

// TwentyQCategoriesResponse: 카테고리 목록 응답
type TwentyQCategoriesResponse struct {
	Categories []string `json:"categories"`
}

// TwentyQSelectTopic: 조건에 맞는 토픽을 선택 요청합니다.
func (c *Client) TwentyQSelectTopic(ctx context.Context, category string, bannedTopics []string, excludedCategories []string) (*TwentyQSelectTopicResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.TwentyQSelectTopic(callCtx, &llmv1.TwentyQSelectTopicRequest{
		Category:           category,
		BannedTopics:       bannedTopics,
		ExcludedCategories: excludedCategories,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq select topic failed: %w", err)
	}

	details := map[string]any(nil)
	if resp.Details != nil {
		details = resp.Details.AsMap()
	}

	return &TwentyQSelectTopicResponse{
		Name:     resp.Name,
		Category: resp.Category,
		Details:  details,
	}, nil
}

// TwentyQGetCategories: 사용 가능한 카테고리 목록을 조회합니다.
func (c *Client) TwentyQGetCategories(ctx context.Context) (*TwentyQCategoriesResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.TwentyQGetCategories(callCtx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("grpc twentyq get categories failed: %w", err)
	}
	return &TwentyQCategoriesResponse{Categories: resp.Categories}, nil
}
