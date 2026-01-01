package security

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
)

func TestCachedInjectionGuard_ValidateOrThrow_CachesIsMalicious(t *testing.T) {
	mockMalicious := false
	var calls int32

	stub := &guardOnlyLLMGRPCStub{
		handler: func(ctx context.Context, _ *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
			atomic.AddInt32(&calls, 1)
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

	base := NewMcpInjectionGuard(client, logger)
	guard := NewCachedInjectionGuard(base, 5*time.Minute, 128, logger)

	sanitized, err := guard.ValidateOrThrow(context.Background(), "  safe   input  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sanitized != "safe input" {
		t.Fatalf("unexpected sanitized output: %q", sanitized)
	}

	sanitized, err = guard.ValidateOrThrow(context.Background(), "  safe   input  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sanitized != "safe input" {
		t.Fatalf("unexpected sanitized output: %q", sanitized)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 guard call, got %d", got)
	}
}

func TestCachedInjectionGuard_IsMalicious_SingleflightCoalesces(t *testing.T) {
	mockMalicious := false
	var calls int32
	started := make(chan struct{})
	release := make(chan struct{})

	stub := &guardOnlyLLMGRPCStub{
		handler: func(ctx context.Context, _ *llmv1.GuardIsMaliciousRequest) (*llmv1.GuardIsMaliciousResponse, error) {
			n := atomic.AddInt32(&calls, 1)
			if n == 1 {
				close(started)
				<-release
			}
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

	base := NewMcpInjectionGuard(client, logger)
	guard := NewCachedInjectionGuard(base, 5*time.Minute, 128, logger)

	const workers = 20
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	resultCh := make(chan bool, workers)
	goCh := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-goCh:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
			malicious, err := guard.IsMalicious(ctx, "  safe   input  ")
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- malicious
		}()
	}

	close(goCh)

	select {
	case <-started:
	case <-ctx.Done():
		t.Fatalf("timeout waiting for first request: %v", ctx.Err())
	}

	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		t.Fatalf("timeout before releasing server: %v", ctx.Err())
	}

	close(release)
	wg.Wait()

	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	close(resultCh)
	for malicious := range resultCh {
		if malicious != mockMalicious {
			t.Fatalf("unexpected malicious result: %v", malicious)
		}
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 guard call, got %d", got)
	}
}
