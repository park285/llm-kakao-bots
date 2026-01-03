package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"

	commonhttputil "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httputil"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	qrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
)

// Admin API 에러 코드
const (
	adminErrorInvalidRequest  = "INVALID_REQUEST"
	adminErrorSessionNotFound = "SESSION_NOT_FOUND"
	adminErrorGameNotFound    = "GAME_NOT_FOUND"
	adminErrorInternalError   = "INTERNAL_ERROR"
)

// AdminStatsResponse: 통합 통계 응답 DTO
type AdminStatsResponse struct {
	TotalGamesPlayed    int     `json:"totalGamesPlayed"`
	TotalGamesCompleted int     `json:"totalGamesCompleted"`
	TotalSurrenders     int     `json:"totalSurrenders"`
	SuccessRate         float64 `json:"successRate"`
	ActiveSessions      int     `json:"activeSessions"`
	TotalParticipants   int     `json:"totalParticipants"`
	Last24HoursGames    int     `json:"last24HoursGames"`
}

// ActiveSessionResponse: 활성 세션 조회 응답 DTO
type ActiveSessionResponse struct {
	ChatID     string `json:"chatId"`
	Category   string `json:"category"`
	Target     string `json:"target"`
	TTLSeconds int64  `json:"ttlSeconds"`
}

// GameHistoryResponse: 게임 히스토리 조회 응답 DTO
type GameHistoryResponse struct {
	SessionID        string    `json:"sessionId"`
	ChatID           string    `json:"chatId"`
	Category         string    `json:"category"`
	Target           string    `json:"target"`
	Result           string    `json:"result"`
	ParticipantCount int       `json:"participantCount"`
	QuestionCount    int       `json:"questionCount"`
	HintCount        int       `json:"hintCount"`
	CompletedAt      time.Time `json:"completedAt"`
}

// LeaderboardEntry: 리더보드 항목 DTO
type LeaderboardEntry struct {
	Rank                int     `json:"rank"`
	UserID              string  `json:"userId"`
	ChatID              string  `json:"chatId"`
	TotalGamesCompleted int     `json:"totalGamesCompleted"`
	SuccessRate         float64 `json:"successRate"`
	BestQuestionCount   *int    `json:"bestQuestionCount,omitempty"`
	BestTarget          *string `json:"bestTarget,omitempty"`
}

// SynonymRequest: 동의어 매핑 요청 DTO
type SynonymRequest struct {
	Canonical string   `json:"canonical"` // 표준 단어
	Aliases   []string `json:"aliases"`   // 별칭 목록
}

// AuditRequest: 판정 리뷰 요청 DTO
type AuditRequest struct {
	QuestionIndex int    `json:"questionIndex"`
	Verdict       string `json:"verdict"` // AI_CORRECT, AI_WRONG, UNCLEAR
	Reason        string `json:"reason"`
	AdminUserID   string `json:"adminUserId"`
}

// RefundRequest: 스탯 복원 요청 DTO
type RefundRequest struct {
	UserID       string `json:"userId"`
	RestoreStats bool   `json:"restoreStats"`
	AdminUserID  string `json:"adminUserId"`
	Reason       string `json:"reason"`
}

// AdminDeps: Admin API 핸들러 의존성
type AdminDeps struct {
	DB           *gorm.DB
	ValkeyClient valkey.Client
	SessionStore *qredis.SessionStore
	Logger       *slog.Logger
}

// RegisterAdminRoutes: Admin API 라우트 등록
func RegisterAdminRoutes(mux *http.ServeMux, deps AdminDeps) {
	// Phase 1: 기존 API
	mux.HandleFunc("GET /admin/stats", func(w http.ResponseWriter, r *http.Request) {
		handleAdminStats(w, r, deps)
	})
	mux.HandleFunc("GET /admin/sessions", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSessions(w, r, deps)
	})
	mux.HandleFunc("GET /admin/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSessionDetail(w, r, deps)
	})
	mux.HandleFunc("DELETE /admin/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSessionDelete(w, r, deps)
	})
	mux.HandleFunc("POST /admin/sessions/{id}/hint", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSessionHint(w, r, deps)
	})
	mux.HandleFunc("POST /admin/sessions/cleanup", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSessionsCleanup(w, r, deps)
	})
	mux.HandleFunc("GET /admin/games", func(w http.ResponseWriter, r *http.Request) {
		handleAdminGames(w, r, deps)
	})
	mux.HandleFunc("GET /admin/games/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleAdminGameDetail(w, r, deps)
	})
	mux.HandleFunc("GET /admin/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		handleAdminLeaderboard(w, r, deps)
	})

	// Phase 3: CMS API
	mux.HandleFunc("POST /admin/synonyms", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSynonymCreate(w, r, deps)
	})
	mux.HandleFunc("GET /admin/synonyms", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSynonymSearch(w, r, deps)
	})
	mux.HandleFunc("DELETE /admin/synonyms/{alias}", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSynonymDelete(w, r, deps)
	})
	mux.HandleFunc("POST /admin/games/{id}/audit", func(w http.ResponseWriter, r *http.Request) {
		handleAdminGameAudit(w, r, deps)
	})
	mux.HandleFunc("POST /admin/games/{id}/refund", func(w http.ResponseWriter, r *http.Request) {
		handleAdminGameRefund(w, r, deps)
	})

	// Phase 4: 추가 통계/관리 API
	mux.HandleFunc("GET /admin/stats/categories", func(w http.ResponseWriter, r *http.Request) {
		handleAdminCategoryStats(w, r, deps)
	})
	mux.HandleFunc("GET /admin/nicknames", func(w http.ResponseWriter, r *http.Request) {
		handleAdminNicknames(w, r, deps)
	})

	// Phase 5: 유저 통계 + 로그 관리
	mux.HandleFunc("GET /admin/users/stats", func(w http.ResponseWriter, r *http.Request) {
		handleAdminUserStatsList(w, r, deps)
	})
	mux.HandleFunc("GET /admin/users/{id}/stats", func(w http.ResponseWriter, r *http.Request) {
		handleAdminUserStatsGet(w, r, deps)
	})
	mux.HandleFunc("DELETE /admin/users/{id}/stats", func(w http.ResponseWriter, r *http.Request) {
		handleAdminUserStatsReset(w, r, deps)
	})
	mux.HandleFunc("GET /admin/audits", func(w http.ResponseWriter, r *http.Request) {
		handleAdminAuditLogs(w, r, deps)
	})
	mux.HandleFunc("GET /admin/refunds", func(w http.ResponseWriter, r *http.Request) {
		handleAdminRefundLogs(w, r, deps)
	})

	deps.Logger.Info("twentyq_admin_api_registered", "routes", 22)
}

// handleAdminStats: 통합 통계 조회
func handleAdminStats(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	start := time.Now()
	deps.Logger.Info("ADMIN_STATS_REQUEST")

	var stats struct {
		TotalPlayed    int64
		TotalCompleted int64
		TotalSurrender int64
		Last24Hours    int64
	}

	deps.DB.WithContext(ctx).Model(&qrepo.GameSession{}).Count(&stats.TotalPlayed)
	deps.DB.WithContext(ctx).Model(&qrepo.GameSession{}).Where("result = ?", "success").Count(&stats.TotalCompleted)
	deps.DB.WithContext(ctx).Model(&qrepo.GameSession{}).Where("result = ?", "surrender").Count(&stats.TotalSurrender)

	since := time.Now().Add(-24 * time.Hour)
	deps.DB.WithContext(ctx).Model(&qrepo.GameSession{}).Where("completed_at > ?", since).Count(&stats.Last24Hours)

	var totalParticipants int64
	deps.DB.WithContext(ctx).Model(&qrepo.UserStats{}).Count(&totalParticipants)

	activeSessions := countActiveSessions(ctx, deps)

	var successRate float64
	if stats.TotalPlayed > 0 {
		successRate = float64(stats.TotalCompleted) / float64(stats.TotalPlayed) * 100
	}

	response := AdminStatsResponse{
		TotalGamesPlayed:    int(stats.TotalPlayed),
		TotalGamesCompleted: int(stats.TotalCompleted),
		TotalSurrenders:     int(stats.TotalSurrender),
		SuccessRate:         successRate,
		ActiveSessions:      activeSessions,
		TotalParticipants:   int(totalParticipants),
		Last24HoursGames:    int(stats.Last24Hours),
	}

	deps.Logger.Info("ADMIN_STATS_SUCCESS", "duration", time.Since(start).Milliseconds())
	_ = commonhttputil.WriteJSON(w, http.StatusOK, response)
}

// handleAdminSessions: 활성 세션 목록 조회 (Valkey SCAN)
func handleAdminSessions(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	start := time.Now()
	deps.Logger.Info("ADMIN_SESSIONS_REQUEST")

	sessions := listActiveSessions(ctx, deps)

	deps.Logger.Info("ADMIN_SESSIONS_SUCCESS", "count", len(sessions), "duration", time.Since(start).Milliseconds())
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// handleAdminSessionDelete: 세션 강제 종료
func handleAdminSessionDelete(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	chatID := r.PathValue("id")
	if chatID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "session id is required")
		return
	}

	deps.Logger.Info("ADMIN_SESSION_DELETE_REQUEST", "chatId", chatID)

	exists, err := deps.SessionStore.Exists(ctx, chatID)
	if err != nil {
		deps.Logger.Error("ADMIN_SESSION_DELETE_CHECK_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to check session")
		return
	}
	if !exists {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorSessionNotFound, "session not found")
		return
	}

	if err := deps.SessionStore.ClearAllData(ctx, chatID); err != nil {
		deps.Logger.Error("ADMIN_SESSION_DELETE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to delete session")
		return
	}

	deps.Logger.Info("ADMIN_SESSION_DELETE_SUCCESS", "chatId", chatID)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "session deleted",
	})
}

// handleAdminGames: 게임 히스토리 조회
func handleAdminGames(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	start := time.Now()
	deps.Logger.Info("ADMIN_GAMES_REQUEST")

	query := r.URL.Query()
	limit := parseIntOrDefault(query.Get("limit"), 50)
	offset := parseIntOrDefault(query.Get("offset"), 0)
	category := query.Get("category")
	result := query.Get("result")

	if limit > 100 {
		limit = 100
	}

	db := deps.DB.WithContext(ctx).Model(&qrepo.GameSession{}).
		Order("completed_at DESC").
		Limit(limit).
		Offset(offset)

	if category != "" {
		db = db.Where("category = ?", category)
	}
	if result != "" {
		db = db.Where("result = ?", result)
	}

	var sessions []qrepo.GameSession
	if err := db.Find(&sessions).Error; err != nil {
		deps.Logger.Error("ADMIN_GAMES_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to fetch games")
		return
	}

	games := make([]GameHistoryResponse, 0, len(sessions))
	for _, s := range sessions {
		games = append(games, GameHistoryResponse{
			SessionID:        s.SessionID,
			ChatID:           s.ChatID,
			Category:         s.Category,
			Target:           s.Target,
			Result:           s.Result,
			ParticipantCount: s.ParticipantCount,
			QuestionCount:    s.QuestionCount,
			HintCount:        s.HintCount,
			CompletedAt:      s.CompletedAt,
		})
	}

	var total int64
	countDB := deps.DB.WithContext(ctx).Model(&qrepo.GameSession{})
	if category != "" {
		countDB = countDB.Where("category = ?", category)
	}
	if result != "" {
		countDB = countDB.Where("result = ?", result)
	}
	countDB.Count(&total)

	deps.Logger.Info("ADMIN_GAMES_SUCCESS", "count", len(games), "duration", time.Since(start).Milliseconds())
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"games":  games,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleAdminLeaderboard: 리더보드 조회
func handleAdminLeaderboard(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	start := time.Now()
	deps.Logger.Info("ADMIN_LEADERBOARD_REQUEST")

	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	var stats []qrepo.UserStats
	if err := deps.DB.WithContext(ctx).Model(&qrepo.UserStats{}).
		Where("total_games_completed > 0").
		Order("total_games_completed DESC, best_score_question_count ASC").
		Limit(limit).
		Find(&stats).Error; err != nil {
		deps.Logger.Error("ADMIN_LEADERBOARD_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to fetch leaderboard")
		return
	}

	entries := make([]LeaderboardEntry, 0, len(stats))
	for i, s := range stats {
		var successRate float64
		if s.TotalGamesStarted > 0 {
			successRate = float64(s.TotalGamesCompleted) / float64(s.TotalGamesStarted) * 100
		}
		entries = append(entries, LeaderboardEntry{
			Rank:                i + 1,
			UserID:              s.UserID,
			ChatID:              s.ChatID,
			TotalGamesCompleted: s.TotalGamesCompleted,
			SuccessRate:         successRate,
			BestQuestionCount:   s.BestScoreQuestionCnt,
			BestTarget:          s.BestScoreTarget,
		})
	}

	deps.Logger.Info("ADMIN_LEADERBOARD_SUCCESS", "count", len(entries), "duration", time.Since(start).Milliseconds())
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"leaderboard": entries,
		"count":       len(entries),
	})
}

// synonymKeyPrefix: 동의어 저장용 Valkey 키 접두사
const synonymKeyPrefix = "20q:synonyms"

// handleAdminSynonymCreate: 동의어 매핑 생성 (Valkey Hash)
func handleAdminSynonymCreate(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	deps.Logger.Info("ADMIN_SYNONYM_CREATE_REQUEST")

	var req SynonymRequest
	if err := commonhttputil.ReadJSON(r, &req, 4096); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "invalid request body")
		return
	}

	canonical := strings.TrimSpace(req.Canonical)
	if canonical == "" || len(req.Aliases) == 0 {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "canonical and aliases are required")
		return
	}

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "valkey client not available")
		return
	}

	// 각 alias를 canonical에 매핑 (HSET 20q:synonyms alias1 canonical ...)
	args := make([]string, 0, len(req.Aliases)*2)
	for _, alias := range req.Aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		args = append(args, alias, canonical)
	}

	if len(args) == 0 {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "no valid aliases provided")
		return
	}

	cmd := client.B().Hset().Key(synonymKeyPrefix).FieldValue().FieldValue(args[0], args[1])
	for i := 2; i < len(args); i += 2 {
		cmd = cmd.FieldValue(args[i], args[i+1])
	}
	if err := client.Do(ctx, cmd.Build()).Error(); err != nil {
		deps.Logger.Error("ADMIN_SYNONYM_CREATE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to create synonym")
		return
	}

	deps.Logger.Info("ADMIN_SYNONYM_CREATE_SUCCESS", "canonical", canonical, "aliasCount", len(req.Aliases))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"message":   "synonym created",
		"canonical": canonical,
		"aliases":   req.Aliases,
	})
}

// handleAdminSynonymSearch: 동의어 검색 (Valkey Hash)
func handleAdminSynonymSearch(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	deps.Logger.Info("ADMIN_SYNONYM_SEARCH_REQUEST", "query", query)

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "valkey client not available")
		return
	}

	// 특정 alias 조회
	if query != "" {
		cmd := client.B().Hget().Key(synonymKeyPrefix).Field(query).Build()
		canonical, err := client.Do(ctx, cmd).ToString()
		if err != nil {
			if valkeyx.IsNil(err) {
				_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
					"status": "ok",
					"result": nil,
				})
				return
			}
			deps.Logger.Error("ADMIN_SYNONYM_SEARCH_FAILED", "err", err)
			_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to search synonym")
			return
		}

		deps.Logger.Info("ADMIN_SYNONYM_SEARCH_SUCCESS", "alias", query, "canonical", canonical)
		_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"result": map[string]string{
				"alias":     query,
				"canonical": canonical,
			},
		})
		return
	}

	// 전체 목록 조회
	cmd := client.B().Hgetall().Key(synonymKeyPrefix).Build()
	result, err := client.Do(ctx, cmd).AsStrMap()
	if err != nil {
		deps.Logger.Error("ADMIN_SYNONYM_SEARCH_ALL_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to list synonyms")
		return
	}

	// canonical 기준으로 그룹화
	grouped := make(map[string][]string)
	for alias, canonical := range result {
		grouped[canonical] = append(grouped[canonical], alias)
	}

	synonyms := make([]map[string]any, 0, len(grouped))
	for canonical, aliases := range grouped {
		synonyms = append(synonyms, map[string]any{
			"canonical": canonical,
			"aliases":   aliases,
		})
	}

	deps.Logger.Info("ADMIN_SYNONYM_SEARCH_ALL_SUCCESS", "count", len(synonyms))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"synonyms": synonyms,
		"count":    len(synonyms),
	})
}

// handleAdminGameAudit: 판정 리뷰 기록
func handleAdminGameAudit(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	sessionID := r.PathValue("id")
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "game session id is required")
		return
	}

	deps.Logger.Info("ADMIN_GAME_AUDIT_REQUEST", "sessionId", sessionID)

	var req AuditRequest
	if err := commonhttputil.ReadJSON(r, &req, 4096); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "invalid request body")
		return
	}

	// 게임 세션 존재 확인
	var session qrepo.GameSession
	if err := deps.DB.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorGameNotFound, "game session not found")
			return
		}
		deps.Logger.Error("ADMIN_GAME_AUDIT_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to find game session")
		return
	}

	// 오디트 로그 저장
	audit := qrepo.AuditLog{
		SessionID:     sessionID,
		QuestionIndex: req.QuestionIndex,
		Verdict:       req.Verdict,
		Reason:        req.Reason,
		AdminUserID:   req.AdminUserID,
		CreatedAt:     time.Now(),
	}

	if err := deps.DB.WithContext(ctx).Create(&audit).Error; err != nil {
		deps.Logger.Error("ADMIN_GAME_AUDIT_SAVE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to save audit log")
		return
	}

	deps.Logger.Info("ADMIN_GAME_AUDIT_SUCCESS", "sessionId", sessionID, "verdict", req.Verdict)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "audit recorded",
		"id":      audit.ID,
	})
}

// handleAdminGameRefund: 유저 스탯 복원
func handleAdminGameRefund(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	sessionID := r.PathValue("id")
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "game session id is required")
		return
	}

	deps.Logger.Info("ADMIN_GAME_REFUND_REQUEST", "sessionId", sessionID)

	var req RefundRequest
	if err := commonhttputil.ReadJSON(r, &req, 4096); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "userId is required")
		return
	}

	// 게임 세션 조회
	var session qrepo.GameSession
	if err := deps.DB.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorGameNotFound, "game session not found")
			return
		}
		deps.Logger.Error("ADMIN_GAME_REFUND_SESSION_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to find game session")
		return
	}

	// 유저 스탯 조회
	var userStats qrepo.UserStats
	if err := deps.DB.WithContext(ctx).Where("user_id = ? AND chat_id = ?", req.UserID, session.ChatID).First(&userStats).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorSessionNotFound, "user stats not found")
			return
		}
		deps.Logger.Error("ADMIN_GAME_REFUND_STATS_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to find user stats")
		return
	}

	// 스탯 복원: 항복으로 끝난 게임이면 completed +1, surrenders -1
	if req.RestoreStats && session.Result == "surrender" {
		userStats.TotalGamesCompleted++
		userStats.TotalSurrenders--
		if userStats.TotalSurrenders < 0 {
			userStats.TotalSurrenders = 0
		}

		if err := deps.DB.WithContext(ctx).Save(&userStats).Error; err != nil {
			deps.Logger.Error("ADMIN_GAME_REFUND_STATS_UPDATE_FAILED", "err", err)
			_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to update user stats")
			return
		}
	}

	// 리펀드 로그 저장
	refundLog := qrepo.RefundLog{
		SessionID:   sessionID,
		UserID:      req.UserID,
		AdminUserID: req.AdminUserID,
		Reason:      req.Reason,
		CreatedAt:   time.Now(),
	}
	if err := deps.DB.WithContext(ctx).Create(&refundLog).Error; err != nil {
		deps.Logger.Warn("ADMIN_GAME_REFUND_LOG_SAVE_FAILED", "err", err)
	}

	deps.Logger.Info("ADMIN_GAME_REFUND_SUCCESS", "sessionId", sessionID, "userId", req.UserID)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "refund applied",
	})
}

// ============================================================================
// Phase 2: Valkey SCAN 구현
// ============================================================================

// countActiveSessions: Valkey SCAN으로 활성 세션 수 조회
func countActiveSessions(ctx context.Context, deps AdminDeps) int {
	client := deps.ValkeyClient
	if client == nil {
		return 0
	}

	pattern := qconfig.RedisKeySessionPrefix + ":*"
	count := 0
	cursor := uint64(0)

	for {
		cmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		result, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			deps.Logger.Error("ADMIN_COUNT_SESSIONS_SCAN_FAILED", "err", err)
			break
		}

		count += len(result.Elements)
		cursor = result.Cursor
		if cursor == 0 {
			break
		}
	}

	return count
}

// listActiveSessions: Valkey SCAN으로 활성 세션 목록 조회
func listActiveSessions(ctx context.Context, deps AdminDeps) []ActiveSessionResponse {
	client := deps.ValkeyClient
	if client == nil {
		return []ActiveSessionResponse{}
	}

	pattern := qconfig.RedisKeySessionPrefix + ":*"
	var keys []string
	cursor := uint64(0)

	for {
		cmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		result, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			deps.Logger.Error("ADMIN_LIST_SESSIONS_SCAN_FAILED", "err", err)
			break
		}

		keys = append(keys, result.Elements...)
		cursor = result.Cursor
		if cursor == 0 {
			break
		}
	}

	sessions := make([]ActiveSessionResponse, 0, len(keys))
	for _, key := range keys {
		// 키에서 chatID 추출: 20q:riddle:session:{chatID}
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			continue
		}
		chatID := parts[len(parts)-1]

		// 세션 데이터 및 TTL 조회
		getCmd := client.B().Get().Key(key).Build()
		raw, err := client.Do(ctx, getCmd).AsBytes()
		if err != nil {
			if !valkeyx.IsNil(err) {
				deps.Logger.Warn("ADMIN_LIST_SESSIONS_GET_FAILED", "key", key, "err", err)
			}
			continue
		}

		var sessionData struct {
			Target   string `json:"target"`
			Category string `json:"category"`
		}
		if err := json.Unmarshal(raw, &sessionData); err != nil {
			deps.Logger.Warn("ADMIN_LIST_SESSIONS_UNMARSHAL_FAILED", "key", key, "err", err)
			continue
		}

		// TTL 조회
		ttlCmd := client.B().Ttl().Key(key).Build()
		ttl, err := client.Do(ctx, ttlCmd).AsInt64()
		if err != nil {
			ttl = -1
		}

		sessions = append(sessions, ActiveSessionResponse{
			ChatID:     chatID,
			Category:   sessionData.Category,
			Target:     sessionData.Target,
			TTLSeconds: ttl,
		})
	}

	return sessions
}

// parseIntOrDefault: 문자열을 정수로 변환, 실패 시 기본값 반환
func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// handleAdminGameDetail: 단일 게임 상세 조회 (세션 + 참여자 로그)
func handleAdminGameDetail(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	sessionID := r.PathValue("id")
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "session id is required")
		return
	}

	deps.Logger.Info("ADMIN_GAME_DETAIL_REQUEST", "sessionId", sessionID)

	// 게임 세션 조회
	var session qrepo.GameSession
	if err := deps.DB.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorGameNotFound, "game session not found")
			return
		}
		deps.Logger.Error("ADMIN_GAME_DETAIL_SESSION_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query session")
		return
	}

	// 참여자 로그 조회
	var logs []qrepo.GameLog
	if err := deps.DB.WithContext(ctx).Where("chat_id = ? AND completed_at = ?", session.ChatID, session.CompletedAt).Find(&logs).Error; err != nil {
		deps.Logger.Error("ADMIN_GAME_DETAIL_LOGS_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query logs")
		return
	}

	// 오디트 로그 조회
	var audits []qrepo.AuditLog
	_ = deps.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at DESC").Find(&audits)

	// 리펀드 로그 조회
	var refunds []qrepo.RefundLog
	_ = deps.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at DESC").Find(&refunds)

	deps.Logger.Info("ADMIN_GAME_DETAIL_SUCCESS", "sessionId", sessionID, "logCount", len(logs))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"session": map[string]any{
			"sessionId":        session.SessionID,
			"chatId":           session.ChatID,
			"category":         session.Category,
			"target":           session.Target,
			"result":           session.Result,
			"participantCount": session.ParticipantCount,
			"questionCount":    session.QuestionCount,
			"hintCount":        session.HintCount,
			"completedAt":      session.CompletedAt,
		},
		"logs":    logs,
		"audits":  audits,
		"refunds": refunds,
	})
}

// handleAdminSynonymDelete: 동의어 삭제
func handleAdminSynonymDelete(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	alias := r.PathValue("alias")
	if alias == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "alias is required")
		return
	}

	alias = strings.TrimSpace(alias)
	deps.Logger.Info("ADMIN_SYNONYM_DELETE_REQUEST", "alias", alias)

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "valkey client not available")
		return
	}

	// 삭제 전 존재 여부 확인
	getCmd := client.B().Hget().Key(synonymKeyPrefix).Field(alias).Build()
	canonical, err := client.Do(ctx, getCmd).ToString()
	if err != nil {
		if valkeyx.IsNil(err) {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, "SYNONYM_NOT_FOUND", "alias not found")
			return
		}
		deps.Logger.Error("ADMIN_SYNONYM_DELETE_CHECK_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to check synonym")
		return
	}

	// 삭제
	delCmd := client.B().Hdel().Key(synonymKeyPrefix).Field(alias).Build()
	if err := client.Do(ctx, delCmd).Error(); err != nil {
		deps.Logger.Error("ADMIN_SYNONYM_DELETE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to delete synonym")
		return
	}

	deps.Logger.Info("ADMIN_SYNONYM_DELETE_SUCCESS", "alias", alias, "canonical", canonical)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"message":   "synonym deleted",
		"alias":     alias,
		"canonical": canonical,
	})
}

// CategoryStatsResponse: 카테고리별 통계 응답
type CategoryStatsResponse struct {
	Category      string  `json:"category"`
	TotalGames    int     `json:"totalGames"`
	SuccessCount  int     `json:"successCount"`
	SurrenderRate float64 `json:"surrenderRate"`
}

// handleAdminCategoryStats: 카테고리별 통계 조회
func handleAdminCategoryStats(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	deps.Logger.Info("ADMIN_CATEGORY_STATS_REQUEST")

	// 카테고리별 집계 쿼리
	type categoryAggregate struct {
		Category     string `gorm:"column:category"`
		TotalGames   int    `gorm:"column:total_games"`
		SuccessCount int    `gorm:"column:success_count"`
	}

	var results []categoryAggregate
	if err := deps.DB.WithContext(ctx).
		Model(&qrepo.GameSession{}).
		Select("category, count(*) as total_games, sum(case when result = 'correct' then 1 else 0 end) as success_count").
		Group("category").
		Order("total_games DESC").
		Scan(&results).Error; err != nil {
		deps.Logger.Error("ADMIN_CATEGORY_STATS_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query category stats")
		return
	}

	stats := make([]CategoryStatsResponse, 0, len(results))
	for _, r := range results {
		surrenderRate := float64(0)
		if r.TotalGames > 0 {
			surrenderRate = float64(r.TotalGames-r.SuccessCount) / float64(r.TotalGames) * 100
		}
		stats = append(stats, CategoryStatsResponse{
			Category:      r.Category,
			TotalGames:    r.TotalGames,
			SuccessCount:  r.SuccessCount,
			SurrenderRate: surrenderRate,
		})
	}

	deps.Logger.Info("ADMIN_CATEGORY_STATS_SUCCESS", "count", len(stats))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"categories": stats,
		"count":      len(stats),
	})
}

// NicknameMapResponse: 닉네임 매핑 응답
type NicknameMapResponse struct {
	UserID     string    `json:"userId"`
	LastSender string    `json:"lastSender"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// handleAdminNicknames: 닉네임 매핑 조회
func handleAdminNicknames(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	chatID := r.URL.Query().Get("chatId")
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 100)
	if limit > 500 {
		limit = 500
	}

	deps.Logger.Info("ADMIN_NICKNAMES_REQUEST", "chatId", chatID, "limit", limit)

	query := deps.DB.WithContext(ctx).Model(&qrepo.UserNicknameMap{}).Order("last_seen_at DESC").Limit(limit)
	if chatID != "" {
		query = query.Where("chat_id = ?", chatID)
	}

	var mappings []qrepo.UserNicknameMap
	if err := query.Find(&mappings).Error; err != nil {
		deps.Logger.Error("ADMIN_NICKNAMES_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query nicknames")
		return
	}

	result := make([]NicknameMapResponse, 0, len(mappings))
	for _, m := range mappings {
		result = append(result, NicknameMapResponse{
			UserID:     m.UserID,
			LastSender: m.LastSender,
			LastSeenAt: m.LastSeenAt,
		})
	}

	deps.Logger.Info("ADMIN_NICKNAMES_SUCCESS", "count", len(result))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"nicknames": result,
		"count":     len(result),
	})
}

// handleAdminSessionDetail: 진행 중인 세션 상세 조회 (Q&A 히스토리 포함)
func handleAdminSessionDetail(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	chatID := r.PathValue("id")
	if chatID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "chat id is required")
		return
	}

	deps.Logger.Info("ADMIN_SESSION_DETAIL_REQUEST", "chatId", chatID)

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "valkey client not available")
		return
	}

	// 세션 데이터 조회
	sessionKey := qconfig.RedisKeySessionPrefix + ":" + chatID
	sessionCmd := client.B().Get().Key(sessionKey).Build()
	sessionRaw, err := client.Do(ctx, sessionCmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorSessionNotFound, "session not found")
			return
		}
		deps.Logger.Error("ADMIN_SESSION_DETAIL_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query session")
		return
	}

	var sessionData struct {
		Target   string `json:"target"`
		Category string `json:"category"`
		Intro    string `json:"intro"`
	}
	if err := json.Unmarshal(sessionRaw, &sessionData); err != nil {
		deps.Logger.Error("ADMIN_SESSION_DETAIL_UNMARSHAL_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to parse session")
		return
	}

	// 히스토리 조회
	historyKey := qconfig.RedisKeyHistoryPrefix + ":" + chatID
	historyCmd := client.B().Get().Key(historyKey).Build()
	historyRaw, _ := client.Do(ctx, historyCmd).AsBytes()
	var history []map[string]any
	if historyRaw != nil {
		_ = json.Unmarshal(historyRaw, &history)
	}

	// 힌트 횟수 조회
	hintKey := qconfig.RedisKeyHints + ":" + chatID
	hintCmd := client.B().Get().Key(hintKey).Build()
	hintCount, _ := client.Do(ctx, hintCmd).AsInt64()

	// 플레이어 목록 조회
	playersKey := qconfig.RedisKeyPlayers + ":" + chatID
	playersCmd := client.B().Get().Key(playersKey).Build()
	playersRaw, _ := client.Do(ctx, playersCmd).AsBytes()
	var players []map[string]string
	if playersRaw != nil {
		_ = json.Unmarshal(playersRaw, &players)
	}

	// TTL 조회
	ttlCmd := client.B().Ttl().Key(sessionKey).Build()
	ttl, _ := client.Do(ctx, ttlCmd).AsInt64()

	deps.Logger.Info("ADMIN_SESSION_DETAIL_SUCCESS", "chatId", chatID, "questionCount", len(history))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"session": map[string]any{
			"chatId":        chatID,
			"target":        sessionData.Target,
			"category":      sessionData.Category,
			"intro":         sessionData.Intro,
			"questionCount": len(history),
			"hintCount":     hintCount,
			"ttlSeconds":    ttl,
		},
		"history": history,
		"players": players,
	})
}

// HintInjectRequest: 힌트 주입 요청
type HintInjectRequest struct {
	Message string `json:"message"`
}

// handleAdminSessionHint: 진행 중인 게임에 힌트 주입
func handleAdminSessionHint(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	chatID := r.PathValue("id")
	if chatID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "chat id is required")
		return
	}

	var req HintInjectRequest
	if err := commonhttputil.ReadJSON(r, &req, 4096); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "message is required")
		return
	}

	deps.Logger.Info("ADMIN_SESSION_HINT_REQUEST", "chatId", chatID)

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "valkey client not available")
		return
	}

	// 세션 존재 확인
	sessionKey := qconfig.RedisKeySessionPrefix + ":" + chatID
	existsCmd := client.B().Exists().Key(sessionKey).Build()
	exists, err := client.Do(ctx, existsCmd).AsInt64()
	if err != nil || exists == 0 {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, adminErrorSessionNotFound, "session not found")
		return
	}

	// 히스토리에 GM 힌트 추가
	historyKey := qconfig.RedisKeyHistoryPrefix + ":" + chatID
	historyCmd := client.B().Get().Key(historyKey).Build()
	historyRaw, _ := client.Do(ctx, historyCmd).AsBytes()
	var history []map[string]any
	if historyRaw != nil {
		_ = json.Unmarshal(historyRaw, &history)
	}

	history = append(history, map[string]any{
		"questionNumber": len(history) + 1,
		"question":       "[GM 힌트]",
		"answer":         req.Message,
		"isChain":        false,
	})

	newHistory, _ := json.Marshal(history)
	setCmd := client.B().Set().Key(historyKey).Value(string(newHistory)).Build()
	if err := client.Do(ctx, setCmd).Error(); err != nil {
		deps.Logger.Error("ADMIN_SESSION_HINT_SAVE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to save hint")
		return
	}

	deps.Logger.Info("ADMIN_SESSION_HINT_SUCCESS", "chatId", chatID, "message", req.Message)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "hint injected",
	})
}

// CleanupRequest: 세션 정리 요청
type CleanupRequest struct {
	OlderThanHours int `json:"olderThanHours"`
}

// handleAdminSessionsCleanup: 오래된 세션 정리
func handleAdminSessionsCleanup(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()

	var req CleanupRequest
	if err := commonhttputil.ReadJSON(r, &req, 1024); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "invalid request body")
		return
	}
	if req.OlderThanHours <= 0 {
		req.OlderThanHours = 24
	}

	deps.Logger.Info("ADMIN_SESSIONS_CLEANUP_REQUEST", "olderThanHours", req.OlderThanHours)

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "valkey client not available")
		return
	}

	pattern := qconfig.RedisKeySessionPrefix + ":*"
	var cursor uint64
	var deletedCount int

	for {
		scanCmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		result, err := client.Do(ctx, scanCmd).AsScanEntry()
		if err != nil {
			break
		}

		for _, key := range result.Elements {
			ttlCmd := client.B().Ttl().Key(key).Build()
			ttl, _ := client.Do(ctx, ttlCmd).AsInt64()

			// TTL이 설정된 키 중 남은 시간이 적은 것만 삭제
			maxTTL := int64(qconfig.RedisSessionTTLSeconds)
			elapsed := maxTTL - ttl
			if elapsed > int64(req.OlderThanHours*3600) {
				delCmd := client.B().Del().Key(key).Build()
				if err := client.Do(ctx, delCmd).Error(); err == nil {
					deletedCount++
				}
			}
		}

		cursor = result.Cursor
		if cursor == 0 {
			break
		}
	}

	deps.Logger.Info("ADMIN_SESSIONS_CLEANUP_SUCCESS", "deletedCount", deletedCount)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"deletedCount": deletedCount,
	})
}

// handleAdminUserStatsList: 유저 통계 목록 조회
func handleAdminUserStatsList(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	chatID := r.URL.Query().Get("chatId")
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	deps.Logger.Info("ADMIN_USER_STATS_LIST_REQUEST", "chatId", chatID, "limit", limit)

	query := deps.DB.WithContext(ctx).Model(&qrepo.UserStats{}).Order("total_games_completed DESC").Limit(limit).Offset(offset)
	if chatID != "" {
		query = query.Where("chat_id = ?", chatID)
	}

	var stats []qrepo.UserStats
	if err := query.Find(&stats).Error; err != nil {
		deps.Logger.Error("ADMIN_USER_STATS_LIST_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query user stats")
		return
	}

	var total int64
	deps.DB.WithContext(ctx).Model(&qrepo.UserStats{}).Count(&total)

	deps.Logger.Info("ADMIN_USER_STATS_LIST_SUCCESS", "count", len(stats))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"stats":  stats,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleAdminUserStatsGet: 특정 유저 통계 조회
func handleAdminUserStatsGet(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	userID := r.PathValue("id")
	chatID := r.URL.Query().Get("chatId")
	if userID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "user id is required")
		return
	}

	deps.Logger.Info("ADMIN_USER_STATS_GET_REQUEST", "userId", userID, "chatId", chatID)

	query := deps.DB.WithContext(ctx).Model(&qrepo.UserStats{}).Where("user_id = ?", userID)
	if chatID != "" {
		query = query.Where("chat_id = ?", chatID)
	}

	var stats []qrepo.UserStats
	if err := query.Find(&stats).Error; err != nil {
		deps.Logger.Error("ADMIN_USER_STATS_GET_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query user stats")
		return
	}

	deps.Logger.Info("ADMIN_USER_STATS_GET_SUCCESS", "userId", userID, "count", len(stats))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"stats":  stats,
	})
}

// handleAdminUserStatsReset: 유저 통계 리셋
func handleAdminUserStatsReset(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	userID := r.PathValue("id")
	chatID := r.URL.Query().Get("chatId")
	if userID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, adminErrorInvalidRequest, "user id is required")
		return
	}

	deps.Logger.Info("ADMIN_USER_STATS_RESET_REQUEST", "userId", userID, "chatId", chatID)

	query := deps.DB.WithContext(ctx).Where("user_id = ?", userID)
	if chatID != "" {
		query = query.Where("chat_id = ?", chatID)
	}

	result := query.Delete(&qrepo.UserStats{})
	if result.Error != nil {
		deps.Logger.Error("ADMIN_USER_STATS_RESET_FAILED", "err", result.Error)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to reset user stats")
		return
	}

	deps.Logger.Info("ADMIN_USER_STATS_RESET_SUCCESS", "userId", userID, "deleted", result.RowsAffected)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"message":      "user stats reset",
		"deletedCount": result.RowsAffected,
	})
}

// handleAdminAuditLogs: 오디트 로그 조회
func handleAdminAuditLogs(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	sessionID := r.URL.Query().Get("sessionId")
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	deps.Logger.Info("ADMIN_AUDIT_LOGS_REQUEST", "sessionId", sessionID, "limit", limit)

	query := deps.DB.WithContext(ctx).Model(&qrepo.AuditLog{}).Order("created_at DESC").Limit(limit).Offset(offset)
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	var logs []qrepo.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		deps.Logger.Error("ADMIN_AUDIT_LOGS_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query audit logs")
		return
	}

	var total int64
	deps.DB.WithContext(ctx).Model(&qrepo.AuditLog{}).Count(&total)

	deps.Logger.Info("ADMIN_AUDIT_LOGS_SUCCESS", "count", len(logs))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleAdminRefundLogs: 리펀드 로그 조회
func handleAdminRefundLogs(w http.ResponseWriter, r *http.Request, deps AdminDeps) {
	ctx := r.Context()
	sessionID := r.URL.Query().Get("sessionId")
	userID := r.URL.Query().Get("userId")
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	deps.Logger.Info("ADMIN_REFUND_LOGS_REQUEST", "sessionId", sessionID, "userId", userID, "limit", limit)

	query := deps.DB.WithContext(ctx).Model(&qrepo.RefundLog{}).Order("created_at DESC").Limit(limit).Offset(offset)
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	var logs []qrepo.RefundLog
	if err := query.Find(&logs).Error; err != nil {
		deps.Logger.Error("ADMIN_REFUND_LOGS_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, adminErrorInternalError, "failed to query refund logs")
		return
	}

	var total int64
	deps.DB.WithContext(ctx).Model(&qrepo.RefundLog{}).Count(&total)

	deps.Logger.Info("ADMIN_REFUND_LOGS_SUCCESS", "count", len(logs))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
