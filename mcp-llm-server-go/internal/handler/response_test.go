package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type sampleRequest struct {
	Name string `json:"name" binding:"required"`
}

func TestBindJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	var req sampleRequest
	if bindJSON(c, &req) {
		t.Fatalf("expected bindJSON to fail")
	}
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestBindJSONAllowEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	var req sampleRequest
	if !bindJSONAllowEmpty(c, &req) {
		t.Fatalf("expected bindJSONAllowEmpty to succeed")
	}
}
