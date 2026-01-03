package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// NewLogger: 기본 slog 로거를 생성합니다. (stdout, tint 핸들러 사용)
func NewLogger() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
}

// EnableFileLogging: 파일 로깅을 활성화하고, 파일과 stdout에 동시에 출력하는 로거를 반환합니다.
// OTel이 활성화된 경우 로그에 trace_id/span_id가 자동으로 추가됩니다.
func EnableFileLogging(cfg commonconfig.LogConfig, fileName string) (*slog.Logger, error) {
	return EnableFileLoggingWithOTel(cfg, fileName, false)
}

// EnableFileLoggingWithOTel: OTel 상관관계 기능을 포함한 파일 로깅을 활성화합니다.
// enableOTel이 true면 로그에 trace_id/span_id가 자동으로 추가됩니다.
func EnableFileLoggingWithOTel(cfg commonconfig.LogConfig, fileName string, enableOTel bool) (*slog.Logger, error) {
	logDir := strings.TrimSpace(cfg.Dir)
	if logDir == "" {
		return nil, nil
	}
	if cfg.MaxSizeMB <= 0 || cfg.MaxBackups <= 0 || cfg.MaxAgeDays <= 0 {
		return nil, fmt.Errorf("invalid log config: size=%d backups=%d age_days=%d", cfg.MaxSizeMB, cfg.MaxBackups, cfg.MaxAgeDays)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}

	// 서비스별 로그 파일
	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fileName),
		MaxSize:    cfg.MaxSizeMB,  // megabytes
		MaxBackups: cfg.MaxBackups, // files
		MaxAge:     cfg.MaxAgeDays, // days
		Compress:   cfg.Compress,
	}

	// 통합 로그 파일 (combined.log) - 모든 서비스의 로그가 여기에 모임
	combinedLogFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "combined.log"),
		MaxSize:    cfg.MaxSizeMB * 3, // 서비스 합산이므로 더 큰 용량
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	// stdout + 서비스별 로그 + 통합 로그에 동시 출력
	w := io.MultiWriter(os.Stdout, logFile, combinedLogFile)

	// 기본 핸들러 생성
	var handler = tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    true,
	})

	// OTel 활성화 시 trace_id/span_id 자동 추가
	if enableOTel {
		handler = NewOTelHandler(handler)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled",
		slog.String("path", logFile.Filename),
		slog.String("combined", combinedLogFile.Filename),
		slog.Bool("otel_correlation", enableOTel),
	)
	return logger, nil
}
