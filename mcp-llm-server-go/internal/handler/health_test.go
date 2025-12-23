package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestHealthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Gemini: config.GeminiConfig{
			APIKeys:      nil,
			DefaultModel: "gemini-3-test",
		},
		HTTP: config.HTTPConfig{HTTP2Enabled: true},
	}

	router := gin.New()
	RegisterHealthRoutes(router, cfg)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}

	modelReq := httptest.NewRequest(http.MethodGet, "/health/models", nil)
	modelResp := httptest.NewRecorder()
	router.ServeHTTP(modelResp, modelReq)
	if modelResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", modelResp.Code)
	}

	var payload ModelConfigResponse
	if err := json.Unmarshal(modelResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.ModelDefault != "gemini-3-test" || payload.ModelHints != "gemini-3-test" {
		t.Fatalf("unexpected models: %+v", payload)
	}
	if payload.TransportMode != "h2c" {
		t.Fatalf("expected h2c, got %s", payload.TransportMode)
	}
}
