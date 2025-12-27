package model

import (
	"strings"

	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

// RiddleSecret: 스무고개 게임의 정답 및 메타데이터를 담고 있는 구조체 (Redis 세션에 저장됨)
type RiddleSecret struct {
	Target      string `json:"target"`
	Category    string `json:"category"`
	Intro       string `json:"intro"`
	Description string `json:"description,omitempty"`
}

// QuestionHistory: 사용자의 질문과 그에 대한 AI의 답변 기록
type QuestionHistory struct {
	QuestionNumber   int     `json:"questionNumber"`
	Question         string  `json:"question"`
	Answer           string  `json:"answer"`
	IsChain          bool    `json:"isChain"`
	ThoughtSignature *string `json:"thoughtSignature,omitempty"`
	UserID           *string `json:"userId,omitempty"`
}

// HintHistory: 게임 중 제공된 힌트의 기록
type HintHistory struct {
	HintNumber int    `json:"hintNumber"`
	Content    string `json:"content"`
}

// RiddleStatusResponse: 현재 게임의 진행 상황(질문/답변 내역, 힌트, 설정 등)을 클라이언트에 전달하기 위한 응답 구조체
type RiddleStatusResponse struct {
	QuestionCount    int               `json:"questionCount"`
	Questions        []QuestionHistory `json:"questions"`
	Hints            []HintHistory     `json:"hints"`
	HintCount        int               `json:"hintCount"`
	MaxHints         int               `json:"maxHints"`
	SelectedCategory *string           `json:"selectedCategory,omitempty"`
}

// PlayerInfo: 게임 참여자 정보 (ID, 닉네임)
type PlayerInfo struct {
	UserID string `json:"userId"`
	Sender string `json:"sender"`
}

// PendingMessage: pending.Message 타입 재정의
type PendingMessage = domainmodels.PendingMessage

// SurrenderVote: surrender.Vote 타입 재정의
type SurrenderVote = domainmodels.SurrenderVote

// FiveScaleKo: 5단계 긍정/부정 답변 타입
type FiveScaleKo int

// FiveScaleAlwaysYes 등: 5단계 응답 상수
const (
	FiveScaleAlwaysYes FiveScaleKo = iota
	FiveScaleMostlyYes
	FiveScaleMostlyNo
	FiveScaleAlwaysNo
	FiveScaleInvalid
	FiveScalePolicyViolation
)

var fiveScaleTokenToValue = map[string]FiveScaleKo{
	"예":              FiveScaleAlwaysYes,
	"아마도 예":          FiveScaleMostlyYes,
	"아마도 아니오":        FiveScaleMostlyNo,
	"아니오":            FiveScaleAlwaysNo,
	"이해할 수 없는 질문입니다": FiveScaleInvalid,
	"정책 위반":          FiveScalePolicyViolation,
}

// ParseFiveScaleKo: AI의 답변 문자열을 분석하여 FiveScaleKo 열거형 값으로 변환한다.
// 문장 부호 제거 및 정규화를 수행한 후 매핑한다.
func ParseFiveScaleKo(raw string) (*FiveScaleKo, bool) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return nil, false
	}

	cleaned = strings.Trim(cleaned, "\"'")
	cleaned = strings.ReplaceAll(cleaned, "\u3000", " ")
	cleaned = strings.ReplaceAll(cleaned, "\u3002", ".")
	cleaned = strings.ReplaceAll(cleaned, "\uff0c", ",")
	cleaned = strings.TrimSpace(cleaned)

	cleaned = strings.TrimSuffix(cleaned, ".")
	cleaned = strings.TrimSuffix(cleaned, "!")
	cleaned = strings.TrimSuffix(cleaned, "?")
	cleaned = strings.TrimSuffix(cleaned, "\u3002")
	cleaned = strings.TrimSuffix(cleaned, "\uff01")
	cleaned = strings.TrimSuffix(cleaned, "\uff1f")
	cleaned = strings.TrimSpace(cleaned)

	value, ok := fiveScaleTokenToValue[cleaned]
	if !ok {
		return nil, false
	}
	return &value, true
}

// FiveScaleToken: FiveScaleKo 열거형 값을 해당하는 한국어 토큰 문자열로 변환한다.
func FiveScaleToken(value FiveScaleKo) string {
	for token, v := range fiveScaleTokenToValue {
		if v == value {
			return token
		}
	}
	return ""
}

// ChainCondition: 체인 질문(연속 질문)의 실행 조건을 정의하는 열거형
type ChainCondition int

const (
	// ChainConditionAlways 무조건 실행 (기본값).
	ChainConditionAlways ChainCondition = iota
	// ChainConditionIfTrue 긍정 답변 시만 실행 ("예" 또는 "아마도 예").
	ChainConditionIfTrue
)

// ShouldContinue: 이전 질문의 답변(scale)에 따라 체인 질문을 계속 진행할지 여부를 결정한다.
func (c ChainCondition) ShouldContinue(scale FiveScaleKo) bool {
	switch c {
	case ChainConditionAlways:
		return true
	case ChainConditionIfTrue:
		return scale == FiveScaleAlwaysYes || scale == FiveScaleMostlyYes
	default:
		return true
	}
}
