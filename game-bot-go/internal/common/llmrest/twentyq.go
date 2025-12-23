package llmrest

import "context"

// TwentyQHintsRequest 는 타입이다.
type TwentyQHintsRequest struct {
	Target   string         `json:"target"`
	Category string         `json:"category"`
	Details  map[string]any `json:"details,omitempty"`
}

// TwentyQHintsResponse 는 타입이다.
type TwentyQHintsResponse struct {
	Hints            []string `json:"hints"`
	ThoughtSignature *string  `json:"thought_signature,omitempty"`
}

// TwentyQAnswerRequest 는 타입이다.
type TwentyQAnswerRequest struct {
	SessionID *string        `json:"session_id,omitempty"`
	ChatID    *string        `json:"chat_id,omitempty"`
	Namespace *string        `json:"namespace,omitempty"`
	Target    string         `json:"target"`
	Category  string         `json:"category"`
	Question  string         `json:"question"`
	Details   map[string]any `json:"details,omitempty"`
}

// TwentyQAnswerResponse 는 타입이다.
type TwentyQAnswerResponse struct {
	Scale            *string `json:"scale,omitempty"`
	RawText          string  `json:"raw_text"`
	ThoughtSignature *string `json:"thought_signature,omitempty"`
}

// TwentyQVerifyRequest 는 타입이다.
type TwentyQVerifyRequest struct {
	Target string `json:"target"`
	Guess  string `json:"guess"`
}

// TwentyQVerifyResponse 는 타입이다.
type TwentyQVerifyResponse struct {
	Result  *string `json:"result,omitempty"`
	RawText string  `json:"raw_text"`
}

// TwentyQNormalizeRequest 는 타입이다.
type TwentyQNormalizeRequest struct {
	Question string `json:"question"`
}

// TwentyQNormalizeResponse 는 타입이다.
type TwentyQNormalizeResponse struct {
	Normalized string `json:"normalized"`
	Original   string `json:"original"`
}

// TwentyQSynonymRequest 는 타입이다.
type TwentyQSynonymRequest struct {
	Target string `json:"target"`
	Guess  string `json:"guess"`
}

// TwentyQSynonymResponse 는 타입이다.
type TwentyQSynonymResponse struct {
	Result  *string `json:"result,omitempty"`
	RawText string  `json:"raw_text"`
}

// TwentyQGenerateHints 는 동작을 수행한다.
func (c *Client) TwentyQGenerateHints(ctx context.Context, target string, category string, details map[string]any) (*TwentyQHintsResponse, error) {
	req := TwentyQHintsRequest{Target: target, Category: category, Details: details}
	var out TwentyQHintsResponse
	if err := c.Post(ctx, "/api/twentyq/hints", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQAnswerQuestion 는 동작을 수행한다.
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

// TwentyQVerifyGuess 는 동작을 수행한다.
func (c *Client) TwentyQVerifyGuess(ctx context.Context, target string, guess string) (*TwentyQVerifyResponse, error) {
	req := TwentyQVerifyRequest{Target: target, Guess: guess}
	var out TwentyQVerifyResponse
	if err := c.Post(ctx, "/api/twentyq/verifications", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQNormalizeQuestion 는 동작을 수행한다.
func (c *Client) TwentyQNormalizeQuestion(ctx context.Context, question string) (*TwentyQNormalizeResponse, error) {
	req := TwentyQNormalizeRequest{Question: question}
	var out TwentyQNormalizeResponse
	if err := c.Post(ctx, "/api/twentyq/normalizations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TwentyQCheckSynonym 는 동작을 수행한다.
func (c *Client) TwentyQCheckSynonym(ctx context.Context, target string, guess string) (*TwentyQSynonymResponse, error) {
	req := TwentyQSynonymRequest{Target: target, Guess: guess}
	var out TwentyQSynonymResponse
	if err := c.Post(ctx, "/api/twentyq/synonym-checks", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
