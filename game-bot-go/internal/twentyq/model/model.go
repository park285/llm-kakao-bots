package model

import (
	"strings"

	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

// RiddleSecret 는 타입이다.
type RiddleSecret struct {
	Target      string `json:"target"`
	Category    string `json:"category"`
	Intro       string `json:"intro"`
	Description string `json:"description,omitempty"`
}

// QuestionHistory 는 타입이다.
type QuestionHistory struct {
	QuestionNumber   int     `json:"questionNumber"`
	Question         string  `json:"question"`
	Answer           string  `json:"answer"`
	IsChain          bool    `json:"isChain"`
	ThoughtSignature *string `json:"thoughtSignature,omitempty"`
	UserID           *string `json:"userId,omitempty"`
}

// HintHistory 는 타입이다.
type HintHistory struct {
	HintNumber int    `json:"hintNumber"`
	Content    string `json:"content"`
}

// RiddleStatusResponse 는 타입이다.
type RiddleStatusResponse struct {
	QuestionCount    int               `json:"questionCount"`
	Questions        []QuestionHistory `json:"questions"`
	Hints            []HintHistory     `json:"hints"`
	HintCount        int               `json:"hintCount"`
	MaxHints         int               `json:"maxHints"`
	SelectedCategory *string           `json:"selectedCategory,omitempty"`
}

// PlayerInfo 는 타입이다.
type PlayerInfo struct {
	UserID string `json:"userId"`
	Sender string `json:"sender"`
}

// PendingMessage 는 타입이다.
type PendingMessage = domainmodels.PendingMessage

// SurrenderVote 는 타입이다.
type SurrenderVote = domainmodels.SurrenderVote

// FiveScaleKo 는 타입이다.
type FiveScaleKo int

// FiveScaleAlwaysYes 는 5단계 응답 상수 목록이다.
const (
	FiveScaleAlwaysYes FiveScaleKo = iota
	FiveScaleMostlyYes
	FiveScaleMostlyNo
	FiveScaleAlwaysNo
	FiveScaleInvalid
)

var fiveScaleTokenToValue = map[string]FiveScaleKo{
	"예":              FiveScaleAlwaysYes,
	"아마도 예":          FiveScaleMostlyYes,
	"아마도 아니오":        FiveScaleMostlyNo,
	"아니오":            FiveScaleAlwaysNo,
	"이해할 수 없는 질문입니다": FiveScaleInvalid,
}

// ParseFiveScaleKo 는 동작을 수행한다.
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

// FiveScaleToken 는 동작을 수행한다.
func FiveScaleToken(value FiveScaleKo) string {
	for token, v := range fiveScaleTokenToValue {
		if v == value {
			return token
		}
	}
	return ""
}

// ChainCondition 체인 질문 실행 조건.
type ChainCondition int

const (
	// ChainConditionAlways 무조건 실행 (기본값).
	ChainConditionAlways ChainCondition = iota
	// ChainConditionIfTrue 긍정 답변 시만 실행 ("예" 또는 "아마도 예").
	ChainConditionIfTrue
)

// ShouldContinue 주어진 응답 스케일에 따라 체인 질문 계속 여부 결정.
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
