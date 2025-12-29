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

// CachedInjectionGuard 는 타입이다.
type CachedInjectionGuard struct {
	inner  InjectionGuard
	cache  *commoncache.TTLLRUCache
	logger *slog.Logger
	sf     singleflight.Group
}

// NewCachedInjectionGuard 는 동작을 수행한다.
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

// IsMalicious 는 동작을 수행한다.
func (g *CachedInjectionGuard) IsMalicious(ctx context.Context, input string) (bool, error) {
	if g == nil || g.inner == nil {
		return false, nil
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

		malicious, err := g.inner.IsMalicious(ctx, input)
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

// ValidateOrThrow 는 동작을 수행한다.
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
