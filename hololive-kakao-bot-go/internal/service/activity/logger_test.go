package activity

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestActivityLogger_LogAndGetRecentLogs(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "activity.log")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	l := NewActivityLogger(filePath, logger)
	l.Log("command", "first", map[string]any{"key": "value"})
	l.Log("system", "second", nil)

	logs, err := l.GetRecentLogs(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].Summary != "first" || logs[1].Summary != "second" {
		t.Fatalf("unexpected log order: %+v", logs)
	}

	limited, err := l.GetRecentLogs(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limited) != 1 || limited[0].Summary != "second" {
		t.Fatalf("unexpected limited logs: %+v", limited)
	}
}

func TestActivityLogger_GetRecentLogsMissingFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "missing.log")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	l := NewActivityLogger(filePath, logger)
	logs, err := l.GetRecentLogs(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected empty logs, got %d", len(logs))
	}
}
