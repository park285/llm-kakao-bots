package watchdog

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"golang.org/x/time/rate"
)

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
