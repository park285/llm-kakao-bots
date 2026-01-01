package security

import (
	"context"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

type guardOnlyLLMGRPCStub struct {
	llmv1.UnimplementedLLMServiceServer
	handler func(ctx context.Context, req *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error)
}

func (s *guardOnlyLLMGRPCStub) GuardIsMalicious(ctx context.Context, req *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
	if s != nil && s.handler != nil {
		return s.handler(ctx, req)
	}
	return &llmv1.GuardIsMaliciousResponse{Malicious: false}, nil
}
