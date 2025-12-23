package server

import (
	"sync"
	"time"
)

// LoginRateLimiter 로그인 시도 횟수 제한
type LoginRateLimiter struct {
	attempts    map[string]*attemptInfo
	mu          sync.RWMutex
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
}

type attemptInfo struct {
	count        int
	firstAttempt time.Time
	lockedUntil  time.Time
}

// NewLoginRateLimiter 새 Rate Limiter 생성
func NewLoginRateLimiter() *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts:    make(map[string]*attemptInfo),
		maxAttempts: 5,                // 5회 시도
		window:      5 * time.Minute,  // 5분 윈도우
		lockout:     15 * time.Minute, // 15분 잠금
	}

	// 주기적 정리 고루틴
	go rl.cleanupLoop()

	return rl
}

// IsAllowed IP의 로그인 시도 허용 여부 확인
func (l *LoginRateLimiter) IsAllowed(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	info, exists := l.attempts[ip]
	now := time.Now()

	if !exists {
		l.attempts[ip] = &attemptInfo{count: 0, firstAttempt: now}
		return true, 0
	}

	// 잠금 상태 확인
	if now.Before(info.lockedUntil) {
		return false, info.lockedUntil.Sub(now)
	}

	// 윈도우 만료 시 리셋
	if now.Sub(info.firstAttempt) > l.window {
		info.count = 0
		info.firstAttempt = now
		info.lockedUntil = time.Time{}
	}

	return info.count < l.maxAttempts, 0
}

// RecordFailure 로그인 실패 기록
func (l *LoginRateLimiter) RecordFailure(ip string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	info, exists := l.attempts[ip]
	if !exists {
		info = &attemptInfo{count: 0, firstAttempt: time.Now()}
		l.attempts[ip] = info
	}

	info.count++

	if info.count >= l.maxAttempts {
		info.lockedUntil = time.Now().Add(l.lockout)
	}

	return info.count
}

// RecordSuccess 로그인 성공 시 기록 초기화
func (l *LoginRateLimiter) RecordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// cleanupLoop 만료된 기록 주기적 정리
func (l *LoginRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

func (l *LoginRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, info := range l.attempts {
		// 윈도우 + 잠금 시간 모두 지나면 삭제
		if now.Sub(info.firstAttempt) > l.window+l.lockout {
			delete(l.attempts, ip)
		}
	}
}
