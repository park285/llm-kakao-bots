package security

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
)

func TestMcpInjectionGuard_IsMalicious(t *testing.T) {
	mockMalicious := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/guard/checks") {
			json.NewEncoder(w).Encode(llmrest.GuardMaliciousResponse{Malicious: mockMalicious})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, _ := llmrest.New(llmrest.Config{BaseURL: ts.URL})
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/guard/checks") {
			json.NewEncoder(w).Encode(llmrest.GuardMaliciousResponse{Malicious: mockMalicious})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, _ := llmrest.New(llmrest.Config{BaseURL: ts.URL})
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
	if !strings.Contains(err.Error(), "malicious") {
		t.Errorf("unexpected error message: %v", err)
	}
}
