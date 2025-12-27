package valkeyx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valkey-io/valkey-go"
)

// ParseLuaArray 는 Lua 결과를 배열로 파싱하고 길이를 검증한다.
func ParseLuaArray(resp valkey.ValkeyResult, expectedLen int) ([]valkey.ValkeyMessage, error) {
	values, err := resp.ToArray()
	if err != nil {
		return nil, fmt.Errorf("parse lua array failed: %w", err)
	}
	if expectedLen > 0 && len(values) != expectedLen {
		return nil, fmt.Errorf("unexpected lua array len: %d", len(values))
	}
	return values, nil
}

// ParseLuaInt64 는 Lua 결과를 int64로 파싱한다.
func ParseLuaInt64(resp valkey.ValkeyResult) (int64, error) {
	value, err := resp.AsInt64()
	if err != nil {
		return 0, fmt.Errorf("parse lua int64 failed: %w", err)
	}
	return value, nil
}

// ParseLuaInt64Message 는 Lua 배열 메시지를 int64로 파싱한다.
func ParseLuaInt64Message(msg valkey.ValkeyMessage) (int64, error) {
	value, err := msg.AsInt64()
	if err != nil {
		return 0, fmt.Errorf("parse lua int64 failed: %w", err)
	}
	return value, nil
}

// ParseLuaInt64Pair 는 Lua 결과를 [int64, int64]로 파싱한다.
func ParseLuaInt64Pair(resp valkey.ValkeyResult) (int64, int64, error) {
	values, err := ParseLuaArray(resp, 2)
	if err != nil {
		return 0, 0, err
	}
	first, err := ParseLuaInt64Message(values[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := ParseLuaInt64Message(values[1])
	if err != nil {
		return 0, 0, err
	}
	return first, second, nil
}

// ParseLuaString 는 Lua 결과를 문자열로 파싱한다.
func ParseLuaString(resp valkey.ValkeyResult) (string, error) {
	value, err := resp.ToString()
	if err != nil {
		return "", fmt.Errorf("parse lua string failed: %w", err)
	}
	return value, nil
}

// ParseLuaScoreToInt64 는 Lua 점수 값을 int64로 파싱한다.
func ParseLuaScoreToInt64(msg valkey.ValkeyMessage) (int64, error) {
	score, err := msg.ToString()
	if err != nil {
		return 0, fmt.Errorf("parse lua score failed: %w", err)
	}
	score = strings.TrimSpace(score)
	if score == "" {
		return 0, errors.New("empty score")
	}
	f, err := strconv.ParseFloat(score, 64)
	if err != nil {
		return 0, fmt.Errorf("parse lua score failed: %w", err)
	}
	return int64(f), nil
}
