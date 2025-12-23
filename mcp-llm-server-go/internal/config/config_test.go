package config

import "testing"

func TestParseAPIKeys(t *testing.T) {
	t.Setenv("GOOGLE_API_KEYS", "k1, k2")
	keys := parseAPIKeys()
	if len(keys) != 2 || keys[0] != "k1" || keys[1] != "k2" {
		t.Fatalf("unexpected keys: %+v", keys)
	}

	t.Setenv("GOOGLE_API_KEYS", "")
	t.Setenv("GOOGLE_API_KEY", "single")
	keys = parseAPIKeys()
	if len(keys) != 1 || keys[0] != "single" {
		t.Fatalf("unexpected single key: %+v", keys)
	}
}

func TestSplitKeys(t *testing.T) {
	keys := splitKeys("a,b c\td\n")
	if len(keys) != 4 {
		t.Fatalf("unexpected keys length: %d", len(keys))
	}
}

func TestGeminiConfigModelSelection(t *testing.T) {
	cfg := GeminiConfig{DefaultModel: "gemini-3-default", HintsModel: "gemini-3-hints"}
	if cfg.ModelForTask("hints") != "gemini-3-hints" {
		t.Fatalf("unexpected model for hints")
	}
	if cfg.ModelForTask("unknown") != "gemini-3-default" {
		t.Fatalf("unexpected default model")
	}
}

func TestTemperatureForModel(t *testing.T) {
	cfg := GeminiConfig{Temperature: 0.5}
	if cfg.TemperatureForModel("gemini-3-test") != 1.0 {
		t.Fatalf("expected min temperature for gemini3")
	}
	if cfg.TemperatureForModel("other-model") != 0.5 {
		t.Fatalf("unexpected temperature")
	}

	cfg = GeminiConfig{Temperature: 1.25}
	if cfg.TemperatureForModel("gemini-3-test") != 1.25 {
		t.Fatalf("expected configured temperature when >=1 for gemini3")
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := &Config{Gemini: GeminiConfig{DefaultModel: "gemini-2-test"}}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestThinkingConfigLevel(t *testing.T) {
	cfg := ThinkingConfig{
		LevelDefault: "low",
		LevelHints:   "medium",
		LevelAnswer:  "high",
		LevelVerify:  "minimal",
	}

	if cfg.Level("hints") != "medium" {
		t.Fatalf("expected 'medium' for hints, got: %s", cfg.Level("hints"))
	}
	if cfg.Level("answer") != "high" {
		t.Fatalf("expected 'high' for answer, got: %s", cfg.Level("answer"))
	}
	if cfg.Level("verify") != "minimal" {
		t.Fatalf("expected 'minimal' for verify, got: %s", cfg.Level("verify"))
	}
	if cfg.Level("unknown") != "low" {
		t.Fatalf("expected 'low' for unknown, got: %s", cfg.Level("unknown"))
	}
}

func TestGeminiConfigPrimaryKey(t *testing.T) {
	cfg := GeminiConfig{APIKeys: []string{"key1", "key2"}}
	if cfg.PrimaryKey() != "key1" {
		t.Fatalf("expected 'key1', got: %s", cfg.PrimaryKey())
	}

	cfg = GeminiConfig{APIKeys: nil}
	if cfg.PrimaryKey() != "" {
		t.Fatalf("expected empty string for nil keys")
	}
}

func TestDatabaseConfigDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "user",
		Password: "pass",
	}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Fatalf("expected non-empty DSN")
	}
	// DSN 형식: postgresql://user:pass@localhost:5432/testdb
	if !containsSubstring(dsn, "localhost:5432") {
		t.Fatalf("DSN should contain host:port: %s", dsn)
	}
	if !containsSubstring(dsn, "/testdb") {
		t.Fatalf("DSN should contain dbname: %s", dsn)
	}
	if !containsSubstring(dsn, "postgresql://") {
		t.Fatalf("DSN should start with postgresql://: %s", dsn)
	}
}

func TestConfigValidateSuccess(t *testing.T) {
	cfg := &Config{
		Gemini: GeminiConfig{DefaultModel: "gemini-3-test"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestModelForTaskAllVariants(t *testing.T) {
	cfg := GeminiConfig{
		DefaultModel: "gemini-3-default",
		HintsModel:   "gemini-3-hints",
		AnswerModel:  "gemini-3-answer",
		VerifyModel:  "gemini-3-verify",
	}

	tests := []struct {
		task     string
		expected string
	}{
		{"hints", "gemini-3-hints"},
		{"answer", "gemini-3-answer"},
		{"verify", "gemini-3-verify"},
		{"normalize", "gemini-3-default"},
		{"", "gemini-3-default"},
	}

	for _, tc := range tests {
		got := cfg.ModelForTask(tc.task)
		if got != tc.expected {
			t.Errorf("ModelForTask(%q) = %q, want %q", tc.task, got, tc.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
