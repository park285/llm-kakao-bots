package util

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig: 로깅 설정 (로그 디렉토리, 로테이션 정책)
type LogConfig struct {
	Dir string

	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// NewLogger: 콘솔 출력용 기본 slog 로거를 생성한다.
func NewLogger() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
}

// NewLoggerWithLevel: 지정된 레벨로 콘솔 출력용 slog 로거를 생성한다.
func NewLoggerWithLevel(level string) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      parseLogLevel(level),
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
}

// EnableFileLogging: 파일 로깅을 활성화하고, 로그 로테이션이 적용된 로거를 반환한다.
// cfg.Dir이 비어있으면 nil을 반환한다.
func EnableFileLogging(cfg LogConfig, fileName string) (*slog.Logger, error) {
	if cfg.Dir == "" {
		return nil, nil
	}
	if cfg.MaxSizeMB <= 0 || cfg.MaxBackups <= 0 || cfg.MaxAgeDays <= 0 {
		return nil, fmt.Errorf("invalid log config: size=%d backups=%d age_days=%d", cfg.MaxSizeMB, cfg.MaxBackups, cfg.MaxAgeDays)
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, fileName),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	w := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    true,
	}))
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled", "path", logFile.Filename)
	return logger, nil
}

// EnableFileLoggingWithLevel: 지정된 레벨과 파일 로깅을 활성화한다.
func EnableFileLoggingWithLevel(cfg LogConfig, fileName, level string) (*slog.Logger, error) {
	if cfg.Dir == "" {
		return NewLoggerWithLevel(level), nil
	}
	if cfg.MaxSizeMB <= 0 || cfg.MaxBackups <= 0 || cfg.MaxAgeDays <= 0 {
		return nil, fmt.Errorf("invalid log config: size=%d backups=%d age_days=%d", cfg.MaxSizeMB, cfg.MaxBackups, cfg.MaxAgeDays)
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, fileName),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	w := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(tint.NewHandler(w, &tint.Options{
		Level:      parseLogLevel(level),
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    true,
	}))
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled", "path", logFile.Filename)
	return logger, nil
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
