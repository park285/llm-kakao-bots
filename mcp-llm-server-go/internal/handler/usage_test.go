package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

func TestParseDays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?days=3", nil)

	days, ok := parseDays(c, 7)
	if !ok || days != 3 {
		t.Fatalf("unexpected days: %d", days)
	}
}

func TestParseDaysInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?days=0", nil)

	_, ok := parseDays(c, 7)
	if ok {
		t.Fatalf("expected parseDays to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBuildDailyResponse(t *testing.T) {
	cfg := &config.Config{Gemini: config.GeminiConfig{DefaultModel: "gemini-3-test"}}
	handler := &UsageHandler{cfg: cfg}

	resp := handler.buildDailyResponse(nil)
	if resp.TotalTokens != 0 || resp.Model != "gemini-3-test" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	row := &usage.DailyUsage{
		UsageDate:       time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		InputTokens:     1,
		OutputTokens:    2,
		ReasoningTokens: 1,
		RequestCount:    3,
	}
	resp = handler.buildDailyResponse(row)
	if resp.TotalTokens != 3 || resp.RequestCount != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestBuildUsageListResponse(t *testing.T) {
	cfg := &config.Config{Gemini: config.GeminiConfig{DefaultModel: "gemini-3-test"}}
	handler := &UsageHandler{cfg: cfg}

	rows := []usage.DailyUsage{
		{InputTokens: 1, OutputTokens: 2, ReasoningTokens: 0, RequestCount: 1, UsageDate: time.Now()},
		{InputTokens: 3, OutputTokens: 4, ReasoningTokens: 1, RequestCount: 2, UsageDate: time.Now()},
	}
	resp := handler.buildUsageListResponse(rows)
	if resp.TotalInputTokens != 4 || resp.TotalOutputTokens != 6 || resp.TotalRequestCount != 3 {
		t.Fatalf("unexpected totals: %+v", resp)
	}
}
