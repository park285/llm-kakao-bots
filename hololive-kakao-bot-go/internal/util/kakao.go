package util

import "strings"

// 카카오 메시지 관련 상수 목록.
const (
	// KakaoSeeMorePadding 는 상수다.
	KakaoSeeMorePadding = 500
	KakaoZeroWidthSpace = "\u200b"
)

// ApplyKakaoSeeMorePadding 는 동작을 수행한다.
func ApplyKakaoSeeMorePadding(text, instruction string) string {
	if TrimSpace(text) == "" {
		return text
	}

	message := TrimSpace(instruction)

	var builder strings.Builder
	builder.Grow(len(text) + KakaoSeeMorePadding + len(message) + 2)

	if message != "" {
		builder.WriteString(message)
	}
	builder.WriteString(strings.Repeat(KakaoZeroWidthSpace, KakaoSeeMorePadding))
	if !strings.HasPrefix(text, "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString(text)

	return builder.String()
}
