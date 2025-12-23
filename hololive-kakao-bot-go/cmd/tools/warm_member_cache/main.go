package main

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/app"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		return
	}

	logger, err := util.NewLogger(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		fmt.Printf("failed to initialize logger: %v\n", err)
		return
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("Manual member list cache refresh started")

	_, cleanup, err := app.InitializeWarmMemberCache(ctx, cfg, logger)
	if err != nil {
		logger.Error("Manual cache refresh failed", zap.Error(err))
		return
	}
	defer cleanup()

	logger.Info("Manual member list cache refresh completed successfully")
}
