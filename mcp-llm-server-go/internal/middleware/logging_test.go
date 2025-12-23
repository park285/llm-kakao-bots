package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

type logEntry struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

type recordingHandler struct {
	level   slog.Level
	attrs   []slog.Attr
	mu      sync.Mutex
	entries []logEntry
}

func (h *recordingHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := map[string]any{}
	for _, attr := range h.attrs {
		attrs[attr.Key] = attr.Value.Any()
	}
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})

	h.mu.Lock()
	h.entries = append(h.entries, logEntry{
		level: record.Level,
		msg:   record.Message,
		attrs: attrs,
	})
	h.mu.Unlock()
	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recordingHandler{
		level:   h.level,
		attrs:   append(append([]slog.Attr{}, h.attrs...), attrs...),
		entries: h.entries, // entries 슬라이스 공유 (실제로는 테스트에서 WithAttrs 반환 핸들러의 entries를 안 씀)
	}
}

func (h *recordingHandler) WithGroup(_ string) slog.Handler {
	return &recordingHandler{
		level:   h.level,
		attrs:   h.attrs,
		entries: h.entries,
	}
}

func (h *recordingHandler) Entries() []logEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries := make([]logEntry, len(h.entries))
	copy(entries, h.entries)
	return entries
}

func TestRequestLoggerLogsInfoOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &recordingHandler{level: slog.LevelInfo}
	logger := slog.New(handler)

	router := gin.New()
	router.Use(RequestID(), RequestLogger(logger))
	router.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	entries := handler.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.level != slog.LevelInfo {
		t.Fatalf("expected info level, got %s", entry.level)
	}
	if entry.msg != "http_request" {
		t.Fatalf("expected http_request message, got %q", entry.msg)
	}

	ctx := entry.attrs
	if ctx["request_id"] != "req-123" {
		t.Fatalf("expected request_id=req-123, got %v", ctx["request_id"])
	}
	if ctx["method"] != "GET" {
		t.Fatalf("expected method=GET, got %v", ctx["method"])
	}
	if ctx["path"] != "/api/test" {
		t.Fatalf("expected path=/api/test, got %v", ctx["path"])
	}
	if fmt.Sprint(ctx["status"]) != "200" {
		t.Fatalf("expected status=200, got %v", ctx["status"])
	}
}

func TestRequestLoggerSkipsHealthOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &recordingHandler{level: slog.LevelInfo}
	logger := slog.New(handler)

	router := gin.New()
	router.Use(RequestID(), RequestLogger(logger))
	router.GET("/health/ready", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	req.Header.Set(RequestIDHeader, "req-health")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	entries := handler.Entries()
	if len(entries) != 0 {
		t.Fatalf("expected no log entry, got %d", len(entries))
	}
}

func TestRequestLoggerLogsWarnOnClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &recordingHandler{level: slog.LevelInfo}
	logger := slog.New(handler)

	router := gin.New()
	router.Use(RequestID(), RequestLogger(logger))
	router.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusBadRequest) })

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(RequestIDHeader, "req-400")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	entries := handler.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.level != slog.LevelWarn {
		t.Fatalf("expected warn level, got %s", entry.level)
	}

	ctx := entry.attrs
	if ctx["request_id"] != "req-400" {
		t.Fatalf("expected request_id=req-400, got %v", ctx["request_id"])
	}
	if fmt.Sprint(ctx["status"]) != "400" {
		t.Fatalf("expected status=400, got %v", ctx["status"])
	}
}

func TestRequestLoggerLogsErrorOnServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &recordingHandler{level: slog.LevelInfo}
	logger := slog.New(handler)

	router := gin.New()
	router.Use(RequestID(), RequestLogger(logger))
	router.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(RequestIDHeader, "req-500")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	entries := handler.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.level != slog.LevelError {
		t.Fatalf("expected error level, got %s", entry.level)
	}

	ctx := entry.attrs
	if ctx["request_id"] != "req-500" {
		t.Fatalf("expected request_id=req-500, got %v", ctx["request_id"])
	}
	if fmt.Sprint(ctx["status"]) != "500" {
		t.Fatalf("expected status=500, got %v", ctx["status"])
	}
}
