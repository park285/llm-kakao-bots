package server

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin 는 동작을 수행한다.
func (h *AdminHandler) HandleLogin(c *gin.Context) {
	ip := c.ClientIP()

	// Rate limit 확인
	allowed, remaining := h.rateLimiter.IsAllowed(ip)
	if !allowed {
		h.logger.Warn("Login rate limited",
			slog.String("ip", ip),
			slog.Duration("remaining", remaining),
		)
		c.Header("Retry-After", strconv.Itoa(int(remaining.Seconds())))
		c.JSON(429, gin.H{
			"error":       "Too many login attempts",
			"retry_after": remaining.Seconds(),
		})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// 사용자명 확인
	if req.Username != h.adminUser {
		h.handleLoginFailure(c, ip, req.Username, "invalid_username")
		return
	}

	// bcrypt 해시 비교
	if err := bcrypt.CompareHashAndPassword([]byte(h.adminPassHash), []byte(req.Password)); err != nil {
		h.handleLoginFailure(c, ip, req.Username, "invalid_password")
		return
	}

	// 성공: rate limiter 초기화
	h.rateLimiter.RecordSuccess(ip)

	// 세션 생성 및 HMAC 서명
	session := h.sessions.CreateSession(c.Request.Context())
	signedSessionID := SignSessionID(session.ID, h.securityCfg.SessionSecret)
	SetSecureCookie(c, sessionCookieName, signedSessionID, 86400, h.securityCfg.ForceHTTPS)

	h.logger.Info("Admin logged in",
		slog.String("username", req.Username),
		slog.String("ip", ip),
	)

	h.activity.Log("auth_login", "Admin login successful", map[string]any{
		"username": req.Username,
		"ip":       ip,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Login successful",
	})
}

// handleLoginFailure 로그인 실패 처리 (중복 코드 제거)
func (h *AdminHandler) handleLoginFailure(c *gin.Context, ip, username, reason string) {
	failCount := h.rateLimiter.RecordFailure(ip)

	h.logger.Warn("Failed login attempt",
		slog.String("username", username),
		slog.String("ip", ip),
		slog.String("reason", reason),
		slog.Int("fail_count", failCount),
	)

	// 점진적 지연: 실패 횟수에 따라 대기
	delay := time.Duration(failCount) * 500 * time.Millisecond
	if delay > 3*time.Second {
		delay = 3 * time.Second // 최대 3초
	}
	time.Sleep(delay)

	c.JSON(200, gin.H{"success": false, "error": "Authentication failed"})
}

// HandleLogout processes admin logout (JSON API)
func (h *AdminHandler) HandleLogout(c *gin.Context) {
	signedSessionID, _ := c.Cookie(sessionCookieName)
	if signedSessionID != "" {
		// 서명 검증 후 삭제
		if sessionID, valid := ValidateSessionSignature(signedSessionID, h.securityCfg.SessionSecret); valid {
			h.sessions.DeleteSession(sessionID)
		}
	}

	ClearSecureCookie(c, sessionCookieName, h.securityCfg.ForceHTTPS)

	h.activity.Log("auth_logout", "Admin logout", map[string]any{
		"ip": c.ClientIP(),
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Logout successful",
	})
}
