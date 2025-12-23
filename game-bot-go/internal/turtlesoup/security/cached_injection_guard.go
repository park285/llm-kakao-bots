package security

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
)

// CachedInjectionGuard 는 타입이다.
type CachedInjectionGuard struct {
	inner  InjectionGuard
	cache  *ttlLRUCache
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
		cache:  newTTLLRUCache(maxEntries, ttl),
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
		return "", tserrors.MalformedInputError{Message: "empty input"}
	}

	malicious, err := g.IsMalicious(ctx, input)
	if err != nil {
		return "", err
	}
	if malicious {
		if g != nil && g.logger != nil {
			g.logger.Warn("injection_blocked", "input", truncateForLog(input))
		}
		return "", tserrors.InputInjectionError{Message: "potentially malicious input detected"}
	}

	return sanitize(input), nil
}

func cacheKey(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type ttlLRUCache struct {
	mu         sync.Mutex
	maxEntries int
	ttl        time.Duration
	items      map[string]*list.Element
	order      *list.List
}

type ttlLRUEntry struct {
	key       string
	value     bool
	expiresAt time.Time
}

func newTTLLRUCache(maxEntries int, ttl time.Duration) *ttlLRUCache {
	if maxEntries <= 0 || ttl <= 0 {
		return nil
	}
	return &ttlLRUCache{
		maxEntries: maxEntries,
		ttl:        ttl,
		items:      make(map[string]*list.Element, maxEntries),
		order:      list.New(),
	}
}

func (c *ttlLRUCache) Get(key string) (bool, bool) {
	if c == nil {
		return false, false
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return false, false
	}

	entry := elem.Value.(ttlLRUEntry)
	if !entry.expiresAt.After(now) {
		c.removeElement(elem)
		return false, false
	}

	c.order.MoveToFront(elem)
	return entry.value, true
}

func (c *ttlLRUCache) Set(key string, value bool) {
	if c == nil {
		return
	}

	expiresAt := time.Now().Add(c.ttl)

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value = ttlLRUEntry{
			key:       key,
			value:     value,
			expiresAt: expiresAt,
		}
		return
	}

	elem := c.order.PushFront(ttlLRUEntry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	})
	c.items[key] = elem

	for len(c.items) > c.maxEntries {
		back := c.order.Back()
		if back == nil {
			break
		}
		c.removeElement(back)
	}
}

func (c *ttlLRUCache) removeElement(elem *list.Element) {
	entry := elem.Value.(ttlLRUEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
}
