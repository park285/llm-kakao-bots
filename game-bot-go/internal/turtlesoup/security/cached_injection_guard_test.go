package security

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	json "github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
)

func TestCachedInjectionGuard_ValidateOrThrow_CachesIsMalicious(t *testing.T) {
	mockMalicious := false
	var calls int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/guard/checks") {
			atomic.AddInt32(&calls, 1)
			_ = json.NewEncoder(w).Encode(llmrest.GuardMaliciousResponse{Malicious: mockMalicious})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, _ := llmrest.New(llmrest.Config{BaseURL: ts.URL})
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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/guard/checks") {
			n := atomic.AddInt32(&calls, 1)
			if n == 1 {
				close(started)
				<-release
			}
			_ = json.NewEncoder(w).Encode(llmrest.GuardMaliciousResponse{Malicious: mockMalicious})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, _ := llmrest.New(llmrest.Config{BaseURL: ts.URL})
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
