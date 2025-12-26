package settings

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestSettingsService_LoadDefaultAndPersist(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "settings.json")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := NewSettingsService(filePath, logger)
	got := svc.Get()
	if got.AlarmAdvanceMinutes != 5 {
		t.Fatalf("expected default 5, got %d", got.AlarmAdvanceMinutes)
	}

	updated := Settings{AlarmAdvanceMinutes: 12}
	if err := svc.Update(updated); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	reloaded := NewSettingsService(filePath, logger)
	got = reloaded.Get()
	if got.AlarmAdvanceMinutes != 12 {
		t.Fatalf("expected persisted 12, got %d", got.AlarmAdvanceMinutes)
	}
}
