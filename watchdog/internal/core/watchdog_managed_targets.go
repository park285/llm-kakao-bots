package watchdog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type orderedMap map[string]any

func (m orderedMap) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')

		valueJSON, err := json.Marshal(m[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valueJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false

	// durability best-effort (POSIX)
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}

func updateConfigFileContainers(path string, containerName string, managed bool) ([]string, error) {
	name := CanonicalContainerName(containerName)
	if name == "" {
		return nil, errors.New("container name is empty")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file read failed: %w", err)
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		trimmed = []byte("{}")
	}

	doc := make(map[string]json.RawMessage)
	if unmarshalErr := json.Unmarshal(trimmed, &doc); unmarshalErr != nil {
		return nil, fmt.Errorf("config file json parse failed: %w", unmarshalErr)
	}

	var containers []string
	if rawContainers, ok := doc["containers"]; ok {
		if parseErr := json.Unmarshal(rawContainers, &containers); parseErr != nil {
			return nil, fmt.Errorf("config file containers parse failed: %w", parseErr)
		}
	}
	containers = normalizeContainers(containers)

	if managed {
		if !containsString(containers, name) {
			containers = append(containers, name)
		}
	} else {
		containers = removeString(containers, name)
	}

	containersForJSON := containers
	if containersForJSON == nil {
		containersForJSON = []string{}
	}
	containersRaw, err := json.Marshal(containersForJSON)
	if err != nil {
		return nil, fmt.Errorf("containers marshal failed: %w", err)
	}
	doc["containers"] = containersRaw

	values := make(map[string]any, len(doc))
	for k, v := range doc {
		values[k] = v
	}

	pretty, err := json.MarshalIndent(orderedMap(values), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("config file marshal failed: %w", err)
	}
	if len(pretty) == 0 || pretty[len(pretty)-1] != '\n' {
		pretty = append(pretty, '\n')
	}

	perm := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	if err := atomicWriteFile(path, pretty, perm); err != nil {
		return nil, fmt.Errorf("config file write failed: %w", err)
	}
	return containers, nil
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func removeString(values []string, target string) []string {
	if len(values) == 0 {
		return nil
	}

	out := values[:0]
	for _, v := range values {
		if v == target {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SetTargetManaged 는 동작을 수행한다.
func (w *Watchdog) SetTargetManaged(ctx context.Context, containerName string, managed bool, requestedBy string, reason string) (ReloadResult, error) {
	path := w.GetConfigPath()
	if strings.TrimSpace(path) == "" {
		return ReloadResult{}, ErrConfigPathNotSet
	}

	w.configFileMu.Lock()
	defer w.configFileMu.Unlock()

	if _, err := updateConfigFileContainers(path, containerName, managed); err != nil {
		return ReloadResult{}, err
	}

	reloadResult, err := w.reloadConfigFromFileUnlocked(ctx)
	if err != nil {
		return ReloadResult{}, err
	}

	action := "target_managed_disable"
	if managed {
		action = "target_managed_enable"
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = action
	}

	w.appendEvent(Event{
		Action:      action,
		Container:   CanonicalContainerName(containerName),
		By:          RestartByManual,
		RequestedBy: requestedBy,
		Reason:      trimmedReason,
		Result:      "ok",
	})

	return reloadResult, nil
}
