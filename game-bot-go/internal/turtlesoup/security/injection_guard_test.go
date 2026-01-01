package security

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"google.golang.org/grpc"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
)

func TestMcpInjectionGuard_IsMalicious(t *testing.T) {
	mockMalicious := false
	stub := &guardOnlyLLMGRPCStub{
		handler: func(ctx context.Context, _ *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
			return &llmv1.GuardIsMaliciousResponse{Malicious: mockMalicious}, nil
		},
	}
	baseURL, _ := testhelper.StartTestGRPCServer(t, func(s *grpc.Server) {
		llmv1.RegisterLLMServiceServer(s, stub)
	})

	client, err := llmrest.New(llmrest.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("llm client init failed: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	guard := NewMcpInjectionGuard(client, logger)

	// Case 1: Not malicious
	mockMalicious = false
	malicious, err := guard.IsMalicious(context.Background(), "safe input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if malicious {
		t.Error("expected false, got true")
	}

	// Case 2: Malicious
	mockMalicious = true
	malicious, err = guard.IsMalicious(context.Background(), "malicious input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !malicious {
		t.Error("expected true, got false")
	}
}

func TestMcpInjectionGuard_ValidateOrThrow(t *testing.T) {
	mockMalicious := false
	stub := &guardOnlyLLMGRPCStub{
		handler: func(ctx context.Context, _ *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
			return &llmv1.GuardIsMaliciousResponse{Malicious: mockMalicious}, nil
		},
	}
	baseURL, _ := testhelper.StartTestGRPCServer(t, func(s *grpc.Server) {
		llmv1.RegisterLLMServiceServer(s, stub)
	})

	client, err := llmrest.New(llmrest.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("llm client init failed: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	guard := NewMcpInjectionGuard(client, logger)

	// Case 1: Valid clean input
	mockMalicious = false
	sanitized, err := guard.ValidateOrThrow(context.Background(), "  safe   input  ")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if sanitized != "safe input" {
		t.Errorf("expected 'safe input', got '%s'", sanitized)
	}

	// Case 2: Empty input
	_, err = guard.ValidateOrThrow(context.Background(), "   ")
	if err == nil {
		t.Error("expected error for empty input")
	}

	// Case 3: Malicious input
	mockMalicious = true
	_, err = guard.ValidateOrThrow(context.Background(), "malicious")
	if err == nil {
		t.Error("expected error for malicious input")
	}
	var injectionErr cerrors.InputInjectionError
	if !errors.As(err, &injectionErr) {
		t.Fatalf("expected InputInjectionError, got: %v", err)
	}
}
