package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin: 관리자 로그인을 처리합니다.
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
	session, err := h.sessions.CreateSession(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to create admin session",
			slog.String("ip", ip),
			slog.Any("error", err),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session store unavailable"})
		return
	}
	signedSessionID := SignSessionID(session.ID, h.securityCfg.SessionSecret)
	SetSecureCookie(c, sessionCookieName, signedSessionID, 0, h.securityCfg.ForceHTTPS) // 0 = 세션 쿠키 (브라우저 종료 시 삭제)

	// Client Hints 수집 (실제 기기 정보)
	clientHints := ParseClientHints(c)
	deviceInfo := clientHints.Summary()
	if deviceInfo == "" {
		deviceInfo = truncateUA(c.Request.UserAgent())
	}

	h.logger.Info("Admin logged in",
		slog.String("username", req.Username),
		slog.String("ip", ip),
		slog.String("device", deviceInfo),
	)

	// 활동 로그에 Client Hints 정보 추가
	logDetails := map[string]any{
		"username": req.Username,
		"ip":       ip,
	}
	if clientHints.HasClientHints() {
		logDetails["device"] = deviceInfo
		for k, v := range clientHints.ToLogFields() {
			logDetails[k] = v
		}
	} else {
		logDetails["ua"] = truncateUA(c.Request.UserAgent())
	}
	h.activity.Log("auth_login", "Admin login successful", logDetails)

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

// HandleLogout: 관리자 로그아웃을 처리합니다. (JSON API)
// 명시적 로그아웃 시에는 Grace Period를 적용하지 않고 DeleteSession으로 즉시 삭제합니다.
// RotateSession이나 expireSession을 사용하면 안 됩니다.
func (h *AdminHandler) HandleLogout(c *gin.Context) {
	signedSessionID, _ := c.Cookie(sessionCookieName)
	if signedSessionID != "" {
		// 서명 검증 후 즉시 삭제 (Grace Period 없음)
		if sessionID, valid := ValidateSessionSignature(signedSessionID, h.securityCfg.SessionSecret); valid {
			h.sessions.DeleteSession(c.Request.Context(), sessionID)
		}
	}

	ClearSecureCookie(c, sessionCookieName, h.securityCfg.ForceHTTPS)

	// Client Hints 수집 (실제 기기 정보)
	clientHints := ParseClientHints(c)
	logDetails := map[string]any{
		"ip": c.ClientIP(),
	}
	if clientHints.HasClientHints() {
		logDetails["device"] = clientHints.Summary()
	}
	h.activity.Log("auth_logout", "Admin logout", logDetails)

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Logout successful",
	})
}

// heartbeatRequest: 하트비트 요청 구조체
type heartbeatRequest struct {
	Idle bool `json:"idle"` // 클라이언트 유휴 상태 여부
}

// heartbeatResponse: 하트비트 응답 구조체
type heartbeatResponse struct {
	Status            string `json:"status"`
	Rotated           bool   `json:"rotated,omitempty"`             // 세션 ID가 갱신되었는지 여부
	AbsoluteExpiresAt int64  `json:"absolute_expires_at,omitempty"` // Unix timestamp (절대 만료 시간)
	IdleRejected      bool   `json:"idle_rejected,omitempty"`       // 유휴 상태로 갱신 거부됨
}

// HandleHeartbeat: 세션 TTL을 갱신합니다. (프론트엔드에서 주기적으로 호출)
// 보안 강화 사항:
// 1. idle=true면 세션 갱신 거부 (클라이언트에서 로그아웃 유도)
// 2. 절대 만료 시간 초과 시 세션 즉시 삭제 및 401 반환
// 3. TokenRotation 활성화 시 새 세션 ID 발급 (토큰 탈취 피해 최소화)
func (h *AdminHandler) HandleHeartbeat(c *gin.Context) {
	signedSessionID, err := c.Cookie(sessionCookieName)
	if err != nil || signedSessionID == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	sessionID, valid := ValidateSessionSignature(signedSessionID, h.securityCfg.SessionSecret)
	if !valid {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	// 요청 파싱 (idle 플래그)
	var req heartbeatRequest
	// JSON 바디가 없어도 허용 (하위 호환성)
	_ = c.ShouldBindJSON(&req)

	ctx := c.Request.Context()

	// RefreshSessionWithValidation 호출 (idle 검증, 절대 만료 검증)
	refreshed, absoluteExpired, err := h.sessions.RefreshSessionWithValidation(ctx, sessionID, req.Idle)
	if err != nil {
		h.logger.Error("Heartbeat refresh error",
			slog.String("session_id", truncateSessionID(sessionID)),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	// 절대 만료 시간 초과
	if absoluteExpired {
		ClearSecureCookie(c, sessionCookieName, h.securityCfg.ForceHTTPS)
		c.JSON(401, gin.H{
			"error":            "Session expired",
			"absolute_expired": true,
		})
		return
	}

	// 유휴 상태로 갱신 거부됨
	if req.Idle && !refreshed {
		c.JSON(200, heartbeatResponse{
			Status:       "idle",
			IdleRejected: true,
		})
		return
	}

	// 세션이 없거나 만료됨
	if !refreshed {
		ClearSecureCookie(c, sessionCookieName, h.securityCfg.ForceHTTPS)
		c.JSON(401, gin.H{"error": "Session expired"})
		return
	}

	// 토큰 갱신 (설정 활성화 시)
	response := heartbeatResponse{Status: "ok"}

	if h.config.SessionTokenRotation {
		newSession, rotateErr := h.sessions.RotateSession(ctx, sessionID)
		if rotateErr != nil {
			h.logger.Warn("Session rotation failed, keeping existing session",
				slog.String("session_id", truncateSessionID(sessionID)),
				slog.Any("error", rotateErr),
			)
			// 실패해도 기존 세션 유지 (graceful degradation)
		} else {
			// 새 세션 ID로 쿠키 갱신
			newSignedSessionID := SignSessionID(newSession.ID, h.securityCfg.SessionSecret)
			SetSecureCookie(c, sessionCookieName, newSignedSessionID, 0, h.securityCfg.ForceHTTPS)
			response.Rotated = true
			response.AbsoluteExpiresAt = newSession.AbsoluteExpiresAt.Unix()
		}
	}

	c.JSON(200, response)
}
