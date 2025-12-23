package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{HTTPAuth: config.HTTPAuthConfig{APIKey: "secret"}}

	router := gin.New()
	router.Use(APIKeyAuth(cfg))
	router.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", resp.Code)
	}

	authed := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	authed.Header.Set("X-API-Key", "secret")
	authedResp := httptest.NewRecorder()
	router.ServeHTTP(authedResp, authed)
	if authedResp.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d", authedResp.Code)
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthResp := httptest.NewRecorder()
	router.ServeHTTP(healthResp, healthReq)
	if healthResp.Code != http.StatusOK {
		t.Fatalf("expected ok for health, got %d", healthResp.Code)
	}
}
