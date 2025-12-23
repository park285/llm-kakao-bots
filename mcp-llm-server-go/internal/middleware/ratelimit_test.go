package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{HTTPRateLimit: config.HTTPRateLimitConfig{
		RequestsPerMinute: 1,
		CacheSize:         10,
		CacheTTLSeconds:   int(time.Minute.Seconds()),
	}}

	router := gin.New()
	router.Use(RateLimit(cfg))
	router.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	first := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	first.RemoteAddr = "1.2.3.4:1234"
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, first)
	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d", firstResp.Code)
	}

	second := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	second.RemoteAddr = "1.2.3.4:1234"
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, second)
	if secondResp.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limit, got %d", secondResp.Code)
	}
}
