package watchdog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"golang.org/x/time/rate"
)

// ContainerState tracks the state of a monitored container.
type ContainerState struct {
	name                   string
	failures               int
	cooldownUntil          time.Time
	restartInProgress      atomic.Bool
	lastStatus             string
	lastChecked            time.Time
	monitoringPaused       bool
	lastRestartAt          time.Time
	lastRestartBy          string
	lastRestartReason      string
	lastRestartRequestedBy string
	lastRestartResult      string
	lastRestartError       string
	mu                     sync.Mutex
}

// Config holds the watchdog configuration.
type Config struct {
	Enabled              bool
	Containers           []string
	IntervalSeconds      int
	MaxFailures          int
	RetryChecks          int
	RetryIntervalSeconds int
	GraceSeconds         int
	CooldownSeconds      int
	RestartTimeoutSec    int
	DockerSocket         string
	UseEvents            bool
	EventMinIntervalSec  int
	StatusReportSeconds  int
	VerboseLogging       bool
}

// Watchdog monitors Docker containers and auto-recovers them.
type Watchdog struct {
	cli          *client.Client
	rootCtx      context.Context
	cfg          Config
	states       map[string]*ContainerState
	targetSet    map[string]struct{}
	listFilters  client.Filters
	eventLimiter *rate.Limiter
	mu           sync.RWMutex
	logger       *slog.Logger
	checkTrigger chan struct{}
	startedAt    time.Time
	configPath   string
	configSource string
	configFileMu sync.Mutex
	eventsMu     sync.Mutex
	events       []Event
}

// SetEnabled updates the global enabled state of the watchdog.
func (w *Watchdog) SetEnabled(enabled bool, requestedBy string, reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cfg.Enabled == enabled {
		return
	}

	w.cfg.Enabled = enabled
	action := "enable"
	if !enabled {
		action = "disable"
	}

	w.logger.Info("watchdog_toggled", "enabled", enabled, "by", requestedBy, "reason", reason)
	w.appendEvent(Event{
		Action:      "watchdog_" + action,
		Container:   "global",
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "ok",
	})
}

func splitList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	trimmed = strings.ReplaceAll(trimmed, ",", " ")
	return strings.Fields(trimmed)
}

// CanonicalContainerName normalizes the container name.
func CanonicalContainerName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.TrimPrefix(name, "/")
	return name
}

func envBool(key string, defaultValue bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func envInt(key string, defaultValue int, minValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		if defaultValue < minValue {
			return minValue
		}
		return defaultValue
	}
	var parsed int
	_, err := fmt.Sscanf(raw, "%d", &parsed)
	if err != nil {
		if defaultValue < minValue {
			return minValue
		}
		return defaultValue
	}
	if parsed < minValue {
		return minValue
	}
	return parsed
}

func loadConfigFromEnv() Config {
	containers := splitList(os.Getenv("WATCHDOG_CONTAINERS"))
	if len(containers) == 0 {
		// Legacy: WATCHDOG_RESTART_CONTAINERS support
		containers = splitList(os.Getenv("WATCHDOG_RESTART_CONTAINERS"))
	}
	if len(containers) > 0 {
		seen := make(map[string]struct{}, len(containers))
		normalized := containers[:0]
		for _, raw := range containers {
			name := CanonicalContainerName(raw)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			normalized = append(normalized, name)
		}
		containers = normalized
	}

	dockerSocket := strings.TrimSpace(os.Getenv("WATCHDOG_DOCKER_SOCKET"))
	if dockerSocket == "" {
		dockerSocket = "/var/run/docker.sock"
	}

	return Config{
		Enabled:              envBool("WATCHDOG_ENABLED", true),
		Containers:           containers,
		IntervalSeconds:      envInt("WATCHDOG_INTERVAL_SECONDS", 30, 1),
		MaxFailures:          envInt("WATCHDOG_MAX_FAILURES", 1, 1),
		RetryChecks:          envInt("WATCHDOG_RETRY_CHECKS", 3, 1),
		RetryIntervalSeconds: envInt("WATCHDOG_RETRY_INTERVAL_SECONDS", 5, 1),
		GraceSeconds:         envInt("WATCHDOG_STARTUP_GRACE_SECONDS", 30, 0),
		CooldownSeconds:      envInt("WATCHDOG_RESTART_COOLDOWN_SECONDS", 120, 0),
		RestartTimeoutSec:    envInt("WATCHDOG_RESTART_TIMEOUT_SECONDS", 30, 5),
		DockerSocket:         dockerSocket,
		UseEvents:            envBool("WATCHDOG_USE_EVENTS", true),
		EventMinIntervalSec:  envInt("WATCHDOG_EVENT_MIN_INTERVAL_SECONDS", 1, 0),
		StatusReportSeconds:  envInt("WATCHDOG_STATUS_REPORT_SECONDS", 60, 0),
		VerboseLogging:       envBool("WATCHDOG_VERBOSE", false),
	}
}

// NewWatchdog creates a new Watchdog instance.
func NewWatchdog(cli *client.Client, cfg Config, configPath string, configSource string, logger *slog.Logger) *Watchdog {
	states := make(map[string]*ContainerState, len(cfg.Containers))
	targetSet := make(map[string]struct{}, len(cfg.Containers))
	listFilters := make(client.Filters)
	for _, name := range cfg.Containers {
		states[name] = &ContainerState{name: name}
		targetSet[name] = struct{}{}
		listFilters = listFilters.Add("name", name)
	}

	var limiter *rate.Limiter
	if cfg.EventMinIntervalSec > 0 {
		limiter = rate.NewLimiter(rate.Every(time.Duration(cfg.EventMinIntervalSec)*time.Second), 1)
	}
	return &Watchdog{
		cli:          cli,
		rootCtx:      context.Background(),
		cfg:          cfg,
		states:       states,
		targetSet:    targetSet,
		listFilters:  listFilters,
		eventLimiter: limiter,
		logger:       logger,
		checkTrigger: make(chan struct{}, 1),
		startedAt:    time.Now(),
		configPath:   strings.TrimSpace(configPath),
		configSource: strings.TrimSpace(configSource),
	}
}

func trimStatusValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return value
	}
	if len(value) > 100 {
		return value[:100] + "..."
	}
	return value
}

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
		status := ""
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
			w.logger.Info("retry_verification_cancelled", "container", state.name)
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
