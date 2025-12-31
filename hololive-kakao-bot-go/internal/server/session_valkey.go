package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

const (
	sessionKeyPrefix = "session:admin:"
)

// ValkeySessionStore 는 Valkey 기반 세션 저장소로 서버 재기동 시에도 세션을 유지한다.
type ValkeySessionStore struct {
	client valkey.Client
	logger *slog.Logger
	ttl    time.Duration
}

// NewValkeySessionStore: 새로운 Valkey 기반 세션 저장소를 생성합니다.
func NewValkeySessionStore(client valkey.Client, logger *slog.Logger) *ValkeySessionStore {
	return &ValkeySessionStore{
		client: client,
		logger: logger,
		ttl:    constants.SessionConfig.ExpiryDuration,
	}
}

// CreateSession: 새로운 관리자 세션을 생성하고 Valkey에 저장합니다.
// AbsoluteExpiresAt을 설정하여 절대 만료 시간을 강제합니다.
func (s *ValkeySessionStore) CreateSession(ctx context.Context) (*Session, error) {
	sessionID := generateValkeySessionID()
	now := time.Now()
	session := &Session{
		ID:                sessionID,
		CreatedAt:         now,
		ExpiresAt:         now.Add(s.ttl),
		AbsoluteExpiresAt: now.Add(constants.SessionConfig.AbsoluteTimeout),
	}

	if err := s.storeSession(ctx, session); err != nil {
		return nil, err
	}

	s.logger.Debug("Session created in Valkey",
		slog.String("session_id", truncateSessionID(sessionID)),
		slog.Duration("ttl", s.ttl),
		slog.Time("absolute_expires_at", session.AbsoluteExpiresAt),
	)
	return session, nil
}

// storeSession: 세션을 Valkey에 저장하는 내부 헬퍼 함수
func (s *ValkeySessionStore) storeSession(ctx context.Context, session *Session) error {
	if ctx == nil {
		ctx = context.Background()
	}
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	key := sessionKeyPrefix + session.ID

	// 세션 데이터를 JSON으로 직렬화
	data, err := json.Marshal(session)
	if err != nil {
		s.logger.Error("Failed to marshal session", slog.Any("error", err))
		return fmt.Errorf("marshal session: %w", err)
	}

	// Valkey에 저장 (TTL 설정)
	cmd := s.client.B().Set().Key(key).Value(string(data)).ExSeconds(int64(s.ttl.Seconds())).Build()
	if err := s.client.Do(storeCtx, cmd).Error(); err != nil {
		s.logger.Error("Failed to store session in Valkey",
			slog.String("session_id", truncateSessionID(session.ID)),
			slog.Any("error", err),
		)
		return err
	}
	return nil
}

// expireSession: 세션의 TTL을 지정된 값으로 설정하는 내부 헬퍼 함수
func (s *ValkeySessionStore) expireSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	expireCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	key := sessionKeyPrefix + sessionID
	resp := s.client.Do(expireCtx, s.client.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build())
	return resp.Error()
}

// GetSession: Valkey에서 세션 데이터를 조회합니다.
func (s *ValkeySessionStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	key := sessionKeyPrefix + sessionID

	resp := s.client.Do(ctx, s.client.B().Get().Key(key).Build())
	if util.IsValkeyNil(resp.Error()) {
		return nil, nil // 세션 없음 (에러 아님)
	}
	if resp.Error() != nil {
		s.logger.Error("Failed to get session",
			slog.String("session_id", truncateSessionID(sessionID)),
			slog.Any("error", resp.Error()),
		)
		return nil, resp.Error()
	}

	data, err := resp.ToString()
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		s.logger.Error("Failed to unmarshal session",
			slog.String("session_id", truncateSessionID(sessionID)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

// ValidateSession: 세션이 Valkey에 존재하고 유효한지 확인합니다.
func (s *ValkeySessionStore) ValidateSession(ctx context.Context, sessionID string) bool {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil || session == nil {
		return false
	}
	// 절대 만료 시간 검증
	if time.Now().After(session.AbsoluteExpiresAt) {
		s.logger.Warn("Session absolute timeout exceeded",
			slog.String("session_id", truncateSessionID(sessionID)),
		)
		s.DeleteSession(ctx, sessionID)
		return false
	}
	return true
}

// DeleteSession: Valkey에서 세션을 삭제합니다.
func (s *ValkeySessionStore) DeleteSession(ctx context.Context, sessionID string) {
	if ctx == nil {
		ctx = context.Background()
	}
	// 부모 context 취소와 분리하여 삭제 작업 보장
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	key := sessionKeyPrefix + sessionID

	if err := s.client.Do(deleteCtx, s.client.B().Del().Key(key).Build()).Error(); err != nil {
		s.logger.Error("Failed to delete session", slog.String("session_id", truncateSessionID(sessionID)), slog.Any("error", err))
	} else {
		s.logger.Debug("Session deleted from Valkey", slog.String("session_id", truncateSessionID(sessionID)))
	}
}

// RefreshSession: 세션의 TTL을 연장합니다. (Heartbeat, 하위 호환성 유지)
func (s *ValkeySessionStore) RefreshSession(ctx context.Context, sessionID string) bool {
	refreshed, _, _ := s.RefreshSessionWithValidation(ctx, sessionID, false)
	return refreshed
}

// RefreshSessionWithValidation: 절대 만료 시간을 검증하고 TTL을 갱신합니다.
// idle=true면 유휴 상태로 간주하여 세션 TTL을 단축합니다 (즉시 만료 유도).
// 반환값: (갱신성공여부, 절대만료여부, 에러)
func (s *ValkeySessionStore) RefreshSessionWithValidation(ctx context.Context, sessionID string, idle bool) (bool, bool, error) {
	// 세션 존재 확인
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return false, false, err
	}
	if session == nil {
		return false, false, nil // 세션 없음
	}

	// 절대 만료 시간 초과 검증
	if time.Now().After(session.AbsoluteExpiresAt) {
		s.logger.Warn("Session absolute timeout exceeded",
			slog.String("session_id", truncateSessionID(sessionID)),
			slog.Time("absolute_expires_at", session.AbsoluteExpiresAt),
		)
		s.DeleteSession(ctx, sessionID)
		return false, true, nil // 절대 만료
	}

	// 유휴 상태면 세션 TTL을 10초로 단축 (즉시 만료 유도, 보안 강화)
	if idle {
		idleTTL := constants.SessionConfig.IdleSessionTTL
		if err := s.expireSession(ctx, sessionID, idleTTL); err != nil {
			s.logger.Error("Failed to shorten session TTL for idle",
				slog.String("session_id", truncateSessionID(sessionID)),
				slog.Any("error", err),
			)
		}
		s.logger.Info("Session TTL shortened (idle)",
			slog.String("session_id", truncateSessionID(sessionID)),
			slog.Duration("ttl", idleTTL),
		)
		return false, false, nil // 갱신 거부 (클라이언트에서 로그아웃 처리)
	}

	// TTL 갱신 (멀티 탭 환경 대응: 다른 탭에서 단축시켰더라도 여기서 강제 복원)
	if err := s.expireSession(ctx, sessionID, s.ttl); err != nil {
		s.logger.Error("Failed to refresh session",
			slog.String("session_id", truncateSessionID(sessionID)),
			slog.Any("error", err),
		)
		return false, false, err
	}

	s.logger.Debug("Session refreshed",
		slog.String("session_id", truncateSessionID(sessionID)),
		slog.Duration("ttl", s.ttl),
	)
	return true, false, nil
}

// RotateSession: 기존 세션을 삭제하고 새 세션을 생성합니다 (토큰 갱신).
// 원본 세션의 AbsoluteExpiresAt을 유지하여 절대 만료 시간은 연장 불가.
// 기존 세션에는 Grace Period (30초)를 적용하여 Race Condition을 방지합니다.
func (s *ValkeySessionStore) RotateSession(ctx context.Context, oldSessionID string) (*Session, error) {
	// 기존 세션 조회
	oldSession, err := s.GetSession(ctx, oldSessionID)
	if err != nil {
		return nil, err
	}
	if oldSession == nil {
		return nil, fmt.Errorf("session not found")
	}

	// [방어적 코드: 중복 회전 방지]
	// 기존 세션의 남은 TTL 확인.
	// GracePeriod(30초) + 여유분(5초) 이내라면 이미 회전된 세션으로 간주하고 회전 중단.
	//
	// NOTE: 현재 HandleHeartbeat 흐름에서는 RefreshSessionWithValidation이 먼저 호출되어
	// TTL을 1시간으로 연장하므로, 정상 흐름에서는 이 조건이 실행되지 않습니다.
	// 다만, 향후 Refresh 로직 변경이나 직접 RotateSession 호출 시를 대비한 방어적 코드입니다.
	// 삭제해도 보안상 문제는 없으나, 네트워크 지연/병렬 요청 시 이중 회전이 발생할 수 있습니다.
	key := sessionKeyPrefix + oldSessionID
	ttlResp := s.client.Do(ctx, s.client.B().Ttl().Key(key).Build())
	if ttl, err := ttlResp.AsInt64(); err == nil && ttl > 0 {
		graceThreshold := int64((constants.SessionConfig.GracePeriod + 5*time.Second).Seconds())
		if ttl <= graceThreshold {
			s.logger.Debug("Session rotation skipped (already rotating/expiring)",
				slog.String("session_id", truncateSessionID(oldSessionID)),
				slog.Int64("ttl", ttl),
			)
			// 중복 회전 방지: 기존 세션 반환 (클라이언트는 200 OK 받고 계속 진행)
			return oldSession, nil
		}
	}

	// 절대 만료 시간 초과 검증
	if time.Now().After(oldSession.AbsoluteExpiresAt) {
		s.DeleteSession(ctx, oldSessionID)
		return nil, fmt.Errorf("session absolute timeout exceeded")
	}

	// 새 세션 생성 (AbsoluteExpiresAt 유지)
	newSessionID := generateValkeySessionID()
	now := time.Now()
	newSession := &Session{
		ID:                newSessionID,
		CreatedAt:         oldSession.CreatedAt, // 원본 생성 시간 유지
		ExpiresAt:         now.Add(s.ttl),
		AbsoluteExpiresAt: oldSession.AbsoluteExpiresAt, // 절대 만료 시간 유지!
	}

	// 새 세션 저장
	if err := s.storeSession(ctx, newSession); err != nil {
		return nil, err
	}

	// 기존 세션에 Grace Period 설정 (Race Condition 방지)
	// 즉시 삭제 대신 30초 후 만료되도록 설정하여 병렬 요청 보호
	gracePeriod := constants.SessionConfig.GracePeriod
	if err := s.expireSession(ctx, oldSessionID, gracePeriod); err != nil {
		s.logger.Warn("Failed to set grace period for old session, deleting immediately",
			slog.String("session_id", truncateSessionID(oldSessionID)),
			slog.Any("error", err),
		)
		s.DeleteSession(ctx, oldSessionID) // Fallback: 즉시 삭제
	} else {
		s.logger.Debug("Grace period set for old session",
			slog.String("session_id", truncateSessionID(oldSessionID)),
			slog.Duration("grace_period", gracePeriod),
		)
	}

	s.logger.Info("Session rotated",
		slog.String("old_session_id", truncateSessionID(oldSessionID)),
		slog.String("new_session_id", truncateSessionID(newSessionID)),
		slog.Time("absolute_expires_at", newSession.AbsoluteExpiresAt),
		slog.Duration("grace_period", gracePeriod),
	)
	return newSession, nil
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
