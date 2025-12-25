package llmrest

// ModelConfigResponse: LLM 모델 설정값 응답 구조체
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

// SessionCreateRequest: 세션 생성 요청 파라미터
type SessionCreateRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	ChatID    *string `json:"chat_id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

// SessionCreateResponse: 세션 생성 응답
type SessionCreateResponse struct {
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
	Created   bool   `json:"created"`
}

// SessionEndResponse: 세션 종료 응답
type SessionEndResponse struct {
	SessionID string `json:"session_id"`
	Removed   bool   `json:"removed"`
}

// GuardRequest: 악성 입력 감지 요청 파라미터
type GuardRequest struct {
	InputText string `json:"input_text"`
}

// GuardMaliciousResponse: 악성 입력 감지 결과 응답
type GuardMaliciousResponse struct {
	Malicious bool `json:"malicious"`
}

// UsageResponse: 토큰 사용량 정보 (단건)
type UsageResponse struct {
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	ReasoningTokens *int    `json:"reasoning_tokens,omitempty"`
	Model           *string `json:"model,omitempty"`
}

// DailyUsageResponse: 일별 토큰 사용량 집계 정보
type DailyUsageResponse struct {
	UsageDate       string  `json:"usage_date"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	TotalTokens     int64   `json:"total_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	RequestCount    int64   `json:"request_count"`
	Model           *string `json:"model,omitempty"`
}

// UsageListResponse: 토큰 사용량 목록 응답
type UsageListResponse struct {
	Usages            []DailyUsageResponse `json:"usages"`
	TotalInputTokens  int64                `json:"total_input_tokens"`
	TotalOutputTokens int64                `json:"total_output_tokens"`
	TotalTokens       int64                `json:"total_tokens"`
	TotalRequestCount int64                `json:"total_request_count"`
	Model             *string              `json:"model,omitempty"`
}
