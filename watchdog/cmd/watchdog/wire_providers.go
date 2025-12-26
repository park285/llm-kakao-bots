package main

import (
	"fmt"
	"log/slog"

	"github.com/moby/moby/client"
	watchdog "llm-watchdog/internal/core"
)

// WatchdogConfigMeta 는 타입이다.
type WatchdogConfigMeta struct {
	Source string
	Path   string
}

// DockerClientInitError 는 타입이다.
type DockerClientInitError struct {
	Host string
	Err  error
}

func (e *DockerClientInitError) Error() string {
	return fmt.Sprintf("docker client init failed (host=%s): %v", e.Host, e.Err)
}

func (e *DockerClientInitError) Unwrap() error {
	return e.Err
}

// WatchdogRuntime 는 타입이다.
type WatchdogRuntime struct {
	DockerHost   string
	DockerClient *client.Client
	Watchdog     *watchdog.Watchdog
}

func newDockerHost(cfg watchdog.Config) string {
	return "unix://" + cfg.DockerSocket
}

func newDockerClient(dockerHost string) (*client.Client, error) {
	cli, err := client.New(
		client.WithHost(dockerHost),
	)
	if err != nil {
		return nil, &DockerClientInitError{Host: dockerHost, Err: err}
	}
	return cli, nil
}

func newWatchdog(cli *client.Client, cfg watchdog.Config, meta WatchdogConfigMeta, logger *slog.Logger) *watchdog.Watchdog {
	return watchdog.NewWatchdog(cli, cfg, meta.Path, meta.Source, logger)
}
