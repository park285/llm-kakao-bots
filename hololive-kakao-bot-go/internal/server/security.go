package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig 보안 설정
type SecurityConfig struct {
	SessionSecret string
	ForceHTTPS    bool
}

// SecurityHeadersMiddleware 보안 헤더 추가 미들웨어
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// CSP는 SPA 환경에서 제한적으로 적용
		c.Header("Content-Security-Policy", "frame-ancestors 'none'")
		c.Next()
	}
}

// SetSecureCookie 보안 쿠키 설정
func SetSecureCookie(c *gin.Context, name, value string, maxAge int, forceHTTPS bool) {
	// TLS 연결 감지 또는 환경변수 기반
	isSecure := c.Request.TLS != nil || forceHTTPS

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		value,
		maxAge,
		"/",
		"",       // domain (자동)
		isSecure, // Secure: HTTPS에서만 전송
		true,     // HttpOnly: JS 접근 차단
	)
}

// ClearSecureCookie 쿠키 삭제
func ClearSecureCookie(c *gin.Context, name string, forceHTTPS bool) {
	isSecure := c.Request.TLS != nil || forceHTTPS
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(name, "", -1, "/", "", isSecure, true)
}

// SignSessionID 세션 ID에 HMAC 서명 추가
func SignSessionID(sessionID, secret string) string {
	if secret == "" {
		return sessionID
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return sessionID + "." + signature
}

// ValidateSessionSignature 세션 ID 서명 검증
func ValidateSessionSignature(fullID, secret string) (string, bool) {
	if secret == "" {
		return fullID, true
	}

	parts := strings.SplitN(fullID, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	sessionID, providedSig := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return "", false
	}

	return sessionID, true
}
