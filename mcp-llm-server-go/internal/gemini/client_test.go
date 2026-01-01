package gemini

import (
	"errors"
	"testing"

	"google.golang.org/genai"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

func TestNormalizeThinkingLevel(t *testing.T) {
	level, ok := normalizeThinkingLevel("low")
	if !ok || level != genai.ThinkingLevelLow {
		t.Fatalf("unexpected thinking level")
	}

	if _, ok := normalizeThinkingLevel("none"); ok {
		t.Fatalf("expected none to be disabled")
	}

	if _, ok := normalizeThinkingLevel("unknown"); ok {
		t.Fatalf("expected unknown to be disabled")
	}
}

func TestIsGemini3(t *testing.T) {
	if !isGemini3("gemini-3-test") {
		t.Fatalf("expected gemini-3 match")
	}
	if isGemini3("gemini-2-test") {
		t.Fatalf("did not expect gemini-2 match")
	}
}

func TestBuildContents(t *testing.T) {
	history := []llm.HistoryEntry{
		{Role: "assistant", Content: "A1"},
		{Role: "user", Content: "Q1"},
		{Role: "SYSTEM", Content: "SYS"},
	}
	contents := buildContents("prompt", history)
	if len(contents) != 4 {
		t.Fatalf("expected 4 contents, got %d", len(contents))
	}
	if contents[0].Role != string(genai.RoleModel) {
		t.Fatalf("expected model role, got %s", contents[0].Role)
	}
	if contents[0].Parts[0].Text != "A1" {
		t.Fatalf("expected A1, got %s", contents[0].Parts[0].Text)
	}
	if contents[1].Role != string(genai.RoleUser) {
		t.Fatalf("expected user role, got %s", contents[1].Role)
	}
	if contents[2].Role != string(genai.RoleUser) {
		t.Fatalf("expected user role for system, got %s", contents[2].Role)
	}
	if contents[3].Role != string(genai.RoleUser) {
		t.Fatalf("expected user role for prompt, got %s", contents[3].Role)
	}
	if contents[3].Parts[0].Text != "prompt" {
		t.Fatalf("expected prompt text, got %s", contents[3].Parts[0].Text)
	}
}

func TestExtractParts(t *testing.T) {
	texts, thoughts := extractParts(nil)
	if texts != nil || thoughts != nil {
		t.Fatalf("expected nil parts for nil response")
	}

	response := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "answer"},
						{Text: "thought", Thought: true},
						{Text: ""},
						nil,
					},
				},
			},
		},
	}
	texts, thoughts = extractParts(response)
	if len(texts) != 1 || texts[0] != "answer" {
		t.Fatalf("unexpected texts: %v", texts)
	}
	if len(thoughts) != 1 || thoughts[0] != "thought" {
		t.Fatalf("unexpected thoughts: %v", thoughts)
	}
}

func TestExtractUsage(t *testing.T) {
	response := &genai.GenerateContentResponse{
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			ThoughtsTokenCount:   3,
			TotalTokenCount:      33,
		},
	}
	usage := extractUsage(response)
	if usage.InputTokens != 10 {
		t.Fatalf("unexpected input tokens: %d", usage.InputTokens)
	}
	if usage.OutputTokens != 23 {
		t.Fatalf("unexpected output tokens: %d", usage.OutputTokens)
	}
	if usage.TotalTokens != 33 {
		t.Fatalf("unexpected total tokens: %d", usage.TotalTokens)
	}
	if usage.ReasoningTokens != 3 {
		t.Fatalf("unexpected reasoning tokens: %d", usage.ReasoningTokens)
	}
}

func TestResolveModel(t *testing.T) {
	cfg := &config.Config{
		Gemini: config.GeminiConfig{
			DefaultModel: "gemini-3-default",
			AnswerModel:  "gemini-3-answer",
		},
	}
	client := &Client{cfg: cfg}

	model, err := client.resolveModel("", "answer")
	if err != nil || model != "gemini-3-answer" {
		t.Fatalf("expected answer model, got model=%s err=%v", model, err)
	}

	model, err = client.resolveModel("gemini-3-override", "answer")
	if err != nil || model != "gemini-3-override" {
		t.Fatalf("expected override model, got model=%s err=%v", model, err)
	}

	model, err = client.resolveModel("gemini-2-test", "answer")
	if !errors.Is(err, ErrInvalidModel) || model != "gemini-2-test" {
		t.Fatalf("expected invalid model error, got model=%s err=%v", model, err)
	}

	emptyCfg := &config.Config{Gemini: config.GeminiConfig{}}
	emptyClient := &Client{cfg: emptyCfg}
	model, err = emptyClient.resolveModel("", "answer")
	if !errors.Is(err, ErrInvalidModel) || model != "" {
		t.Fatalf("expected empty invalid model, got model=%s err=%v", model, err)
	}
}

func TestPickConsensusWinner(t *testing.T) {
	tests := []struct {
		name          string
		scores        map[string]consensusScore
		expectedValue string
	}{
		{
			name:   "empty",
			scores: map[string]consensusScore{
				// empty
			},
			expectedValue: "",
		},
		{
			// 핵심 변경: Count(다수결)가 WeightSum보다 우선
			// "확신에 찬 소수"가 다수를 이기지 못함
			name: "count_priority_over_weight",
			scores: map[string]consensusScore{
				"정답": {Count: 2, WeightSum: 0.8, MaxConfidence: 0.4},
				"오답": {Count: 1, WeightSum: 0.95, MaxConfidence: 0.95},
			},
			expectedValue: "정답", // Count 2 > 1 → 정답 승리
		},
		{
			// 동일 Count일 때 WeightSum으로 결정
			name: "weight_when_count_tied",
			scores: map[string]consensusScore{
				"정답": {Count: 2, WeightSum: 1.5, MaxConfidence: 0.8},
				"오답": {Count: 2, WeightSum: 1.0, MaxConfidence: 0.6},
			},
			expectedValue: "정답", // Count 동일 → WeightSum 1.5 > 1.0
		},
		{
			name: "tie_break_by_count",
			scores: map[string]consensusScore{
				"A": {Count: 1, WeightSum: 1.0, MaxConfidence: 1.0},
				"B": {Count: 2, WeightSum: 1.0, MaxConfidence: 0.5},
			},
			expectedValue: "B", // Count 2 > 1
		},
		{
			name: "tie_break_by_max_confidence",
			scores: map[string]consensusScore{
				"A": {Count: 2, WeightSum: 1.0, MaxConfidence: 0.6},
				"B": {Count: 2, WeightSum: 1.0, MaxConfidence: 0.8},
			},
			expectedValue: "B", // Count, Weight 동일 → MaxConfidence 0.8 > 0.6
		},
		{
			// 명시적 우선순위 맵 테스트: 정답 > 근접 > 오답
			name: "priority_map_accept_over_close",
			scores: map[string]consensusScore{
				"정답": {Count: 1, WeightSum: 1.0, MaxConfidence: 0.5},
				"근접": {Count: 1, WeightSum: 1.0, MaxConfidence: 0.5},
			},
			expectedValue: "정답", // 동점 → 우선순위 맵: 정답(3) > 근접(2)
		},
		{
			name: "priority_map_close_over_reject",
			scores: map[string]consensusScore{
				"근접": {Count: 1, WeightSum: 1.0, MaxConfidence: 0.5},
				"오답": {Count: 1, WeightSum: 1.0, MaxConfidence: 0.5},
			},
			expectedValue: "근접", // 동점 → 우선순위 맵: 근접(2) > 오답(1)
		},
		{
			name: "priority_map_accept_over_reject",
			scores: map[string]consensusScore{
				"정답": {Count: 1, WeightSum: 0.5, MaxConfidence: 0.5},
				"오답": {Count: 1, WeightSum: 0.5, MaxConfidence: 0.5},
			},
			expectedValue: "정답", // 동점 → 우선순위 맵: 정답(3) > 오답(1)
		},
		{
			// 알 수 없는 값은 우선순위 0으로 처리됨
			name: "unknown_values_fallback",
			scores: map[string]consensusScore{
				"a": {Count: 1, WeightSum: 1.0, MaxConfidence: 0.5},
				"b": {Count: 1, WeightSum: 1.0, MaxConfidence: 0.5},
			},
			expectedValue: "a", // 둘 다 우선순위 0 → 먼저 처리된 값 유지 (비결정적이지만 테스트에서는 map 순서에 의존)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, _ := pickConsensusWinner(tc.scores)
			if value != tc.expectedValue {
				t.Fatalf("expected %q, got %q", tc.expectedValue, value)
			}
		})
	}
}
