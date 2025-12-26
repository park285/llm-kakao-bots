package watchdog

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type fileConfig struct {
	Enabled              *bool     `json:"enabled"`
	Containers           *[]string `json:"containers"`
	IntervalSeconds      *int      `json:"intervalSeconds"`
	MaxFailures          *int      `json:"maxFailures"`
	RetryChecks          *int      `json:"retryChecks"`
	RetryIntervalSeconds *int      `json:"retryIntervalSeconds"`
	GraceSeconds         *int      `json:"graceSeconds"`
	CooldownSeconds      *int      `json:"cooldownSeconds"`
	RestartTimeoutSec    *int      `json:"restartTimeoutSec"`
	DockerSocket         *string   `json:"dockerSocket"`
	UseEvents            *bool     `json:"useEvents"`
	EventMinIntervalSec  *int      `json:"eventMinIntervalSec"`
	StatusReportSeconds  *int      `json:"statusReportSeconds"`
	VerboseLogging       *bool     `json:"verboseLogging"`
}

// LoadConfigWithSource 는 동작을 수행한다.
func LoadConfigWithSource(logger *slog.Logger) (Config, string, string, error) {
	cfg := loadConfigFromEnv()

	path := strings.TrimSpace(os.Getenv("WATCHDOG_CONFIG_PATH"))
	if path == "" {
		return cfg, "env", "", nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", path, fmt.Errorf("config file read failed: %w", err)
	}

	var fc fileConfig
	if unmarshalErr := json.Unmarshal(raw, &fc); unmarshalErr != nil {
		return Config{}, "", path, fmt.Errorf("config file json parse failed: %w", unmarshalErr)
	}

	merged, mergeErr := mergeFileConfig(cfg, fc)
	if mergeErr != nil {
		return Config{}, "", path, mergeErr
	}

	logger.Info("watchdog_config_loaded", "source", "file", "path", path)
	return merged, "file", path, nil
}

func mergeFileConfig(base Config, fc fileConfig) (Config, error) {
	out := base

	if fc.Enabled != nil {
		out.Enabled = *fc.Enabled
	}
	if fc.Containers != nil {
		normalized := normalizeContainers(*fc.Containers)
		out.Containers = normalized
	}
	if fc.IntervalSeconds != nil {
		out.IntervalSeconds = max(*fc.IntervalSeconds, 1)
	}
	if fc.MaxFailures != nil {
		out.MaxFailures = max(*fc.MaxFailures, 1)
	}
	if fc.RetryChecks != nil {
		out.RetryChecks = max(*fc.RetryChecks, 1)
	}
	if fc.RetryIntervalSeconds != nil {
		out.RetryIntervalSeconds = max(*fc.RetryIntervalSeconds, 1)
	}
	if fc.GraceSeconds != nil {
		out.GraceSeconds = max(*fc.GraceSeconds, 0)
	}
	if fc.CooldownSeconds != nil {
		out.CooldownSeconds = max(*fc.CooldownSeconds, 0)
	}
	if fc.RestartTimeoutSec != nil {
		out.RestartTimeoutSec = max(*fc.RestartTimeoutSec, 5)
	}
	if fc.DockerSocket != nil {
		socket := strings.TrimSpace(*fc.DockerSocket)
		if socket == "" {
			return Config{}, errors.New("dockerSocket is empty")
		}
		out.DockerSocket = socket
	}
	if fc.UseEvents != nil {
		out.UseEvents = *fc.UseEvents
	}
	if fc.EventMinIntervalSec != nil {
		out.EventMinIntervalSec = max(*fc.EventMinIntervalSec, 0)
	}
	if fc.StatusReportSeconds != nil {
		out.StatusReportSeconds = max(*fc.StatusReportSeconds, 0)
	}
	if fc.VerboseLogging != nil {
		out.VerboseLogging = *fc.VerboseLogging
	}

	return out, nil
}

func normalizeContainers(input []string) []string {
	if len(input) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, raw := range input {
		name := CanonicalContainerName(raw)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
