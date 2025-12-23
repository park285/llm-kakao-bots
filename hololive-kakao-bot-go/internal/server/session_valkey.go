package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/valkey-io/valkey-go"
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
)

const (
	sessionKeyPrefix = "session:admin:"
)

// ValkeySessionStore 는 Valkey 기반 세션 저장소로 서버 재기동 시에도 세션을 유지한다.
type ValkeySessionStore struct {
	client valkey.Client
	logger *zap.Logger
	ttl    time.Duration
}

// NewValkeySessionStore creates a new Valkey-based session store
func NewValkeySessionStore(client valkey.Client, logger *zap.Logger) *ValkeySessionStore {
	return &ValkeySessionStore{
		client: client,
		logger: logger,
		ttl:    constants.SessionConfig.ExpiryDuration,
	}
}

// CreateSession creates a new admin session and stores it in Valkey
func (s *ValkeySessionStore) CreateSession() *Session {
	sessionID := generateValkeySessionID()
	now := time.Now()
	session := &Session{
		ID:        sessionID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := sessionKeyPrefix + sessionID

	// 세션 데이터를 JSON으로 직렬화
	data, err := json.Marshal(session)
	if err != nil {
		s.logger.Error("Failed to marshal session", zap.Error(err))
		return session
	}

	// Valkey에 저장 (TTL 설정)
	cmd := s.client.B().Set().Key(key).Value(string(data)).ExSeconds(int64(s.ttl.Seconds())).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		s.logger.Error("Failed to store session in Valkey", zap.String("session_id", truncateSessionID(sessionID)), zap.Error(err))
	} else {
		s.logger.Debug("Session created in Valkey", zap.String("session_id", truncateSessionID(sessionID)), zap.Duration("ttl", s.ttl))
	}

	return session
}

// ValidateSession checks if a session exists and is valid in Valkey
func (s *ValkeySessionStore) ValidateSession(sessionID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := sessionKeyPrefix + sessionID

	resp := s.client.Do(ctx, s.client.B().Get().Key(key).Build())
	if valkey.IsValkeyNil(resp.Error()) {
		return false // 세션 없음
	}
	if resp.Error() != nil {
		s.logger.Error("Failed to validate session", zap.String("session_id", truncateSessionID(sessionID)), zap.Error(resp.Error()))
		return false
	}

	value, err := resp.ToString()
	if err != nil {
		s.logger.Error("Failed to read session value", zap.String("session_id", truncateSessionID(sessionID)), zap.Error(err))
		return false
	}

	var session Session
	if err := json.Unmarshal([]byte(value), &session); err != nil {
		s.logger.Error("Failed to unmarshal session", zap.String("session_id", truncateSessionID(sessionID)), zap.Error(err))
		return false
	}

	// 만료 시간 확인 (Valkey TTL이 있지만 이중 확인)
	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(sessionID)
		return false
	}

	return true
}

// DeleteSession removes a session from Valkey
func (s *ValkeySessionStore) DeleteSession(sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := sessionKeyPrefix + sessionID

	if err := s.client.Do(ctx, s.client.B().Del().Key(key).Build()).Error(); err != nil {
		s.logger.Error("Failed to delete session", zap.String("session_id", truncateSessionID(sessionID)), zap.Error(err))
	} else {
		s.logger.Debug("Session deleted from Valkey", zap.String("session_id", truncateSessionID(sessionID)))
	}
}

func generateValkeySessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// truncateSessionID 세션 ID 앞 8자리만 반환 (로그 보안)
func truncateSessionID(sessionID string) string {
	if len(sessionID) <= 8 {
		return sessionID
	}
	return sessionID[:8] + "..."
}
