package watchdog

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// GetConfig 는 동작을 수행한다.
func (w *Watchdog) GetConfig() Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg
}

// GetStartedAt 는 동작을 수행한다.
func (w *Watchdog) GetStartedAt() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.startedAt
}

// GetConfigSource 는 동작을 수행한다.
func (w *Watchdog) GetConfigSource() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.configSource
}

// GetConfigPath 는 동작을 수행한다.
func (w *Watchdog) GetConfigPath() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.configPath
}

// GetState 는 동작을 수행한다.
func (w *Watchdog) GetState(containerName string) (*ContainerState, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	state, ok := w.states[containerName]
	return state, ok
}

// SnapshotStates 는 동작을 수행한다.
func (w *Watchdog) SnapshotStates() []*ContainerState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	states := make([]*ContainerState, 0, len(w.states))
	for _, state := range w.states {
		states = append(states, state)
	}
	return states
}

// GetEventLimiter 는 동작을 수행한다.
func (w *Watchdog) GetEventLimiter() *rate.Limiter {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.eventLimiter
}

// PollInterval 는 동작을 수행한다.
func (w *Watchdog) PollInterval() time.Duration {
	seconds := w.GetConfig().IntervalSeconds
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(seconds) * time.Second
}

// GetRootContext 는 동작을 수행한다.
func (w *Watchdog) GetRootContext() context.Context {
	w.mu.RLock()
	ctx := w.rootCtx
	w.mu.RUnlock()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}