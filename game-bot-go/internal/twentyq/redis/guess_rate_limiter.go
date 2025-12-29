package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"

	luautil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/lua"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
)

const (
	guessRateLimitTTL = 30 * time.Second // 30초 제한
)

// GuessRateLimiter: 정답 시도에 대한 개인별 Rate Limit를 관리합니다.
type GuessRateLimiter struct {
	client   valkey.Client
	prefix   string
	registry *luautil.Registry
}

// NewGuessRateLimiter: 새로운 GuessRateLimiter를 생성합니다.
func NewGuessRateLimiter(client valkey.Client, prefix string) *GuessRateLimiter {
	registry := luautil.NewRegistry([]luautil.Script{
		{Name: luautil.ScriptGuessRateLimit, Source: assets.GuessRateLimitLua},
	})
	_ = registry.Preload(context.Background(), client)
	return &GuessRateLimiter{
		client:   client,
		prefix:   prefix,
		registry: registry,
	}
}

// guessRateLimitKey: 사용자별 Rate Limit 키를 생성합니다.
func (r *GuessRateLimiter) guessRateLimitKey(chatID, userID string) string {
	return fmt.Sprintf("%s:guess_limit:%s:%s", r.prefix, chatID, userID)
}

// CheckAndSet: 정답 시도가 허용되는지 확인하고, 허용되면 Rate Limit를 설정합니다.
// 반환: (허용 여부, 남은 시간(초), 에러)
func (r *GuessRateLimiter) CheckAndSet(ctx context.Context, chatID, userID string) (bool, int64, error) {
	key := r.guessRateLimitKey(chatID, userID)
	ttlSec := int64(guessRateLimitTTL.Seconds())
	ttlArg := strconv.FormatInt(ttlSec, 10)

	// Lua 스크립트 실행 (1 RTT)
	// 반환값: {allowed(1|0), remaining_ms}
	resp, err := r.registry.Exec(ctx, r.client, luautil.ScriptGuessRateLimit, []string{key}, []string{ttlArg})
	if err != nil {
		return false, 0, wrapRedisError("guess_rate_limit_exec", err)
	}

	allowedValue, remainingMs, err := valkeyx.ParseLuaInt64Pair(resp)
	if err != nil {
		return false, 0, wrapRedisError("guess_rate_limit_parse", err)
	}

	allowed, remainingSeconds, err := parseRateLimitResult(allowedValue, remainingMs)
	if err != nil {
		return false, 0, wrapRedisError("guess_rate_limit_result", err)
	}
	if allowed {
		return true, 0, nil
	}
	return false, remainingSeconds, nil
}

// GetLimitSeconds: Rate Limit 제한 시간(초)을 반환합니다.
func (r *GuessRateLimiter) GetLimitSeconds() int64 {
	return int64(guessRateLimitTTL.Seconds())
}

// GetRemainingTime: 남은 Rate Limit 시간을 확인합니다.
func (r *GuessRateLimiter) GetRemainingTime(ctx context.Context, chatID, userID string) (int64, error) {
	key := r.guessRateLimitKey(chatID, userID)
	ttlResp := r.client.Do(ctx, r.client.B().Pttl().Key(key).Build())
	ttlMillis, err := ttlResp.AsInt64()
	if err != nil {
		return 0, nil
	}
	if ttlMillis <= 0 {
		return 0, nil
	}

	remainingSeconds := ttlMillis / 1000
	if ttlMillis%1000 != 0 {
		remainingSeconds++
	}
	return remainingSeconds, nil
}

func parseRateLimitResult(allowedValue int64, remainingMs int64) (bool, int64, error) {
	if allowedValue != 0 && allowedValue != 1 {
		return false, 0, fmt.Errorf("unexpected allowed flag: %d", allowedValue)
	}

	if remainingMs < 0 {
		remainingMs = 0
	}

	remainingSeconds := remainingMs / 1000
	if remainingMs%1000 != 0 {
		remainingSeconds++
	}

	return allowedValue == 1, remainingSeconds, nil
}
