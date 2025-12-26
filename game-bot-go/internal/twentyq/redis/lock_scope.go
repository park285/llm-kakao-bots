package redis

import (
	"context"
	"sync"
)

type lockScopeKey struct{}

type lockScope struct {
	mu   sync.Mutex
	held map[string]*heldLock
}

type heldLock struct {
	mode      lockMode
	token     string
	count     int
	stopRenew context.CancelFunc
}

func newLockScope() *lockScope {
	return &lockScope{held: make(map[string]*heldLock)}
}

func withLockScope(ctx context.Context, scope *lockScope) context.Context {
	return context.WithValue(ctx, lockScopeKey{}, scope)
}

func lockScopeFromContext(ctx context.Context) (*lockScope, bool) {
	scope, ok := ctx.Value(lockScopeKey{}).(*lockScope)
	return scope, ok
}

func (s *lockScope) incrementIfHeld(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return false
	}

	lock.count++
	return true
}

func (s *lockScope) decrement(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return
	}

	lock.count--
	if lock.count <= 0 {
		delete(s.held, key)
	}
}

func (s *lockScope) set(key string, lock heldLock) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.held[key] = &lock
}

func (s *lockScope) isHeld(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.held[key]
	return ok
}

func (s *lockScope) releaseIfLast(key string) (held heldLock, shouldRelease bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return heldLock{}, false
	}

	lock.count--
	if lock.count > 0 {
		return heldLock{}, false
	}

	delete(s.held, key)
	return *lock, true
}
