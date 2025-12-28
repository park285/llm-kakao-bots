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

func TestCompileRulepackErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))

	tests := []struct {
		name    string
		raw     rawRulepack
		wantErr bool
	}{
		{
			name: "Unknown rule type",
			raw: rawRulepack{
				Rules: []rawRule{
					{ID: "r1", Type: "unknown", Pattern: "test", Weight: 0.5},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid regex rule - missing ID",
			raw: rawRulepack{
				Rules: []rawRule{
					{ID: "", Type: "regex", Pattern: "test", Weight: 0.5},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid regex rule - missing pattern",
			raw: rawRulepack{
				Rules: []rawRule{
					{ID: "r1", Type: "regex", Pattern: "", Weight: 0.5},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid phrases rule - missing ID",
			raw: rawRulepack{
				Rules: []rawRule{
					{ID: "", Type: "phrases", Phrases: []string{"test"}, Weight: 0.5},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid phrases rule - empty phrases",
			raw: rawRulepack{
				Rules: []rawRule{
					{ID: "r1", Type: "phrases", Phrases: []string{}, Weight: 0.5},
				},
			},
			wantErr: true,
		},
		{
			name: "Default values - no threshold",
			raw: rawRulepack{
				Version: 0,
				Rules: []rawRule{
					{ID: "r1", Type: "regex", Pattern: "test", Weight: 0.5},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid regex pattern - continues with warning",
			raw: rawRulepack{
				Rules: []rawRule{
					{ID: "r1", Type: "regex", Pattern: "[invalid", Weight: 0.5},
					{ID: "r2", Type: "regex", Pattern: "valid", Weight: 0.5},
				},
			},
			wantErr: false, // 무효한 regex는 건너뛰고 계속 진행
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileRulepack(tc.raw, logger)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
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

func TestLoadRulepacksErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))

	t.Run("Empty directory", func(t *testing.T) {
		dir := t.TempDir()
		packs := loadRulepacks(dir, logger)
		if len(packs) != 0 {
			t.Errorf("expected 0 packs for empty dir, got %d", len(packs))
		}
	})

	t.Run("Invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		rulePath := filepath.Join(dir, "invalid.yml")
		data := []byte("invalid: yaml: content: [")
		if err := os.WriteFile(rulePath, data, 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		packs := loadRulepacks(dir, logger)
		if len(packs) != 0 {
			t.Errorf("expected 0 packs for invalid YAML, got %d", len(packs))
		}
	})

	t.Run("File with compile error", func(t *testing.T) {
		dir := t.TempDir()
		rulePath := filepath.Join(dir, "error.yml")
		data := []byte("version: 1\nrules:\n  - id: r1\n    type: unknown\n    weight: 0.5\n")
		if err := os.WriteFile(rulePath, data, 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		packs := loadRulepacks(dir, logger)
		if len(packs) != 0 {
			t.Errorf("expected 0 packs for compile error, got %d", len(packs))
		}
	})

	t.Run("Non-existent directory", func(t *testing.T) {
		packs := loadRulepacks("/non/existent/path", logger)
		if len(packs) != 0 {
			t.Errorf("expected 0 packs for non-existent dir, got %d", len(packs))
		}
	})

	t.Run("YAML file extension variations", func(t *testing.T) {
		dir := t.TempDir()
		// .yml 파일
		yml := filepath.Join(dir, "test1.yml")
		if err := os.WriteFile(yml, []byte("version: 1\nrules:\n  - id: r1\n    type: regex\n    pattern: test1\n    weight: 0.5\n"), 0o644); err != nil {
			t.Fatalf("failed to write yml: %v", err)
		}
		// .yaml 파일
		yaml := filepath.Join(dir, "test2.yaml")
		if err := os.WriteFile(yaml, []byte("version: 1\nrules:\n  - id: r2\n    type: regex\n    pattern: test2\n    weight: 0.5\n"), 0o644); err != nil {
			t.Fatalf("failed to write yaml: %v", err)
		}
		packs := loadRulepacks(dir, logger)
		if len(packs) != 2 {
			t.Errorf("expected 2 packs (.yml and .yaml), got %d", len(packs))
		}
	})
}

func TestFindRulepackFiles(t *testing.T) {
	t.Run("Directory with matching files", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}

		files := findRulepackFiles(dir)
		if len(files) != 2 {
			t.Errorf("expected 2 yml/yaml files, got %d", len(files))
		}
	})

	t.Run("Empty directory", func(t *testing.T) {
		dir := t.TempDir()
		files := findRulepackFiles(dir)
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})
}
