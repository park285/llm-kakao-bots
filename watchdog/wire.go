//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"llm-watchdog/watchdog"

	"github.com/google/wire"
)

func initializeWatchdogRuntime(cfg watchdog.Config, meta WatchdogConfigMeta, logger *slog.Logger) (*WatchdogRuntime, error) {
	wire.Build(
		newDockerHost,
		newDockerClient,
		newWatchdog,
		wire.Struct(new(WatchdogRuntime), "*"),
	)
	return nil, nil
}
