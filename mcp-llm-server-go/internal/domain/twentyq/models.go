package twentyq

import (
	"strings"

	domainmodels "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/models"
)

// AnswerScale 는 스무고개 답변 척도 타입이다.
type AnswerScale string

const (
	// AnswerYes 는 긍정 답변이다.
	AnswerYes AnswerScale = domainmodels.AnswerYesText
	// AnswerProbablyYes 는 아마도 예 답변이다.
	AnswerProbablyYes AnswerScale = "아마도 예"
	// AnswerProbablyNo 는 아마도 아니오 답변이다.
	AnswerProbablyNo AnswerScale = "아마도 아니오"
	// AnswerNo 는 부정 답변이다.
	AnswerNo AnswerScale = domainmodels.AnswerNoText
	// AnswerPolicyViolation 는 정책 위반 질문이다. 히스토리에 기록되지 않는다.
	AnswerPolicyViolation AnswerScale = "정책 위반"
)

var answerScales = []AnswerScale{
	AnswerYes,
	AnswerProbablyYes,
	AnswerProbablyNo,
	AnswerNo,
	AnswerPolicyViolation,
}

// ParseAnswerScale 는 답변 척도를 파싱한다.
func ParseAnswerScale(text string) (AnswerScale, bool) {
	text = strings.TrimSpace(text)
	for _, scale := range answerScales {
		if strings.Contains(text, string(scale)) {
			return scale, true
		}
	}
	return "", false
}

// VerifyResult 는 정답 검증 결과 타입이다.
type VerifyResult string

const (
	// VerifyAccept 는 정답 판정이다.
	VerifyAccept VerifyResult = "정답"
	// VerifyClose 는 근접 판정이다.
	VerifyClose VerifyResult = "근접"
	// VerifyReject 는 오답 판정이다.
	VerifyReject VerifyResult = "오답"
)

// VerifyResultName 는 검증 결과를 영문 코드로 변환한다.
func VerifyResultName(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case string(VerifyAccept):
		return "ACCEPT", true
	case string(VerifyClose):
		return "CLOSE", true
	case string(VerifyReject):
		return "REJECT", true
	default:
		return "", false
	}
}

// SynonymResult 는 유사어 판정 결과 타입이다.
type SynonymResult string

const (
	// SynonymEquivalent 는 동일 판정이다.
	SynonymEquivalent SynonymResult = "동일"
	// SynonymNotEquivalent 는 상이 판정이다.
	SynonymNotEquivalent SynonymResult = "상이"
)

// SynonymResultName 는 유사어 판정을 영문 코드로 변환한다.
func SynonymResultName(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case string(SynonymEquivalent):
		return "EQUIVALENT", true
	case string(SynonymNotEquivalent):
		return "NOT_EQUIVALENT", true
	default:
		return "", false
	}
}

// HintsOutput 은 힌트 출력 스키마다.
type HintsOutput struct {
	Reasoning string   `json:"reasoning"`
	Hints     []string `json:"hints"`
}

// NormalizeOutput 은 정규화 출력 스키마다.
type NormalizeOutput struct {
	Normalized string `json:"normalized"`
}

// VerifyOutput 은 검증 출력 스키마다.
type VerifyOutput struct {
	Reasoning string `json:"reasoning"`
	Result    string `json:"result"`
}

// SynonymOutput 은 유사어 출력 스키마다.
type SynonymOutput struct {
	Result string `json:"result"`
}

// hintsSchema 는 힌트 스키마다 (reasoning + hints 배열).
var hintsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"reasoning": map[string]any{
			"type":        "string",
			"description": "Thought process for creating the poetic hint",
		},
		"hints": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
	},
	"required": []string{"reasoning", "hints"},
}

var normalizeSchema = domainmodels.RequiredStringFieldSchema("normalized")

var verifySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"reasoning": map[string]any{
			"type":        "string",
			"description": "Step-by-step thought process for the verification decision",
		},
		"result": map[string]any{
			"type": "string",
			"enum": []string{
				string(VerifyAccept),
				string(VerifyClose),
				string(VerifyReject),
			},
		},
	},
	"required": []string{"reasoning", "result"},
}

var synonymSchema = domainmodels.RequiredStringFieldSchema("result")

// HintsSchema 는 힌트 JSON 스키마를 반환한다.
func HintsSchema() map[string]any {
	return hintsSchema
}

// NormalizeSchema 는 정규화 JSON 스키마를 반환한다.
func NormalizeSchema() map[string]any {
	return normalizeSchema
}

// VerifySchema 는 검증 JSON 스키마를 반환한다.
func VerifySchema() map[string]any {
	return verifySchema
}

// SynonymSchema 는 유사어 JSON 스키마를 반환한다.
func SynonymSchema() map[string]any {
	return synonymSchema
}

// AnswerOutput 은 답변 출력 스키마다.
type AnswerOutput struct {
	Reasoning  string  `json:"reasoning"`
	Answer     string  `json:"answer"`
	Confidence float64 `json:"confidence"`
}

// answerSchema 는 답변 스키마다 (reasoning + enum 제약 + confidence).
var answerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"reasoning": map[string]any{
			"type":        "string",
			"description": "Step-by-step thought process explaining how you arrived at the answer",
		},
		"answer": map[string]any{
			"type": "string",
			"enum": []string{
				string(AnswerYes),
				string(AnswerProbablyYes),
				string(AnswerProbablyNo),
				string(AnswerNo),
				string(AnswerPolicyViolation),
			},
		},
		"confidence": map[string]any{
			"type":        "number",
			"minimum":     0.0,
			"maximum":     1.0,
			"description": "Confidence level 0.0-1.0. Use < 0.5 if uncertain, prefer 아마도 scales when low confidence.",
		},
	},
	"required": []string{"reasoning", "answer", "confidence"},
}

// AnswerSchema 는 답변 JSON 스키마를 반환한다.
func AnswerSchema() map[string]any {
	return answerSchema
}
