package watchdog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moby/moby/client"
)

const (
	// RestartByAuto 는 자동 재시작 구분 값이다.
	RestartByAuto = "auto"
	// RestartByManual 는 수동 재시작 구분 값이다.
	RestartByManual = "manual"
)

// ErrContainerNotManaged 는 패키지 변수다.
var ErrContainerNotManaged = errors.New("container is not managed")

// PauseMonitoring 는 동작을 수행한다.
func (w *Watchdog) PauseMonitoring(containerName string) error {
	state, ok := w.GetState(containerName)
	if !ok {
		return ErrContainerNotManaged
	}
	state.mu.Lock()
	state.monitoringPaused = true
	state.failures = 0
	state.mu.Unlock()
	w.appendEvent(Event{
		Action:    "monitor_pause",
		Container: containerName,
		Result:    "ok",
	})
	return nil
}

// ResumeMonitoring 는 동작을 수행한다.
func (w *Watchdog) ResumeMonitoring(containerName string) error {
	state, ok := w.GetState(containerName)
	if !ok {
		return ErrContainerNotManaged
	}
	state.mu.Lock()
	state.monitoringPaused = false
	state.mu.Unlock()
	w.appendEvent(Event{
		Action:    "monitor_resume",
		Container: containerName,
		Result:    "ok",
	})
	w.TriggerHealthCheck()
	return nil
}

// StopContainer 는 동작을 수행한다.
func (w *Watchdog) StopContainer(ctx context.Context, containerName string, timeoutSeconds int, requestedBy string, reason string) error {
	state, ok := w.GetState(containerName)
	if !ok {
		return ErrContainerNotManaged
	}

	state.mu.Lock()
	previousPaused := state.monitoringPaused
	state.monitoringPaused = true
	state.failures = 0
	state.mu.Unlock()

	timeout := timeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}
	stopTimeout := time.Duration(timeout) * time.Second
	stopCtx, cancel := context.WithTimeout(ctx, stopTimeout+5*time.Second)
	defer cancel()

	_, err := w.cli.ContainerStop(stopCtx, containerName, client.ContainerStopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		state.mu.Lock()
		state.monitoringPaused = previousPaused
		state.mu.Unlock()
		return fmt.Errorf("docker stop failed: %w", err)
	}

	w.appendEvent(Event{
		Action:      "stop",
		Container:   containerName,
		By:          RestartByManual,
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "ok",
	})
	return nil
}

// StartContainer 는 동작을 수행한다.
func (w *Watchdog) StartContainer(ctx context.Context, containerName string, requestedBy string, reason string) error {
	_, ok := w.GetState(containerName)
	if !ok {
		return ErrContainerNotManaged
	}

	startCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if _, err := w.cli.ContainerStart(startCtx, containerName, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("docker start failed: %w", err)
	}

	_ = w.ResumeMonitoring(containerName)
	w.appendEvent(Event{
		Action:      "start",
		Container:   containerName,
		By:          RestartByManual,
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "ok",
	})
	return nil
}

// RequestRestart 는 동작을 수행한다.
func (w *Watchdog) RequestRestart(ctx context.Context, containerName string, by string, reason string, requestedBy string, force bool) (bool, string, error) {
	state, ok := w.GetState(containerName)
	if !ok {
		return false, "", ErrContainerNotManaged
	}
	return w.startRestart(ctx, state, by, reason, requestedBy, force)
}

func (w *Watchdog) startRestart(ctx context.Context, state *ContainerState, by string, reason string, requestedBy string, force bool) (bool, string, error) {
	cfg := w.GetConfig()
	now := time.Now()

	state.mu.Lock()
	isPaused := state.monitoringPaused
	cooldownUntil := state.cooldownUntil
	state.mu.Unlock()

	if isPaused && by == RestartByAuto {
		w.appendEvent(Event{
			Action:    "restart",
			Container: state.name,
			By:        by,
			Reason:    "paused",
			Result:    "skipped",
		})
		return false, "paused", nil
	}

	if !force && now.Before(cooldownUntil) {
		remaining := time.Until(cooldownUntil).Round(time.Second)
		w.appendEvent(Event{
			Action:    "restart",
			Container: state.name,
			By:        by,
			Reason:    "cooldown",
			Result:    "skipped",
		})
		return false, fmt.Sprintf("cooldown(%s)", remaining), nil
	}

	if !state.restartInProgress.CompareAndSwap(false, true) {
		w.appendEvent(Event{
			Action:    "restart",
			Container: state.name,
			By:        by,
			Reason:    "in_progress",
			Result:    "skipped",
		})
		return false, "in_progress", nil
	}

	state.mu.Lock()
	state.lastRestartAt = now
	state.lastRestartBy = by
	state.lastRestartReason = reason
	state.lastRestartRequestedBy = requestedBy
	state.lastRestartResult = "initiated"
	state.lastRestartError = ""
	state.mu.Unlock()

	w.appendEvent(Event{
		Action:      "restart",
		Container:   state.name,
		By:          by,
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "initiated",
	})

	go w.executeRestart(state, cfg, by, reason, requestedBy)
	return true, "accepted", nil
}

func (w *Watchdog) executeRestart(state *ContainerState, cfg Config, by, reason, requestedBy string) {
	defer state.restartInProgress.Store(false)

	restartTimeout := time.Duration(cfg.RestartTimeoutSec+10) * time.Second
	restartCtx, cancel := context.WithTimeout(w.GetRootContext(), restartTimeout)
	defer cancel()

	started := time.Now()
	w.logger.Warn("restart_initiated", "container", state.name, "by", by)

	err := w.RestartContainer(restartCtx, state.name)
	elapsed := time.Since(started).Round(time.Millisecond)

	if err != nil {
		w.handleRestartFailure(state, by, reason, requestedBy, elapsed, err)
		return
	}
	w.handleRestartSuccess(state, cfg, by, reason, requestedBy, elapsed)
}

func (w *Watchdog) handleRestartFailure(state *ContainerState, by, reason, requestedBy string, elapsed time.Duration, err error) {
	w.logger.Error("restart_failed", "container", state.name, "by", by, "elapsed", elapsed, "err", err)
	state.mu.Lock()
	state.lastRestartResult = "failed"
	state.lastRestartError = err.Error()
	state.mu.Unlock()
	w.appendEvent(Event{
		Action:      "restart",
		Container:   state.name,
		By:          by,
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "failed",
		Error:       err.Error(),
	})
}

func (w *Watchdog) handleRestartSuccess(state *ContainerState, cfg Config, by, reason, requestedBy string, elapsed time.Duration) {
	w.logger.Info("restart_ok", "container", state.name, "by", by, "elapsed", elapsed)
	state.mu.Lock()
	state.failures = 0
	state.lastRestartResult = "ok"
	state.lastRestartError = ""
	if cfg.CooldownSeconds > 0 {
		state.cooldownUntil = time.Now().Add(time.Duration(cfg.CooldownSeconds) * time.Second)
	}
	state.mu.Unlock()

	w.appendEvent(Event{
		Action:      "restart",
		Container:   state.name,
		By:          by,
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "ok",
	})
}
