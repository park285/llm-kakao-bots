package httpapi

import "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"

// StartGameRequest 는 타입이다.
type StartGameRequest struct {
	SessionID  string  `json:"sessionId"`
	UserID     string  `json:"userId"`
	ChatID     string  `json:"chatId"`
	Category   *string `json:"category,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
	Theme      *string `json:"theme,omitempty"`
}

// AskQuestionRequest 는 타입이다.
type AskQuestionRequest struct {
	SessionID string `json:"sessionId"`
	Question  string `json:"question"`
}

// SubmitSolutionRequest 는 타입이다.
type SubmitSolutionRequest struct {
	SessionID string `json:"sessionId"`
	Answer    string `json:"answer"`
}

// HintRequest 는 타입이다.
type HintRequest struct {
	SessionID string `json:"sessionId"`
}

// GameStateResponse 는 타입이다.
type GameStateResponse struct {
	SessionID      string `json:"sessionId"`
	UserID         string `json:"userId"`
	ChatID         string `json:"chatId"`
	ScenarioTitle  string `json:"scenarioTitle"`
	Scenario       string `json:"scenario"`
	QuestionCount  int    `json:"questionCount"`
	HintsUsed      int    `json:"hintsUsed"`
	IsSolved       bool   `json:"isSolved"`
	ElapsedSeconds int64  `json:"elapsedSeconds"`
}

// QuestionResponse 는 타입이다.
type QuestionResponse struct {
	Answer        string `json:"answer"`
	QuestionCount int    `json:"questionCount"`
}

// SolutionResponse 는 타입이다.
type SolutionResponse struct {
	Result   string  `json:"result"`
	Solution *string `json:"solution,omitempty"`
}

// HintResponse 는 타입이다.
type HintResponse struct {
	Hint           string `json:"hint"`
	HintsUsed      int    `json:"hintsUsed"`
	HintsRemaining int    `json:"hintsRemaining"`
}

// LlmDebugTransport 는 타입이다.
type LlmDebugTransport struct {
	BaseURL               string `json:"baseUrl"`
	HTTP2Enabled          bool   `json:"http2Enabled"`
	TimeoutSeconds        int64  `json:"timeoutSeconds"`
	ConnectTimeoutSeconds int64  `json:"connectTimeoutSeconds"`
}

// LlmDebugResponse 는 타입이다.
type LlmDebugResponse struct {
	LlmRest           LlmDebugTransport            `json:"llmRest"`
	ModelConfig       *llmrest.ModelConfigResponse `json:"modelConfig,omitempty"`
	ModelConfigStatus string                       `json:"modelConfigStatus"`
}
