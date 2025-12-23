package handler

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gin-gonic/gin"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
)

func TestGuardHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.yml")
	data := []byte("version: 1\nthreshold: 0.5\nrules:\n  - id: r1\n    type: regex\n    pattern: evil\n    weight: 0.6\n")
	if err := os.WriteFile(rulePath, data, 0o644); err != nil {
		t.Fatalf("failed to write rulepack: %v", err)
	}

	cfg := &config.Config{Guard: config.GuardConfig{Enabled: true, Threshold: 0.5, RulepacksDir: dir}}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	g, err := guard.NewGuard(cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := NewGuardHandler(g)
	router := gin.New()
	handler.RegisterRoutes(router)

	body := []byte(`{"input_text":"evil"}`)
	checkReq := httptest.NewRequest(http.MethodPost, "/api/guard/checks", bytes.NewBuffer(body))
	checkReq.Header.Set("Content-Type", "application/json")
	checkResp := httptest.NewRecorder()
	router.ServeHTTP(checkResp, checkReq)
	if checkResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", checkResp.Code)
	}

	var checkPayload map[string]any
	if err := json.Unmarshal(checkResp.Body.Bytes(), &checkPayload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if value, ok := checkPayload["malicious"].(bool); !ok || !value {
		t.Fatalf("expected malicious true")
	}

	evalReq := httptest.NewRequest(http.MethodPost, "/api/guard/evaluations", bytes.NewBuffer(body))
	evalReq.Header.Set("Content-Type", "application/json")
	evalResp := httptest.NewRecorder()
	router.ServeHTTP(evalResp, evalReq)
	if evalResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", evalResp.Code)
	}

	var evalPayload GuardResponse
	if err := json.Unmarshal(evalResp.Body.Bytes(), &evalPayload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !evalPayload.Malicious || len(evalPayload.Hits) == 0 {
		t.Fatalf("expected evaluation to be malicious")
	}
}
