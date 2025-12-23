package guard

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestGuardEvaluateAndEnsureSafe(t *testing.T) {
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.yml")
	data := []byte("version: 1\nthreshold: 0.5\nrules:\n  - id: r1\n    type: regex\n    pattern: evil\n    weight: 0.6\n")
	if err := os.WriteFile(rulePath, data, 0o644); err != nil {
		t.Fatalf("failed to write rulepack: %v", err)
	}

	cfg := &config.Config{
		Guard: config.GuardConfig{
			Enabled:         true,
			Threshold:       0.5,
			RulepacksDir:    dir,
			CacheMaxSize:    10,
			CacheTTLSeconds: 60,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	guard, err := NewGuard(cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evaluation := guard.Evaluate("evil payload")
	if !evaluation.Malicious() {
		t.Fatalf("expected malicious evaluation")
	}
	if err := guard.EnsureSafe("evil payload"); err == nil {
		t.Fatalf("expected blocked error")
	}

	safeEval := guard.Evaluate("hello")
	if safeEval.Malicious() {
		t.Fatalf("expected safe evaluation")
	}
}
