package server

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/activity"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/settings"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
)

// AdminHandler handles admin API requests
type AdminHandler struct {
	repo          *member.Repository
	memberCache   *member.Cache
	valkeyCache   *cache.Service
	alarm         *notification.AlarmService
	holodex       *holodex.Service
	youtube       *youtube.Service
	activity      *activity.Logger
	settings      *settings.Service
	acl           *acl.Service
	config        *config.Config
	sessions      SessionProvider
	rateLimiter   *LoginRateLimiter
	securityCfg   *SecurityConfig
	adminUser     string
	adminPassHash string
	logger        *slog.Logger
	startTime     time.Time
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	repo *member.Repository,
	memberCache *member.Cache,
	valkeyCache *cache.Service,
	alarm *notification.AlarmService,
	holodex *holodex.Service,
	youtube *youtube.Service,
	activity *activity.Logger,
	settings *settings.Service,
	aclSvc *acl.Service,
	cfg *config.Config,
	sessions SessionProvider,
	rateLimiter *LoginRateLimiter,
	securityCfg *SecurityConfig,
	adminUser, adminPassHash string,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		repo:          repo,
		memberCache:   memberCache,
		valkeyCache:   valkeyCache,
		alarm:         alarm,
		holodex:       holodex,
		youtube:       youtube,
		activity:      activity,
		settings:      settings,
		acl:           aclSvc,
		config:        cfg,
		sessions:      sessions,
		rateLimiter:   rateLimiter,
		securityCfg:   securityCfg,
		adminUser:     adminUser,
		adminPassHash: adminPassHash,
		logger:        logger,
		startTime:     time.Now(),
	}
}

// aliasRequest represents a unified alias request
type aliasRequest struct {
	Type  string `json:"type" binding:"required,oneof=ko ja"`
	Alias string `json:"alias" binding:"required,min=1"`
}

// handleAliasOperation processes alias add/remove operations with shared logic
func (h *AdminHandler) handleAliasOperation(
	c *gin.Context,
	repoFunc func(context.Context, int, string, string) error,
	operationName string,
) {
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("Invalid member ID", slog.String("id", c.Param("id")), slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid member ID"})
		return
	}

	var req aliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	if err := repoFunc(ctx, memberID, req.Type, req.Alias); err != nil {
		h.logger.Error("Failed to "+operationName+" alias",
			slog.Int("member_id", memberID),
			slog.String("type", req.Type),
			slog.String("alias", req.Alias),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Failed to " + operationName + " alias"})
		return
	}

	if err := h.memberCache.InvalidateAliasCache(ctx, req.Alias); err != nil {
		h.logger.Warn("Failed to invalidate alias cache", slog.Any("error", err))
	}

	h.logger.Info("Alias "+operationName,
		slog.Int("member_id", memberID),
		slog.String("type", req.Type),
		slog.String("alias", req.Alias),
	)

	h.activity.Log("member_alias_"+operationName, fmt.Sprintf("Member alias %s: %s (ID: %d)", operationName, req.Alias, memberID), map[string]any{
		"member_id": memberID,
		"type":      req.Type,
		"alias":     req.Alias,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Alias " + operationName + " successfully",
	})
}

// AddAlias adds an alias to a member
// AddAlias adds an alias to a member
func (h *AdminHandler) AddAlias(c *gin.Context) {
	h.handleAliasOperation(c, h.repo.AddAlias, "add")
}

// RemoveAlias removes an alias from a member
// RemoveAlias removes an alias from a member
func (h *AdminHandler) RemoveAlias(c *gin.Context) {
	h.handleAliasOperation(c, h.repo.RemoveAlias, "remove")
}

// SetGraduation 는 졸업 상태를 갱신한다.
//
//nolint:dupl // Similar patterns for different update operations
func (h *AdminHandler) SetGraduation(c *gin.Context) {
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("Invalid member ID", slog.String("id", c.Param("id")), slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid member ID"})
		return
	}

	var req struct {
		IsGraduated bool `json:"isGraduated"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	if err := h.repo.SetGraduation(ctx, memberID, req.IsGraduated); err != nil {
		h.logger.Error("Failed to set graduation status",
			slog.Int("member_id", memberID),
			slog.Bool("is_graduated", req.IsGraduated),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Failed to set graduation status"})
		return
	}

	if err := h.memberCache.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh cache after graduation update", slog.Any("error", err))
	}

	h.logger.Info("Graduation status updated",
		slog.Int("member_id", memberID),
		slog.Bool("is_graduated", req.IsGraduated),
	)

	statusStr := "graduated"
	if !req.IsGraduated {
		statusStr = "active"
	}
	h.activity.Log("member_graduation", fmt.Sprintf("Member status changed to %s (ID: %d)", statusStr, memberID), map[string]any{
		"member_id":    memberID,
		"is_graduated": req.IsGraduated,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Graduation status updated successfully",
	})
}

// UpdateChannelID 는 채널 ID를 갱신한다.
//
//nolint:dupl // Similar patterns for different update operations
func (h *AdminHandler) UpdateChannelID(c *gin.Context) {
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("Invalid member ID", slog.String("id", c.Param("id")), slog.Any("error", err))
		c.JSON(400, gin.H{"error": "Invalid member ID"})
		return
	}

	var req struct {
		ChannelID string `json:"channelId" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	if err := h.repo.UpdateChannelID(ctx, memberID, req.ChannelID); err != nil {
		h.logger.Error("Failed to update channel ID",
			slog.Int("member_id", memberID),
			slog.String("channel_id", req.ChannelID),
			slog.Any("error", err),
		)
		c.JSON(500, gin.H{"error": "Failed to update channel ID"})
		return
	}

	if err := h.memberCache.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh cache after channel ID update", slog.Any("error", err))
	}

	h.logger.Info("Channel ID updated",
		slog.Int("member_id", memberID),
		slog.String("channel_id", req.ChannelID),
	)

	h.activity.Log("member_channel_update", fmt.Sprintf("Member channel ID updated to %s (ID: %d)", req.ChannelID, memberID), map[string]any{
		"member_id":  memberID,
		"channel_id": req.ChannelID,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Channel ID updated successfully",
	})
}

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
	session := h.sessions.CreateSession()
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

// GetMembers returns all members as JSON
func (h *AdminHandler) GetMembers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	members, err := h.repo.GetAllMembers(ctx)
	if err != nil {
		h.logger.Error("Failed to get members", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get members"})
		return
	}

	c.JSON(200, gin.H{
		"status":  "ok",
		"members": members,
	})
}

// GetAlarms returns all alarms as JSON
func (h *AdminHandler) GetAlarms(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	// Get all alarm registry keys
	alarmKeys, err := h.alarm.GetAllAlarmKeys(ctx)
	if err != nil {
		h.logger.Error("Failed to get alarm keys", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get alarms"})
		return
	}

	c.JSON(200, gin.H{
		"status": "ok",
		"alarms": alarmKeys,
	})
}

// DeleteAlarm deletes a specific alarm
func (h *AdminHandler) DeleteAlarm(c *gin.Context) {
	var req struct {
		RoomID    string `json:"roomId" binding:"required"`
		UserID    string `json:"userId" binding:"required"`
		ChannelID string `json:"channelId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	removed, err := h.alarm.RemoveAlarm(ctx, req.RoomID, req.UserID, req.ChannelID)
	if err != nil {
		h.logger.Error("Failed to delete alarm", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to delete alarm"})
		return
	}

	h.activity.Log("alarm_delete", fmt.Sprintf("Alarm deleted: %s / %s", req.UserID, req.ChannelID), map[string]any{
		"room_id":    req.RoomID,
		"user_id":    req.UserID,
		"channel_id": req.ChannelID,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"removed": removed,
	})
}

// GetRooms returns configured room list
func (h *AdminHandler) GetRooms(c *gin.Context) {
	if h.acl == nil {
		c.JSON(503, gin.H{"error": "ACL service not available"})
		return
	}
	aclEnabled, rooms := h.acl.GetACLStatus()
	c.JSON(200, gin.H{
		"status":     "ok",
		"rooms":      rooms,
		"aclEnabled": aclEnabled,
	})
}

// AddRoom adds a new room to the whitelist
func (h *AdminHandler) AddRoom(c *gin.Context) {
	if h.acl == nil {
		c.JSON(503, gin.H{"error": "ACL service not available"})
		return
	}

	var req struct {
		Room string `json:"room" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	added, err := h.acl.AddRoom(ctx, req.Room)
	if err != nil {
		h.logger.Error("Failed to add room", slog.String("room", req.Room), slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to add room"})
		return
	}

	if !added {
		c.JSON(200, gin.H{"status": "ok", "message": "Room already exists"})
		return
	}

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Room added successfully",
	})

	h.activity.Log("room_add", "Room added to whitelist: "+req.Room, map[string]any{"room": req.Room})
}

// RemoveRoom removes a room from the whitelist
func (h *AdminHandler) RemoveRoom(c *gin.Context) {
	if h.acl == nil {
		c.JSON(503, gin.H{"error": "ACL service not available"})
		return
	}

	var req struct {
		Room string `json:"room" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	_, err := h.acl.RemoveRoom(ctx, req.Room)
	if err != nil {
		h.logger.Error("Failed to remove room", slog.String("room", req.Room), slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to remove room"})
		return
	}

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Room removed successfully",
	})

	h.activity.Log("room_remove", "Room removed from whitelist: "+req.Room, map[string]any{"room": req.Room})
}

// SetACL enables or disables room ACL
func (h *AdminHandler) SetACL(c *gin.Context) {
	if h.acl == nil {
		c.JSON(503, gin.H{"error": "ACL service not available"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.acl.SetEnabled(ctx, req.Enabled); err != nil {
		h.logger.Error("Failed to set ACL", slog.Bool("enabled", req.Enabled), slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to set ACL"})
		return
	}

	h.logger.Info("Room ACL updated", slog.Bool("enabled", req.Enabled))

	h.activity.Log("acl_update", fmt.Sprintf("Room ACL state changed to %v", req.Enabled), map[string]any{"enabled": req.Enabled})
	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "ACL setting updated successfully",
		"enabled": req.Enabled,
	})
}

// GetStats returns bot statistics (parallel fetch for performance)
func (h *AdminHandler) GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	var (
		members   []*domain.Member
		alarmKeys []*notification.AlarmEntry
		wg        sync.WaitGroup
	)

	// 병렬로 데이터 조회
	wg.Add(2)
	go func() {
		defer wg.Done()
		members, _ = h.repo.GetAllMembers(ctx)
	}()
	go func() {
		defer wg.Done()
		alarmKeys, _ = h.alarm.GetAllAlarmKeys(ctx)
	}()
	wg.Wait()

	uptime := time.Since(h.startTime).Truncate(time.Second).String()

	// ACL 서비스에서 rooms 수 조회
	var roomCount int
	if h.acl != nil {
		_, rooms := h.acl.GetACLStatus()
		roomCount = len(rooms)
	}

	c.JSON(200, gin.H{
		"status":  "ok",
		"members": len(members),
		"alarms":  len(alarmKeys),
		"rooms":   roomCount,
		"version": "v1.1.0-go",
		"uptime":  uptime,
	})
}

// SetRoomName sets a display name for a room ID
func (h *AdminHandler) SetRoomName(c *gin.Context) {
	var req struct {
		RoomID   string `json:"roomId" binding:"required"`
		RoomName string `json:"roomName" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	// Save to Valkey via AlarmService
	if err := h.alarm.SetRoomName(ctx, req.RoomID, req.RoomName); err != nil {
		h.logger.Error("Failed to set room name", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to set room name"})
		return
	}

	h.logger.Info("Room name set",
		slog.String("room_id", req.RoomID),
		slog.String("room_name", req.RoomName),
	)

	h.activity.Log("name_update", fmt.Sprintf("Room name set: %s -> %s", req.RoomID, req.RoomName), map[string]any{
		"room_id":   req.RoomID,
		"room_name": req.RoomName,
	})

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "Room name set successfully",
	})
}

// SetUserName sets a display name for a user ID
func (h *AdminHandler) SetUserName(c *gin.Context) {
	var req struct {
		UserID   string `json:"userId" binding:"required"`
		UserName string `json:"userName" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.RequestTimeout.AdminRequest)
	defer cancel()

	// Save to Valkey via AlarmService
	if err := h.alarm.SetUserName(ctx, req.UserID, req.UserName); err != nil {
		h.logger.Error("Failed to set user name", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to set user name"})
		return
	}

	h.logger.Info("User name set",
		slog.String("user_id", req.UserID),
		slog.String("user_name", req.UserName),
	)

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "User name set successfully",
	})
}

// GetLiveStreams returns currently live streams
func (h *AdminHandler) GetLiveStreams(c *gin.Context) {
	ctx := c.Request.Context()
	streams, err := h.holodex.GetLiveStreams(ctx)
	if err != nil {
		h.logger.Error("Failed to get live streams", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get live streams"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "streams": streams})
}

// GetUpcomingStreams returns upcoming streams
func (h *AdminHandler) GetUpcomingStreams(c *gin.Context) {
	ctx := c.Request.Context()
	streams, err := h.holodex.GetUpcomingStreams(ctx, 24)
	if err != nil {
		h.logger.Error("Failed to get upcoming streams", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get upcoming streams"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "streams": streams})
}

const channelStatsCacheKey = "admin:channel_stats"
const channelStatsCacheTTL = 10 * time.Minute

// GetChannelStats returns channel statistics (cached for 10 minutes)
func (h *AdminHandler) GetChannelStats(c *gin.Context) {
	ctx := c.Request.Context()
	if h.youtube == nil {
		c.JSON(503, gin.H{"error": "YouTube service not available"})
		return
	}

	// 캐시 확인
	if h.valkeyCache != nil {
		var cachedStats map[string]*youtube.ChannelStats
		if err := h.valkeyCache.Get(ctx, channelStatsCacheKey, &cachedStats); err == nil && cachedStats != nil {
			h.logger.Debug("Channel stats cache hit", slog.Int("count", len(cachedStats)))
			c.JSON(200, gin.H{"status": "ok", "stats": cachedStats})
			return
		}
	}

	members, err := h.repo.GetAllMembers(ctx)
	if err != nil {
		h.logger.Error("Failed to get members", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get members"})
		return
	}

	var channelIDs []string
	for _, m := range members {
		if m.ChannelID != "" && !m.IsGraduated {
			channelIDs = append(channelIDs, m.ChannelID)
		}
	}

	stats, err := h.youtube.GetChannelStatistics(ctx, channelIDs)
	if err != nil {
		h.logger.Error("Failed to get channel stats", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get channel stats"})
		return
	}

	// 캐시 저장
	if h.valkeyCache != nil && stats != nil {
		if err := h.valkeyCache.Set(ctx, channelStatsCacheKey, stats, channelStatsCacheTTL); err != nil {
			h.logger.Warn("Failed to cache channel stats", slog.Any("error", err))
		}
	}

	c.JSON(200, gin.H{"status": "ok", "stats": stats})
}

// AddMember adds a new member
func (h *AdminHandler) AddMember(c *gin.Context) {
	var req domain.Member
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.repo.CreateMember(ctx, &req); err != nil {
		h.logger.Error("Failed to add member", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to add member"})
		return
	}

	if err := h.memberCache.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh member cache", slog.Any("error", err))
	}

	h.activity.Log("member_add", "Member added: "+req.Name, map[string]any{"name": req.Name})

	c.JSON(200, gin.H{"status": "ok", "message": "Member added successfully"})
}

// GetLogs returns activity logs
func (h *AdminHandler) GetLogs(c *gin.Context) {
	logs, err := h.activity.GetRecentLogs(100)
	if err != nil {
		h.logger.Error("Failed to get logs", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to get logs"})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "logs": logs})
}

// GetSettings returns current settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	s := h.settings.Get()
	c.JSON(200, gin.H{"status": "ok", "settings": s})
}

// UpdateSettings updates settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req settings.Settings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := h.settings.Update(req); err != nil {
		h.logger.Error("Failed to update settings", slog.Any("error", err))
		c.JSON(500, gin.H{"error": "Failed to update settings"})
		return
	}

	h.activity.Log("settings_update", "Settings updated", nil)

	c.JSON(200, gin.H{"status": "ok", "message": "Settings updated"})
}
