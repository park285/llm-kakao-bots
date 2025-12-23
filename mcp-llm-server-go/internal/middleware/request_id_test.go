package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGenerated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	id := resp.Header().Get(RequestIDHeader)
	if id == "" {
		t.Fatalf("expected request id header")
	}
	if resp.Body.String() != id {
		t.Fatalf("expected body to match request id")
	}
}

func TestRequestIDPreserved(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	id := resp.Header().Get(RequestIDHeader)
	if id != "req-123" {
		t.Fatalf("expected request id to be preserved")
	}
}
