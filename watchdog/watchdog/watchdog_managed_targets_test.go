package watchdog

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateConfigFileContainers_AddAndPreserveUnknown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	initial := `{
  "enabled": true,
  "containers": ["a", "/b", "a"],
  "intervalSeconds": 30,
  "extra": { "foo": "bar" }
}`
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatalf("write initial: %v", err)
	}

	updated, err := updateConfigFileContainers(path, "c", true)
	if err != nil {
		t.Fatalf("update containers: %v", err)
	}
	if !containsString(updated, "a") || !containsString(updated, "b") || !containsString(updated, "c") {
		t.Fatalf("unexpected containers: %#v", updated)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}
	if _, ok := doc["extra"]; !ok {
		t.Fatalf("expected extra field preserved")
	}

	arr, ok := doc["containers"].([]any)
	if !ok {
		t.Fatalf("containers type mismatch: %T", doc["containers"])
	}
	if len(arr) != 3 {
		t.Fatalf("containers length mismatch: %d", len(arr))
	}
}

func TestUpdateConfigFileContainers_EmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"containers":["x"]}`), 0644); err != nil {
		t.Fatalf("write initial: %v", err)
	}

	_, err := updateConfigFileContainers(path, "x", false)
	if err != nil {
		t.Fatalf("remove container: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}

	value, ok := doc["containers"]
	if !ok {
		t.Fatalf("containers key missing")
	}
	arr, ok := value.([]any)
	if !ok {
		t.Fatalf("containers should be array, got %T", value)
	}
	if len(arr) != 0 {
		t.Fatalf("expected empty containers, got %d", len(arr))
	}
}

func TestUpdateConfigFileContainers_InvalidContainersType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"containers":"nope"}`), 0644); err != nil {
		t.Fatalf("write initial: %v", err)
	}

	if _, err := updateConfigFileContainers(path, "x", true); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSetTargetManaged_WritesAndReloads(t *testing.T) {
	t.Setenv("WATCHDOG_CONTAINERS", "")
	t.Setenv("WATCHDOG_RESTART_CONTAINERS", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"containers":["alpha"]}`), 0644); err != nil {
		t.Fatalf("write initial: %v", err)
	}

	cfg := loadConfigFromEnv()
	cfg.Containers = []string{"alpha"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := NewWatchdog(nil, cfg, path, "file", logger)

	if _, err := w.SetTargetManaged(context.Background(), "beta", true, "admin@example.com", "test"); err != nil {
		t.Fatalf("set managed true: %v", err)
	}
	if !containsString(w.GetConfig().Containers, "alpha") || !containsString(w.GetConfig().Containers, "beta") {
		t.Fatalf("unexpected runtime containers: %#v", w.GetConfig().Containers)
	}

	if _, err := w.SetTargetManaged(context.Background(), "alpha", false, "admin@example.com", ""); err != nil {
		t.Fatalf("set managed false: %v", err)
	}
	if containsString(w.GetConfig().Containers, "alpha") {
		t.Fatalf("alpha should be removed: %#v", w.GetConfig().Containers)
	}
}