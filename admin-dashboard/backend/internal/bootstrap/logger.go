package bootstrap

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"github.com/park285/llm-kakao-bots/admin-dashboard/internal/logging"
)

// NewLogger: 기본 stdout 로거를 생성합니다.
func NewLogger() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
}

// NewLoggerWithConfig: 설정 기반 로거를 생성합니다 (OTel 지원).
func NewLoggerWithConfig(cfg logging.Config, enableOTel bool) (*slog.Logger, error) {
	logger, err := logging.NewLoggerWithOTel(cfg, enableOTel)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	return logger, nil
}

// LoggingConfigFromEnv: 환경 변수에서 로깅 설정을 로드합니다.
func LoggingConfigFromEnv(logDir, logLevel string) logging.Config {
	cfg := logging.DefaultConfig()
	if logDir != "" {
		cfg.Dir = logDir
	}
	if logLevel != "" {
		cfg.Level = logLevel
	}
	return cfg
}
