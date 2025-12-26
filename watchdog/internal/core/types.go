package watchdog

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

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
