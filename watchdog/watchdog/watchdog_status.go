package watchdog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// TargetStatus is a snapshot of target container status for frontend use.
type TargetStatus struct {
	Name             string              `json:"name"`
	MonitoringPaused bool                `json:"monitoringPaused"`
	Watchdog         TargetWatchdogState `json:"watchdog"`
	Docker           TargetDockerState   `json:"docker"`
}

// TargetWatchdogState 는 타입이다.
type TargetWatchdogState struct {
	Failures          int       `json:"failures"`
	LastStatus        string    `json:"lastStatus"`
	LastCheckedAt     time.Time `json:"lastCheckedAt,omitempty"`
	CooldownUntil     time.Time `json:"cooldownUntil,omitempty"`
	RestartInProgress bool      `json:"restartInProgress"`

	LastRestartAt          time.Time `json:"lastRestartAt,omitempty"`
	LastRestartBy          string    `json:"lastRestartBy,omitempty"`
	LastRestartRequestedBy string    `json:"lastRestartRequestedBy,omitempty"`
	LastRestartReason      string    `json:"lastRestartReason,omitempty"`
	LastRestartResult      string    `json:"lastRestartResult,omitempty"`
	LastRestartError       string    `json:"lastRestartError,omitempty"`
}

// TargetDockerState 는 타입이다.
type TargetDockerState struct {
	Found        bool      `json:"found"`
	ID           string    `json:"id,omitempty"`
	Image        string    `json:"image,omitempty"`
	State        string    `json:"state,omitempty"`
	Status       string    `json:"status,omitempty"`
	Health       string    `json:"health,omitempty"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	FinishedAt   time.Time `json:"finishedAt,omitempty"`
	ExitCode     int       `json:"exitCode,omitempty"`
	RestartCount int       `json:"restartCount,omitempty"`
	UptimeSec    int64     `json:"uptimeSec,omitempty"`
}

// ListTargetsStatus 는 동작을 수행한다.
func (w *Watchdog) ListTargetsStatus(ctx context.Context) ([]TargetStatus, error) {
	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	summaries, err := w.ListTargetContainerSummaries(listCtx)
	if err != nil {
		return nil, fmt.Errorf("docker container list failed: %w", err)
	}

	states := w.SnapshotStates()
	out := make([]TargetStatus, 0, len(states))
	for _, state := range states {
		status := w.buildTargetStatus(ctx, state, summaries[state.name])
		out = append(out, status)
	}
	return out, nil
}

// GetTargetStatus 는 동작을 수행한다.
func (w *Watchdog) GetTargetStatus(ctx context.Context, containerName string) (TargetStatus, error) {
	state, ok := w.GetState(containerName)
	if !ok {
		return TargetStatus{}, ErrContainerNotManaged
	}

	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	summaries, err := w.ListTargetContainerSummaries(listCtx)
	if err != nil {
		return TargetStatus{}, fmt.Errorf("docker container list failed: %w", err)
	}

	return w.buildTargetStatus(ctx, state, summaries[containerName]), nil
}

func (w *Watchdog) buildTargetStatus(ctx context.Context, state *ContainerState, summary *container.Summary) TargetStatus {
	out := TargetStatus{
		Name: state.name,
	}

	state.mu.Lock()
	out.MonitoringPaused = state.monitoringPaused
	out.Watchdog.Failures = state.failures
	out.Watchdog.LastStatus = state.lastStatus
	out.Watchdog.LastCheckedAt = state.lastChecked
	out.Watchdog.CooldownUntil = state.cooldownUntil
	out.Watchdog.RestartInProgress = state.restartInProgress.Load()

	out.Watchdog.LastRestartAt = state.lastRestartAt
	out.Watchdog.LastRestartBy = state.lastRestartBy
	out.Watchdog.LastRestartRequestedBy = state.lastRestartRequestedBy
	out.Watchdog.LastRestartReason = state.lastRestartReason
	out.Watchdog.LastRestartResult = state.lastRestartResult
	out.Watchdog.LastRestartError = state.lastRestartError
	state.mu.Unlock()

	if summary == nil {
		out.Docker.Found = false
		return out
	}
	out.Docker.Found = true
	out.Docker.Image = summary.Image
	out.Docker.State = string(summary.State)
	out.Docker.Status = trimStatusValue(summary.Status)
	if summary.Health != nil {
		out.Docker.Health = string(summary.Health.Status)
	}

	inspectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	inspectResult, err := w.cli.ContainerInspect(inspectCtx, state.name, client.ContainerInspectOptions{})
	if err != nil {
		out.Docker.Found = true
		out.Docker.Status = fmt.Sprintf("%s (inspect_failed)", out.Docker.Status)
		return out
	}
	inspect := inspectResult.Container

	out.Docker.ID = inspect.ID
	if inspect.State != nil {
		out.Docker.State = string(inspect.State.Status)
		out.Docker.ExitCode = inspect.State.ExitCode
		out.Docker.RestartCount = inspect.RestartCount
		if startedAt, ok := parseDockerTime(inspect.State.StartedAt); ok {
			out.Docker.StartedAt = startedAt
			out.Docker.UptimeSec = int64(time.Since(startedAt).Seconds())
		}
		if finishedAt, ok := parseDockerTime(inspect.State.FinishedAt); ok {
			out.Docker.FinishedAt = finishedAt
		}
	}

	return out
}

func parseDockerTime(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || value == "0001-01-01T00:00:00Z" {
		return time.Time{}, false
	}

	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t, true
}
