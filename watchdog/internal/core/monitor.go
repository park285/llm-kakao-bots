package watchdog

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// runPollingMonitor polls all target containers with a single timer.
func (w *Watchdog) runPollingMonitor(ctx context.Context) {
	w.runHealthCheckCycle(ctx)

	timer := time.NewTimer(w.PollInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.GetConfig().VerboseLogging {
				w.logger.Info("monitor_stop", "reason", "shutdown")
			}
			return
		case <-timer.C:
			w.runHealthCheckCycle(ctx)
			timer.Reset(w.PollInterval())
		case <-w.checkTrigger:
			limiter := w.GetEventLimiter()
			if limiter != nil && !limiter.Allow() {
				if w.GetConfig().VerboseLogging {
					w.logger.Info("check_trigger_throttled",
						"min_interval", time.Duration(w.GetConfig().EventMinIntervalSec)*time.Second,
					)
				}
				continue
			}
			w.runHealthCheckCycle(ctx)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(w.PollInterval())
		}
	}
}

func (w *Watchdog) runStatusReporter(ctx context.Context) {
	for {
		cfg := w.GetConfig()
		wait := time.Duration(cfg.StatusReportSeconds) * time.Second
		if cfg.StatusReportSeconds <= 0 {
			wait = 5 * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			if cfg.StatusReportSeconds > 0 {
				w.logStatusReport()
			}
		}
	}
}

func (w *Watchdog) logStatusReport() {
	w.mu.RLock()
	defer w.mu.RUnlock()

	cfg := w.GetConfig()
	healthyCount := 0
	unhealthyCount := 0
	statuses := make([]string, 0, len(w.states))

	for _, state := range w.states {
		state.mu.Lock()
		status := state.lastStatus
		failures := state.failures
		lastChecked := state.lastChecked
		state.mu.Unlock()

		if status == "" {
			status = "pending"
		}

		checkedAgo := ""
		if !lastChecked.IsZero() {
			checkedAgo = fmt.Sprintf(" checked_ago=%s", time.Since(lastChecked).Round(time.Second))
		}

		if failures > 0 {
			unhealthyCount++
			statuses = append(statuses, fmt.Sprintf("%s(FAIL:%d,%s%s)", state.name, failures, status, checkedAgo))
		} else {
			healthyCount++
			statuses = append(statuses, fmt.Sprintf("%s(OK,%s%s)", state.name, status, checkedAgo))
		}
	}

	w.logger.Info("status_report",
		"healthy", healthyCount,
		"unhealthy", unhealthyCount,
		"check_interval", time.Duration(cfg.IntervalSeconds)*time.Second,
		"report_interval", time.Duration(cfg.StatusReportSeconds)*time.Second,
		"max_failures", cfg.MaxFailures,
		"details", statuses,
	)
}

// Run starts the watchdog.
func (w *Watchdog) Run(ctx context.Context) {
	var wg sync.WaitGroup

	cfg := w.GetConfig()
	if cfg.UseEvents {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runEventListener(ctx)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		w.runPollingMonitor(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		w.runStatusReporter(ctx)
	}()

	<-ctx.Done()
	w.logger.Info("shutting_down")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("shutdown_complete")
	case <-time.After(5 * time.Second):
		w.logger.Warn("shutdown_timeout")
	}
}
