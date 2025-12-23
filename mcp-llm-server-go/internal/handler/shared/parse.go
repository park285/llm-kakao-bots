package shared

import (
	"fmt"
	"strings"

	"github.com/goccy/go-json"
)

type noEscapeHTMLJSON struct{}

func (noEscapeHTMLJSON) Marshal(v any) ([]byte, error) {
	var builder strings.Builder
	enc := json.NewEncoder(&builder)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}
	return []byte(strings.TrimRight(builder.String(), "\n")), nil
}

var jsonNoEscapeHTML = noEscapeHTMLJSON{}

// ParseStringSlice 는 map에서 문자열 슬라이스 필드를 파싱한다.
func ParseStringSlice(payload map[string]any, field string) ([]string, error) {
	raw, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("missing field %s", field)
	}
	switch value := raw.(type) {
	case []string:
		return value, nil
	case []any:
		items := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("invalid element in %s", field)
			}
			items = append(items, text)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("invalid field type for %s", field)
	}
}

// ParseStringField 는 map에서 문자열 필드를 파싱한다.
func ParseStringField(payload map[string]any, field string) (string, error) {
	raw, ok := payload[field]
	if !ok {
		return "", fmt.Errorf("missing field %s", field)
	}
	text, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("invalid field type for %s", field)
	}
	return text, nil
}

// SerializeDetails 는 details map을 JSON 문자열로 직렬화한다.
func SerializeDetails(details map[string]any) (string, error) {
	if len(details) == 0 {
		return "", nil
	}
	data, err := jsonNoEscapeHTML.Marshal(details)
	if err != nil {
		return "", fmt.Errorf("encode details: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// TrimRunes 는 문자열을 최대 maxRunes 개의 룬으로 자른다.
func TrimRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
