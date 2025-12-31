package llmrest

import (
	"context"
	"fmt"

	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
)

type TurtleSoupAnswerRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	ChatID    *string `json:"chat_id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Scenario  string  `json:"scenario"`
	Solution  string  `json:"solution"`
	Question  string  `json:"question"`
}

// TurtleSoupHistoryItem: 바다거북 스프 질문/답변 이력 항목
type TurtleSoupHistoryItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// TurtleSoupAnswerResponse: 바다거북 스프 답변 응답
type TurtleSoupAnswerResponse struct {
	Answer        string                  `json:"answer"`
	RawText       string                  `json:"raw_text"`
	QuestionCount int                     `json:"question_count"`
	History       []TurtleSoupHistoryItem `json:"history"`
}

// TurtleSoupHintRequest: 바다거북 스프 힌트 요청 파라미터
type TurtleSoupHintRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	ChatID    *string `json:"chat_id,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Scenario  string  `json:"scenario"`
	Solution  string  `json:"solution"`
	Level     int     `json:"level"`
}

// TurtleSoupHintResponse: 바다거북 스프 힌트 응답
type TurtleSoupHintResponse struct {
	Hint  string `json:"hint"`
	Level int    `json:"level"`
}

// TurtleSoupValidateRequest: 정답 검증 요청 파라미터
type TurtleSoupValidateRequest struct {
	SessionID    *string `json:"session_id,omitempty"`
	ChatID       *string `json:"chat_id,omitempty"`
	Namespace    *string `json:"namespace,omitempty"`
	Solution     string  `json:"solution"`
	PlayerAnswer string  `json:"player_answer"`
}

// TurtleSoupValidateResponse: 정답 검증 응답
type TurtleSoupValidateResponse struct {
	Result  string `json:"result"`
	RawText string `json:"raw_text"`
}

// TurtleSoupRewriteRequest: 시나리오 재작성/최적화 요청 파라미터
type TurtleSoupRewriteRequest struct {
	Title      string `json:"title"`
	Scenario   string `json:"scenario"`
	Solution   string `json:"solution"`
	Difficulty int    `json:"difficulty"`
}

// TurtleSoupRewriteResponse: 시나리오 재작성 응답
type TurtleSoupRewriteResponse struct {
	Scenario         string `json:"scenario"`
	Solution         string `json:"solution"`
	OriginalScenario string `json:"original_scenario"`
	OriginalSolution string `json:"original_solution"`
}

// TurtleSoupPuzzleGenerationRequest: 퍼즐 자동 생성 요청 파라미터
type TurtleSoupPuzzleGenerationRequest struct {
	Category   *string `json:"category,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
	Theme      *string `json:"theme,omitempty"`
}

// TurtleSoupPuzzleGenerationResponse: 퍼즐 자동 생성 응답
type TurtleSoupPuzzleGenerationResponse struct {
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Solution   string   `json:"solution"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"`
	Hints      []string `json:"hints"`
}

// TurtleSoupPuzzlePresetResponse: 프리셋 퍼즐 정보 응답
type TurtleSoupPuzzlePresetResponse struct {
	ID         *int    `json:"id,omitempty"`
	Title      *string `json:"title,omitempty"`
	Question   *string `json:"question,omitempty"`
	Answer     *string `json:"answer,omitempty"`
	Difficulty *int    `json:"difficulty,omitempty"`
}

// TurtleSoupAnswerQuestion: 사용자의 질문에 대한 예/아니오 답변을 요청합니다.
func (c *Client) TurtleSoupAnswerQuestion(
	ctx context.Context,
	chatID string,
	namespace string,
	scenario string,
	solution string,
	question string,
) (*TurtleSoupAnswerResponse, error) {
	if c.grpcClient != nil {
		callCtx, cancel := c.grpcCallContext(ctx)
		defer cancel()

		req := &llmv1.TurtleSoupAnswerQuestionRequest{
			ChatId:    &chatID,
			Namespace: &namespace,
			Scenario:  scenario,
			Solution:  solution,
			Question:  question,
		}
		resp, err := c.grpcClient.TurtleSoupAnswerQuestion(callCtx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc turtlesoup answer failed: %w", err)
		}

		history := make([]TurtleSoupHistoryItem, 0, len(resp.History))
		for _, item := range resp.History {
			if item == nil {
				continue
			}
			history = append(history, TurtleSoupHistoryItem{Question: item.Question, Answer: item.Answer})
		}

		return &TurtleSoupAnswerResponse{
			Answer:        resp.Answer,
			RawText:       resp.RawText,
			QuestionCount: int(resp.QuestionCount),
			History:       history,
		}, nil
	}

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

// TurtleSoupGenerateHint: 힌트 생성을 요청합니다.
func (c *Client) TurtleSoupGenerateHint(
	ctx context.Context,
	chatID string,
	namespace string,
	scenario string,
	solution string,
	level int,
) (*TurtleSoupHintResponse, error) {
	if c.grpcClient != nil {
		callCtx, cancel := c.grpcCallContext(ctx)
		defer cancel()

		req := &llmv1.TurtleSoupGenerateHintRequest{
			ChatId:    &chatID,
			Namespace: &namespace,
			Scenario:  scenario,
			Solution:  solution,
			Level:     int32(level),
		}
		resp, err := c.grpcClient.TurtleSoupGenerateHint(callCtx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc turtlesoup hint failed: %w", err)
		}

		return &TurtleSoupHintResponse{Hint: resp.Hint, Level: int(resp.Level)}, nil
	}

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

// TurtleSoupValidateSolution: 사용자의 정답 시도를 검증합니다.
func (c *Client) TurtleSoupValidateSolution(
	ctx context.Context,
	chatID string,
	namespace string,
	solution string,
	playerAnswer string,
) (*TurtleSoupValidateResponse, error) {
	if c.grpcClient != nil {
		callCtx, cancel := c.grpcCallContext(ctx)
		defer cancel()

		req := &llmv1.TurtleSoupValidateSolutionRequest{
			ChatId:       &chatID,
			Namespace:    &namespace,
			Solution:     solution,
			PlayerAnswer: playerAnswer,
		}
		resp, err := c.grpcClient.TurtleSoupValidateSolution(callCtx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc turtlesoup validate failed: %w", err)
		}

		return &TurtleSoupValidateResponse{Result: resp.Result, RawText: resp.RawText}, nil
	}

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

// TurtleSoupRewriteScenario: 사용자가 입력한 시나리오를 게임에 맞게 최적화/재작성합니다.
func (c *Client) TurtleSoupRewriteScenario(
	ctx context.Context,
	title string,
	scenario string,
	solution string,
	difficulty int,
) (*TurtleSoupRewriteResponse, error) {
	if c.grpcClient != nil {
		callCtx, cancel := c.grpcCallContext(ctx)
		defer cancel()

		req := &llmv1.TurtleSoupRewriteScenarioRequest{
			Title:      title,
			Scenario:   scenario,
			Solution:   solution,
			Difficulty: int32(difficulty),
		}
		resp, err := c.grpcClient.TurtleSoupRewriteScenario(callCtx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc turtlesoup rewrite failed: %w", err)
		}

		return &TurtleSoupRewriteResponse{
			Scenario:         resp.Scenario,
			Solution:         resp.Solution,
			OriginalScenario: resp.OriginalScenario,
			OriginalSolution: resp.OriginalSolution,
		}, nil
	}

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

// TurtleSoupGeneratePuzzle: 새로운 퍼즐을 자동으로 생성합니다.
func (c *Client) TurtleSoupGeneratePuzzle(ctx context.Context, req TurtleSoupPuzzleGenerationRequest) (*TurtleSoupPuzzleGenerationResponse, error) {
	if c.grpcClient != nil {
		callCtx, cancel := c.grpcCallContext(ctx)
		defer cancel()

		grpcReq := &llmv1.TurtleSoupGeneratePuzzleRequest{
			Category: req.Category,
			Theme:    req.Theme,
		}
		if req.Difficulty != nil {
			value := int32(*req.Difficulty)
			grpcReq.Difficulty = &value
		}

		resp, err := c.grpcClient.TurtleSoupGeneratePuzzle(callCtx, grpcReq)
		if err != nil {
			return nil, fmt.Errorf("grpc turtlesoup generate puzzle failed: %w", err)
		}

		return &TurtleSoupPuzzleGenerationResponse{
			Title:      resp.Title,
			Scenario:   resp.Scenario,
			Solution:   resp.Solution,
			Category:   resp.Category,
			Difficulty: int(resp.Difficulty),
			Hints:      resp.Hints,
		}, nil
	}

	var out TurtleSoupPuzzleGenerationResponse
	if err := c.Post(ctx, "/api/turtle-soup/puzzles", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TurtleSoupGetRandomPuzzle: 프리셋 퍼즐 중 하나를 랜덤으로 가져옵니다.
func (c *Client) TurtleSoupGetRandomPuzzle(ctx context.Context, difficulty *int) (*TurtleSoupPuzzlePresetResponse, error) {
	if c.grpcClient != nil {
		callCtx, cancel := c.grpcCallContext(ctx)
		defer cancel()

		req := &llmv1.TurtleSoupGetRandomPuzzleRequest{}
		if difficulty != nil {
			value := int32(*difficulty)
			req.Difficulty = &value
		}

		resp, err := c.grpcClient.TurtleSoupGetRandomPuzzle(callCtx, req)
		if err != nil {
			return nil, fmt.Errorf("grpc turtlesoup get random puzzle failed: %w", err)
		}

		var id *int
		if resp.Id != nil {
			value := int(*resp.Id)
			id = &value
		}

		var diff *int
		if resp.Difficulty != nil {
			value := int(*resp.Difficulty)
			diff = &value
		}

		return &TurtleSoupPuzzlePresetResponse{
			ID:         id,
			Title:      resp.Title,
			Question:   resp.Question,
			Answer:     resp.Answer,
			Difficulty: diff,
		}, nil
	}

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
