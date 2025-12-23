package server

import (
	"time"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "admin_session"

// SessionProvider 인터페이스 - 세션 저장소 공통 인터페이스
type SessionProvider interface {
	CreateSession() *Session
	ValidateSession(sessionID string) bool
	DeleteSession(sessionID string)
}

// Session represents an admin session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AdminAuthMiddleware validates admin session for API endpoints
func AdminAuthMiddleware(sessions SessionProvider, sessionSecret string) gin.HandlerFunc {
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

		if !sessions.ValidateSession(sessionID) {
			// 보안 쿠키 삭제 (ForceHTTPS는 런타임에 결정)
			isSecure := c.Request.TLS != nil
			ClearSecureCookie(c, sessionCookieName, isSecure)
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}
