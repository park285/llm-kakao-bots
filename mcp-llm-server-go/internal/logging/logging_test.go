package logging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

func TestNewLoggerCreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.LoggingConfig{
		LogDir:     dir,
		Level:      "info",
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
		Compress:   true,
	}
	_, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, "server.log")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected log file, got error: %v", err)
	}
}
