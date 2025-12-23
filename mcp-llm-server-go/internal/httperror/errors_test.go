package httperror

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

func TestFromErrorMapping(t *testing.T) {
	apiErr := FromError(&guard.BlockedError{Score: 0.9, Threshold: 0.8})
	if apiErr == nil || apiErr.Code != ErrorCodeGuardBlocked {
		t.Fatalf("expected guard blocked error")
	}

	apiErr = FromError(session.ErrSessionNotFound)
	if apiErr == nil || apiErr.Code != ErrorCodeSession || apiErr.Status != http.StatusNotFound {
		t.Fatalf("expected session error with 404")
	}

	apiErr = FromError(gemini.ErrMissingAPIKey)
	if apiErr == nil || apiErr.Code != ErrorCodeLLM {
		t.Fatalf("expected llm error")
	}

	apiErr = FromError(context.DeadlineExceeded)
	if apiErr == nil || apiErr.Code != ErrorCodeLLMTimeout {
		t.Fatalf("expected timeout error")
	}
}

func TestResponseIncludesRequestID(t *testing.T) {
	status, payload := Response(NewMissingField("id"), "req-1")
	if status != 400 {
		t.Fatalf("unexpected status: %d", status)
	}
	if payload.RequestID == nil || *payload.RequestID != "req-1" {
		t.Fatalf("expected request id")
	}
}

func TestNewMissingField(t *testing.T) {
	err := NewMissingField("username")
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	if err.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 status, got: %d", err.Status)
	}
	if err.Code != ErrorCodeMissingField {
		t.Fatalf("expected missing field error code")
	}
}

func TestNewInvalidInput(t *testing.T) {
	err := NewInvalidInput("must be positive")
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	if err.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 status, got: %d", err.Status)
	}
}

func TestNewValidationError(t *testing.T) {
	originalErr := errors.New("field validation failed")
	err := NewValidationError(originalErr)
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	// NewValidationError 는 422 Unprocessable Entity 반환
	if err.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 status, got: %d", err.Status)
	}
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError("something went wrong")
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	if err.Status != http.StatusInternalServerError {
		t.Fatalf("expected 500 status, got: %d", err.Status)
	}
	if err.Code != ErrorCodeInternal {
		t.Fatalf("expected internal error code")
	}
}

func TestAPIErrorError(t *testing.T) {
	err := NewMissingField("test")
	msg := err.Error()
	if msg == "" {
		t.Fatalf("expected non-empty error message")
	}
}

func TestFromErrorNil(t *testing.T) {
	apiErr := FromError(nil)
	if apiErr != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestFromErrorGeneric(t *testing.T) {
	genericErr := errors.New("some generic error")
	apiErr := FromError(genericErr)
	if apiErr == nil {
		t.Fatalf("expected non-nil error")
	}
	if apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("expected 500 for generic error")
	}
}

func TestResponseWithEmptyRequestID(t *testing.T) {
	status, payload := Response(NewInternalError("test"), "")
	if status != 500 {
		t.Fatalf("unexpected status: %d", status)
	}
	if payload.RequestID != nil {
		t.Fatalf("expected nil request id for empty string")
	}
}
