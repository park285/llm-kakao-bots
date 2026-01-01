package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	commoncache "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/cache"
	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

// CachedInjectionGuard: LRU + TTL 캐시를 적용한 Injection Guard 래퍼입니다.
// singleflight를 사용하여 동일 입력에 대한 중복 호출을 방지합니다.
type CachedInjectionGuard struct {
	inner  InjectionGuard
	cache  *commoncache.TTLLRUCache
	logger *slog.Logger
	sf     singleflight.Group
}

// NewCachedInjectionGuard: CachedInjectionGuard 인스턴스를 생성합니다.
func NewCachedInjectionGuard(
	inner InjectionGuard,
	ttl time.Duration,
	maxEntries int,
	logger *slog.Logger,
) *CachedInjectionGuard {
	if logger == nil {
		logger = slog.Default()
	}
	return &CachedInjectionGuard{
		inner:  inner,
		cache:  commoncache.NewTTLLRUCache(maxEntries, ttl),
		logger: logger,
	}
}

// IsMalicious: 캐시를 확인한 후 악성 여부를 검사합니다.
// 캐시 미스 시 inner guard를 호출하고 결과를 캐싱합니다.
func (g *CachedInjectionGuard) IsMalicious(ctx context.Context, input string) (bool, error) {
	if g == nil || g.inner == nil {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalized := strings.TrimSpace(input)
	if normalized == "" {
		malicious, err := g.inner.IsMalicious(ctx, input)
		if err != nil {
			return false, fmt.Errorf("cached guard isMalicious failed: %w", err)
		}
		return malicious, nil
	}

	key := cacheKey(normalized)
	if g.cache != nil {
		if value, ok := g.cache.Get(key); ok {
			return value, nil
		}
	}

	resultCh := g.sf.DoChan(key, func() (any, error) {
		if g.cache != nil {
			if value, ok := g.cache.Get(key); ok {
				return value, nil
			}
		}

		checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()

		malicious, err := g.inner.IsMalicious(checkCtx, input)
		if err != nil {
			return false, fmt.Errorf("cached guard isMalicious failed: %w", err)
		}
		if g.cache != nil {
			g.cache.Set(key, malicious)
		}
		return malicious, nil
	})

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return false, result.Err
		}
		malicious, ok := result.Val.(bool)
		if !ok {
			return false, fmt.Errorf("cached guard invalid singleflight result type: %T", result.Val)
		}
		return malicious, nil
	case <-ctx.Done():
		return false, fmt.Errorf("cached guard context done: %w", ctx.Err())
	}
}

// ValidateOrThrow: 입력을 검증하고 악성이면 에러를 반환합니다.
func (g *CachedInjectionGuard) ValidateOrThrow(ctx context.Context, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", cerrors.MalformedInputError{Message: "empty input"}
	}

	malicious, err := g.IsMalicious(ctx, input)
	if err != nil {
		return "", err
	}
	if malicious {
		if g != nil && g.logger != nil {
			g.logger.Warn("injection_blocked", "input", truncateForLog(input))
		}
		return "", cerrors.InputInjectionError{Message: "potentially malicious input detected"}
	}

	return sanitize(input), nil
}

func cacheKey(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
