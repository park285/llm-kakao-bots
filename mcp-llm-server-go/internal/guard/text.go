package guard

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

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

// isBase64Char: Base64 문자셋 검사 (A-Za-z0-9+/-_)
func isBase64Char(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '+' || c == '/' || c == '-' || c == '_'
}

// containsSuspiciousBase64: 입력값 내에 숨겨진 악성 Base64 페이로드가 있는지 탐지
// 패턴 추출 방식: 입력 전체가 아닌, Base64 의심 패턴만 추출하여 검사
// 의미 기반 필터링: 디코딩된 결과가 '읽을 수 있는 텍스트'일 때만 차단
func containsSuspiciousBase64(input string) bool {
	n := len(input)
	i := 0

	// 수동 스캐너로 Base64 패턴 추출 (Zero-Alloc)
	for i < n {
		// Base64 문자가 아니면 스킵
		if !isBase64Char(input[i]) {
			i++
			continue
		}

		// Base64 시퀀스 시작점 찾음
		start := i
		for i < n && isBase64Char(input[i]) {
			i++
		}

		// 패딩(=) 처리
		paddingCount := 0
		for i < n && input[i] == '=' && paddingCount < 2 {
			i++
			paddingCount++
		}

		seqLen := i - start
		// 최소 20자 이상이어야 의미 있는 Base64
		if seqLen < 20 {
			continue
		}

		// 디코딩 시도
		match := input[start:i]
		decodedBytes, err := tryDecodeBase64(match)
		if err != nil {
			continue
		}

		// 디코딩된 결과가 '읽을 수 있는 텍스트'인지 확인
		if isReadableText(decodedBytes) {
			return true
		}
	}

	return false
}

// tryDecodeBase64: URL-Safe 문자 치환 및 패딩 보정 후 디코딩 (Zero-Alloc 최적화)
func tryDecodeBase64(s string) ([]byte, error) {
	n := len(s)
	if n == 0 {
		return nil, fmt.Errorf("base64 decode: empty input")
	}

	// 패딩 계산: Base64 길이는 4의 배수여야 함
	padNeeded := (4 - n%4) % 4

	// 버퍼 할당 (URL-Safe 치환 + 패딩)
	buf := make([]byte, n+padNeeded)

	// URL-Safe 문자('-', '_')를 표준 문자('+', '/')로 치환 (바이트 단위)
	for i := 0; i < n; i++ {
		switch s[i] {
		case '-':
			buf[i] = '+'
		case '_':
			buf[i] = '/'
		default:
			buf[i] = s[i]
		}
	}

	// 패딩 추가
	for i := 0; i < padNeeded; i++ {
		buf[n+i] = '='
	}

	// 디코딩 (in-place 가능하지만 안전을 위해 별도 버퍼)
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(buf)))
	written, err := base64.StdEncoding.Decode(decoded, buf)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return decoded[:written], nil
}

// isReadableText: 바이트 배열이 사람이 읽을 수 있는 텍스트인지 판별 (Zero-Alloc 최적화)
// UTF-8 유효성 검사 + 출력 가능 문자 비율 검사
func isReadableText(data []byte) bool {
	n := len(data)
	if n == 0 {
		return false
	}

	printableCount := 0
	totalChars := 0
	i := 0

	// UTF-8 유효성 검사와 가독성 검사를 단일 루프로 통합
	for i < n {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			// 유효하지 않은 UTF-8 → 바이너리 데이터
			return false
		}
		i += size
		totalChars++

		// 출력 가능한 문자(한글, 영문, 숫자 등)이거나 공백 문자일 경우 카운트
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printableCount++
		}
	}

	// 가독성 비율 검사 (정수 연산: printableCount * 100 > totalChars * 90)
	// 전체 문자의 90% 이상이 읽을 수 있는 문자라면 '의도된 텍스트'로 판단
	return totalChars > 0 && printableCount*100 > totalChars*90
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

// containsEmoji: 입력 문자열에 이모지가 포함되어 있는지 확인합니다.
// gomoji 라이브러리를 사용하여 최신 유니코드 이모지 표준을 자동 지원합니다.
func containsEmoji(text string) bool {
	return gomoji.ContainsEmoji(text)
}

// isJamoOnly: 입력이 한글 자모로만 구성되어 있는지 확인합니다.
// 자모 외에 공백, 숫자, 구두점은 허용됩니다. (예: "ㅈㅓㅇㄷㅏㅂ 123!" → true)
// 완성형 한글(가-힣)이 포함되면 false를 반환합니다.
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

// composeJamoSequences: 혼합 문자열에서 연속 자모 시퀀스를 완성형으로 조합합니다.
// 예: "시스템 ㅍㅡㄹㅗㅁㅍㅡㅌㅡ" → "시스템 프롬프트"
// 조합에 실패한 자모는 원본 그대로 유지됩니다.
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
