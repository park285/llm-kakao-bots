package llm

// HistoryEntry: 대화 히스토리 항목입니다.
type HistoryEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage: 토큰 사용량 정보를 담습니다.
type Usage struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
	ReasoningTokens int `json:"reasoning_tokens"`
	CachedTokens    int `json:"cached_tokens"` // 암시적 캐싱된 토큰 수 (CachedContentTokenCount)
}

// CacheHitRatio: 캐시 적중률을 계산합니다 (0.0 ~ 1.0).
// InputTokens가 0이면 0을 반환합니다.
func (u Usage) CacheHitRatio() float64 {
	if u.InputTokens == 0 {
		return 0
	}
	return float64(u.CachedTokens) / float64(u.InputTokens)
}

// ChatResult: LLM 응답과 사용량을 담습니다.
type ChatResult struct {
	Text         string
	Usage        Usage
	Reasoning    string
	HasReasoning bool
}
