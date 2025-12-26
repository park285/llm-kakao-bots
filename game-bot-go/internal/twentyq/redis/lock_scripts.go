package redis

import (
	"context"
	"fmt"

	qassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
)

func (m *LockManager) loadScripts(ctx context.Context) error {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()

	if m.acquireReadSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockAcquireReadLua)
		if err != nil {
			return fmt.Errorf("load acquire_read script: %w", err)
		}
		m.acquireReadSHA = sha
	}
	if m.acquireWriteSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockAcquireWriteLua)
		if err != nil {
			return fmt.Errorf("load acquire_write script: %w", err)
		}
		m.acquireWriteSHA = sha
	}
	if m.releaseReadSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockReleaseReadLua)
		if err != nil {
			return fmt.Errorf("load release_read script: %w", err)
		}
		m.releaseReadSHA = sha
	}
	if m.releaseWriteSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockReleaseLua)
		if err != nil {
			return fmt.Errorf("load release_write script: %w", err)
		}
		m.releaseWriteSHA = sha
	}
	if m.renewReadSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockRenewReadLua)
		if err != nil {
			return fmt.Errorf("load renew_read script: %w", err)
		}
		m.renewReadSHA = sha
	}
	if m.renewWriteSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockRenewWriteLua)
		if err != nil {
			return fmt.Errorf("load renew_write script: %w", err)
		}
		m.renewWriteSHA = sha
	}
	return nil
}

func (m *LockManager) loadScript(ctx context.Context, script string) (string, error) {
	cmd := m.client.B().ScriptLoad().Script(script).Build()
	sha, err := m.client.Do(ctx, cmd).ToString()
	if err != nil {
		return "", fmt.Errorf("script load failed: %w", err)
	}
	return sha, nil
}

func (m *LockManager) clearScriptCache() {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()
	m.acquireReadSHA = ""
	m.acquireWriteSHA = ""
	m.releaseReadSHA = ""
	m.releaseWriteSHA = ""
	m.renewReadSHA = ""
	m.renewWriteSHA = ""
}
