package watchdog

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// HandleDockerEvent handles Docker events.
func (w *Watchdog) HandleDockerEvent(ctx context.Context, event events.Message) {
	if !w.GetConfig().UseEvents {
		return
	}

	containerName := CanonicalContainerName(event.Actor.Attributes["name"])
	if containerName == "" {
		return
	}

	w.mu.RLock()
	state, exists := w.states[containerName]
	w.mu.RUnlock()
	if !exists {
		return
	}

	switch event.Action {
	case events.ActionDie, events.ActionKill, events.ActionStop:
		if w.GetConfig().VerboseLogging {
			w.logger.Info("event", "container", containerName, "action", event.Action)
		}
		w.TriggerHealthCheck()

	case events.ActionHealthStatusUnhealthy:
		if w.GetConfig().VerboseLogging {
			w.logger.Warn("event", "container", containerName, "action", "health_status:unhealthy")
		}
		w.TriggerHealthCheck()

	case events.ActionStart, events.ActionRestart:
		if w.GetConfig().VerboseLogging {
			w.logger.Info("event", "container", containerName, "action", event.Action)
		}
		state.mu.Lock()
		state.failures = 0
		state.mu.Unlock()
		w.TriggerHealthCheck()

	case events.ActionHealthStatusHealthy:
		if w.GetConfig().VerboseLogging {
			w.logger.Info("event", "container", containerName, "action", "health_status:healthy")
		}
		prevFailures := 0
		state.mu.Lock()
		prevFailures = state.failures
		state.failures = 0
		state.mu.Unlock()
		if prevFailures > 0 {
			w.logger.Info("recover", "container", containerName, "via_event", true)
		}
		w.TriggerHealthCheck()
	}
}

func (w *Watchdog) runEventListener(ctx context.Context) {
	filters := make(client.Filters)
	filters = filters.Add("type", "container")
	cfg := w.GetConfig()
	for _, name := range cfg.Containers {
		filters = filters.Add("container", name)
	}

	w.logger.Info("events_start", "containers", cfg.Containers)

	retryBackoff := backoff.NewExponentialBackOff()
	retryBackoff.InitialInterval = 1 * time.Second
	retryBackoff.MaxInterval = 30 * time.Second
	retryBackoff.Multiplier = 2.0
	retryBackoff.RandomizationFactor = 0.2
	retryBackoff.MaxElapsedTime = 0

	eventsResult := w.cli.Events(ctx, client.EventsListOptions{
		Filters: filters,
	})

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("events_stop", "reason", "shutdown")
			return
		case err := <-eventsResult.Err:
			if err != nil && ctx.Err() == nil {
				wait := retryBackoff.NextBackOff()
				w.logger.Warn("events_error", "err", err, "retry_in", wait.Round(time.Second))
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					w.logger.Info("events_stop", "reason", "shutdown")
					return
				case <-timer.C:
				}
				eventsResult = w.cli.Events(ctx, client.EventsListOptions{
					Filters: filters,
				})
			}
		case event := <-eventsResult.Messages:
			retryBackoff.Reset()
			w.HandleDockerEvent(ctx, event)
		}
	}
}
