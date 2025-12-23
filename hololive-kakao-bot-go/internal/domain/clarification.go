package domain

// ClarificationResponse 는 타입이다.
type ClarificationResponse struct {
	Message   string `json:"message"`
	Candidate string `json:"candidate,omitempty"`
}

// Clarification 는 타입이다.
type Clarification struct {
	IsHololiveRelated bool   `json:"is_hololive_related"`
	Message           string `json:"message"`
	Candidate         string `json:"candidate,omitempty"`
}
