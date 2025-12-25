package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kapu/hololive-kakao-bot-go/internal/app"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// slog 기반 로거 초기화 (파일 로깅 포함)
	logger, err := util.EnableFileLoggingWithLevel(util.LogConfig{
		Dir:        cfg.Logging.Dir,
		MaxSizeMB:  cfg.Logging.MaxSizeMB,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAgeDays: cfg.Logging.MaxAgeDays,
		Compress:   cfg.Logging.Compress,
	}, "bot.log", cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Hololive KakaoTalk Bot starting...",
		slog.String("version", cfg.Version),
		slog.String("log_level", cfg.Logging.Level),
	)

	buildCtx, buildCancel := context.WithTimeout(context.Background(), constants.AppTimeout.Build)
	runtime, err := app.BuildRuntime(buildCtx, cfg, logger)
	buildCancel()
	if err != nil {
		logger.Error("Failed to assemble application services", slog.Any("error", err))
		os.Exit(1)
	}
	defer runtime.Close()

	runtime.Run()
}
