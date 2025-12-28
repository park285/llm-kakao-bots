package lua

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
	"golang.org/x/sync/errgroup"
)

// Script 는 Registry 에 등록되는 Lua 스크립트 정의다.
type Script struct {
	Name     string
	Source   string
	ReadOnly bool
	NoSHA    bool
	LoadSHA1 bool
}

// Registry 는 Lua 스크립트 실행을 단일 경로로 관리한다.
type Registry struct {
	scripts map[string]*valkey.Lua
	metas   map[string]Script
}

// NewRegistry 는 스크립트 목록으로 Registry 를 생성한다.
func NewRegistry(scripts []Script) *Registry {
	registry := &Registry{
		scripts: make(map[string]*valkey.Lua, len(scripts)),
		metas:   make(map[string]Script, len(scripts)),
	}
	for _, script := range scripts {
		registry.scripts[script.Name] = buildLua(script)
		registry.metas[script.Name] = script
	}
	return registry
}

// Exec 는 등록된 스크립트를 실행한다.
// Redis 오류는 ValkeyResult.Error()로 확인한다.
func (r *Registry) Exec(ctx context.Context, client valkey.Client, name string, keys []string, args []string) (valkey.ValkeyResult, error) {
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

// Preload 는 등록된 Lua 스크립트를 SCRIPT LOAD 로 서버에 적재한다.
// 노드별 병렬 로드를 수행하여 부트스트랩 시간을 단축한다.
func (r *Registry) Preload(ctx context.Context, client valkey.Client) error {
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

	// 병렬 로드: 노드 x 스크립트 조합을 동시에 처리
	g, gctx := errgroup.WithContext(ctx)
	for name, meta := range r.metas {
		if meta.NoSHA {
			continue
		}
		scriptName := name
		scriptSource := meta.Source
		for _, node := range nodes {
			nodeClient := node
			g.Go(func() error {
				cmd := nodeClient.B().ScriptLoad().Script(scriptSource).Build()
				if err := nodeClient.Do(gctx, cmd).Error(); err != nil {
					return fmt.Errorf("lua preload failed (%s): %w", scriptName, err)
				}
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("lua preload: %w", err)
	}
	return nil
}

// Script 는 등록된 스크립트를 반환한다.
func (r *Registry) Script(name string) (*valkey.Lua, bool) {
	if r == nil {
		return nil, false
	}
	script, ok := r.scripts[name]
	return script, ok
}

func buildLua(script Script) *valkey.Lua {
	if script.NoSHA {
		if script.ReadOnly {
			return valkey.NewLuaScriptReadOnlyNoSha(script.Source)
		}
		return valkey.NewLuaScriptNoSha(script.Source)
	}

	opts := []valkey.LuaOption{}
	if script.LoadSHA1 {
		opts = append(opts, valkey.WithLoadSHA1(true))
	}

	if script.ReadOnly {
		return valkey.NewLuaScriptReadOnly(script.Source, opts...)
	}
	return valkey.NewLuaScript(script.Source, opts...)
}
