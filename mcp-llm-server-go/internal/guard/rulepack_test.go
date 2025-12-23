package guard

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestCompileRulepack(t *testing.T) {
	raw := rawRulepack{
		Threshold: 0.5,
		Rules: []rawRule{
			{ID: "r1", Type: "regex", Pattern: "evil", Weight: 0.6},
			{ID: "r2", Type: "phrases", Phrases: []string{"bad", "worse"}, Weight: 0.2},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	pack, err := compileRulepack(raw, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pack.RegexRules) != 1 {
		t.Fatalf("expected regex rules")
	}
	if pack.PhraseMatcher == nil || len(pack.Phrases) != 2 {
		t.Fatalf("expected phrase matcher")
	}
	if pack.PhraseWeights["bad"] != 0.2 {
		t.Fatalf("unexpected phrase weight")
	}
}

func TestLoadRulepacks(t *testing.T) {
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.yml")
	data := []byte("version: 1\nthreshold: 0.5\nrules:\n  - id: r1\n    type: regex\n    pattern: evil\n    weight: 0.6\n")
	if err := os.WriteFile(rulePath, data, 0o644); err != nil {
		t.Fatalf("failed to write rulepack: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	packs := loadRulepacks(dir, logger)
	if len(packs) != 1 {
		t.Fatalf("expected 1 pack, got %d", len(packs))
	}
}
