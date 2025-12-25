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
)

var answerScales = []AnswerScale{
	AnswerYes,
	AnswerProbablyYes,
	AnswerProbablyNo,
	AnswerNo,
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
	Hints []string `json:"hints"`
}

// NormalizeOutput 은 정규화 출력 스키마다.
type NormalizeOutput struct {
	Normalized string `json:"normalized"`
}

// VerifyOutput 은 검증 출력 스키마다.
type VerifyOutput struct {
	Result string `json:"result"`
}

// SynonymOutput 은 유사어 출력 스키마다.
type SynonymOutput struct {
	Result string `json:"result"`
}

var hintsSchema = domainmodels.RequiredStringArrayFieldSchema("hints")

var normalizeSchema = domainmodels.RequiredStringFieldSchema("normalized")

var verifySchema = domainmodels.RequiredStringFieldSchema("result")

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
	Answer string `json:"answer"`
}

// answerSchema 는 답변 스키마다 (enum 제약).
var answerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answer": map[string]any{
			"type": "string",
			"enum": []string{
				string(AnswerYes),
				string(AnswerProbablyYes),
				string(AnswerProbablyNo),
				string(AnswerNo),
			},
		},
	},
	"required": []string{"answer"},
}

// AnswerSchema 는 답변 JSON 스키마를 반환한다.
func AnswerSchema() map[string]any {
	return answerSchema
}
