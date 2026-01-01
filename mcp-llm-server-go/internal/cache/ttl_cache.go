package cache

import (
	"container/list"
	"sync"
	"time"
)

const ttlCachePurgeLimit = 8

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// TTLCache 캐시는 만료 시간과 최대 크기를 가진 LRU 캐시다.
type TTLCache[K comparable, V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	maxSize int
	order   *list.List
	items   map[K]*list.Element
	enabled bool
}

// NewTTLCache: 만료 시간과 최대 크기를 갖는 TTLCache를 생성합니다.
// maxSize 또는 ttl이 0 이하이면 비활성 캐시를 반환합니다.
func NewTTLCache[K comparable, V any](maxSize int, ttl time.Duration) *TTLCache[K, V] {
	if maxSize <= 0 || ttl <= 0 {
		return &TTLCache[K, V]{enabled: false}
	}
	return &TTLCache[K, V]{
		ttl:     ttl,
		maxSize: maxSize,
		order:   list.New(),
		items:   make(map[K]*list.Element, maxSize),
		enabled: true,
	}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	var zero V
	if c == nil || !c.enabled {
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return zero, false
	}

	ent := element.Value.(*entry[K, V])
	if time.Now().After(ent.expiresAt) {
		c.removeElement(element)
		return zero, false
	}

	c.order.MoveToFront(element)
	return ent.value, true
}

func (c *TTLCache[K, V]) Set(key K, value V) {
	if c == nil || !c.enabled {
		return
	}
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		ent := element.Value.(*entry[K, V])
		ent.value = value
		ent.expiresAt = now.Add(c.ttl)
		c.order.MoveToFront(element)
		return
	}

	ent := &entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: now.Add(c.ttl),
	}
	element := c.order.PushFront(ent)
	c.items[key] = element
	c.purgeExpired(now, ttlCachePurgeLimit)
	c.evictIfNeeded()
}

// Modify: key의 값을 원자적으로 갱신하고 갱신된 값을 반환합니다.
// update 함수는 캐시 내부 락을 잡은 상태로 호출되므로, update 내부에서 긴 연산을 수행하지 않아야 한다.
func (c *TTLCache[K, V]) Modify(key K, update func(current V, exists bool) V) (V, bool) {
	var zero V
	if c == nil || !c.enabled || update == nil {
		return zero, false
	}
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		ent := element.Value.(*entry[K, V])
		if now.After(ent.expiresAt) {
			c.removeElement(element)
		} else {
			ent.value = update(ent.value, true)
			ent.expiresAt = now.Add(c.ttl)
			c.order.MoveToFront(element)
			return ent.value, true
		}
	}

	value := update(zero, false)
	ent := &entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: now.Add(c.ttl),
	}
	element := c.order.PushFront(ent)
	c.items[key] = element
	c.purgeExpired(now, ttlCachePurgeLimit)
	c.evictIfNeeded()
	return value, true
}

func (c *TTLCache[K, V]) Delete(key K) {
	if c == nil || !c.enabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return
	}
	c.removeElement(element)
}

func (c *TTLCache[K, V]) Len() int {
	if c == nil || !c.enabled {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

func (c *TTLCache[K, V]) purgeExpired(now time.Time, limit int) {
	for i := 0; i < limit; i++ {
		element := c.order.Back()
		if element == nil {
			return
		}
		ent := element.Value.(*entry[K, V])
		if now.Before(ent.expiresAt) {
			return
		}
		c.removeElement(element)
	}
}

func (c *TTLCache[K, V]) evictIfNeeded() {
	for len(c.items) > c.maxSize {
		element := c.order.Back()
		if element == nil {
			return
		}
		c.removeElement(element)
	}
}

func (c *TTLCache[K, V]) removeElement(element *list.Element) {
	c.order.Remove(element)
	ent := element.Value.(*entry[K, V])
	delete(c.items, ent.key)
}
