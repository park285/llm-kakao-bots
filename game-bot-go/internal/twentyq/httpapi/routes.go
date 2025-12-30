package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"gorm.io/gorm"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
	qsvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/service"
)

const (
	headerSessionID    = "X-Session-Id"
	headerUserID       = "X-User-Id"
	internalChatPrefix = "internal:"
	fallbackAddr       = "local"
	maxBodyBytes       = 1 << 20
	minHintRequest     = 1
	maxHintRequest     = 10
)

// Register HTTP API 라우트 등록.
func Register(
	mux *http.ServeMux,
	riddleService *qsvc.RiddleService,
	db *gorm.DB,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) {
	// GET /health - 헬스체크
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"goroutines": runtime.NumGoroutine(),
		})
	})

	// POST /api/twentyq/riddles - 게임 시작
	mux.HandleFunc("POST /api/twentyq/riddles", func(w http.ResponseWriter, r *http.Request) {
		handleCreate(w, r, riddleService, msgProvider, logger)
	})

	// POST /api/twentyq/riddles/hints - 힌트 요청
	mux.HandleFunc("POST /api/twentyq/riddles/hints", func(w http.ResponseWriter, r *http.Request) {
		handleHints(w, r, riddleService, msgProvider, logger)
	})

	// POST /api/twentyq/riddles/answers - 질문/답변
	mux.HandleFunc("POST /api/twentyq/riddles/answers", func(w http.ResponseWriter, r *http.Request) {
		handleAnswer(w, r, riddleService, msgProvider, logger)
	})

	// GET /api/twentyq/riddles - 상태 조회
	mux.HandleFunc("GET /api/twentyq/riddles", func(w http.ResponseWriter, r *http.Request) {
		handleStatus(w, r, riddleService, msgProvider, logger)
	})

	// GET /api/twentyq/stats/rooms/{chatId}/users/{userId} - 사용자 통계
	mux.HandleFunc("GET /api/twentyq/stats/rooms/{chatId}/users/{userId}", func(w http.ResponseWriter, r *http.Request) {
		handleUserStats(w, r, db, logger)
	})

	logger.Info("twentyq_http_api_registered")
}

type (
	// RiddleCreateRequest: 스무고개 게임 생성 요청 DTO
	RiddleCreateRequest struct {
		Category *string `json:"category,omitempty"`
	}

	// RiddleCreateResponse: 게임 생성 결과 응답 DTO
	RiddleCreateResponse struct {
		Message string `json:"message"`
	}

	// RiddleHintsRequest: 힌트 요청 DTO
	RiddleHintsRequest struct {
		Count int `json:"count"`
	}

	// RiddleHintsResponse: 힌트 생성 결과 응답 DTO
	RiddleHintsResponse struct {
		Hints []string `json:"hints"`
	}

	// RiddleAnswerRequest: 사용자의 질문/정답 제출 요청 DTO
	RiddleAnswerRequest struct {
		Question string `json:"question"`
	}

	// RiddleAnswerResponse: 질문에 대한 AI의 답변(Correct, Incorrect 등) 응답 DTO
	RiddleAnswerResponse struct {
		Scale string `json:"scale"`
	}
)

func handleCreate(
	w http.ResponseWriter,
	r *http.Request,
	riddleService *qsvc.RiddleService,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) {
	chatID := resolveChatID(r)
	logger.Info("CREATE_REQUEST", "chatId", chatID)

	// 요청 파싱
	var req RiddleCreateRequest
	if r.Body != nil && r.ContentLength > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Debug("CREATE_PARSE_FAILED", "err", err)
		}
	}

	var categories []string
	if req.Category != nil && strings.TrimSpace(*req.Category) != "" {
		categories = []string{strings.TrimSpace(*req.Category)}
	}

	// 게임 시작
	start := time.Now()
	message, err := riddleService.Start(r.Context(), chatID, chatID, categories)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		logger.Error("CREATE_FAILED", "chatId", chatID, "err", err, "duration", duration)
		respondJSON(w, http.StatusInternalServerError, RiddleCreateResponse{Message: msgProvider.Get(qmessages.ErrorGeneric)})
		return
	}

	logger.Info("CREATE_SUCCESS", "chatId", chatID, "duration", duration)
	respondJSON(w, http.StatusOK, RiddleCreateResponse{Message: message})
}

func handleHints(
	w http.ResponseWriter,
	r *http.Request,
	riddleService *qsvc.RiddleService,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) {
	chatID := resolveChatID(r)
	logger.Info("HINTS_REQUEST", "chatId", chatID)

	// 요청 파싱
	var req RiddleHintsRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Count < minHintRequest || req.Count > maxHintRequest {
		respondError(w, http.StatusBadRequest, "count must be between 1 and 10")
		return
	}

	// 힌트 생성
	start := time.Now()
	hint, err := riddleService.GenerateHint(r.Context(), chatID)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		logger.Error("HINTS_FAILED", "chatId", chatID, "err", err, "duration", duration)
		respondJSON(w, http.StatusOK, RiddleHintsResponse{Hints: []string{msgProvider.Get(qmessages.ErrorGeneric)}})
		return
	}

	logger.Info("HINTS_SUCCESS", "chatId", chatID, "duration", duration)
	respondJSON(w, http.StatusOK, RiddleHintsResponse{Hints: []string{hint}})
}

func handleAnswer(
	w http.ResponseWriter,
	r *http.Request,
	riddleService *qsvc.RiddleService,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) {
	chatID := resolveChatID(r)
	userID := resolveUserID(r)
	logger.Info("ANSWER_REQUEST", "chatId", chatID, "userId", userID)

	// 요청 파싱
	var req RiddleAnswerRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Question) == "" {
		respondError(w, http.StatusBadRequest, "question is required")
		return
	}

	// 질문 처리
	start := time.Now()
	response, err := riddleService.Answer(r.Context(), chatID, userID, nil, req.Question)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		logger.Error("ANSWER_FAILED", "chatId", chatID, "err", err, "duration", duration)
		respondJSON(w, http.StatusOK, RiddleAnswerResponse{Scale: msgProvider.Get(qmessages.ErrorGeneric)})
		return
	}

	logger.Info("ANSWER_SUCCESS", "chatId", chatID, "duration", duration)
	respondJSON(w, http.StatusOK, RiddleAnswerResponse{Scale: response})
}

func handleStatus(
	w http.ResponseWriter,
	r *http.Request,
	riddleService *qsvc.RiddleService,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) {
	chatID := resolveChatID(r)
	logger.Info("STATUS_REQUEST", "chatId", chatID)

	start := time.Now()
	status, err := riddleService.Status(r.Context(), chatID)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		logger.Warn("STATUS_FAILED", "chatId", chatID, "err", err, "duration", duration)
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error":   "session_not_found",
			"message": msgProvider.Get(qmessages.ErrorNoSession),
		})
		return
	}

	logger.Info("STATUS_SUCCESS", "chatId", chatID, "duration", duration)
	// Status()는 문자열을 반환하므로 그대로 전달
	respondJSON(w, http.StatusOK, map[string]string{"status": status})
}

func handleUserStats(
	w http.ResponseWriter,
	r *http.Request,
	db *gorm.DB,
	logger *slog.Logger,
) {
	chatID := r.PathValue("chatId")
	userID := r.PathValue("userId")

	if chatID == "" || userID == "" {
		respondError(w, http.StatusBadRequest, "chatId and userId are required")
		return
	}

	logger.Info("USER_STATS_REQUEST", "chatId", chatID, "userId", userID)

	start := time.Now()
	compositeID := qrepo.CompositeUserStatsID(chatID, userID)

	var stats qrepo.UserStats
	if err := db.WithContext(r.Context()).First(&stats, "id = ?", compositeID).Error; err != nil {
		duration := time.Since(start).Milliseconds()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("USER_STATS_NOT_FOUND", "userId", userID, "duration", duration)
			respondError(w, http.StatusNotFound, "user stats not found")
			return
		}
		logger.Error("USER_STATS_FAILED", "userId", userID, "err", err, "duration", duration)
		respondError(w, http.StatusInternalServerError, "failed to fetch user stats")
		return
	}

	duration := time.Since(start).Milliseconds()
	logger.Info("USER_STATS_SUCCESS", "userId", userID, "totalStarted", stats.TotalGamesStarted, "totalCompleted", stats.TotalGamesCompleted, "duration", duration)

	// Kotlin DTO 형태로 변환
	response := UserStatsResponse{
		UserID:              stats.UserID,
		TotalGamesStarted:   stats.TotalGamesStarted,
		TotalGamesCompleted: stats.TotalGamesCompleted,
		TotalSurrenders:     stats.TotalSurrenders,
		TotalQuestionsAsked: stats.TotalQuestionsAsked,
		TotalHintsUsed:      stats.TotalHintsUsed,
		TotalWrongGuesses:   stats.TotalWrongGuesses,
	}

	if stats.BestScoreQuestionCnt != nil && stats.BestScoreTarget != nil {
		response.BestScore = &BestScoreResponse{
			QuestionCount:   *stats.BestScoreQuestionCnt,
			WrongGuessCount: derefIntOrZero(stats.BestScoreWrongGuess),
			Target:          *stats.BestScoreTarget,
			Category:        derefStringOrEmpty(stats.BestScoreCategory),
			AchievedAt:      stats.BestScoreAchievedAt,
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// UserStatsResponse: 특정 사용자의 전체 스무고개 게임 통계 응답 DTO
type UserStatsResponse struct {
	UserID              string             `json:"userId"`
	TotalGamesStarted   int                `json:"totalGamesStarted"`
	TotalGamesCompleted int                `json:"totalGamesCompleted"`
	TotalSurrenders     int                `json:"totalSurrenders"`
	TotalQuestionsAsked int                `json:"totalQuestionsAsked"`
	TotalHintsUsed      int                `json:"totalHintsUsed"`
	TotalWrongGuesses   int                `json:"totalWrongGuesses"`
	BestScore           *BestScoreResponse `json:"bestScore,omitempty"`
}

// BestScoreResponse: 사용자의 최고 기록(최소 질문 성공 등) 정보 응답 DTO
type BestScoreResponse struct {
	QuestionCount   int        `json:"questionCount"`
	WrongGuessCount int        `json:"wrongGuessCount"`
	Target          string     `json:"target"`
	Category        string     `json:"category"`
	AchievedAt      *time.Time `json:"achievedAt,omitempty"`
}

func derefIntOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefStringOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func resolveChatID(r *http.Request) string {
	chatID := strings.TrimSpace(r.Header.Get(headerSessionID))
	if chatID != "" {
		return chatID
	}
	remoteAddr := getRemoteAddr(r)
	if remoteAddr != "" {
		return internalChatPrefix + remoteAddr
	}
	return internalChatPrefix + fallbackAddr
}

func resolveUserID(r *http.Request) string {
	userID := strings.TrimSpace(r.Header.Get(headerUserID))
	if userID != "" {
		return userID
	}
	remoteAddr := getRemoteAddr(r)
	if remoteAddr != "" {
		return remoteAddr
	}
	return "unknown"
}

func getRemoteAddr(r *http.Request) string {
	// X-Forwarded-For 헤더 확인 (프록시 뒤에서 실행 시)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	// X-Real-IP 헤더 확인
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr에서 IP 추출
	if r.RemoteAddr != "" {
		addr := r.RemoteAddr
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			return addr[:idx]
		}
		return addr
	}
	return ""
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
