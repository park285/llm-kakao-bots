package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// ConfigLoader: 설정을 로드하는 함수 타입
type ConfigLoader[C any] func() (*C, error)

// LogConfigGetter: 설정에서 로깅 설정을 추출하는 함수 타입
type LogConfigGetter[C any] func(*C) commonconfig.LogConfig

// AppInitializer: 애플리케이션 초기화 함수 타입 (ServerApp과 정리 함수 반환)
type AppInitializer[C any] func(context.Context, *C, *slog.Logger) (*ServerApp, func(), error)

// RunBotEntrypoint: 봇 애플리케이션의 공통 시작점.
// .env 로드, 설정 로드, 로거 설정, 앱 초기화 및 실행을 담당합니다.
func RunBotEntrypoint[C any](
	ctx context.Context,
	logger *slog.Logger,
	logFileName string,
	loadConfig ConfigLoader[C],
	getLogConfig LogConfigGetter[C],
	initialize AppInitializer[C],
) (*slog.Logger, error) {
	if err := commonconfig.LoadDotenvIfPresent(); err != nil {
		return logger, fmt.Errorf("load dotenv failed: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return logger, fmt.Errorf("load config failed: %w", err)
	}

	var logCfg commonconfig.LogConfig
	if getLogConfig != nil {
		logCfg = getLogConfig(cfg)
	}

	if strings.TrimSpace(logCfg.Dir) != "" {
		fileLogger, logErr := EnableFileLogging(logCfg, logFileName)
		if logErr != nil {
			return logger, fmt.Errorf("enable file logging failed: %w", logErr)
		}
		if fileLogger != nil {
			logger = fileLogger
		}
	}

	serverApp, cleanup, err := initialize(ctx, cfg, logger)
	if err != nil {
		return logger, fmt.Errorf("initialize app failed: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	if err := serverApp.Run(ctx); err != nil {
		return logger, fmt.Errorf("run app failed: %w", err)
	}
	return logger, nil
}
