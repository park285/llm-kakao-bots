package llmrest

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

// ErrGRPCClientRequired: gRPC 클라이언트가 초기화되지 않았을 때 반환되는 에러
var ErrGRPCClientRequired = errors.New("grpc client required")

func (c *Client) GetModelConfig(ctx context.Context) (*ModelConfigResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.GetModelConfig(callCtx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("grpc get model config failed: %w", err)
	}

	out := &ModelConfigResponse{
		ModelDefault:          resp.ModelDefault,
		ModelHints:            resp.ModelHints,
		ModelAnswer:           resp.ModelAnswer,
		ModelVerify:           resp.ModelVerify,
		Temperature:           resp.Temperature,
		ConfiguredTemperature: resp.ConfiguredTemperature,
		TimeoutSeconds:        int(resp.TimeoutSeconds),
		MaxRetries:            int(resp.MaxRetries),
		HTTP2Enabled:          resp.Http2Enabled,
		TransportMode:         resp.TransportMode,
	}
	return out, nil
}

// EndSession: 세션 ID로 세션을 종료합니다.
func (c *Client) EndSession(ctx context.Context, sessionID string) (*SessionEndResponse, error) {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return nil, fmt.Errorf("invalid session id: %q", sessionID)
	}

	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.EndSession(callCtx, &llmv1.EndSessionRequest{SessionId: trimmed})
	if err != nil {
		return nil, fmt.Errorf("grpc end session failed: %w", err)
	}
	return &SessionEndResponse{Message: resp.Message, ID: resp.Id}, nil
}

// EndSessionByChat: 채팅 ID로 세션을 종료합니다.
func (c *Client) EndSessionByChat(ctx context.Context, namespace string, chatID string) (*SessionEndResponse, error) {
	sessionID := strings.TrimSpace(namespace) + ":" + strings.TrimSpace(chatID)
	return c.EndSession(ctx, sessionID)
}

// GuardIsMalicious: 입력 텍스트의 악의적 체크 여부를 확인합니다.
func (c *Client) GuardIsMalicious(ctx context.Context, text string) (bool, error) {
	if c.grpcClient == nil {
		return false, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.GuardIsMalicious(callCtx, &llmv1.GuardIsMaliciousRequest{InputText: text})
	if err != nil {
		return false, fmt.Errorf("grpc guard check failed: %w", err)
	}
	return resp.Malicious, nil
}

// GetTotalUsage: 전체 누적 사용량을 조회합니다.
func (c *Client) GetTotalUsage(ctx context.Context, _ map[string]string) (*UsageResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.GetTotalUsage(callCtx, &llmv1.GetTotalUsageRequest{Days: 0})
	if err != nil {
		return nil, fmt.Errorf("grpc get total usage failed: %w", err)
	}

	reasoning := int(resp.ReasoningTokens)
	var reasoningPtr *int
	if resp.ReasoningTokens != 0 {
		reasoningPtr = &reasoning
	}

	model := strings.TrimSpace(resp.Model)
	var modelPtr *string
	if model != "" {
		modelPtr = &model
	}

	return &UsageResponse{
		InputTokens:     int(resp.InputTokens),
		OutputTokens:    int(resp.OutputTokens),
		TotalTokens:     int(resp.TotalTokens),
		ReasoningTokens: reasoningPtr,
		Model:           modelPtr,
	}, nil
}

// GetDailyUsage: 일별 사용량을 조회합니다.
func (c *Client) GetDailyUsage(ctx context.Context, _ map[string]string) (*DailyUsageResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.GetDailyUsage(callCtx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("grpc get daily usage failed: %w", err)
	}

	model := strings.TrimSpace(resp.Model)
	var modelPtr *string
	if model != "" {
		modelPtr = &model
	}

	return &DailyUsageResponse{
		UsageDate:       resp.UsageDate,
		InputTokens:     resp.InputTokens,
		OutputTokens:    resp.OutputTokens,
		TotalTokens:     resp.TotalTokens,
		ReasoningTokens: resp.ReasoningTokens,
		RequestCount:    resp.RequestCount,
		Model:           modelPtr,
	}, nil
}

// GetRecentUsage: 지정된 일수 내 사용량을 조회합니다.
func (c *Client) GetRecentUsage(ctx context.Context, days int, _ map[string]string) (*UsageListResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.GetRecentUsage(callCtx, &llmv1.GetRecentUsageRequest{Days: int32(days)})
	if err != nil {
		return nil, fmt.Errorf("grpc get recent usage failed: %w", err)
	}

	usages := make([]DailyUsageResponse, 0, len(resp.Usages))
	for _, item := range resp.Usages {
		if item == nil {
			continue
		}

		model := strings.TrimSpace(item.Model)
		var modelPtr *string
		if model != "" {
			modelPtr = &model
		}

		usages = append(usages, DailyUsageResponse{
			UsageDate:       item.UsageDate,
			InputTokens:     item.InputTokens,
			OutputTokens:    item.OutputTokens,
			TotalTokens:     item.TotalTokens,
			ReasoningTokens: item.ReasoningTokens,
			RequestCount:    item.RequestCount,
			Model:           modelPtr,
		})
	}

	model := strings.TrimSpace(resp.Model)
	var modelPtr *string
	if model != "" {
		modelPtr = &model
	}

	return &UsageListResponse{
		Usages:            usages,
		TotalInputTokens:  resp.TotalInputTokens,
		TotalOutputTokens: resp.TotalOutputTokens,
		TotalTokens:       resp.TotalTokens,
		TotalRequestCount: resp.TotalRequestCount,
		Model:             modelPtr,
	}, nil
}

// GetUsageTotalFromDB: DB에서 지정된 일수 내 누적 사용량을 조회합니다.
func (c *Client) GetUsageTotalFromDB(ctx context.Context, days int, _ map[string]string) (*UsageResponse, error) {
	if c.grpcClient == nil {
		return nil, ErrGRPCClientRequired
	}

	callCtx, cancel := c.grpcCallContext(ctx)
	defer cancel()

	resp, err := c.grpcClient.GetTotalUsage(callCtx, &llmv1.GetTotalUsageRequest{Days: int32(days)})
	if err != nil {
		return nil, fmt.Errorf("grpc get total usage failed: %w", err)
	}

	reasoning := int(resp.ReasoningTokens)
	var reasoningPtr *int
	if resp.ReasoningTokens != 0 {
		reasoningPtr = &reasoning
	}

	model := strings.TrimSpace(resp.Model)
	var modelPtr *string
	if model != "" {
		modelPtr = &model
	}

	return &UsageResponse{
		InputTokens:     int(resp.InputTokens),
		OutputTokens:    int(resp.OutputTokens),
		TotalTokens:     int(resp.TotalTokens),
		ReasoningTokens: reasoningPtr,
		Model:           modelPtr,
	}, nil
}
