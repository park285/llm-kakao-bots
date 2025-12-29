package handler

// TwentyQHintsRequest: 힌트 요청 본문입니다.
type TwentyQHintsRequest struct {
	Target   string         `json:"target" binding:"required"`
	Category string         `json:"category" binding:"required"`
	Details  map[string]any `json:"details"`
}

// TwentyQHintsResponse: 힌트 응답 본문입니다.
type TwentyQHintsResponse struct {
	Hints            []string `json:"hints"`
	ThoughtSignature *string  `json:"thought_signature"`
}

// TwentyQAnswerRequest: 정답 요청 본문입니다.
type TwentyQAnswerRequest struct {
	SessionID *string        `json:"session_id"`
	ChatID    *string        `json:"chat_id"`
	Namespace *string        `json:"namespace"`
	Target    string         `json:"target" binding:"required"`
	Category  string         `json:"category" binding:"required"`
	Question  string         `json:"question" binding:"required"`
	Details   map[string]any `json:"details"`
}

// TwentyQAnswerResponse: 정답 응답 본문입니다.
type TwentyQAnswerResponse struct {
	Scale            *string `json:"scale"`
	RawText          string  `json:"raw_text"`
	ThoughtSignature *string `json:"thought_signature"`
}

// TwentyQVerifyRequest: 정답 검증 요청 본문입니다.
type TwentyQVerifyRequest struct {
	Target string `json:"target" binding:"required"`
	Guess  string `json:"guess" binding:"required"`
}

// TwentyQVerifyResponse: 정답 검증 응답 본문입니다.
type TwentyQVerifyResponse struct {
	Result  *string `json:"result"`
	RawText string  `json:"raw_text"`
}

// TwentyQNormalizeRequest: 질문 정규화 요청 본문입니다.
type TwentyQNormalizeRequest struct {
	Question string `json:"question" binding:"required"`
}

// TwentyQNormalizeResponse: 질문 정규화 응답 본문입니다.
type TwentyQNormalizeResponse struct {
	Normalized string `json:"normalized"`
	Original   string `json:"original"`
}

// TwentyQSynonymRequest: 유사어 확인 요청 본문입니다.
type TwentyQSynonymRequest struct {
	Target string `json:"target" binding:"required"`
	Guess  string `json:"guess" binding:"required"`
}

// TwentyQSynonymResponse: 유사어 확인 응답 본문입니다.
type TwentyQSynonymResponse struct {
	Result  *string `json:"result"`
	RawText string  `json:"raw_text"`
}
