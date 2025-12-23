package httperror

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

// ErrorCode 는 API 오류 코드다.
type ErrorCode string

const (
	// ErrorCodeInternal 는 내부 오류 코드다.
	ErrorCodeInternal ErrorCode = "INTERNAL_ERROR"
	// ErrorCodeValidation 는 검증 오류 코드다.
	ErrorCodeValidation ErrorCode = "VALIDATION_ERROR"
	// ErrorCodeUnauthorized 는 인증 오류 코드다.
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrorCodeHTTPRateLimit 는 요청 제한 오류 코드다.
	ErrorCodeHTTPRateLimit ErrorCode = "HTTP_RATE_LIMIT"
	// ErrorCodeLLM 는 LLM 오류 코드다.
	ErrorCodeLLM ErrorCode = "LLM_ERROR"
	// ErrorCodeLLMTimeout 는 LLM 타임아웃 코드다.
	ErrorCodeLLMTimeout ErrorCode = "LLM_TIMEOUT"
	// ErrorCodeLLMParsing 는 LLM 파싱 오류 코드다.
	ErrorCodeLLMParsing ErrorCode = "LLM_PARSING_ERROR"
	// ErrorCodeLLMModel 는 LLM 모델 오류 코드다.
	ErrorCodeLLMModel ErrorCode = "LLM_MODEL_ERROR"
	// ErrorCodeSession 는 세션 오류 코드다.
	ErrorCodeSession ErrorCode = "SESSION_ERROR"
	// ErrorCodeSessionNotFound 는 세션 미존재 코드다.
	ErrorCodeSessionNotFound ErrorCode = "SESSION_NOT_FOUND"
	// ErrorCodeSessionLimit 는 세션 제한 초과 코드다.
	ErrorCodeSessionLimit ErrorCode = "SESSION_LIMIT_EXCEEDED"
	// ErrorCodeSessionExpired 는 세션 만료 코드다.
	ErrorCodeSessionExpired ErrorCode = "SESSION_EXPIRED"
	// ErrorCodeGuard 는 가드 오류 코드다.
	ErrorCodeGuard ErrorCode = "GUARD_ERROR"
	// ErrorCodeGuardBlocked 는 가드 차단 코드다.
	ErrorCodeGuardBlocked ErrorCode = "GUARD_BLOCKED"
	// ErrorCodeGuardConfig 는 가드 설정 오류 코드다.
	ErrorCodeGuardConfig ErrorCode = "GUARD_CONFIG_ERROR"
	// ErrorCodeInvalidInput 는 입력 오류 코드다.
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"
	// ErrorCodeMissingField 는 필드 누락 코드다.
	ErrorCodeMissingField ErrorCode = "MISSING_FIELD"
)

// ErrorResponse 는 API 오류 응답 본문이다.
type ErrorResponse struct {
	ErrorCode string         `json:"error_code"`
	ErrorType string         `json:"error_type"`
	Message   string         `json:"message"`
	RequestID *string        `json:"request_id"`
	Details   map[string]any `json:"details"`
}

// Error 는 내부 표준 오류 타입이다.
type Error struct {
	Code    ErrorCode
	Status  int
	Type    string
	Message string
	Details map[string]any
}

// Error 는 오류 메시지를 반환한다.
func (e *Error) Error() string {
	return e.Message
}

// Response 는 오류를 HTTP 응답으로 변환한다.
func Response(err error, requestID string) (int, ErrorResponse) {
	apiErr := FromError(err)
	if apiErr == nil {
		apiErr = NewInternalError("unknown error")
	}

	var requestIDPtr *string
	if requestID != "" {
		requestIDPtr = &requestID
	}

	return apiErr.Status, ErrorResponse{
		ErrorCode: string(apiErr.Code),
		ErrorType: apiErr.Type,
		Message:   apiErr.Message,
		RequestID: requestIDPtr,
		Details:   apiErr.Details,
	}
}

// FromError 는 오류를 내부 오류 타입으로 변환한다.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}

	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}

	var blocked *guard.BlockedError
	if errors.As(err, &blocked) {
		return NewGuardBlocked(blocked.Score, blocked.Threshold)
	}

	if errors.Is(err, session.ErrSessionNotFound) {
		return NewSessionError("Session not found", http.StatusNotFound)
	}

	if errors.Is(err, gemini.ErrInvalidModel) {
		return NewLLMModelError("Invalid model")
	}

	if errors.Is(err, gemini.ErrMissingAPIKey) {
		return NewLLMError("Missing Gemini API key", http.StatusServiceUnavailable)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return NewLLMTimeoutError("LLM request timed out")
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		return NewValidationError(err)
	}

	return NewInternalError(err.Error())
}

// NewInternalError 는 내부 오류를 생성한다.
func NewInternalError(message string) *Error {
	return &Error{
		Code:    ErrorCodeInternal,
		Status:  http.StatusInternalServerError,
		Type:    "InternalError",
		Message: message,
		Details: nil,
	}
}

// NewValidationError 는 검증 오류를 생성한다.
func NewValidationError(err error) *Error {
	return &Error{
		Code:    ErrorCodeValidation,
		Status:  http.StatusUnprocessableEntity,
		Type:    "ValidationError",
		Message: "Input validation failed",
		Details: validationDetails(err),
	}
}

// NewMissingField 는 누락 필드 오류를 생성한다.
func NewMissingField(field string) *Error {
	return &Error{
		Code:    ErrorCodeMissingField,
		Status:  http.StatusBadRequest,
		Type:    "MissingFieldError",
		Message: fmt.Sprintf("Field '%s' required", field),
		Details: map[string]any{"field": field},
	}
}

// NewInvalidInput 는 입력 오류를 생성한다.
func NewInvalidInput(message string) *Error {
	return &Error{
		Code:    ErrorCodeInvalidInput,
		Status:  http.StatusBadRequest,
		Type:    "InvalidInputError",
		Message: message,
		Details: nil,
	}
}

// NewUnauthorized 는 인증 오류를 생성한다.
func NewUnauthorized(details map[string]any) *Error {
	return &Error{
		Code:    ErrorCodeUnauthorized,
		Status:  http.StatusUnauthorized,
		Type:    "UnauthorizedError",
		Message: "Invalid API key",
		Details: details,
	}
}

// NewRateLimitExceeded 는 요청 제한 오류를 생성한다.
func NewRateLimitExceeded(details map[string]any) *Error {
	return &Error{
		Code:    ErrorCodeHTTPRateLimit,
		Status:  http.StatusTooManyRequests,
		Type:    "HTTPRateLimitExceededError",
		Message: "Rate limit exceeded",
		Details: details,
	}
}

// NewGuardBlocked 는 가드 차단 오류를 생성한다.
func NewGuardBlocked(score float64, threshold float64) *Error {
	return &Error{
		Code:    ErrorCodeGuardBlocked,
		Status:  http.StatusBadRequest,
		Type:    "GuardBlockedError",
		Message: fmt.Sprintf("Input blocked by injection guard (score=%.2f, threshold=%.2f)", score, threshold),
		Details: map[string]any{"score": score, "threshold": threshold},
	}
}

// NewSessionNotFound 는 세션 미존재 오류를 생성한다.
func NewSessionNotFound(sessionID string) *Error {
	return &Error{
		Code:    ErrorCodeSessionNotFound,
		Status:  http.StatusNotFound,
		Type:    "SessionNotFoundError",
		Message: fmt.Sprintf("Session '%s' not found", sessionID),
		Details: map[string]any{"session_id": sessionID},
	}
}

// NewSessionError 는 세션 오류를 생성한다.
func NewSessionError(message string, status int) *Error {
	return &Error{
		Code:    ErrorCodeSession,
		Status:  status,
		Type:    "SessionError",
		Message: message,
		Details: nil,
	}
}

// NewLLMModelError 는 LLM 모델 오류를 생성한다.
func NewLLMModelError(message string) *Error {
	return &Error{
		Code:    ErrorCodeLLMModel,
		Status:  http.StatusBadRequest,
		Type:    "LLMModelError",
		Message: message,
		Details: nil,
	}
}

// NewLLMTimeoutError 는 LLM 타임아웃 오류를 생성한다.
func NewLLMTimeoutError(message string) *Error {
	return &Error{
		Code:    ErrorCodeLLMTimeout,
		Status:  http.StatusGatewayTimeout,
		Type:    "LLMTimeoutError",
		Message: message,
		Details: nil,
	}
}

// NewLLMError 는 LLM 오류를 생성한다.
func NewLLMError(message string, status int) *Error {
	return &Error{
		Code:    ErrorCodeLLM,
		Status:  status,
		Type:    "LLMError",
		Message: message,
		Details: nil,
	}
}

// FieldError 는 필드 오류 상세 정보다.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value"`
}

func validationDetails(err error) map[string]any {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		fields := make([]FieldError, 0, len(validationErrors))
		for _, validationErr := range validationErrors {
			fields = append(fields, FieldError{
				Field:   validationErr.Field(),
				Message: validationErr.Error(),
				Value:   validationErr.Value(),
			})
		}
		return map[string]any{"errors": fields}
	}

	return map[string]any{
		"errors": []FieldError{
			{
				Field:   "body",
				Message: err.Error(),
				Value:   nil,
			},
		},
	}
}
