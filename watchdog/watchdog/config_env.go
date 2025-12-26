package watchdog

import (
	"fmt"
	"os"
	"strings"
)

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
