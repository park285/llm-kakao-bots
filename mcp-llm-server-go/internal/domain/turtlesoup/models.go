package turtlesoup

import (
	"strings"

	domainmodels "github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/models"
)

// AnswerType: 정답 판정 결과 타입입니다.
type AnswerType string

const (
	// AnswerYes 는 긍정 답변입니다.
	AnswerYes AnswerType = domainmodels.AnswerYesText
	// AnswerNo 는 부정 답변입니다.
	AnswerNo AnswerType = domainmodels.AnswerNoText
	// AnswerIrrelevant 는 무관함 답변입니다.
	AnswerIrrelevant AnswerType = "관계없습니다"
	// AnswerSomewhatRelated 는 부분 관련 답변입니다.
	AnswerSomewhatRelated AnswerType = "조금은 관계있습니다"
	// AnswerWrongPremise 는 전제 오류 답변입니다.
	AnswerWrongPremise AnswerType = "전제가 틀렸습니다"
	// AnswerCannotAnswer 는 응답 불가 답변입니다.
	AnswerCannotAnswer AnswerType = "답변할 수 없습니다"
	// AnswerImportantMessage 는 중요 질문 메시지다.
	AnswerImportantMessage = "중요한 질문입니다"
)

var turtleBaseAnswers = []AnswerType{
	AnswerSomewhatRelated,
	AnswerIrrelevant,
	AnswerWrongPremise,
	AnswerCannotAnswer,
	AnswerNo,
	AnswerYes,
}

// IsImportantAnswer: 중요 질문 메시지 포함 여부를 판단합니다.
func IsImportantAnswer(rawText string) bool {
	normalized := strings.ReplaceAll(rawText, " ", "")
	return strings.Contains(normalized, strings.ReplaceAll(AnswerImportantMessage, " ", "")) ||
		strings.Contains(normalized, "중요합니다")
}

// ParseBaseAnswer: 답변 문장에서 기본 답변 타입을 추출합니다.
func ParseBaseAnswer(rawText string) (AnswerType, bool) {
	raw := strings.TrimSpace(rawText)
	for _, answer := range turtleBaseAnswers {
		if strings.HasPrefix(raw, string(answer)) {
			return answer, true
		}
	}
	for _, answer := range turtleBaseAnswers {
		if strings.Contains(raw, string(answer)) {
			return answer, true
		}
	}
	return "", false
}

// FormatAnswerText: 기본 답변과 중요 표시를 합쳐 문자열로 만든다.
func FormatAnswerText(base AnswerType, isImportant bool) string {
	if base == "" {
		return ""
	}
	if !isImportant {
		return string(base)
	}
	if base == AnswerNo {
		return "아니오 하지만 중요한 질문입니다!"
	}
	return string(base) + ", 중요한 질문입니다!"
}

// ValidationResult: 정답 검증 결과 타입입니다.
type ValidationResult string

const (
	// ValidationYes 는 정답 판정입니다.
	ValidationYes ValidationResult = "YES"
	// ValidationNo 는 오답 판정입니다.
	ValidationNo ValidationResult = "NO"
	// ValidationClose 는 유사 판정입니다.
	ValidationClose ValidationResult = "CLOSE"
)

// ParseValidationResult: 검증 결과를 파싱합니다.
func ParseValidationResult(rawText string) (ValidationResult, bool) {
	upper := strings.ToUpper(rawText)
	for _, candidate := range []ValidationResult{ValidationYes, ValidationNo, ValidationClose} {
		if strings.Contains(upper, string(candidate)) {
			return candidate, true
		}
	}
	return "", false
}

// HintOutput: 힌트 출력 스키마다.
type HintOutput struct {
	Hint string `json:"hint"`
}

// PuzzleOutput: 퍼즐 생성 출력 스키마다.
type PuzzleOutput struct {
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Solution   string   `json:"solution"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"`
	Hints      []string `json:"hints"`
}

// RewriteOutput: 리라이트 출력 스키마다.
type RewriteOutput struct {
	Scenario string `json:"scenario"`
	Solution string `json:"solution"`
}

var hintSchema = domainmodels.RequiredStringFieldSchema("hint")

var puzzleSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title":      map[string]any{"type": "string"},
		"scenario":   map[string]any{"type": "string"},
		"solution":   map[string]any{"type": "string"},
		"category":   map[string]any{"type": "string"},
		"difficulty": map[string]any{"type": "integer"},
		"hints": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required": []string{"title", "scenario", "solution", "category", "difficulty", "hints"},
}

var rewriteSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"scenario": map[string]any{"type": "string"},
		"solution": map[string]any{"type": "string"},
	},
	"required": []string{"scenario", "solution"},
}

// HintSchema: 힌트 JSON 스키마를 반환합니다.
func HintSchema() map[string]any {
	return hintSchema
}

// PuzzleSchema: 퍼즐 JSON 스키마를 반환합니다.
func PuzzleSchema() map[string]any {
	return puzzleSchema
}

// RewriteSchema: 리라이트 JSON 스키마를 반환합니다.
func RewriteSchema() map[string]any {
	return rewriteSchema
}

// AnswerOutput: 답변 출력 스키마다.
type AnswerOutput struct {
	Answer    string `json:"answer"`
	Important bool   `json:"important"`
}

// answerSchema 는 답변 스키마다 (enum 제약).
var answerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answer": map[string]any{
			"type": "string",
			"enum": []string{
				string(AnswerYes),
				string(AnswerNo),
				string(AnswerIrrelevant),
				string(AnswerSomewhatRelated),
				string(AnswerWrongPremise),
				string(AnswerCannotAnswer),
			},
		},
		"important": map[string]any{
			"type": "boolean",
		},
	},
	"required": []string{"answer", "important"},
}

// AnswerSchema: 답변 JSON 스키마를 반환합니다.
func AnswerSchema() map[string]any {
	return answerSchema
}

// ValidateOutput: 검증 출력 스키마다.
type ValidateOutput struct {
	Result string `json:"result"`
}

// validateSchema 는 검증 스키마다 (enum 제약).
var validateSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"result": map[string]any{
			"type": "string",
			"enum": []string{
				string(ValidationYes),
				string(ValidationNo),
				string(ValidationClose),
			},
		},
	},
	"required": []string{"result"},
}

// ValidateSchema: 검증 JSON 스키마를 반환합니다.
func ValidateSchema() map[string]any {
	return validateSchema
}
