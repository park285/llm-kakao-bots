package lockutil

import (
	"context"
	"sync"
)

type scopeKey struct{}

// HeldLock 는 Scope 내에 보관되는 락 상태를 표현한다.
type HeldLock struct {
	Token     string
	Count     int
	Mode      int
	StopRenew context.CancelFunc
}

// Scope 는 재진입 락 상태를 Context 범위로 관리한다.
type Scope struct {
	mu   sync.Mutex
	held map[string]*HeldLock
}

// NewScope 는 새 Scope 를 생성한다.
func NewScope() *Scope {
	return &Scope{held: make(map[string]*HeldLock)}
}

// WithScope 는 Context 에 Scope 를 보관한다.
func WithScope(ctx context.Context, scope *Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope)
}

// ScopeFromContext 는 Context 에서 Scope 를 가져온다.
func ScopeFromContext(ctx context.Context) (*Scope, bool) {
	scope, ok := ctx.Value(scopeKey{}).(*Scope)
	return scope, ok && scope != nil
}

// IncrementIfHeld 는 이미 보유 중인 락이면 count 를 증가시키고 true 를 반환한다.
func (s *Scope) IncrementIfHeld(key string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return false
	}
	lock.Count++
	return true
}

// Decrement 는 count 를 감소시키고 0 이하이면 제거한다.
func (s *Scope) Decrement(key string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return
	}
	lock.Count--
	if lock.Count <= 0 {
		delete(s.held, key)
	}
}

// Set 은 락을 Scope 에 등록한다.
func (s *Scope) Set(key string, lock HeldLock) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if lock.Count <= 0 {
		lock.Count = 1
	}
	s.held[key] = &lock
}

// IsHeld 는 해당 key 가 보유 중인지 확인한다.
func (s *Scope) IsHeld(key string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.held[key]
	return ok
}

// ReleaseIfLast 는 count 를 줄이고 마지막이면 보관된 정보를 반환한다.
func (s *Scope) ReleaseIfLast(key string) (HeldLock, bool) {
	if s == nil {
		return HeldLock{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return HeldLock{}, false
	}

	lock.Count--
	if lock.Count > 0 {
		return HeldLock{}, false
	}

	delete(s.held, key)
	return *lock, true
}
