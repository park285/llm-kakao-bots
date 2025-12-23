package util

import "strings"

// TruncateString 는 동작을 수행한다.
func TruncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// TrimSpace 는 동작을 수행한다.
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// Normalize 는 동작을 수행한다.
func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeSuffix 는 동작을 수행한다.
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

// NormalizeKey 는 동작을 수행한다.
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

// Slugify 는 동작을 수행한다.
func Slugify(name string) string {
	name = Normalize(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "!", "")
	return name
}

// Contains 는 동작을 수행한다.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// StripLeadingHeader 는 텍스트 앞부분의 헤더 문자열을 제거한다.
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
