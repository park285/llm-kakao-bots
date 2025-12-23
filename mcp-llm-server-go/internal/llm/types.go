package llm

// HistoryEntry 는 대화 히스토리 항목이다.
type HistoryEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage 는 토큰 사용량 정보를 담는다.
type Usage struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
	ReasoningTokens int `json:"reasoning_tokens"`
}

// ChatResult 는 LLM 응답과 사용량을 담는다.
type ChatResult struct {
	Text         string
	Usage        Usage
	Reasoning    string
	HasReasoning bool
}
