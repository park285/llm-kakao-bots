package parser

import (
	"regexp"
	"strings"
)

// BaseParser: 명령어 파서의 공통 기반 구조체입니다.
// 도메인별 파서가 이를 임베드하여 확장합니다.
type BaseParser struct {
	Prefix        string
	EscapedPrefix string
}

// NewBaseParser: 주어진 prefix로 기본 파서를 생성합니다.
func NewBaseParser(prefix string, defaultPrefix string) BaseParser {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = defaultPrefix
	}
	return BaseParser{
		Prefix:        p,
		EscapedPrefix: regexp.QuoteMeta(p),
	}
}

// HasPrefix: 텍스트가 파서의 prefix로 시작하는지 확인합니다.
func (b *BaseParser) HasPrefix(text string) bool {
	return strings.HasPrefix(text, b.Prefix)
}

// TrimMessage: 메시지를 trim하고 prefix 확인 결과를 반환합니다.
// 빈 문자열이거나 prefix가 없으면 빈 문자열을 반환합니다.
func (b *BaseParser) TrimMessage(message string) string {
	text := strings.TrimSpace(message)
	if text == "" {
		return ""
	}
	if !b.HasPrefix(text) {
		return ""
	}
	return text
}

// BuildPattern: escapedPrefix를 포함한 정규식 패턴을 생성합니다.
func (b *BaseParser) BuildPattern(pattern string) *regexp.Regexp {
	return regexp.MustCompile("^" + b.EscapedPrefix + pattern)
}

// BuildPatternCaseInsensitive: 대소문자 무시 정규식 패턴을 생성합니다.
func (b *BaseParser) BuildPatternCaseInsensitive(pattern string) *regexp.Regexp {
	return regexp.MustCompile("(?i)^" + b.EscapedPrefix + pattern)
}

// MatchSimple: 정규식 매칭 결과가 있으면 true를 반환합니다.
func MatchSimple(re *regexp.Regexp, text string) bool {
	return re.MatchString(text)
}

// ExtractFirstGroup: 첫 번째 캐처 그룹을 추출합니다.
// 매칭되지 않거나 그룹이 없으면 빈 문자열을 반환합니다.
func ExtractFirstGroup(re *regexp.Regexp, text string) string {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// SplitByComma: 쉼표로 구분된 문자열을 분리하여 트림된 슬라이스로 반환합니다.
func SplitByComma(body string) []string {
	parts := strings.Split(body, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
