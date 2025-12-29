package toon

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Encode: 값을 Toon 포맷 문자열로 변환합니다.
func Encode(value any) string {
	return encode(value, 0)
}

// EncodeSecret 는 스무고개 비밀 정보를 Toon 포맷으로 만든다.
func EncodeSecret(target string, category string, details map[string]any) string {
	lines := []string{
		"target: " + Encode(target),
		"category: " + Encode(category),
	}
	if len(details) > 0 {
		lines = append(lines, "details:")
		encodedDetails := encode(details, 2)
		lines = append(lines, strings.Split(encodedDetails, "\n")...)
	}
	return strings.Join(lines, "\n")
}

// EncodePuzzle 는 퍼즐 정보를 Toon 포맷으로 만든다.
func EncodePuzzle(scenario string, solution string, category string, difficulty *int) string {
	lines := []string{
		"scenario: " + Encode(scenario),
		"solution: " + Encode(solution),
	}
	if category != "" {
		lines = append(lines, "category: "+Encode(category))
	}
	if difficulty != nil {
		lines = append(lines, "difficulty: "+Encode(*difficulty))
	}
	return strings.Join(lines, "\n")
}

func encode(value any, indent int) string {
	if primitive, ok := formatPrimitive(value); ok {
		return primitive
	}

	if slice, ok := toSlice(value); ok {
		return encodeSlice(slice, indent)
	}

	if mapping, ok := toStringMap(value); ok {
		return encodeMap(mapping, indent)
	}

	return fmt.Sprint(value)
}

func encodeSlice(slice []any, indent int) string {
	if len(slice) == 0 {
		return "[]"
	}
	if allPrimitive(slice) {
		return encodePrimitiveSlice(slice)
	}
	if maps, ok := toStringMapSlice(slice); ok {
		if encoded, ok := encodeSliceTable(maps, indent); ok {
			return encoded
		}
	}
	return encodeListSlice(slice, indent)
}

func encodePrimitiveSlice(slice []any) string {
	items := make([]string, 0, len(slice))
	for _, item := range slice {
		items = append(items, encode(item, 0))
	}
	return fmt.Sprintf("[%d]: %s", len(slice), strings.Join(items, ","))
}

func encodeSliceTable(maps []map[string]any, indent int) (string, bool) {
	keys, same := uniformKeys(maps)
	if !same {
		return "", false
	}
	header := fmt.Sprintf("[%d]{%s}:", len(maps), strings.Join(keys, ","))
	rows := make([]string, 0, len(maps))
	for _, item := range maps {
		rowValues := make([]string, 0, len(keys))
		for _, key := range keys {
			rowValues = append(rowValues, encode(item[key], 0))
		}
		rows = append(rows, strings.Join(rowValues, ","))
	}
	prefix := strings.Repeat(" ", indent)
	lines := []string{header}
	for _, row := range rows {
		lines = append(lines, prefix+" "+row)
	}
	return strings.Join(lines, "\n"), true
}

func encodeListSlice(slice []any, indent int) string {
	prefix := strings.Repeat(" ", indent)
	lines := []string{fmt.Sprintf("[%d]:", len(slice))}
	for _, item := range slice {
		lines = append(lines, fmt.Sprintf("%s - %s", prefix, encode(item, indent+2)))
	}
	return strings.Join(lines, "\n")
}

func encodeMap(mapping map[string]any, indent int) string {
	if len(mapping) == 0 {
		return "{}"
	}
	prefix := strings.Repeat(" ", indent)
	keys := sortedKeys(mapping)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		entry := mapping[key]
		if nested, ok := toStringMap(entry); ok && len(nested) > 0 {
			lines = append(lines, fmt.Sprintf("%s%s:", prefix, key))
			lines = append(lines, encodeNestedMapLines(nested, indent+2, prefix)...)
			continue
		}
		if slice, ok := toSlice(entry); ok && len(slice) > 0 {
			if maps, ok := toStringMapSlice(slice); ok {
				if tableLines, ok := encodeMapSliceTable(key, maps, prefix); ok {
					lines = append(lines, tableLines...)
					continue
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, key, encode(entry, indent)))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, key, encode(entry, indent)))
	}
	return strings.Join(lines, "\n")
}

func encodeNestedMapLines(nested map[string]any, indent int, prefix string) []string {
	nestedKeys := sortedKeys(nested)
	lines := make([]string, 0, len(nestedKeys))
	for _, subKey := range nestedKeys {
		encodedValue := encode(nested[subKey], indent)
		lines = append(lines, fmt.Sprintf("%s  %s: %s", prefix, subKey, encodedValue))
	}
	return lines
}

func encodeMapSliceTable(key string, maps []map[string]any, prefix string) ([]string, bool) {
	keys, same := uniformKeys(maps)
	if !same {
		return nil, false
	}
	header := fmt.Sprintf("%s%s[%d]{%s}:", prefix, key, len(maps), strings.Join(keys, ","))
	lines := []string{header}
	for _, item := range maps {
		rowValues := make([]string, 0, len(keys))
		for _, colKey := range keys {
			rowValues = append(rowValues, encode(item[colKey], 0))
		}
		lines = append(lines, fmt.Sprintf("%s  %s", prefix, strings.Join(rowValues, ",")))
	}
	return lines, true
}

func formatPrimitive(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "null", true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	case string:
		return encodeString(v), true
	case int:
		return strconv.Itoa(v), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return "", false
	}
}

func encodeString(value string) string {
	if strings.ContainsAny(value, ",:\n\"'") {
		escaped := strings.ReplaceAll(value, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	return value
}

func allPrimitive(values []any) bool {
	for _, value := range values {
		if _, ok := formatPrimitive(value); !ok {
			return false
		}
	}
	return true
}

func toSlice(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		return v, true
	case []string:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out, true
	case []int:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out, true
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

func toStringMap(value any) (map[string]any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return v, true
	case map[string]string:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[key] = val
		}
		return out, true
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		out[key.String()] = rv.MapIndex(key).Interface()
	}
	return out, true
}

func toStringMapSlice(values []any) ([]map[string]any, bool) {
	maps := make([]map[string]any, 0, len(values))
	for _, item := range values {
		mapping, ok := toStringMap(item)
		if !ok {
			return nil, false
		}
		maps = append(maps, mapping)
	}
	return maps, true
}

func sortedKeys(mapping map[string]any) []string {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func uniformKeys(values []map[string]any) ([]string, bool) {
	if len(values) == 0 {
		return nil, false
	}
	keys := sortedKeys(values[0])
	for _, item := range values[1:] {
		if len(item) != len(keys) {
			return nil, false
		}
		for _, key := range keys {
			if _, ok := item[key]; !ok {
				return nil, false
			}
		}
	}
	return keys, true
}
