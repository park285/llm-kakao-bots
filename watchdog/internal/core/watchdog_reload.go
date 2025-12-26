package watchdog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"golang.org/x/time/rate"
)

// ReloadResult 는 타입이다.
type ReloadResult struct {
	LoadedAt               time.Time `json:"loadedAt"`
	Source                 string    `json:"source"`
	Path                   string    `json:"path,omitempty"`
	AppliedFields          []string  `json:"appliedFields,omitempty"`
	RequiresRestartFields  []string  `json:"requiresRestartFields,omitempty"`
	EffectiveConfigSummary any       `json:"effectiveConfigSummary,omitempty"`
}

// ErrConfigPathNotSet 는 패키지 변수다.
var ErrConfigPathNotSet = errors.New("WATCHDOG_CONFIG_PATH is not set")

// ReloadConfigFromFile 는 동작을 수행한다.
func (w *Watchdog) ReloadConfigFromFile(ctx context.Context) (ReloadResult, error) {
	w.configFileMu.Lock()
	defer w.configFileMu.Unlock()
	return w.reloadConfigFromFileUnlocked(ctx)
}

func (w *Watchdog) reloadConfigFromFileUnlocked(ctx context.Context) (ReloadResult, error) {
	path := w.GetConfigPath()
	if strings.TrimSpace(path) == "" {
		return ReloadResult{}, ErrConfigPathNotSet
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return ReloadResult{}, fmt.Errorf("config file read failed: %w", err)
	}

	var fc fileConfig
	if unmarshalErr := json.Unmarshal(raw, &fc); unmarshalErr != nil {
		return ReloadResult{}, fmt.Errorf("config file json parse failed: %w", unmarshalErr)
	}

	base := loadConfigFromEnv()
	next, mergeErr := mergeFileConfig(base, fc)
	if mergeErr != nil {
		return ReloadResult{}, mergeErr
	}

	applied, requiresRestart := w.applyRuntimeConfig(next)
	if len(applied) > 0 {
		w.TriggerHealthCheck()
	}

	return ReloadResult{
		LoadedAt:              time.Now(),
		Source:                "file",
		Path:                  path,
		AppliedFields:         applied,
		RequiresRestartFields: requiresRestart,
		EffectiveConfigSummary: map[string]any{
			"enabled":              w.GetConfig().Enabled,
			"containers":           w.GetConfig().Containers,
			"intervalSeconds":      w.GetConfig().IntervalSeconds,
			"maxFailures":          w.GetConfig().MaxFailures,
			"retryChecks":          w.GetConfig().RetryChecks,
			"retryIntervalSeconds": w.GetConfig().RetryIntervalSeconds,
			"cooldownSeconds":      w.GetConfig().CooldownSeconds,
			"restartTimeoutSec":    w.GetConfig().RestartTimeoutSec,
			"useEvents":            w.GetConfig().UseEvents,
			"eventMinIntervalSec":  w.GetConfig().EventMinIntervalSec,
			"statusReportSeconds":  w.GetConfig().StatusReportSeconds,
			"verboseLogging":       w.GetConfig().VerboseLogging,
		},
	}, nil
}

func (w *Watchdog) applyRuntimeConfig(next Config) ([]string, []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	old := w.cfg
	applied := make([]string, 0, 8)
	requiresRestart := checkImmutableFields(old, &next)

	applied = append(applied, w.applySimpleFields(old, next)...)
	applied = append(applied, w.applyContainersList(old, next)...)

	if old.UseEvents != next.UseEvents {
		w.cfg.UseEvents = next.UseEvents
		applied = append(applied, "useEvents")
	}

	w.configSource = "file"
	return applied, requiresRestart
}

func checkImmutableFields(old Config, next *Config) []string {
	requiresRestart := make([]string, 0, 4)

	if strings.TrimSpace(old.DockerSocket) != strings.TrimSpace(next.DockerSocket) {
		requiresRestart = append(requiresRestart, "dockerSocket")
		next.DockerSocket = old.DockerSocket
	}

	if old.GraceSeconds != next.GraceSeconds {
		requiresRestart = append(requiresRestart, "graceSeconds")
		next.GraceSeconds = old.GraceSeconds
	}

	if !old.UseEvents && next.UseEvents {
		requiresRestart = append(requiresRestart, "useEvents")
	}

	return requiresRestart
}

func (w *Watchdog) applySimpleFields(old, next Config) []string {
	applied := make([]string, 0, 10)

	if old.Enabled != next.Enabled {
		w.cfg.Enabled = next.Enabled
		applied = append(applied, "enabled")
	}
	if old.IntervalSeconds != next.IntervalSeconds {
		w.cfg.IntervalSeconds = next.IntervalSeconds
		applied = append(applied, "intervalSeconds")
	}
	if old.MaxFailures != next.MaxFailures {
		w.cfg.MaxFailures = next.MaxFailures
		applied = append(applied, "maxFailures")
	}
	if old.RetryChecks != next.RetryChecks {
		w.cfg.RetryChecks = next.RetryChecks
		applied = append(applied, "retryChecks")
	}
	if old.RetryIntervalSeconds != next.RetryIntervalSeconds {
		w.cfg.RetryIntervalSeconds = next.RetryIntervalSeconds
		applied = append(applied, "retryIntervalSeconds")
	}
	if old.CooldownSeconds != next.CooldownSeconds {
		w.cfg.CooldownSeconds = next.CooldownSeconds
		applied = append(applied, "cooldownSeconds")
	}
	if old.RestartTimeoutSec != next.RestartTimeoutSec {
		w.cfg.RestartTimeoutSec = next.RestartTimeoutSec
		applied = append(applied, "restartTimeoutSec")
	}
	if old.EventMinIntervalSec != next.EventMinIntervalSec {
		w.cfg.EventMinIntervalSec = next.EventMinIntervalSec
		applied = append(applied, "eventMinIntervalSec")
		var limiter *rate.Limiter
		if next.EventMinIntervalSec > 0 {
			limiter = rate.NewLimiter(rate.Every(time.Duration(next.EventMinIntervalSec)*time.Second), 1)
		}
		w.eventLimiter = limiter
	}
	if old.StatusReportSeconds != next.StatusReportSeconds {
		w.cfg.StatusReportSeconds = next.StatusReportSeconds
		applied = append(applied, "statusReportSeconds")
	}
	if old.VerboseLogging != next.VerboseLogging {
		w.cfg.VerboseLogging = next.VerboseLogging
		applied = append(applied, "verboseLogging")
	}

	return applied
}

func (w *Watchdog) applyContainersList(old, next Config) []string {
	if reflect.DeepEqual(old.Containers, next.Containers) {
		return nil
	}

	newStates := make(map[string]*ContainerState, len(next.Containers))
	newSet := make(map[string]struct{}, len(next.Containers))
	newFilters := make(client.Filters)
	for _, name := range next.Containers {
		if name == "" {
			continue
		}
		if existing, ok := w.states[name]; ok {
			newStates[name] = existing
		} else {
			newStates[name] = &ContainerState{name: name}
		}
		newSet[name] = struct{}{}
		newFilters = newFilters.Add("name", name)
	}
	w.states = newStates
	w.targetSet = newSet
	w.listFilters = newFilters
	w.cfg.Containers = next.Containers

	return []string{"containers"}
}
