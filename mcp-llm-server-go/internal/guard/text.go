package guard

import (
	"strings"
	"unicode"

	"github.com/forPelevin/gomoji"
	"github.com/mtibben/confusables"
	"github.com/ymw0407/jamo/pkg/jamo"
	"golang.org/x/text/unicode/norm"
)

// jamoTable: 한글 자모 범위를 통합한 테이블
// unicode.Is()를 사용하면 이진 탐색을 수행하여 매우 빠릅니다.
var jamoTable = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x1100, Hi: 0x11FF, Stride: 1}, // Hangul Jamo
		{Lo: 0x3130, Hi: 0x318F, Stride: 1}, // Hangul Compatibility Jamo
		{Lo: 0xA960, Hi: 0xA97F, Stride: 1}, // Hangul Jamo Extended-A
		{Lo: 0xD7B0, Hi: 0xD7FF, Stride: 1}, // Hangul Jamo Extended-B
	},
}

// isASCIIOnly: 문자열이 ASCII만 포함하는지 확인 (Zero Allocation)
func isASCIIOnly(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func isPureBase64(text string) bool {
	// 최소 길이 조건 체크
	if len(text) < 20 {
		return false
	}

	// [Fast Path] ASCII 검사
	if isASCIIOnly(text) {
		return validateBase64Loop(text)
	}

	// ASCII가 아니라면(공격 시도 가능성), 정규화 후 검증
	normalized := normalizeTextForBase64(text)
	return validateBase64Loop(normalized)
}

// validateBase64Loop: Regex 대체용 검증 루프
// 공백 스킵, 문자셋 확인, 패딩 규칙 검사, 길이 검사를 한 번의 순회(One-pass)로 처리합니다.
func validateBase64Loop(s string) bool {
	validLen := 0
	paddingLen := 0
	hasPadding := false

	for _, r := range s {
		// 공백은 무시 (stripAllWhitespace 효과)
		if unicode.IsSpace(r) {
			continue
		}

		// 패딩(=) 처리
		if r == '=' {
			hasPadding = true
			paddingLen++
			continue
		}

		// 패딩이 나온 후에 일반 문자가 오면 안 됨 (Regex의 "={0,2}$" 규칙 준수)
		if hasPadding {
			return false
		}

		// Base64 문자셋 확인 (A-Za-z0-9+/ 와 -_ 허용)
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '+' || r == '/' || r == '-' || r == '_' {
			validLen++
			continue
		}

		// 허용되지 않은 문자
		return false
	}

	// Base64 유효성 검사:
	// 1. "{20,}" 조건: 유효 문자 길이가 20 이상
	// 2. "={0,2}" 조건: 패딩은 최대 2개
	// 3. 4의 배수 조건: Base64 인코딩 원리상 데이터 길이는 4의 배수
	totalLen := validLen + paddingLen
	return validLen >= 20 && paddingLen <= 2 && totalLen%4 == 0
}

// normalizeTextForBase64: Base64 검사용 정규화 (Homoglyph 변환)
func normalizeTextForBase64(text string) string {
	skeleton := confusables.Skeleton(text)
	return norm.NFKC.String(skeleton)
}

// hangulTable: 한글 범위 (Jamo 포함하지 않음 - 완성형만)
var hangulTable = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0xAC00, Hi: 0xD7A3, Stride: 1}, // Hangul Syllables (가-힣)
	},
}

func normalizeText(text string) string {
	// [Fast Path] ASCII만 포함된 경우 Skeleton 변환 불필요
	if isASCIIOnly(text) {
		return stripControlChars(text)
	}

	// NFD 입력 우회 방지: 먼저 NFC로 정규화
	// 예: "한\u1100\u1173\u11AF" (NFD) → "한글" (NFC)
	nfcText := norm.NFC.String(text)

	// Non-ASCII: 한글 보존하면서 Homoglyph 정규화 수행
	normalized := normalizeWithKoreanPreserved(nfcText)
	return stripControlChars(normalized)
}

// normalizeWithKoreanPreserved: 한글 문자는 보존하면서 나머지만 skeleton 변환
func normalizeWithKoreanPreserved(text string) string {
	var result strings.Builder
	var nonKoreanBuffer strings.Builder
	result.Grow(len(text))

	flushNonKorean := func() {
		if nonKoreanBuffer.Len() == 0 {
			return
		}
		// 비한글 텍스트에만 skeleton + NFKC 적용
		skeleton := confusables.Skeleton(nonKoreanBuffer.String())
		result.WriteString(norm.NFKC.String(skeleton))
		nonKoreanBuffer.Reset()
	}

	for _, r := range text {
		if unicode.Is(hangulTable, r) || unicode.Is(jamoTable, r) {
			// 한글(완성형 또는 자모)은 그대로 보존
			flushNonKorean()
			result.WriteRune(r)
		} else {
			// 비한글은 버퍼에 누적
			nonKoreanBuffer.WriteRune(r)
		}
	}
	flushNonKorean() // 마지막 버퍼 처리

	return result.String()
}

// stripControlChars: 불필요한 할당 방지
func stripControlChars(text string) string {
	// 1. 제어 문자가 없는지 먼저 스캔
	hasControl := false
	for _, r := range text {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) {
			hasControl = true
			break
		}
	}
	if !hasControl {
		return text
	}

	// 2. 제어 문자가 있을 때만 빌더 사용
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// containsEmoji: 입력 문자열에 이모지가 포함되어 있는지 확인한다.
// gomoji 라이브러리를 사용하여 최신 유니코드 이모지 표준을 자동 지원한다.
func containsEmoji(text string) bool {
	return gomoji.ContainsEmoji(text)
}

// isJamoOnly: 입력이 한글 자모로만 구성되어 있는지 확인한다.
// 자모 외에 공백, 숫자, 구두점은 허용된다. (예: "ㅈㅓㅇㄷㅏㅂ 123!" → true)
// 완성형 한글(가-힣)이 포함되면 false를 반환한다.
func isJamoOnly(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	hasJamo := false
	for _, r := range trimmed {
		if unicode.Is(jamoTable, r) {
			hasJamo = true
			continue
		}
		if unicode.IsSpace(r) || unicode.IsDigit(r) || unicode.IsPunct(r) {
			continue
		}
		return false
	}
	return hasJamo
}

// composeJamoSequences: 혼합 문자열에서 연속 자모 시퀀스를 완성형으로 조합한다.
// 예: "시스템 ㅍㅡㄹㅗㅁㅍㅡㅌㅡ" → "시스템 프롬프트"
// 조합에 실패한 자모는 원본 그대로 유지된다.
func composeJamoSequences(text string) string {
	var result strings.Builder
	var jamoBuffer strings.Builder
	result.Grow(len(text))

	flushJamo := func() {
		if jamoBuffer.Len() == 0 {
			return
		}
		jamoStr := jamoBuffer.String()
		composed, err := jamo.ComposeHangeul(jamoStr)
		if err == nil && len(composed) > 0 {
			// 첫 번째 조합 결과 사용 (가장 일반적인 해석)
			result.WriteString(composed[0])
		} else {
			// 조합 실패 시 원본 자모 유지
			result.WriteString(jamoStr)
		}
		jamoBuffer.Reset()
	}

	for _, r := range text {
		if unicode.Is(jamoTable, r) {
			jamoBuffer.WriteRune(r)
		} else {
			flushJamo()
			result.WriteRune(r)
		}
	}
	flushJamo() // 마지막 버퍼 처리

	return result.String()
}
