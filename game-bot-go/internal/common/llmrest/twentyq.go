package llmrest

import "context"

// TwentyQHintsRequest: 스무고개 힌트 생성 요청 파라미터
type TwentyQHintsRequest struct {
	Target   string         `json:"target"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details,omitempty"`
}

// TwentyQHintsResponse: 스무고개 힌트 생성 응답
type TwentyQHintsResponse struct {
	Hints            []string `json:"hints"`
	ThoughtSignature *string  `json:"thought_signature,omitempty"`
}

// TwentyQAnswerRequest: 스무고개 질문 답변 요청 파라미터
type TwentyQAnswerRequest struct {
	SessionID *string        `json:"session_id,omitempty"`
	ChatID    *string        `json:"chat_id,omitempty"`
	Namespace *string        `json:"namespace,omitempty"`
	Target    string         `json:"target"`
	Category  string         `json:"category"`
	Question  string         `json:"question"`
	Details   map[string]any `json:"details,omitempty"`
}

// TwentyQAnswerResponse: 스무고개 질문 답변 응답
type TwentyQAnswerResponse struct {
	Scale            *string `json:"scale,omitempty"`
	RawText          string  `json:"raw_text"`
	ThoughtSignature *string `json:"thought_signature,omitempty"`
}

// TwentyQVerifyRequest: 정답 추측 검증 요청 파라미터
type TwentyQVerifyRequest struct {
	Target string `json:"target"`
	Guess  string `json:"guess"`
}

// TwentyQVerifyResponse: 정답 추측 검증 응답
type TwentyQVerifyResponse struct {
	Result  *string `json:"result,omitempty"`
	RawText string  `json:"raw_text"`
}

// TwentyQNormalizeRequest: 질문 정규화 요청 파라미터
type TwentyQNormalizeRequest struct {
	Question string `json:"question"`
}

// TwentyQNormalizeResponse: 질문 정규화 응답
type TwentyQNormalizeResponse struct {
	Normalized string `json:"normalized"`
	Original   string `json:"original"`
}

// TwentyQSynonymRequest: 동의어 확인 요청 파라미터
type TwentyQSynonymRequest struct {
	Target string `json:"target"`
	Guess  string `json:"guess"`
}

// TwentyQSynonymResponse: 동의어 확인 응답
type TwentyQSynonymResponse struct {
	Result  *string `json:"result,omitempty"`
	RawText string  `json:"raw_text"`
}

// TwentyQGenerateHints: 힌트를 생성 요청을 전송합니다.
func (c *Client) TwentyQGenerateHints(ctx context.Context, target string, category string, details map[string]any) (*TwentyQHintsResponse, error) {
	req := TwentyQHintsRequest{Target: target, Category: category, Details: details}
	var out TwentyQHintsResponse
	if err := c.Post(ctx, "/api/twentyq/hints", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQAnswerQuestion: 질문에 대한 답변 요청을 전송합니다.
func (c *Client) TwentyQAnswerQuestion(
	ctx context.Context,
	chatID string,
	namespace string,
	target string,
	category string,
	question string,
	details map[string]any,
) (*TwentyQAnswerResponse, error) {
	req := TwentyQAnswerRequest{
		ChatID:    &chatID,
		Namespace: &namespace,
		Target:    target,
		Category:  category,
		Question:  question,
		Details:   details,
	}

	var out TwentyQAnswerResponse
	if err := c.Post(ctx, "/api/twentyq/answers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQVerifyGuess: 정답 추측 검증 요청을 전송합니다.
func (c *Client) TwentyQVerifyGuess(ctx context.Context, target string, guess string) (*TwentyQVerifyResponse, error) {
	req := TwentyQVerifyRequest{Target: target, Guess: guess}
	var out TwentyQVerifyResponse
	if err := c.Post(ctx, "/api/twentyq/verifications", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQNormalizeQuestion: 질문 정규화 요청을 전송합니다.
func (c *Client) TwentyQNormalizeQuestion(ctx context.Context, question string) (*TwentyQNormalizeResponse, error) {
	req := TwentyQNormalizeRequest{Question: question}
	var out TwentyQNormalizeResponse
	if err := c.Post(ctx, "/api/twentyq/normalizations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQCheckSynonym: 동의어 여부 확인 요청을 전송합니다.
func (c *Client) TwentyQCheckSynonym(ctx context.Context, target string, guess string) (*TwentyQSynonymResponse, error) {
	req := TwentyQSynonymRequest{Target: target, Guess: guess}
	var out TwentyQSynonymResponse
	if err := c.Post(ctx, "/api/twentyq/synonym-checks", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQSelectTopicRequest: 토픽 선택 요청 파라미터
type TwentyQSelectTopicRequest struct {
	Category           string   `json:"category"`
	BannedTopics       []string `json:"bannedTopics"`
	ExcludedCategories []string `json:"excludedCategories"`
}

// TwentyQSelectTopicResponse: 토픽 선택 응답
type TwentyQSelectTopicResponse struct {
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details"`
}

// TwentyQCategoriesResponse: 카테고리 목록 응답
type TwentyQCategoriesResponse struct {
	Categories []string `json:"categories"`
}

// TwentyQSelectTopic: 조건에 맞는 토픽을 선택 요청합니다.
func (c *Client) TwentyQSelectTopic(ctx context.Context, category string, bannedTopics []string, excludedCategories []string) (*TwentyQSelectTopicResponse, error) {
	req := TwentyQSelectTopicRequest{
		Category:           category,
		BannedTopics:       bannedTopics,
		ExcludedCategories: excludedCategories,
	}
	var out TwentyQSelectTopicResponse
	if err := c.Post(ctx, "/api/twentyq/topics/select", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQGetCategories: 사용 가능한 카테고리 목록을 조회합니다.
func (c *Client) TwentyQGetCategories(ctx context.Context) (*TwentyQCategoriesResponse, error) {
	var out TwentyQCategoriesResponse
	if err := c.Get(ctx, "/api/twentyq/topics/categories", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
