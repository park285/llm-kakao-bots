package llmrest

import (
	"fmt"

	"github.com/goccy/go-json"
)

// ModelConfigResponse 는 타입이다.
type ModelConfigResponse struct {
	ModelDefault          string   `json:"model_default"`
	ModelHints            *string  `json:"model_hints,omitempty"`
	ModelAnswer           *string  `json:"model_answer,omitempty"`
	ModelVerify           *string  `json:"model_verify,omitempty"`
	Temperature           float64  `json:"temperature"`
	ConfiguredTemperature *float64 `json:"configured_temperature,omitempty"`
	TimeoutSeconds        int      `json:"timeout_seconds"`
	MaxRetries            int      `json:"max_retries"`
	HTTP2Enabled          bool     `json:"http2_enabled"`
	TransportMode         *string  `json:"transport_mode,omitempty"`
}

// SessionCreateRequest 는 타입이다.
type SessionCreateRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	ChatID    *string `json:"chat_id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

// SessionCreateResponse 는 타입이다.
type SessionCreateResponse struct {
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
	Created   bool   `json:"created"`
}

// SessionEndResponse 는 타입이다.
type SessionEndResponse struct {
	SessionID string `json:"session_id"`
	Removed   bool   `json:"removed"`
}

// GuardRequest 는 타입이다.
type GuardRequest struct {
	InputText string `json:"input_text"`
}

// GuardMaliciousResponse 는 타입이다.
type GuardMaliciousResponse struct {
	Malicious bool `json:"malicious"`
}

// UsageResponse 는 타입이다.
type UsageResponse struct {
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	ReasoningTokens *int    `json:"reasoning_tokens,omitempty"`
	Model           *string `json:"model,omitempty"`
}

// DailyUsageResponse 는 타입이다.
type DailyUsageResponse struct {
	UsageDate       string  `json:"usage_date"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	TotalTokens     int64   `json:"total_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	RequestCount    int64   `json:"request_count"`
	Model           *string `json:"model,omitempty"`
}

// UsageListResponse 는 타입이다.
type UsageListResponse struct {
	Usages            []DailyUsageResponse `json:"usages"`
	TotalInputTokens  int64                `json:"total_input_tokens"`
	TotalOutputTokens int64                `json:"total_output_tokens"`
	TotalTokens       int64                `json:"total_tokens"`
	TotalRequestCount int64                `json:"total_request_count"`
	Model             *string              `json:"model,omitempty"`
}

// LlmErrorResponse 는 타입이다.
type LlmErrorResponse struct {
	ErrorCode string         `json:"error_code"`
	ErrorType string         `json:"error_type"`
	Message   string         `json:"message"`
	RequestID *string        `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// ParseLlmErrorResponse 는 동작을 수행한다.
func ParseLlmErrorResponse(raw []byte) (*LlmErrorResponse, error) {
	var res LlmErrorResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("unmarshal llm error response failed: %w", err)
	}
	return &res, nil
}
