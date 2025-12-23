package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

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

	logger, err := util.NewLogger(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("Hololive KakaoTalk Bot starting...",
		zap.String("version", cfg.Version),
		zap.String("log_level", cfg.Logging.Level),
	)

	buildCtx, buildCancel := context.WithTimeout(context.Background(), constants.AppTimeout.Build)
	runtime, err := app.BuildRuntime(buildCtx, cfg, logger)
	buildCancel()
	if err != nil {
		logger.Error("Failed to assemble application services", zap.Error(err))
		os.Exit(1)
	}
	defer runtime.Close()

	runtime.Run()
}
