package watchdog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// CheckContainerHealthFromSummary evaluates container health from a summary.
func (w *Watchdog) CheckContainerHealthFromSummary(summary *container.Summary) (bool, string) {
	if summary == nil {
		return false, "not_found"
	}

	if summary.State != container.StateRunning {
		if summary.State == container.StateRestarting {
			return false, "restarting"
		}
		state := string(summary.State)
		status := trimStatusValue(summary.Status)
		if status != "" {
			return false, fmt.Sprintf("not_running(state=%s,status=%s)", state, status)
		}
		return false, fmt.Sprintf("not_running(state=%s)", state)
	}

	if summary.Health == nil || summary.Health.Status == container.NoHealthcheck {
		return true, "running_no_healthcheck"
	}

	switch summary.Health.Status {
	case container.Healthy:
		return true, "healthy"
	case container.Starting:
		return true, "starting"
	case container.Unhealthy:
		return false, "unhealthy"
	default:
		return false, fmt.Sprintf("unknown(%s)", summary.Health.Status)
	}
}

// TriggerHealthCheck triggers an immediate health check.
func (w *Watchdog) TriggerHealthCheck() {
	select {
	case w.checkTrigger <- struct{}{}:
	default:
	}
}

// ListTargetContainerSummaries lists summaries for target containers.
func (w *Watchdog) ListTargetContainerSummaries(ctx context.Context) (map[string]*container.Summary, error) {
	w.mu.RLock()
	filters := w.listFilters
	targetSet := w.targetSet
	w.mu.RUnlock()

	if len(targetSet) == 0 {
		return map[string]*container.Summary{}, nil
	}

	result, err := w.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	cfg := w.GetConfig()
	byName := make(map[string]*container.Summary, len(cfg.Containers))
	for i := range result.Items {
		item := &result.Items[i]
		for _, rawName := range item.Names {
			name := strings.TrimPrefix(rawName, "/")
			if _, ok := targetSet[name]; ok {
				byName[name] = item
			}
		}
	}

	return byName, nil
}

// RestartContainer restarts a container.
func (w *Watchdog) RestartContainer(ctx context.Context, containerName string) error {
	timeout := w.GetConfig().RestartTimeoutSec
	_, err := w.cli.ContainerRestart(ctx, containerName, client.ContainerRestartOptions{
		Timeout: &timeout,
	})
	return err
}

// ProcessHealthCheck updates status based on summary and triggers restart if needed.
func (w *Watchdog) ProcessHealthCheck(ctx context.Context, state *ContainerState, now time.Time, summary *container.Summary) {
	state.mu.Lock()
	if state.monitoringPaused {
		state.lastChecked = now
		var status string
		switch {
		case summary == nil:
			status = "paused_not_found"
		case summary.State == container.StateRunning:
			status = "paused_running"
		default:
			status = fmt.Sprintf("paused_%s", summary.State)
		}
		state.lastStatus = status
		state.failures = 0
		state.mu.Unlock()
		if w.GetConfig().VerboseLogging {
			w.logger.Info("monitor_paused", "container", state.name, "status", status)
		}
		return
	}
	state.mu.Unlock()

	healthy, status := w.CheckContainerHealthFromSummary(summary)

	state.mu.Lock()
	state.lastChecked = now
	state.lastStatus = status
	prevFailures := state.failures

	if healthy {
		state.failures = 0
		state.mu.Unlock()

		if prevFailures > 0 {
			w.logger.Info("recover",
				"container", state.name,
				"status", status,
				"prev_failures", prevFailures,
			)
		} else if w.GetConfig().VerboseLogging {
			w.logger.Info("healthy",
				"container", state.name,
				"status", status,
			)
		}
		return
	}

	state.failures++
	failures := state.failures
	state.mu.Unlock()

	cfg := w.GetConfig()
	w.logger.Warn("unhealthy",
		"container", state.name,
		"status", status,
		"failures", failures,
		"threshold", cfg.MaxFailures,
	)

	// 재확인 루프: 첫 실패 후 retryIntervalSeconds 간격으로 retryChecks회 재확인
	// 도중에 healthy 감지 → 복구 처리 및 중단
	// 전부 실패 → 재시작
	if failures >= cfg.MaxFailures {
		w.runRetryVerification(ctx, state)
	}
}

// runRetryVerification 은 첫 실패 후 짧은 간격으로 재확인을 수행한다.
// 재확인 중 healthy가 감지되면 복구 처리 후 중단하고,
// 모든 재확인이 실패하면 MaybeRestart를 호출한다.
func (w *Watchdog) runRetryVerification(ctx context.Context, state *ContainerState) {
	cfg := w.GetConfig()
	retryInterval := time.Duration(cfg.RetryIntervalSeconds) * time.Second

	w.logger.Info("retry_verification_start",
		"container", state.name,
		"retry_checks", cfg.RetryChecks,
		"retry_interval", retryInterval,
	)

	for i := 1; i <= cfg.RetryChecks; i++ {
		select {
		case <-ctx.Done():
			w.logger.Info("retry_verification_canceled", "container", state.name)
			return
		case <-time.After(retryInterval):
		}

		// 단일 컨테이너 상태 조회
		listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		summaries, err := w.ListTargetContainerSummaries(listCtx)
		cancel()

		if err != nil {
			w.logger.Error("retry_check_failed", "container", state.name, "attempt", i, "err", err)
			continue
		}

		summary := summaries[state.name]
		healthy, status := w.CheckContainerHealthFromSummary(summary)

		state.mu.Lock()
		state.lastChecked = time.Now()
		state.lastStatus = status
		state.mu.Unlock()

		if healthy {
			// 복구 감지 → 실패 카운터 초기화 및 루프 중단
			state.mu.Lock()
			prevFailures := state.failures
			state.failures = 0
			state.mu.Unlock()

			w.logger.Info("recover_during_retry",
				"container", state.name,
				"status", status,
				"attempt", i,
				"prev_failures", prevFailures,
			)
			return
		}

		w.logger.Warn("retry_still_unhealthy",
			"container", state.name,
			"status", status,
			"attempt", i,
			"max_attempts", cfg.RetryChecks,
		)
	}

	// 모든 재확인 실패 → 재시작
	w.logger.Warn("retry_verification_failed",
		"container", state.name,
		"retry_checks", cfg.RetryChecks,
	)
	w.MaybeRestart(ctx, state, time.Now())
}

// MaybeRestart restarts the container if failure threshold is reached.
func (w *Watchdog) MaybeRestart(ctx context.Context, state *ContainerState, now time.Time) {
	cfg := w.GetConfig()
	if !cfg.Enabled {
		return
	}

	state.mu.Lock()
	failures := state.failures
	cooldownUntil := state.cooldownUntil
	lastStatus := state.lastStatus
	state.mu.Unlock()

	if failures < cfg.MaxFailures {
		return
	}

	if now.Before(cooldownUntil) {
		remaining := time.Until(cooldownUntil).Round(time.Second)
		w.logger.Warn("restart_skipped",
			"container", state.name,
			"reason", "cooldown",
			"remaining", remaining,
		)
		return
	}

	reason := fmt.Sprintf("healthcheck failures=%d status=%s threshold=%d", failures, lastStatus, cfg.MaxFailures)
	ok, msg, err := w.startRestart(ctx, state, RestartByAuto, reason, "", false)
	if err != nil {
		w.logger.Error("restart_request_failed", "container", state.name, "err", err)
		return
	}
	if !ok && cfg.VerboseLogging {
		w.logger.Info("restart_skipped", "container", state.name, "reason", msg)
	}
}

func (w *Watchdog) runHealthCheckCycle(ctx context.Context) {
	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	summaries, err := w.ListTargetContainerSummaries(listCtx)
	if err != nil {
		w.logger.Error("container_list_failed", "err", err)
		return
	}

	now := time.Now()
	for _, state := range w.SnapshotStates() {
		w.ProcessHealthCheck(ctx, state, now, summaries[state.name])
	}
}
