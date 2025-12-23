package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// IntFromEnv 는 동작을 수행한다.
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

// Int64FromEnv 는 동작을 수행한다.
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

// DurationSecondsFromEnv 는 동작을 수행한다.
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

// DurationMillisFromEnv 는 동작을 수행한다.
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

// BoolFromEnv 는 동작을 수행한다.
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

// StringFromEnv 는 동작을 수행한다.
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

// StringListFromEnv 는 동작을 수행한다.
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

// StringFromEnvFirstNonEmpty 는 동작을 수행한다.
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

// IntFromEnvFirstNonEmpty 는 동작을 수행한다.
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

// Int64FromEnvFirstNonEmpty 는 동작을 수행한다.
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

// BoolFromEnvFirstNonEmpty 는 동작을 수행한다.
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

// StringListFromEnvFirstNonEmpty 는 동작을 수행한다.
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
