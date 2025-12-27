package mq

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
)

const (
	scriptProcessWithIdempotency = "mq_process_with_idempotency"
	scriptCompleteProcessing     = "mq_complete_processing"
)

type luaRegistry struct {
	scripts map[string]*valkey.Lua
	sources map[string]string
}

func newLuaRegistry() *luaRegistry {
	sources := map[string]string{
		scriptProcessWithIdempotency: processWithIdempotencyLua,
		scriptCompleteProcessing:     completeProcessingLua,
	}
	return &luaRegistry{
		scripts: map[string]*valkey.Lua{
			scriptProcessWithIdempotency: valkey.NewLuaScript(processWithIdempotencyLua),
			scriptCompleteProcessing:     valkey.NewLuaScript(completeProcessingLua),
		},
		sources: sources,
	}
}

func (r *luaRegistry) Exec(ctx context.Context, client valkey.Client, name string, keys []string, args []string) (valkey.ValkeyResult, error) {
	if r == nil {
		return valkey.ValkeyResult{}, fmt.Errorf("lua registry is nil")
	}
	if client == nil {
		return valkey.ValkeyResult{}, fmt.Errorf("valkey client is nil")
	}
	script, ok := r.scripts[name]
	if !ok {
		return valkey.ValkeyResult{}, fmt.Errorf("unknown lua script: %s", name)
	}
	return script.Exec(ctx, client, keys, args), nil
}

func (r *luaRegistry) Preload(ctx context.Context, client valkey.Client) error {
	if r == nil {
		return fmt.Errorf("lua registry is nil")
	}
	if client == nil {
		return fmt.Errorf("valkey client is nil")
	}

	nodes := client.Nodes()
	if len(nodes) == 0 {
		nodes = map[string]valkey.Client{"default": client}
	}

	var firstErr error
	for name, source := range r.sources {
		for _, node := range nodes {
			cmd := node.B().ScriptLoad().Script(source).Build()
			if err := node.Do(ctx, cmd).Error(); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("lua preload failed (%s): %w", name, err)
				}
			}
		}
	}
	return firstErr
}
