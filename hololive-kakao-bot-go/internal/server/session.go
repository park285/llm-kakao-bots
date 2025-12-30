package server

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "admin_session"

// SessionProvider 인터페이스 - 세션 저장소 공통 인터페이스
type SessionProvider interface {
	CreateSession(ctx context.Context) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	ValidateSession(ctx context.Context, sessionID string) bool
	DeleteSession(ctx context.Context, sessionID string)
	RefreshSession(ctx context.Context, sessionID string) bool // Heartbeat용 TTL 갱신 (deprecated: RefreshSessionWithValidation 사용)
	// RefreshSessionWithValidation: 절대 만료 시간을 검증하고 TTL을 갱신합니다.
	// idle=true면 세션 갱신을 거부합니다 (유휴 상태).
	// 반환값: (성공여부, 절대만료여부, 에러)
	RefreshSessionWithValidation(ctx context.Context, sessionID string, idle bool) (refreshed bool, absoluteExpired bool, err error)
	// RotateSession: 기존 세션을 삭제하고 새 세션을 생성합니다 (토큰 갱신).
	// 원본 세션의 AbsoluteExpiresAt을 유지합니다.
	RotateSession(ctx context.Context, oldSessionID string) (*Session, error)
}

// Session: 관리자 세션 정보를 담는 구조체입니다.
// AbsoluteExpiresAt: 절대 만료 시간 (하트비트로 연장 불가, OWASP 권고)
type Session struct {
	ID                string    `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	AbsoluteExpiresAt time.Time `json:"absolute_expires_at"`
}

// AdminAuthMiddleware: API 엔드포인트의 관리자 세션을 검증하는 미들웨어입니다.
func AdminAuthMiddleware(sessions SessionProvider, sessionSecret string, forceHTTPS bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		signedSessionID, err := c.Cookie(sessionCookieName)
		if err != nil || signedSessionID == "" {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// HMAC 서명 검증
		sessionID, valid := ValidateSessionSignature(signedSessionID, sessionSecret)
		if !valid {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if !sessions.ValidateSession(c.Request.Context(), sessionID) {
			// 보안 쿠키 삭제 (TLS termination 환경 포함)
			ClearSecureCookie(c, sessionCookieName, forceHTTPS)
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}
