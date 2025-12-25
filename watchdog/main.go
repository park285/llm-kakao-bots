package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"llm-watchdog/admin"
	"llm-watchdog/watchdog"

	"github.com/lmittmann/tint"
	"github.com/moby/moby/client"
	"golang.org/x/sync/errgroup"
	"gopkg.in/natefinch/lumberjack.v2"
)

func isLegacyWatchdogLog(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() {
		_ = file.Close()
	}()

	buf := make([]byte, 256)
	n, _ := file.Read(buf)
	if n == 0 {
		return false
	}

	line := strings.TrimSpace(string(buf[:n]))
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}

	if len(line) >= 5 && line[4] == '/' {
		return true
	}
	if strings.Contains(line, "WATCHDOG_") {
		return true
	}
	return false
}

func setupLogger() (*slog.Logger, func()) {
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
	slog.SetDefault(logger)

	logDir := strings.TrimSpace(os.Getenv("LOG_DIR"))
	if logDir == "" {
		return logger, func() {}
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Error("log_dir_create_failed", "dir", logDir, "err", err)
		return logger, func() {}
	}

	logFilePath := filepath.Join(logDir, "watchdog.log")
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    1, // megabytes
		MaxBackups: 0, // keep all
		MaxAge:     0, // keep all
		Compress:   true,
	}

	if isLegacyWatchdogLog(logFilePath) {
		if rotateErr := logFile.Rotate(); rotateErr != nil {
			logger.Warn("log_rotate_failed", "path", logFilePath, "err", rotateErr)
		}
	}

	w := io.MultiWriter(os.Stdout, logFile)
	logger = slog.New(tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    true,
	}))
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled", "path", logFilePath)

	return logger, func() {
		_ = logFile.Close()
	}
}

func main() {
	logger, closeLogger := setupLogger()
	defer closeLogger()

	adminCfg := admin.LoadAdminConfig()
	if err := adminCfg.ValidateForEnable(); err != nil {
		logger.Error("admin_config_invalid", "err", err)
		os.Exit(2)
	}

	cfg, configSource, configPath, err := watchdog.LoadConfigWithSource(logger)
	if err != nil {
		logger.Error("config_load_failed", "path", configPath, "err", err)
		os.Exit(2)
	}

	if !cfg.Enabled && !adminCfg.Enabled {
		logger.Info("watchdog_disabled", "reason", "watchdog_enabled=false")
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		<-ctx.Done()
		return
	}

	if len(cfg.Containers) == 0 && !adminCfg.Enabled {
		logger.Error("no_targets", "reason", "watchdog_containers_empty")
		os.Exit(2)
	}
	if len(cfg.Containers) == 0 {
		logger.Warn("no_targets", "reason", "watchdog_containers_empty_admin_only")
	}

	runtime, err := initializeWatchdogRuntime(cfg, WatchdogConfigMeta{
		Source: configSource,
		Path:   configPath,
	}, logger)
	if err != nil {
		var dockerInitErr *DockerClientInitError
		if errors.As(err, &dockerInitErr) {
			logger.Error("docker_client_init_failed", "host", dockerInitErr.Host, "err", dockerInitErr.Err)
			os.Exit(2)
		}
		logger.Error("init_failed", "err", err)
		os.Exit(2)
	}
	defer func() {
		if err := runtime.DockerClient.Close(); err != nil {
			logger.Warn("docker_client_close_failed", "err", err)
		}
	}()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	ping, err := runtime.DockerClient.Ping(pingCtx, client.PingOptions{})
	pingCancel()
	if err != nil {
		logger.Error("docker_ping_failed", "err", err)
		os.Exit(2)
	}
	logger.Info("docker_connected", "api_version", ping.APIVersion, "os", ping.OSType)

	w := runtime.Watchdog

	logger.Info("watchdog_start",
		"enabled", cfg.Enabled,
		"admin_enabled", adminCfg.Enabled,
		"config_source", configSource,
		"config_path", configPath,
		"containers", cfg.Containers,
		"interval", time.Duration(cfg.IntervalSeconds)*time.Second,
		"max_failures", cfg.MaxFailures,
		"cooldown", time.Duration(cfg.CooldownSeconds)*time.Second,
		"grace", time.Duration(cfg.GraceSeconds)*time.Second,
		"events", cfg.UseEvents,
		"event_min_interval", time.Duration(cfg.EventMinIntervalSec)*time.Second,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.Enabled && cfg.GraceSeconds > 0 {
		logger.Info("watchdog_grace", "seconds", cfg.GraceSeconds)
		select {
		case <-ctx.Done():
			logger.Info("shutdown", "during_grace", true)
			return
		case <-time.After(time.Duration(cfg.GraceSeconds) * time.Second):
		}
	}

	validateCtx, validateCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := w.ValidateContainers(validateCtx); err != nil {
		validateCancel()
		logger.Error("validation_failed", "err", err)
		os.Exit(2)
	}
	validateCancel()

	g, gctx := errgroup.WithContext(ctx)
	w.SetRootContext(gctx)
	if adminCfg.Enabled {
		g.Go(func() error {
			return admin.RunServer(gctx, adminCfg, w, logger)
		})
	}
	if cfg.Enabled {
		g.Go(func() error {
			w.Run(gctx)
			return nil
		})
	} else {
		g.Go(func() error {
			<-gctx.Done()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}
