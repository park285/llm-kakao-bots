package lockutil

import (
	"context"
	"sync"
)

type scopeKey struct{}

// HeldLock: Scope 내에 보관되는 락 상태를 표현합니다.
type HeldLock struct {
	Token     string
	Count     int
	Mode      int
	StopRenew context.CancelFunc
}

// Scope: 재진입 락 상태를 Context 범위로 관리합니다.
type Scope struct {
	mu   sync.Mutex
	held map[string]*HeldLock
}

// NewScope: 새 Scope를 생성합니다.
func NewScope() *Scope {
	return &Scope{held: make(map[string]*HeldLock)}
}

// WithScope: Context에 Scope를 보관합니다.
func WithScope(ctx context.Context, scope *Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope)
}

// ScopeFromContext: Context에서 Scope를 가져옵니다.
func ScopeFromContext(ctx context.Context) (*Scope, bool) {
	scope, ok := ctx.Value(scopeKey{}).(*Scope)
	return scope, ok && scope != nil
}

// IncrementIfHeld: 이미 보유 중인 락이면 count를 증가시키고 true를 반환합니다.
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

// Decrement: count를 감소시키고 0 이하이면 제거합니다.
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

// Set: 락을 Scope에 등록합니다.
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

// IsHeld: 해당 key가 보유 중인지 확인합니다.
func (s *Scope) IsHeld(key string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.held[key]
	return ok
}

// ReleaseIfLast: count를 줄이고 마지막이면 보관된 정보를 반환합니다.
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
