package handler

// TurtleSoupAnswerRequest: 정답 요청 본문입니다.
type TurtleSoupAnswerRequest struct {
	SessionID *string `json:"session_id"`
	ChatID    *string `json:"chat_id"`
	Namespace *string `json:"namespace"`
	Scenario  string  `json:"scenario" binding:"required"`
	Solution  string  `json:"solution" binding:"required"`
	Question  string  `json:"question" binding:"required"`
}

// TurtleSoupHistoryItem: 질문/답변 히스토리 항목입니다.
type TurtleSoupHistoryItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// TurtleSoupAnswerResponse: 정답 응답 본문입니다.
type TurtleSoupAnswerResponse struct {
	Answer        string                  `json:"answer"`
	RawText       string                  `json:"raw_text"`
	QuestionCount int                     `json:"question_count"`
	History       []TurtleSoupHistoryItem `json:"history"`
}

// TurtleSoupHintRequest: 힌트 요청 본문입니다.
type TurtleSoupHintRequest struct {
	SessionID *string `json:"session_id"`
	ChatID    *string `json:"chat_id"`
	Namespace *string `json:"namespace"`
	Scenario  string  `json:"scenario" binding:"required"`
	Solution  string  `json:"solution" binding:"required"`
	Level     int     `json:"level" binding:"required,gte=1,lte=3"`
}

// TurtleSoupHintResponse: 힌트 응답 본문입니다.
type TurtleSoupHintResponse struct {
	Hint  string `json:"hint"`
	Level int    `json:"level"`
}

// TurtleSoupValidateRequest: 검증 요청 본문입니다.
type TurtleSoupValidateRequest struct {
	SessionID    *string `json:"session_id"`
	ChatID       *string `json:"chat_id"`
	Namespace    *string `json:"namespace"`
	Solution     string  `json:"solution" binding:"required"`
	PlayerAnswer string  `json:"player_answer" binding:"required"`
}

// TurtleSoupValidateResponse: 검증 응답 본문입니다.
type TurtleSoupValidateResponse struct {
	Result  string `json:"result"`
	RawText string `json:"raw_text"`
}

// TurtleSoupRevealRequest: 해설 요청 본문입니다.
type TurtleSoupRevealRequest struct {
	SessionID *string `json:"session_id"`
	ChatID    *string `json:"chat_id"`
	Namespace *string `json:"namespace"`
	Scenario  string  `json:"scenario" binding:"required"`
	Solution  string  `json:"solution" binding:"required"`
}

// TurtleSoupRevealResponse: 해설 응답 본문입니다.
type TurtleSoupRevealResponse struct {
	Narrative string `json:"narrative"`
}

// TurtleSoupPuzzleGenerationRequest: 퍼즐 생성 요청 본문입니다.
type TurtleSoupPuzzleGenerationRequest struct {
	Category   *string `json:"category,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
	Theme      *string `json:"theme,omitempty"`
}

// TurtleSoupPuzzleGenerationResponse: 퍼즐 생성 응답 본문입니다.
type TurtleSoupPuzzleGenerationResponse struct {
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Solution   string   `json:"solution"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"`
	Hints      []string `json:"hints"`
}

// TurtleSoupRewriteRequest: 리라이트 요청 본문입니다.
type TurtleSoupRewriteRequest struct {
	Title      string `json:"title" binding:"required"`
	Scenario   string `json:"scenario" binding:"required"`
	Solution   string `json:"solution" binding:"required"`
	Difficulty int    `json:"difficulty" binding:"required,gte=1,lte=5"`
}

// TurtleSoupRewriteResponse: 리라이트 응답 본문입니다.
type TurtleSoupRewriteResponse struct {
	Scenario         string `json:"scenario"`
	Solution         string `json:"solution"`
	OriginalScenario string `json:"original_scenario"`
	OriginalSolution string `json:"original_solution"`
}
