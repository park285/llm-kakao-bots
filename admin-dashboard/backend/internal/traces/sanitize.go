// Package traces: 트레이스 데이터 Sanitization
package traces

import (
	"net/url"
	"strings"
)

// ===== Tag Sanitization =====

// SanitizeTags: 태그에서 민감 정보 필터링
func SanitizeTags(tags []Tag) []Tag {
	result := make([]Tag, len(tags))
	for i, tag := range tags {
		result[i] = Tag{
			Key:   tag.Key,
			Type:  tag.Type,
			Value: sanitizeTagValue(tag.Key, tag.Value),
		}
	}
	return result
}

// sensitiveTagKeys: 완전 마스킹 대상 태그 키
var sensitiveTagKeys = map[string]bool{
	"db.statement":                      true,
	"db.query":                          true,
	"http.request.header.authorization": true,
	"http.request.header.cookie":        true,
	"http.request.header.x-api-key":     true,
}

// urlTagKeys: URL 쿼리 파라미터 마스킹 대상 태그 키
var urlTagKeys = map[string]bool{
	"url.full":    true,
	"http.url":    true,
	"http.target": true,
}

// errorTagKeys: 에러 메시지 sanitization 대상 태그 키
var errorTagKeys = map[string]bool{
	"error.message":        true,
	"exception.message":    true,
	"error.stack":          true,
	"exception.stacktrace": true,
}

func sanitizeTagValue(key string, value any) any {
	strVal, ok := value.(string)
	if !ok {
		return value
	}

	// 완전 마스킹 대상
	if sensitiveTagKeys[key] {
		return "[REDACTED]"
	}

	// URL 쿼리 파라미터 마스킹
	if urlTagKeys[key] {
		return maskQueryParams(strVal)
	}

	// 에러 메시지 sanitization
	if errorTagKeys[key] {
		return SanitizeErrorMessage(strVal)
	}

	return value
}

// ===== Log Sanitization =====

// SanitizeLogs: 로그에서 민감 정보 필터링
func SanitizeLogs(logs []Log) []Log {
	if len(logs) == 0 {
		return logs
	}

	result := make([]Log, len(logs))
	for i, log := range logs {
		fields := make([]LogField, len(log.Fields))
		for j, field := range log.Fields {
			fields[j] = LogField{
				Key:   field.Key,
				Value: sanitizeLogValue(field.Key, field.Value),
			}
		}
		result[i] = Log{
			Timestamp: log.Timestamp,
			Fields:    fields,
		}
	}
	return result
}

func sanitizeLogValue(key string, value any) any {
	strVal, ok := value.(string)
	if !ok {
		return value
	}

	// 에러 관련 필드는 sanitize
	lowerKey := strings.ToLower(key)
	if strings.Contains(lowerKey, "error") ||
		strings.Contains(lowerKey, "exception") ||
		strings.Contains(lowerKey, "message") ||
		strings.Contains(lowerKey, "stack") {
		return SanitizeErrorMessage(strVal)
	}

	return value
}

// ===== URL Query Parameter Masking =====

// maskQueryParams: URL에서 쿼리 파라미터 값 마스킹
func maskQueryParams(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	if parsed.RawQuery == "" {
		return rawURL
	}

	// 쿼리 파라미터 마스킹
	query := parsed.Query()
	for key := range query {
		query.Set(key, "***")
	}
	parsed.RawQuery = query.Encode()

	return parsed.String()
}

// ===== Error Message Sanitization =====

// sensitiveSchemes: 마스킹 대상 연결 문자열 스키마
var sensitiveSchemes = []string{
	"postgres", "postgresql", "mysql", "redis", "valkey", "mongodb", "amqp",
}

// sensitivePatterns: 마스킹 대상 키=값 패턴
var sensitivePatterns = []string{
	"password=", "passwd=", "secret=", "api_key=", "apikey=",
	"token=", "access_token=", "refresh_token=", "auth=",
}

// SanitizeErrorMessage: 에러 메시지에서 민감 정보 제거
func SanitizeErrorMessage(msg string) string {
	result := msg

	// 연결 문자열 마스킹: postgres://user:pass@host/db → postgres://[REDACTED]
	result = maskConnectionStrings(result)

	// 키=값 형태 마스킹: password=secret123 → password=[REDACTED]
	result = maskKeyValuePatterns(result)

	return result
}

func maskConnectionStrings(msg string) string {
	if !strings.Contains(msg, "://") {
		return msg
	}

	result := msg
	for _, scheme := range sensitiveSchemes {
		prefix := scheme + "://"
		for {
			idx := strings.Index(result, prefix)
			if idx == -1 {
				break
			}

			// 연결 문자열 끝 찾기
			endIdx := idx + len(prefix)
			for endIdx < len(result) {
				ch := result[endIdx]
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '"' || ch == '\'' || ch == ')' {
					break
				}
				endIdx++
			}

			// 마스킹 적용
			result = result[:idx] + scheme + "://[REDACTED]" + result[endIdx:]
		}
	}
	return result
}

func maskKeyValuePatterns(msg string) string {
	result := msg

	for _, pattern := range sensitivePatterns {
		lowerResult := strings.ToLower(result)
		for {
			idx := strings.Index(lowerResult, pattern)
			if idx == -1 {
				break
			}

			// 값 범위 찾기
			valueStart := idx + len(pattern)
			valueEnd := valueStart
			inQuote := false
			quoteChar := byte(0)

			if valueStart < len(result) && (result[valueStart] == '"' || result[valueStart] == '\'') {
				inQuote = true
				quoteChar = result[valueStart]
				valueStart++
				valueEnd = valueStart
			}

			for valueEnd < len(result) {
				if inQuote {
					if result[valueEnd] == quoteChar {
						break
					}
				} else {
					ch := result[valueEnd]
					if ch == ' ' || ch == '\t' || ch == '\n' || ch == '&' || ch == ';' || ch == ')' || ch == ',' {
						break
					}
				}
				valueEnd++
			}

			// 마스킹 적용
			masked := result[:idx+len(pattern)] + "[REDACTED]"
			if inQuote && valueEnd < len(result) {
				masked += string(quoteChar)
				valueEnd++
			}
			result = masked + result[valueEnd:]
			lowerResult = strings.ToLower(result)
		}
	}

	return result
}
