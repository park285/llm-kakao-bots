package cache

import (
	"container/list"
	"sync"
	"time"
)

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
}

// NewTTLCache 는 만료 시간과 최대 크기를 갖는 TTLCache 를 생성한다.
func NewTTLCache[K comparable, V any](maxSize int, ttl time.Duration) *TTLCache[K, V] {
	if maxSize <= 0 {
		maxSize = 1
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	return &TTLCache[K, V]{
		ttl:     ttl,
		maxSize: maxSize,
		order:   list.New(),
		items:   make(map[K]*list.Element, maxSize),
	}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	var zero V
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		ent := element.Value.(*entry[K, V])
		ent.value = value
		ent.expiresAt = time.Now().Add(c.ttl)
		c.order.MoveToFront(element)
		return
	}

	ent := &entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	element := c.order.PushFront(ent)
	c.items[key] = element
	c.evictIfNeeded()
}

func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return
	}
	c.removeElement(element)
}

func (c *TTLCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
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
