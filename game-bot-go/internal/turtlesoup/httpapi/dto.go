package httpapi

import "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"

// StartGameRequest: 게임 시작 요청 DTO
type StartGameRequest struct {
	SessionID  string  `json:"sessionId"`
	UserID     string  `json:"userId"`
	ChatID     string  `json:"chatId"`
	Category   *string `json:"category,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
	Theme      *string `json:"theme,omitempty"`
}

// AskQuestionRequest: 질문 요청 DTO
type AskQuestionRequest struct {
	SessionID string `json:"sessionId"`
	Question  string `json:"question"`
}

// SubmitSolutionRequest: 정답 제출 요청 DTO
type SubmitSolutionRequest struct {
	SessionID string `json:"sessionId"`
	Answer    string `json:"answer"`
}

// HintRequest: 힌트 요청 DTO
type HintRequest struct {
	SessionID string `json:"sessionId"`
}

// GameStateResponse: 게임 상태 응답 DTO
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

// QuestionResponse: 질문 처리 결과 응답 DTO
type QuestionResponse struct {
	Answer        string `json:"answer"`
	QuestionCount int    `json:"questionCount"`
}

// SolutionResponse: 정답 제출 결과 응답 DTO
type SolutionResponse struct {
	Result   string  `json:"result"`
	Solution *string `json:"solution,omitempty"`
}

// HintResponse: 힌트 요청 결과 응답 DTO
type HintResponse struct {
	Hint           string `json:"hint"`
	HintsUsed      int    `json:"hintsUsed"`
	HintsRemaining int    `json:"hintsRemaining"`
}

// LlmDebugTransport: LLM 디버그 전송 설정 정보 DTO
type LlmDebugTransport struct {
	BaseURL               string `json:"baseUrl"`
	TimeoutSeconds        int64  `json:"timeoutSeconds"`
	ConnectTimeoutSeconds int64  `json:"connectTimeoutSeconds"`
}

// LlmDebugResponse: LLM 디버그 정보 전체 응답 DTO
type LlmDebugResponse struct {
	LlmRest           LlmDebugTransport            `json:"llmRest"`
	ModelConfig       *llmrest.ModelConfigResponse `json:"modelConfig,omitempty"`
	ModelConfigStatus string                       `json:"modelConfigStatus"`
}
