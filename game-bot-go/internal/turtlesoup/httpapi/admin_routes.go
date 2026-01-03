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
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
	tsrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/repository"
)

const (
	turtleAdminErrorInvalidRequest  = "INVALID_REQUEST"
	turtleAdminErrorSessionNotFound = "SESSION_NOT_FOUND"
	turtleAdminErrorInternalError   = "INTERNAL_ERROR"
)

// TurtleAdminStatsResponse: 통합 통계 응답 DTO
type TurtleAdminStatsResponse struct {
	ActiveSessions   int     `json:"activeSessions"`
	TotalSolved      int     `json:"totalSolved"`
	TotalFailed      int     `json:"totalFailed"`
	SolveRate        float64 `json:"solveRate"`
	AvgQuestions     float64 `json:"avgQuestions"`
	AvgHintsPerGame  float64 `json:"avgHintsPerGame"`
	Last24HoursSolve int     `json:"last24HoursSolve"`
}

// TurtleActiveSessionResponse: 활성 세션 조회 응답 DTO
type TurtleActiveSessionResponse struct {
	SessionID     string `json:"sessionId"`
	ChatID        string `json:"chatId"`
	QuestionCount int    `json:"questionCount"`
	HintCount     int    `json:"hintCount"`
	TTLSeconds    int64  `json:"ttlSeconds"`
}

// TurtleCleanupRequest: 세션 정리 요청 DTO
type TurtleCleanupRequest struct {
	OlderThanHours int `json:"olderThanHours"`
}

// TurtleCleanupResponse: 세션 정리 응답 DTO
type TurtleCleanupResponse struct {
	DeletedCount int    `json:"deletedCount"`
	Message      string `json:"message"`
}

// TurtleInjectRequest: GM 힌트 주입 요청 DTO
type TurtleInjectRequest struct {
	Message string `json:"message"`
	AsBot   bool   `json:"asBot"`
}

// PuzzleCreateRequest: 퍼즐 생성 요청
type PuzzleCreateRequest struct {
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Solution   string   `json:"solution"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"`
	Hints      []string `json:"hints"`
	AuthorID   string   `json:"authorId"`
}

// TurtleAdminDeps: TurtleSoup Admin API 핸들러 의존성
type TurtleAdminDeps struct {
	DB           *gorm.DB
	ValkeyClient valkey.Client
	SessionStore *tsredis.SessionStore
	Logger       *slog.Logger
}

// RegisterTurtleAdminRoutes: TurtleSoup Admin API 라우트 등록
func RegisterTurtleAdminRoutes(mux *http.ServeMux, deps TurtleAdminDeps) {
	mux.HandleFunc("GET /admin/stats", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminStats(w, r, deps)
	})
	mux.HandleFunc("GET /admin/sessions", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminSessions(w, r, deps)
	})
	mux.HandleFunc("POST /admin/sessions/cleanup", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminCleanup(w, r, deps)
	})
	mux.HandleFunc("DELETE /admin/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminSessionDelete(w, r, deps)
	})
	mux.HandleFunc("POST /admin/sessions/{id}/inject", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminInject(w, r, deps)
	})

	// Puzzle CMS
	mux.HandleFunc("GET /admin/puzzles", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminPuzzleList(w, r, deps)
	})
	mux.HandleFunc("POST /admin/puzzles", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminPuzzleCreate(w, r, deps)
	})
	mux.HandleFunc("GET /admin/puzzles/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminPuzzleGet(w, r, deps)
	})
	mux.HandleFunc("PUT /admin/puzzles/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminPuzzleUpdate(w, r, deps)
	})
	mux.HandleFunc("DELETE /admin/puzzles/{id}", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminPuzzleDelete(w, r, deps)
	})
	mux.HandleFunc("GET /admin/puzzles/stats", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminPuzzleStats(w, r, deps)
	})

	// Archives
	mux.HandleFunc("GET /admin/archives", func(w http.ResponseWriter, r *http.Request) {
		handleTurtleAdminArchives(w, r, deps)
	})

	deps.Logger.Info("turtlesoup_admin_api_registered", "routes", 12)
}

func handleTurtleAdminStats(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	start := time.Now()
	deps.Logger.Info("TURTLE_ADMIN_STATS_REQUEST")

	activeSessions := countTurtleActiveSessions(ctx, deps)

	response := TurtleAdminStatsResponse{
		ActiveSessions:   activeSessions,
		TotalSolved:      0,
		TotalFailed:      0,
		SolveRate:        0,
		AvgQuestions:     0,
		AvgHintsPerGame:  0,
		Last24HoursSolve: 0,
	}

	deps.Logger.Info("TURTLE_ADMIN_STATS_SUCCESS", "duration", time.Since(start).Milliseconds())
	_ = commonhttputil.WriteJSON(w, http.StatusOK, response)
}

func handleTurtleAdminSessions(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	start := time.Now()
	deps.Logger.Info("TURTLE_ADMIN_SESSIONS_REQUEST")

	sessions := listTurtleActiveSessions(ctx, deps)

	deps.Logger.Info("TURTLE_ADMIN_SESSIONS_SUCCESS", "count", len(sessions), "duration", time.Since(start).Milliseconds())
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func handleTurtleAdminCleanup(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	deps.Logger.Info("TURTLE_ADMIN_CLEANUP_REQUEST")

	var req TurtleCleanupRequest
	if err := commonhttputil.ReadJSON(r, &req, 1024); err != nil {
		req.OlderThanHours = 24
	}
	if req.OlderThanHours < 1 {
		req.OlderThanHours = 1
	}

	client := deps.ValkeyClient
	if client == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "valkey client not available")
		return
	}

	// 오래된 세션 cleanup: TTL이 이미 Redis에서 관리되므로 SCAN은 불필요
	// 대신 TTL이 없는 키를 찾아서 삭제
	pattern := tsconfig.RedisKeySessionPrefix + ":*"
	deletedCount := 0
	cursor := uint64(0)
	threshold := time.Duration(req.OlderThanHours) * time.Hour

	for {
		cmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		result, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			deps.Logger.Error("TURTLE_ADMIN_CLEANUP_SCAN_FAILED", "err", err)
			break
		}

		for _, key := range result.Elements {
			ttlCmd := client.B().Ttl().Key(key).Build()
			ttl, err := client.Do(ctx, ttlCmd).AsInt64()
			if err != nil {
				continue
			}
			// TTL이 없거나 threshold 이상 남은 경우만 남김
			// threshold 미만인 경우는 곧 만료될 것이므로 삭제 대상
			if ttl == -1 || (ttl > 0 && ttl < int64(threshold.Seconds())) {
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

	deps.Logger.Info("TURTLE_ADMIN_CLEANUP_SUCCESS", "deletedCount", deletedCount)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, TurtleCleanupResponse{
		DeletedCount: deletedCount,
		Message:      "cleanup completed",
	})
}

func handleTurtleAdminSessionDelete(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	sessionID := r.PathValue("id")
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "session id is required")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_SESSION_DELETE_REQUEST", "sessionId", sessionID)

	exists, err := deps.SessionStore.SessionExists(ctx, sessionID)
	if err != nil {
		deps.Logger.Error("TURTLE_ADMIN_SESSION_DELETE_CHECK_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to check session")
		return
	}
	if !exists {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, turtleAdminErrorSessionNotFound, "session not found")
		return
	}

	if err := deps.SessionStore.DeleteSession(ctx, sessionID); err != nil {
		deps.Logger.Error("TURTLE_ADMIN_SESSION_DELETE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to delete session")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_SESSION_DELETE_SUCCESS", "sessionId", sessionID)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "session deleted",
	})
}

func handleTurtleAdminInject(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	sessionID := r.PathValue("id")
	if sessionID == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "session id is required")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_INJECT_REQUEST", "sessionId", sessionID)

	var req TurtleInjectRequest
	if err := commonhttputil.ReadJSON(r, &req, 4096); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "message is required")
		return
	}

	state, err := deps.SessionStore.LoadGameState(ctx, sessionID)
	if err != nil {
		deps.Logger.Error("TURTLE_ADMIN_INJECT_LOAD_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to load session")
		return
	}
	if state == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, turtleAdminErrorSessionNotFound, "session not found")
		return
	}

	// GM 힌트를 History에 추가
	state.History = append(state.History, tsmodel.HistoryEntry{
		Question: "[GM] " + req.Message,
		Answer:   "시스템 힌트",
	})
	state.HintsUsed++

	if err := deps.SessionStore.SaveGameState(ctx, *state); err != nil {
		deps.Logger.Error("TURTLE_ADMIN_INJECT_SAVE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to save session")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_INJECT_SUCCESS", "sessionId", sessionID, "message", req.Message)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "hint injected",
	})
}

// countTurtleActiveSessions: Valkey SCAN으로 활성 세션 수 조회
func countTurtleActiveSessions(ctx context.Context, deps TurtleAdminDeps) int {
	client := deps.ValkeyClient
	if client == nil {
		return 0
	}

	pattern := tsconfig.RedisKeySessionPrefix + ":*"
	count := 0
	cursor := uint64(0)

	for {
		cmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		result, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			deps.Logger.Error("TURTLE_ADMIN_COUNT_SESSIONS_SCAN_FAILED", "err", err)
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

// listTurtleActiveSessions: Valkey SCAN으로 활성 세션 목록 조회
func listTurtleActiveSessions(ctx context.Context, deps TurtleAdminDeps) []TurtleActiveSessionResponse {
	client := deps.ValkeyClient
	if client == nil {
		return []TurtleActiveSessionResponse{}
	}

	pattern := tsconfig.RedisKeySessionPrefix + ":*"
	var keys []string
	cursor := uint64(0)

	for {
		cmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()
		result, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			deps.Logger.Error("TURTLE_ADMIN_LIST_SESSIONS_SCAN_FAILED", "err", err)
			break
		}

		keys = append(keys, result.Elements...)
		cursor = result.Cursor
		if cursor == 0 {
			break
		}
	}

	sessions := make([]TurtleActiveSessionResponse, 0, len(keys))
	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) < 3 {
			continue
		}
		sessionID := parts[len(parts)-1]

		getCmd := client.B().Get().Key(key).Build()
		raw, err := client.Do(ctx, getCmd).AsBytes()
		if err != nil {
			if !valkeyx.IsNil(err) {
				deps.Logger.Warn("TURTLE_ADMIN_LIST_SESSIONS_GET_FAILED", "key", key, "err", err)
			}
			continue
		}

		var sessionData struct {
			ChatID        string `json:"chatId"`
			QuestionCount int    `json:"questionCount"`
			HintsUsed     int    `json:"hintsUsed"`
		}
		if err := json.Unmarshal(raw, &sessionData); err != nil {
			deps.Logger.Warn("TURTLE_ADMIN_LIST_SESSIONS_UNMARSHAL_FAILED", "key", key, "err", err)
			continue
		}

		ttlCmd := client.B().Ttl().Key(key).Build()
		ttl, err := client.Do(ctx, ttlCmd).AsInt64()
		if err != nil {
			ttl = -1
		}

		sessions = append(sessions, TurtleActiveSessionResponse{
			SessionID:     sessionID,
			ChatID:        sessionData.ChatID,
			QuestionCount: sessionData.QuestionCount,
			HintCount:     sessionData.HintsUsed,
			TTLSeconds:    ttl,
		})
	}

	return sessions
}

// handleTurtleAdminPuzzleList: 퍼즐 목록 조회
func handleTurtleAdminPuzzleList(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)
	if limit > 100 {
		limit = 100
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_LIST_REQUEST", "status", status, "limit", limit, "offset", offset)

	if deps.DB == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "db not available")
		return
	}

	repo := tsrepo.New(deps.DB)
	puzzles, total, err := repo.ListPuzzles(ctx, status, limit, offset)
	if err != nil {
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_LIST_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to list puzzles")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_LIST_SUCCESS", "count", len(puzzles))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"puzzles": puzzles,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// handleTurtleAdminPuzzleCreate: 퍼즐 생성
func handleTurtleAdminPuzzleCreate(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_CREATE_REQUEST")

	var req PuzzleCreateRequest
	if err := commonhttputil.ReadJSON(r, &req, 65536); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "invalid request body")
		return
	}

	if req.Title == "" || req.Scenario == "" || req.Solution == "" {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "title, scenario, solution are required")
		return
	}

	if deps.DB == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "db not available")
		return
	}

	hintsJSON := "[]"
	if len(req.Hints) > 0 {
		if b, err := json.Marshal(req.Hints); err == nil {
			hintsJSON = string(b)
		}
	}

	category := req.Category
	if category == "" {
		category = "MYSTERY"
	}
	difficulty := req.Difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 3
	}

	puzzle := &tsrepo.Puzzle{
		Title:      req.Title,
		Scenario:   req.Scenario,
		Solution:   req.Solution,
		Category:   category,
		Difficulty: difficulty,
		HintsJSON:  hintsJSON,
		Status:     "draft",
		AuthorID:   req.AuthorID,
	}

	repo := tsrepo.New(deps.DB)
	if err := repo.CreatePuzzle(ctx, puzzle); err != nil {
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_CREATE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to create puzzle")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_CREATE_SUCCESS", "id", puzzle.ID, "title", puzzle.Title)
	_ = commonhttputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"status":  "ok",
		"message": "puzzle created",
		"puzzle":  puzzle,
	})
}

// handleTurtleAdminPuzzleGet: 단일 퍼즐 조회
func handleTurtleAdminPuzzleGet(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "invalid puzzle id")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_GET_REQUEST", "id", id)

	if deps.DB == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "db not available")
		return
	}

	repo := tsrepo.New(deps.DB)
	puzzle, err := repo.GetPuzzle(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, "PUZZLE_NOT_FOUND", "puzzle not found")
			return
		}
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_GET_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to get puzzle")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_GET_SUCCESS", "id", id)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"puzzle": puzzle,
	})
}

// handleTurtleAdminPuzzleUpdate: 퍼즐 수정
func handleTurtleAdminPuzzleUpdate(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "invalid puzzle id")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_UPDATE_REQUEST", "id", id)

	var req map[string]any
	if err := commonhttputil.ReadJSON(r, &req, 65536); err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "invalid request body")
		return
	}

	if deps.DB == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "db not available")
		return
	}

	repo := tsrepo.New(deps.DB)
	puzzle, err := repo.GetPuzzle(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, "PUZZLE_NOT_FOUND", "puzzle not found")
			return
		}
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_UPDATE_GET_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to get puzzle")
		return
	}

	// 허용 필드만 업데이트
	if v, ok := req["title"].(string); ok && v != "" {
		puzzle.Title = v
	}
	if v, ok := req["scenario"].(string); ok && v != "" {
		puzzle.Scenario = v
	}
	if v, ok := req["solution"].(string); ok && v != "" {
		puzzle.Solution = v
	}
	if v, ok := req["category"].(string); ok && v != "" {
		puzzle.Category = v
	}
	if v, ok := req["difficulty"].(float64); ok {
		puzzle.Difficulty = int(v)
	}
	if v, ok := req["status"].(string); ok && v != "" {
		puzzle.Status = v
	}
	if v, ok := req["hints"].([]any); ok {
		hints := make([]string, 0, len(v))
		for _, h := range v {
			if s, ok := h.(string); ok {
				hints = append(hints, s)
			}
		}
		if b, err := json.Marshal(hints); err == nil {
			puzzle.HintsJSON = string(b)
		}
	}

	if err := repo.UpdatePuzzle(ctx, puzzle); err != nil {
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_UPDATE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to update puzzle")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_UPDATE_SUCCESS", "id", id)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "puzzle updated",
		"puzzle":  puzzle,
	})
}

// handleTurtleAdminPuzzleDelete: 퍼즐 삭제
func handleTurtleAdminPuzzleDelete(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusBadRequest, turtleAdminErrorInvalidRequest, "invalid puzzle id")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_DELETE_REQUEST", "id", id)

	if deps.DB == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "db not available")
		return
	}

	repo := tsrepo.New(deps.DB)

	// 존재 확인
	_, err = repo.GetPuzzle(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = commonhttputil.WriteErrorJSON(w, http.StatusNotFound, "PUZZLE_NOT_FOUND", "puzzle not found")
			return
		}
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_DELETE_GET_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to get puzzle")
		return
	}

	if err := repo.DeletePuzzle(ctx, id); err != nil {
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_DELETE_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to delete puzzle")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_DELETE_SUCCESS", "id", id)
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "puzzle deleted",
	})
}

// handleTurtleAdminPuzzleStats: 퍼즐 통계 조회
func handleTurtleAdminPuzzleStats(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_STATS_REQUEST")

	if deps.DB == nil {
		_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"message": "db not configured",
			"stats":   nil,
		})
		return
	}

	repo := tsrepo.New(deps.DB)
	stats, err := repo.GetPuzzleStats(ctx)
	if err != nil {
		deps.Logger.Error("TURTLE_ADMIN_PUZZLE_STATS_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to get puzzle stats")
		return
	}

	categoryStats, _ := repo.GetCategoryStats(ctx)

	deps.Logger.Info("TURTLE_ADMIN_PUZZLE_STATS_SUCCESS")
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"stats":         stats,
		"categoryStats": categoryStats,
	})
}

// handleTurtleAdminArchives: 게임 아카이브 조회
func handleTurtleAdminArchives(w http.ResponseWriter, r *http.Request, deps TurtleAdminDeps) {
	ctx := r.Context()
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)
	result := r.URL.Query().Get("result")
	if limit > 100 {
		limit = 100
	}

	deps.Logger.Info("TURTLE_ADMIN_ARCHIVES_REQUEST", "limit", limit, "offset", offset, "result", result)

	if deps.DB == nil {
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "db not available")
		return
	}

	query := deps.DB.WithContext(ctx).Model(&tsrepo.GameArchive{}).Order("completed_at DESC")
	if result != "" {
		query = query.Where("result = ?", result)
	}

	var total int64
	query.Count(&total)

	var archives []tsrepo.GameArchive
	if err := query.Limit(limit).Offset(offset).Find(&archives).Error; err != nil {
		deps.Logger.Error("TURTLE_ADMIN_ARCHIVES_QUERY_FAILED", "err", err)
		_ = commonhttputil.WriteErrorJSON(w, http.StatusInternalServerError, turtleAdminErrorInternalError, "failed to query archives")
		return
	}

	deps.Logger.Info("TURTLE_ADMIN_ARCHIVES_SUCCESS", "count", len(archives))
	_ = commonhttputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"archives": archives,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

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
