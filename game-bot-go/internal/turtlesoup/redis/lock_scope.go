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
	token string
	count int
}

func newLockScope() *lockScope {
	return &lockScope{held: make(map[string]*heldLock)}
}

func withLockScope(ctx context.Context, scope *lockScope) context.Context {
	return context.WithValue(ctx, lockScopeKey{}, scope)
}

func lockScopeFromContext(ctx context.Context) (*lockScope, bool) {
	scope, ok := ctx.Value(lockScopeKey{}).(*lockScope)
	return scope, ok && scope != nil
}

func (s *lockScope) incrementIfHeld(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.held[key]
	if !ok {
		return false
	}
	entry.count++
	return true
}

func (s *lockScope) set(key string, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.held[key] = &heldLock{token: token, count: 1}
}

func (s *lockScope) decrement(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.held[key]
	if !ok {
		return
	}
	entry.count--
	if entry.count <= 0 {
		delete(s.held, key)
	}
}

func (s *lockScope) releaseIfLast(key string) (token string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.held[key]
	if !ok {
		return "", false
	}

	entry.count--
	if entry.count > 0 {
		return "", false
	}

	delete(s.held, key)
	return entry.token, true
}
