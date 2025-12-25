package session

import (
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

// createSessionMemory 메모리 백엔드 세션 생성
func (s *Store) createSessionMemory(meta Meta) error {
	now := time.Now()
	expiresAt := s.computeExpiry(now)

	s.mu.Lock()
	s.pruneExpiredLocked(now)
	s.meta[meta.ID] = meta
	if !expiresAt.IsZero() {
		s.metaExpiresAt[meta.ID] = expiresAt
	} else {
		delete(s.metaExpiresAt, meta.ID)
	}
	s.mu.Unlock()
	return nil
}

// getSessionMemory 메모리 백엔드 세션 메타데이터 조회
func (s *Store) getSessionMemory(sessionID string) (*Meta, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}

	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	expiresAt, ok := s.metaExpiresAt[sessionID]
	if ok && !expiresAt.IsZero() && now.After(expiresAt) {
		delete(s.metaExpiresAt, sessionID)
		delete(s.meta, sessionID)
		s.mu.Unlock()
		return nil, ErrSessionNotFound
	}

	meta, ok := s.meta[sessionID]
	if !ok {
		s.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	s.mu.Unlock()

	copied := meta
	return &copied, nil
}

// updateSessionMemory 메모리 백엔드 세션 메타데이터 업데이트
func (s *Store) updateSessionMemory(meta Meta) error {
	now := time.Now()
	meta.UpdatedAt = now
	expiresAt := s.computeExpiry(now)

	s.mu.Lock()
	s.pruneExpiredLocked(now)
	s.meta[meta.ID] = meta
	if !expiresAt.IsZero() {
		s.metaExpiresAt[meta.ID] = expiresAt
	} else {
		delete(s.metaExpiresAt, meta.ID)
	}
	s.mu.Unlock()
	return nil
}

// deleteSessionMemory 메모리 백엔드 세션 삭제
func (s *Store) deleteSessionMemory(sessionID string) error {
	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	delete(s.meta, sessionID)
	delete(s.history, sessionID)
	delete(s.metaExpiresAt, sessionID)
	delete(s.historyExpireAt, sessionID)
	s.mu.Unlock()
	return nil
}

// getHistoryMemory 메모리 백엔드 히스토리 조회
func (s *Store) getHistoryMemory(sessionID string) []llm.HistoryEntry {
	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	expiresAt, ok := s.historyExpireAt[sessionID]
	if ok && !expiresAt.IsZero() && now.After(expiresAt) {
		delete(s.historyExpireAt, sessionID)
		delete(s.history, sessionID)
		s.mu.Unlock()
		return nil
	}

	history := s.history[sessionID]
	if len(history) == 0 {
		s.mu.Unlock()
		return nil
	}
	copied := append([]llm.HistoryEntry(nil), history...)
	s.mu.Unlock()
	return copied
}

// appendHistoryMemory 메모리 백엔드 히스토리 추가
func (s *Store) appendHistoryMemory(sessionID string, entries ...llm.HistoryEntry) error {
	now := time.Now()
	expiresAt := s.computeExpiry(now)

	s.mu.Lock()
	s.pruneExpiredLocked(now)
	existing := s.history[sessionID]
	existing = append(existing, entries...)

	maxPairs := 0
	if s.cfg != nil {
		maxPairs = s.cfg.Session.HistoryMaxPairs
	}
	if maxPairs > 0 {
		maxEntries := maxPairs * 2
		if len(existing) > maxEntries {
			existing = existing[len(existing)-maxEntries:]
		}
	}

	s.history[sessionID] = existing
	if !expiresAt.IsZero() {
		s.historyExpireAt[sessionID] = expiresAt
	} else {
		delete(s.historyExpireAt, sessionID)
	}
	s.mu.Unlock()
	return nil
}

// sessionCountMemory 메모리 백엔드 세션 수 조회
func (s *Store) sessionCountMemory() int {
	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	count := len(s.meta)
	s.mu.Unlock()
	return count
}

// computeExpiry TTL 기반 만료 시간 계산
func (s *Store) computeExpiry(now time.Time) time.Time {
	ttl := time.Duration(0)
	if s != nil {
		ttl = s.ttl()
	}
	if ttl <= 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}

// pruneExpiredLocked 만료된 세션 정리 (락 보유 상태에서 호출)
func (s *Store) pruneExpiredLocked(now time.Time) {
	for sessionID, expiresAt := range s.metaExpiresAt {
		if expiresAt.IsZero() || now.Before(expiresAt) {
			continue
		}
		delete(s.metaExpiresAt, sessionID)
		delete(s.meta, sessionID)
	}

	for sessionID, expiresAt := range s.historyExpireAt {
		if expiresAt.IsZero() || now.Before(expiresAt) {
			continue
		}
		delete(s.historyExpireAt, sessionID)
		delete(s.history, sessionID)
	}
}
