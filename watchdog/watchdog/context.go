package watchdog

import (
	"context"
	"fmt"
)

// SetRootContext sets the root context for the watchdog.
func (w *Watchdog) SetRootContext(ctx context.Context) {
	w.rootCtx = ctx
}

// ValidateContainers verifies that all target containers exist.
func (w *Watchdog) ValidateContainers(ctx context.Context) error {
	if len(w.GetConfig().Containers) == 0 {
		return nil
	}

	summaries, err := w.ListTargetContainerSummaries(ctx)
	if err != nil {
		return fmt.Errorf("validate containers via list: %w", err)
	}

	for _, name := range w.GetConfig().Containers {
		summary := summaries[name]
		if summary == nil {
			w.logger.Warn("container_not_found", "container", name)
			continue
		}
		state := string(summary.State)
		status := trimStatusValue(summary.Status)
		if status != "" {
			w.logger.Info("container_found", "container", name, "state", state, "status", status)
			continue
		}
		w.logger.Info("container_found", "container", name, "state", state)
	}
	return nil
}
