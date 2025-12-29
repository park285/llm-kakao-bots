package prompt

import (
	"fmt"
	"strings"
)

// FormatTemplate: 템플릿 문자열을 값으로 치환합니다.
func FormatTemplate(template string, values map[string]string) (string, error) {
	var builder strings.Builder
	builder.Grow(len(template))

	for i := 0; i < len(template); {
		switch template[i] {
		case '{':
			if i+1 < len(template) && template[i+1] == '{' {
				builder.WriteByte('{')
				i += 2
				continue
			}
			end := strings.IndexByte(template[i+1:], '}')
			if end < 0 {
				return "", fmt.Errorf("invalid template: missing '}'")
			}
			key := template[i+1 : i+1+end]
			value, ok := values[key]
			if !ok {
				return "", fmt.Errorf("missing template value for %q", key)
			}
			builder.WriteString(value)
			i += end + 2
		case '}':
			if i+1 < len(template) && template[i+1] == '}' {
				builder.WriteByte('}')
				i += 2
				continue
			}
			return "", fmt.Errorf("invalid template: unexpected '}'")
		default:
			builder.WriteByte(template[i])
			i++
		}
	}

	return builder.String(), nil
}

var xmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	"'", "&apos;",
)

// EscapeXML: XML 텍스트로 안전하게 이스케이프합니다.
func EscapeXML(value string) string {
	return xmlEscaper.Replace(value)
}

// WrapXML: 값을 XML 태그로 감쌉니다.
func WrapXML(tag string, value string) string {
	return "<" + tag + ">" + EscapeXML(value) + "</" + tag + ">"
}

// ValidateSystemStatic: 시스템 프롬프트의 템플릿 사용 여부를 검사합니다.
func ValidateSystemStatic(name string, system string) error {
	for i := 0; i < len(system); {
		switch system[i] {
		case '{':
			if i+1 < len(system) && system[i+1] == '{' {
				i += 2
				continue
			}
			end := strings.IndexByte(system[i+1:], '}')
			if end < 0 {
				return fmt.Errorf("%s: invalid system prompt template syntax", name)
			}
			key := system[i+1 : i+1+end]
			return fmt.Errorf("%s: system prompt must not contain template variables %q", name, key)
		case '}':
			if i+1 < len(system) && system[i+1] == '}' {
				i += 2
				continue
			}
			return fmt.Errorf("%s: invalid system prompt template syntax", name)
		default:
			i++
		}
	}
	return nil
}
