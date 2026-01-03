package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/health"
	qapp "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/app"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// Version: 빌드 시 ldflags로 주입됨 (예: -ldflags="-X main.Version=1.0.0")
var Version = "dev"

func main() {
	health.Init(Version)

	logger := bootstrap.NewLogger()
	slog.SetDefault(logger)

	finalLogger, err := bootstrap.RunBotEntrypoint(
		context.Background(),
		logger,
		"twentyq.log",
		qconfig.LoadFromEnv,
		func(cfg *qconfig.Config) qconfig.LogConfig { return cfg.Log },
		qapp.Initialize,
	)
	if err != nil {
		logger = finalLogger
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}
