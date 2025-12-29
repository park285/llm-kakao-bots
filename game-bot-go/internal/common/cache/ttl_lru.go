package cache

import (
	"container/list"
	"sync"
	"time"
)

// TTLLRUCache: TTL 기반 LRU 캐시입니다.
type TTLLRUCache struct {
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

// NewTTLLRUCache: TTL LRU 캐시를 생성합니다.
func NewTTLLRUCache(maxEntries int, ttl time.Duration) *TTLLRUCache {
	if maxEntries <= 0 || ttl <= 0 {
		return nil
	}
	return &TTLLRUCache{
		maxEntries: maxEntries,
		ttl:        ttl,
		items:      make(map[string]*list.Element, maxEntries),
		order:      list.New(),
	}
}

// Get: 캐시에서 값을 조회합니다.
func (c *TTLLRUCache) Get(key string) (bool, bool) {
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

// Set: 캐시에 값을 저장합니다.
func (c *TTLLRUCache) Set(key string, value bool) {
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

func (c *TTLLRUCache) removeElement(elem *list.Element) {
	entry := elem.Value.(ttlLRUEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
}
