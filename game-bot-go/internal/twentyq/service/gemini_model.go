package service

import "strings"

const tokensPerMillion = 1_000_000.0

// GeminiModel: Gemini 모델별 비용 계산 타입을 정의
type GeminiModel int

// GeminiModelFlash25: Gemini 모델 상수 (2.5 Flash, 2.5 Pro 등)
const (
	GeminiModelFlash25 GeminiModel = iota
	GeminiModelPro25
	GeminiModelPro30
	GeminiModelFlash30
)

// ParseGeminiModel: 문자열 모델명을 GeminiModel Enum으로 파싱한다.
func ParseGeminiModel(value *string) GeminiModel {
	if value == nil {
		return GeminiModelFlash25
	}

	normalized :=
		strings.TrimSpace(*value)
	normalized = strings.ToLower(normalized)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.TrimPrefix(normalized, "google/")
	if normalized == "" {
		return GeminiModelFlash25
	}

	switch {
	case normalized == "pro":
		return GeminiModelPro30
	case strings.Contains(normalized, "flash"):
		if strings.Contains(normalized, "flash-30") ||
			strings.Contains(normalized, "3-flash") ||
			(strings.Contains(normalized, "3.0") && strings.Contains(normalized, "flash")) ||
			strings.Contains(normalized, "gemini-3") {
			return GeminiModelFlash30
		}
		return GeminiModelFlash25
	case strings.Contains(normalized, "pro-25") || (strings.Contains(normalized, "2.5") && strings.Contains(normalized, "pro")):
		return GeminiModelPro25
	case strings.Contains(normalized, "pro"):
		return GeminiModelPro30
	case strings.Contains(normalized, "2.5"):
		return GeminiModelFlash25
	default:
		return GeminiModelFlash25
	}
}

// ResolveGeminiModel: 사용자 override와 서버 설정 중 우선순위에 따라 모델을 결정한다.
func ResolveGeminiModel(modelOverride *string, serverModel *string) GeminiModel {
	if modelOverride != nil && strings.TrimSpace(*modelOverride) != "" {
		return ParseGeminiModel(modelOverride)
	}
	if serverModel != nil && strings.TrimSpace(*serverModel) != "" {
		return ParseGeminiModel(serverModel)
	}
	return GeminiModelFlash25
}

// DisplayName: 사용자에게 표시할 모델 이름을 반환한다.
func (m GeminiModel) DisplayName() string {
	switch m {
	case GeminiModelFlash30:
		return "3.0 Flash"
	case GeminiModelPro25:
		return "2.5 Pro"
	case GeminiModelPro30:
		return "3.0 Pro"
	default:
		return "2.5 Flash"
	}
}

func (m GeminiModel) prices() (inputPrice float64, outputPrice float64) {
	switch m {
	case GeminiModelFlash30:
		return 0.50, 3.00
	case GeminiModelPro25:
		return 1.25, 10.00
	case GeminiModelPro30:
		return 2.00, 12.00
	default:
		return 0.30, 2.50
	}
}

// CalculateCostUsd: 사용한 토큰 양에 따른 예상 비용(USD)을 계산한다.
func (m GeminiModel) CalculateCostUsd(inputTokens int64, outputTokens int64, reasoningTokens int64) float64 {
	inputPrice, outputPrice := m.prices()
	totalOutput := float64(outputTokens)
	return (float64(inputTokens)*inputPrice + totalOutput*outputPrice) / tokensPerMillion
}
