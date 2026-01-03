package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// IntFromEnv: 환경 변수에서 정수 값을 읽어옵니다.
func IntFromEnv(key string, defaultValue int) (int, error) {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("invalid int env %s=%q: %w", key, rawValue, err)
	}

	return value, nil
}

// Int64FromEnv: 환경 변수에서 64비트 정수 값을 읽어옵니다.
func Int64FromEnv(key string, defaultValue int64) (int64, error) {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64 env %s=%q: %w", key, rawValue, err)
	}

	return value, nil
}

// Float64FromEnv: 환경 변수에서 64비트 실수 값을 읽어옵니다.
func Float64FromEnv(key string, defaultValue float64) (float64, error) {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float64 env %s=%q: %w", key, rawValue, err)
	}

	return value, nil
}

// DurationSecondsFromEnv: 환경 변수에서 초 단위 시간을 읽어 Duration으로 변환합니다.
func DurationSecondsFromEnv(key string, defaultSeconds int64) (time.Duration, error) {
	valueSeconds, err := Int64FromEnv(key, defaultSeconds)
	if err != nil {
		return 0, err
	}
	if valueSeconds < 0 {
		return 0, fmt.Errorf("invalid duration seconds env %s=%d", key, valueSeconds)
	}
	return time.Duration(valueSeconds) * time.Second, nil
}

// DurationMillisFromEnv: 환경 변수에서 밀리초 단위 시간을 읽어 Duration으로 변환합니다.
func DurationMillisFromEnv(key string, defaultMillis int64) (time.Duration, error) {
	valueMillis, err := Int64FromEnv(key, defaultMillis)
	if err != nil {
		return 0, err
	}
	if valueMillis < 0 {
		return 0, fmt.Errorf("invalid duration millis env %s=%d", key, valueMillis)
	}
	return time.Duration(valueMillis) * time.Millisecond, nil
}

// BoolFromEnv: 환경 변수에서 불리언 값을 읽어옵니다. (true/1/yes/y, false/0/no/n)
func BoolFromEnv(key string, defaultValue bool) (bool, error) {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return defaultValue, nil
	}

	switch strings.ToLower(rawValue) {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool env %s=%q", key, rawValue)
	}
}

// StringFromEnv: 환경 변수에서 문자열 값을 읽어옵니다.
func StringFromEnv(key string, defaultValue string) string {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return defaultValue
	}

	return rawValue
}

// StringListFromEnv: 환경 변수에서 구분자(공백, 콤마 등)로 분리된 문자열 목록을 읽어옵니다.
func StringListFromEnv(key string, defaultValue []string) []string {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return defaultValue
	}

	parts := strings.FieldsFunc(rawValue, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})

	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}

	if len(items) == 0 {
		return defaultValue
	}

	return items
}

// StringFromEnvFirstNonEmpty: 여러 환경 변수 키 중 첫 번째로 값이 존재하는 것을 반환합니다.
func StringFromEnvFirstNonEmpty(keys []string, defaultValue string) string {
	for _, key := range keys {
		rawValue, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		rawValue = strings.TrimSpace(rawValue)
		if rawValue == "" {
			continue
		}

		return rawValue
	}
	return defaultValue
}

// IntFromEnvFirstNonEmpty: 여러 환경 변수 키 중 첫 번째로 값이 존재하는 정수를 반환합니다.
func IntFromEnvFirstNonEmpty(keys []string, defaultValue int) (int, error) {
	for _, key := range keys {
		rawValue, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		rawValue = strings.TrimSpace(rawValue)
		if rawValue == "" {
			continue
		}

		value, err := strconv.Atoi(rawValue)
		if err != nil {
			return 0, fmt.Errorf("invalid int env %s=%q: %w", key, rawValue, err)
		}

		return value, nil
	}
	return defaultValue, nil
}

// Int64FromEnvFirstNonEmpty: 여러 환경 변수 키 중 첫 번째로 값이 존재하는 64비트 정수를 반환합니다.
func Int64FromEnvFirstNonEmpty(keys []string, defaultValue int64) (int64, error) {
	for _, key := range keys {
		rawValue, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		rawValue = strings.TrimSpace(rawValue)
		if rawValue == "" {
			continue
		}

		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid int64 env %s=%q: %w", key, rawValue, err)
		}

		return value, nil
	}
	return defaultValue, nil
}

// BoolFromEnvFirstNonEmpty: 여러 환경 변수 키 중 첫 번째로 값이 존재하는 불리언을 반환합니다.
func BoolFromEnvFirstNonEmpty(keys []string, defaultValue bool) (bool, error) {
	for _, key := range keys {
		rawValue, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		rawValue = strings.TrimSpace(rawValue)
		if rawValue == "" {
			continue
		}

		switch strings.ToLower(rawValue) {
		case "true", "1", "yes", "y":
			return true, nil
		case "false", "0", "no", "n":
			return false, nil
		default:
			return false, fmt.Errorf("invalid bool env %s=%q", key, rawValue)
		}
	}
	return defaultValue, nil
}

// StringListFromEnvFirstNonEmpty: 여러 환경 변수 키 중 첫 번째로 값이 존재하는 문자열 목록을 반환합니다.
func StringListFromEnvFirstNonEmpty(keys []string, defaultValue []string) []string {
	for _, key := range keys {
		rawValue, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		rawValue = strings.TrimSpace(rawValue)
		if rawValue == "" {
			continue
		}

		parts := strings.FieldsFunc(rawValue, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})

		items := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			items = append(items, part)
		}

		if len(items) == 0 {
			continue
		}

		return items
	}

	return defaultValue
}
