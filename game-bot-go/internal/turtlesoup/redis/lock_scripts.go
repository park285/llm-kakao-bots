package redis

import (
	"context"
	"fmt"

	tsassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/assets"
)

func (m *LockManager) loadReleaseScript(ctx context.Context) (string, error) {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()

	if m.releaseSHA != "" {
		return m.releaseSHA, nil
	}

	cmd := m.client.B().ScriptLoad().Script(tsassets.LockReleaseLua).Build()
	sha, err := m.client.Do(ctx, cmd).ToString()
	if err != nil {
		return "", fmt.Errorf("load release script: %w", err)
	}
	m.releaseSHA = sha
	return sha, nil
}

func (m *LockManager) clearScriptCache() {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()
	m.releaseSHA = ""
}
