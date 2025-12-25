package errors

import "fmt"

// Error codes
const (
	CodeBotError    = "BOT_ERROR"
	CodeAPIError    = "API_ERROR"
	CodeValidation  = "VALIDATION_ERROR"
	CodeCache       = "CACHE_ERROR"
	CodeService     = "SERVICE_ERROR"
	CodeKeyRotation = "KEY_ROTATION_ERROR"
)

// BotError 는 타입이다.
type BotError struct {
	Message    string
	Code       string
	StatusCode int
	Context    map[string]any
	Cause      error
}

// ErrorCode 는 동작을 수행한다.
func (e *BotError) ErrorCode() string {
	if e == nil {
		return ""
	}
	return e.Code
}

func (e *BotError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *BotError) Unwrap() error {
	return e.Cause
}

// NewBotError 는 동작을 수행한다.
func NewBotError(message, code string, statusCode int, context map[string]any) *BotError {
	return &BotError{
		Message:    message,
		Code:       code,
		StatusCode: statusCode,
		Context:    context,
	}
}

// WithCause 는 동작을 수행한다.
func (e *BotError) WithCause(cause error) *BotError {
	e.Cause = cause
	return e
}

// APIError 는 타입이다.
type APIError struct {
	*BotError
}

// NewAPIError 는 동작을 수행한다.
func NewAPIError(message string, statusCode int, context map[string]any) *APIError {
	return &APIError{
		BotError: &BotError{
			Message:    message,
			Code:       CodeAPIError,
			StatusCode: statusCode,
			Context:    context,
		},
	}
}

// ValidationError 는 타입이다.
type ValidationError struct {
	*BotError
	Field string
	Value any
}

// NewValidationError 는 동작을 수행한다.
func NewValidationError(message, field string, value any) *ValidationError {
	return &ValidationError{
		BotError: &BotError{
			Message:    message,
			Code:       CodeValidation,
			StatusCode: 400,
			Context: map[string]any{
				"field": field,
				"value": value,
			},
		},
		Field: field,
		Value: value,
	}
}

// CacheError 는 타입이다.
type CacheError struct {
	*BotError
	Operation string
	Key       string
}

// NewCacheError 는 동작을 수행한다.
func NewCacheError(message, operation, key string, cause error) *CacheError {
	return &CacheError{
		BotError: &BotError{
			Message:    message,
			Code:       CodeCache,
			StatusCode: 500,
			Context: map[string]any{
				"operation": operation,
				"key":       key,
			},
			Cause: cause,
		},
		Operation: operation,
		Key:       key,
	}
}

// ServiceError 는 타입이다.
type ServiceError struct {
	*BotError
	Service   string
	Operation string
}

// NewServiceError 는 동작을 수행한다.
func NewServiceError(message, service, operation string, cause error) *ServiceError {
	return &ServiceError{
		BotError: &BotError{
			Message:    message,
			Code:       CodeService,
			StatusCode: 500,
			Context: map[string]any{
				"service":   service,
				"operation": operation,
			},
			Cause: cause,
		},
		Service:   service,
		Operation: operation,
	}
}

// KeyRotationError 는 타입이다.
type KeyRotationError struct {
	*APIError
}

// NewKeyRotationError 는 동작을 수행한다.
func NewKeyRotationError(message string, statusCode int, context map[string]any) *KeyRotationError {
	return &KeyRotationError{
		APIError: &APIError{
			BotError: &BotError{
				Message:    message,
				Code:       CodeKeyRotation,
				StatusCode: statusCode,
				Context:    context,
			},
		},
	}
}
