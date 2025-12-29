package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

// Manager 세션 관리자
type Manager struct {
	store  *Store
	gemini *gemini.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewManager 세션 관리자 생성
func NewManager(
	store *Store,
	geminiClient *gemini.Client,
	cfg *config.Config,
	logger *slog.Logger,
) *Manager {
	return &Manager{
		store:  store,
		gemini: geminiClient,
		cfg:    cfg,
		logger: logger,
	}
}

// CreateSessionRequest 세션 생성 요청
type CreateSessionRequest struct {
	SystemPrompt string `json:"system_prompt,omitempty"`
	Model        string `json:"model,omitempty"`
}

// Info: 세션 정보 응답입니다.
type Info struct {
	ID           string             `json:"id"`
	SystemPrompt string             `json:"system_prompt,omitempty"`
	Model        string             `json:"model,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	MessageCount int                `json:"message_count"`
	History      []llm.HistoryEntry `json:"history,omitempty"`
}

// ChatRequest 세션 채팅 요청
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

// ChatResponse 세션 채팅 응답
type ChatResponse struct {
	Response     string    `json:"response"`
	Model        string    `json:"model"`
	Usage        llm.Usage `json:"usage"`
	MessageCount int       `json:"message_count"`
}

// Create 세션 생성
func (m *Manager) Create(ctx context.Context, req CreateSessionRequest) (*Info, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	meta := Meta{
		ID:           sessionID,
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
	}

	if err := m.store.CreateSession(ctx, meta); err != nil {
		return nil, err
	}

	m.logger.Debug("session_created", "session_id", sessionID)

	return &Info{
		ID:           meta.ID,
		SystemPrompt: meta.SystemPrompt,
		Model:        meta.Model,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
		MessageCount: meta.MessageCount,
	}, nil
}

// Get 세션 정보 조회
func (m *Manager) Get(ctx context.Context, sessionID string) (*Info, error) {
	meta, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	history, err := m.store.GetHistory(ctx, sessionID)
	if err != nil {
		history = nil // 히스토리 조회 실패해도 메타는 반환
	}

	return &Info{
		ID:           meta.ID,
		SystemPrompt: meta.SystemPrompt,
		Model:        meta.Model,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
		MessageCount: meta.MessageCount,
		History:      history,
	}, nil
}

// Chat 세션 기반 채팅
func (m *Manager) Chat(ctx context.Context, sessionID string, req ChatRequest) (*ChatResponse, error) {
	meta, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	history, err := m.store.GetHistory(ctx, sessionID)
	if err != nil {
		history = make([]llm.HistoryEntry, 0)
	}

	// Gemini 요청
	geminiReq := gemini.Request{
		Prompt:       req.Message,
		SystemPrompt: meta.SystemPrompt,
		History:      history,
		Model:        meta.Model,
	}

	result, model, err := m.gemini.ChatWithUsage(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("chat with usage: %w", err)
	}

	// 히스토리에 사용자 메시지 + 응답 추가
	userEntry := llm.HistoryEntry{Role: "user", Content: req.Message}
	assistantEntry := llm.HistoryEntry{Role: "assistant", Content: result.Text}

	if err := m.store.AppendHistory(ctx, sessionID, userEntry, assistantEntry); err != nil {
		m.logger.Warn("history_append_failed", "err", err)
	}

	meta.MessageCount += 2
	meta.UpdatedAt = time.Now()
	if err := m.store.UpdateSession(ctx, *meta); err != nil {
		m.logger.Warn("session_update_failed", "err", err)
	}

	return &ChatResponse{
		Response:     result.Text,
		Model:        model,
		Usage:        result.Usage,
		MessageCount: meta.MessageCount,
	}, nil
}

// Delete 세션 삭제
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	if err := m.store.DeleteSession(ctx, sessionID); err != nil {
		return err
	}

	m.logger.Debug("session_deleted", "session_id", sessionID)
	return nil
}

// Count 현재 세션 수
func (m *Manager) Count(ctx context.Context) int {
	count, err := m.store.SessionCount(ctx)
	if err != nil {
		return 0
	}
	return count
}

// generateSessionID 랜덤 세션 ID 생성
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
