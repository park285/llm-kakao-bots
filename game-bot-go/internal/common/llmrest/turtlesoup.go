package llmrest

import (
	"context"
	"fmt"
)

// TurtleSoupAnswerRequest 는 타입이다.
type TurtleSoupAnswerRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	ChatID    *string `json:"chat_id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Scenario  string  `json:"scenario"`
	Solution  string  `json:"solution"`
	Question  string  `json:"question"`
}

// TurtleSoupHistoryItem 는 타입이다.
type TurtleSoupHistoryItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// TurtleSoupAnswerResponse 는 타입이다.
type TurtleSoupAnswerResponse struct {
	Answer        string                  `json:"answer"`
	RawText       string                  `json:"raw_text"`
	QuestionCount int                     `json:"question_count"`
	History       []TurtleSoupHistoryItem `json:"history"`
}

// TurtleSoupHintRequest 는 타입이다.
type TurtleSoupHintRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	ChatID    *string `json:"chat_id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Scenario  string  `json:"scenario"`
	Solution  string  `json:"solution"`
	Level     int     `json:"level"`
}

// TurtleSoupHintResponse 는 타입이다.
type TurtleSoupHintResponse struct {
	Hint  string `json:"hint"`
	Level int    `json:"level"`
}

// TurtleSoupValidateRequest 는 타입이다.
type TurtleSoupValidateRequest struct {
	SessionID    *string `json:"session_id,omitempty"`
	ChatID       *string `json:"chat_id,omitempty"`
	Namespace    *string `json:"namespace,omitempty"`
	Solution     string  `json:"solution"`
	PlayerAnswer string  `json:"player_answer"`
}

// TurtleSoupValidateResponse 는 타입이다.
type TurtleSoupValidateResponse struct {
	Result  string `json:"result"`
	RawText string `json:"raw_text"`
}

// TurtleSoupRewriteRequest 는 타입이다.
type TurtleSoupRewriteRequest struct {
	Title      string `json:"title"`
	Scenario   string `json:"scenario"`
	Solution   string `json:"solution"`
	Difficulty int    `json:"difficulty"`
}

// TurtleSoupRewriteResponse 는 타입이다.
type TurtleSoupRewriteResponse struct {
	Scenario         string `json:"scenario"`
	Solution         string `json:"solution"`
	OriginalScenario string `json:"original_scenario"`
	OriginalSolution string `json:"original_solution"`
}

// TurtleSoupPuzzleGenerationRequest 는 타입이다.
type TurtleSoupPuzzleGenerationRequest struct {
	Category   *string `json:"category,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
	Theme      *string `json:"theme,omitempty"`
}

// TurtleSoupPuzzleGenerationResponse 는 타입이다.
type TurtleSoupPuzzleGenerationResponse struct {
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Solution   string   `json:"solution"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"`
	Hints      []string `json:"hints"`
}

// TurtleSoupPuzzlePresetResponse 는 타입이다.
type TurtleSoupPuzzlePresetResponse struct {
	ID         *int    `json:"id,omitempty"`
	Title      *string `json:"title,omitempty"`
	Question   *string `json:"question,omitempty"`
	Answer     *string `json:"answer,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
}

// TurtleSoupAnswerQuestion 는 동작을 수행한다.
func (c *Client) TurtleSoupAnswerQuestion(
	ctx context.Context,
	chatID string,
	namespace string,
	scenario string,
	solution string,
	question string,
) (*TurtleSoupAnswerResponse, error) {
	req := TurtleSoupAnswerRequest{
		ChatID:    &chatID,
		Namespace: &namespace,
		Scenario:  scenario,
		Solution:  solution,
		Question:  question,
	}

	var out TurtleSoupAnswerResponse
	if err := c.Post(ctx, "/api/turtle-soup/answers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TurtleSoupGenerateHint 는 동작을 수행한다.
func (c *Client) TurtleSoupGenerateHint(
	ctx context.Context,
	chatID string,
	namespace string,
	scenario string,
	solution string,
	level int,
) (*TurtleSoupHintResponse, error) {
	req := TurtleSoupHintRequest{
		ChatID:    &chatID,
		Namespace: &namespace,
		Scenario:  scenario,
		Solution:  solution,
		Level:     level,
	}

	var out TurtleSoupHintResponse
	if err := c.Post(ctx, "/api/turtle-soup/hints", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TurtleSoupValidateSolution 는 동작을 수행한다.
func (c *Client) TurtleSoupValidateSolution(
	ctx context.Context,
	chatID string,
	namespace string,
	solution string,
	playerAnswer string,
) (*TurtleSoupValidateResponse, error) {
	req := TurtleSoupValidateRequest{
		ChatID:       &chatID,
		Namespace:    &namespace,
		Solution:     solution,
		PlayerAnswer: playerAnswer,
	}

	var out TurtleSoupValidateResponse
	if err := c.Post(ctx, "/api/turtle-soup/validations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TurtleSoupRewriteScenario 는 동작을 수행한다.
func (c *Client) TurtleSoupRewriteScenario(
	ctx context.Context,
	title string,
	scenario string,
	solution string,
	difficulty int,
) (*TurtleSoupRewriteResponse, error) {
	req := TurtleSoupRewriteRequest{
		Title:      title,
		Scenario:   scenario,
		Solution:   solution,
		Difficulty: difficulty,
	}

	var out TurtleSoupRewriteResponse
	if err := c.Post(ctx, "/api/turtle-soup/rewrites", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TurtleSoupGeneratePuzzle 는 동작을 수행한다.
func (c *Client) TurtleSoupGeneratePuzzle(ctx context.Context, req TurtleSoupPuzzleGenerationRequest) (*TurtleSoupPuzzleGenerationResponse, error) {
	var out TurtleSoupPuzzleGenerationResponse
	if err := c.Post(ctx, "/api/turtle-soup/puzzles", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TurtleSoupGetRandomPuzzle 는 동작을 수행한다.
func (c *Client) TurtleSoupGetRandomPuzzle(ctx context.Context, difficulty *int) (*TurtleSoupPuzzlePresetResponse, error) {
	path := "/api/turtle-soup/puzzles/random"
	if difficulty != nil {
		path = fmt.Sprintf("%s?difficulty=%d", path, *difficulty)
	}

	var out TurtleSoupPuzzlePresetResponse
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
