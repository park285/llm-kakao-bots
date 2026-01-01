package util

import "strings"

// TruncateString: 주어진 문자열을 최대 길이(Rune 기준)로 자르고, 초과 시 "..."을 붙여 반환합니다.
func TruncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// TrimSpace: 문자열 양쪽 끝의 공백을 제거한다. (strings.TrimSpace 래퍼)
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// Normalize: 문자열을 소문자로 변환하고 양쪽 공백을 제거합니다.
func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeSuffix: 문자열에서 "짱", "쨩"과 같은 한국어 호칭 접미사를 제거하고 정규화합니다.
func NormalizeSuffix(s string) string {
	normalized := Normalize(s)

	if strings.HasSuffix(normalized, "짱") {
		return normalized[:len(normalized)-len("짱")]
	}

	if strings.HasSuffix(normalized, "쨩") {
		return normalized[:len(normalized)-len("쨩")]
	}

	return normalized
}

// NormalizeKey: 검색 키 생성을 위해 특수문자, 공백 등을 제거하여 문자열을 정규화합니다.
func NormalizeKey(name string) string {
	name = Normalize(name)
	if name == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range name {
		switch r {
		case ' ', '-', '_', '.', '!', '☆', '・', '\u2018', '\u2019', '\'', 'ー', '—':
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// Slugify: URL 등에 사용하기 적합하도록 문자열을 변환한다. (공백 -> "-", 특수문자 제거)
func Slugify(name string) string {
	name = Normalize(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "!", "")
	return name
}

// Contains: 문자열 슬라이스에 특정 문자열이 포함되어 있는지 확인합니다.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// StripLeadingHeader: 텍스트 앞부분의 헤더 문자열을 제거합니다.
// 여러 개행 패턴을 시도하여 가장 적절한 방식으로 제거한다.
func StripLeadingHeader(text, header string) string {
	if TrimSpace(text) == "" || TrimSpace(header) == "" {
		return text
	}
	candidates := []string{
		header + "\r\n\r\n",
		header + "\n\n",
		header + "\r\n",
		header + "\n",
		header,
	}
	for _, candidate := range candidates {
		if strings.HasPrefix(text, candidate) {
			return strings.TrimPrefix(text, candidate)
		}
	}
	return text
}
