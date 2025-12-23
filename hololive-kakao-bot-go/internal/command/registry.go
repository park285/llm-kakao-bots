package command

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// ErrUnknownCommand 는 패키지 변수다.
var ErrUnknownCommand = errors.New("unknown command")

// Registry 는 타입이다.
type Registry struct {
	mu        sync.RWMutex
	handlers  map[string]Command
	aliasKeys map[string]string
}

// NewRegistry 는 동작을 수행한다.
func NewRegistry() *Registry {
	return &Registry{
		handlers:  make(map[string]Command),
		aliasKeys: make(map[string]string),
	}
}

// Register 는 동작을 수행한다.
func (r *Registry) Register(handler Command) {
	if handler == nil {
		return
	}

	name := util.Normalize(handler.Name())

	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = handler
}

// Execute 는 동작을 수행한다.
func (r *Registry) Execute(ctx context.Context, cmdCtx *domain.CommandContext, key string, params map[string]any) error {
	if r == nil {
		return fmt.Errorf("command registry is nil")
	}

	handler := r.getHandler(key)
	if handler == nil {
		return fmt.Errorf("%w: %s", ErrUnknownCommand, key)
	}

	if err := handler.Execute(ctx, cmdCtx, params); err != nil {
		return fmt.Errorf("failed to execute command %s: %w", key, err)
	}
	return nil
}

// Count 는 동작을 수행한다.
func (r *Registry) Count() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}

func (r *Registry) getHandler(key string) Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if key == "" {
		return nil
	}
	if handler, ok := r.handlers[util.Normalize(key)]; ok {
		return handler
	}
	return nil
}
