package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	guessRateLimitTTL = 30 * time.Second // 30초 제한
)

// GuessRateLimiter: 정답 시도에 대한 개인별 Rate Limit를 관리한다.
type GuessRateLimiter struct {
	client valkey.Client
	prefix string
}

// NewGuessRateLimiter: 새로운 GuessRateLimiter를 생성한다.
func NewGuessRateLimiter(client valkey.Client, prefix string) *GuessRateLimiter {
	return &GuessRateLimiter{
		client: client,
		prefix: prefix,
	}
}

// guessRateLimitKey: 사용자별 Rate Limit 키를 생성한다.
func (r *GuessRateLimiter) guessRateLimitKey(chatID, userID string) string {
	return fmt.Sprintf("%s:guess_limit:%s:%s", r.prefix, chatID, userID)
}

// CheckAndSet: 정답 시도가 허용되는지 확인하고, 허용되면 Rate Limit를 설정한다.
// 반환: (허용 여부, 남은 시간(초), 에러)
func (r *GuessRateLimiter) CheckAndSet(ctx context.Context, chatID, userID string) (bool, int64, error) {
	key := r.guessRateLimitKey(chatID, userID)

	// TTL 확인
	ttlResp := r.client.Do(ctx, r.client.B().Ttl().Key(key).Build())
	ttl, err := ttlResp.AsInt64()
	if err == nil && ttl > 0 {
		// Rate limit 적용 중
		return false, ttl, nil
	}

	// Rate limit 설정 (SET NX EX)
	setResp := r.client.Do(ctx, r.client.B().Set().Key(key).Value("1").Nx().Ex(guessRateLimitTTL).Build())
	result, err := setResp.ToString()
	if err != nil {
		// NX 실패 시 (이미 존재)
		ttlResp = r.client.Do(ctx, r.client.B().Ttl().Key(key).Build())
		ttl, _ = ttlResp.AsInt64()
		if ttl > 0 {
			return false, ttl, nil
		}
		// 경쟁 조건 - 재시도하지 않고 허용
		return true, 0, nil
	}

	if result == "OK" {
		return true, 0, nil
	}

	return false, 0, nil
}

// GetLimitSeconds: Rate Limit 제한 시간(초)을 반환한다.
func (r *GuessRateLimiter) GetLimitSeconds() int64 {
	return int64(guessRateLimitTTL.Seconds())
}

// GetRemainingTime: 남은 Rate Limit 시간을 확인한다.
func (r *GuessRateLimiter) GetRemainingTime(ctx context.Context, chatID, userID string) (int64, error) {
	key := r.guessRateLimitKey(chatID, userID)
	ttlResp := r.client.Do(ctx, r.client.B().Ttl().Key(key).Build())
	ttl, err := ttlResp.AsInt64()
	if err != nil {
		return 0, nil
	}
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}
