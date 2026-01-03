package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/telemetry"
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

	// OpenTelemetry 설정을 먼저 읽음 (로거 초기화 전에 OTel 활성화 여부 확인)
	// 서비스명: logFileName에서 .log 확장자를 제거하여 사용 (예: twentyq.log -> twentyq-bot)
	serviceName := strings.TrimSuffix(logFileName, ".log") + "-bot"
	telemetryCfg, err := commonconfig.ReadTelemetryConfigFromEnv(serviceName)
	if err != nil {
		return logger, fmt.Errorf("read telemetry config failed: %w", err)
	}

	// 로거 초기화 (OTel 활성화 시 trace_id/span_id 상관관계 추가)
	var logCfg commonconfig.LogConfig
	if getLogConfig != nil {
		logCfg = getLogConfig(cfg)
	}

	if strings.TrimSpace(logCfg.Dir) != "" {
		fileLogger, logErr := EnableFileLoggingWithOTel(logCfg, logFileName, telemetryCfg.Enabled)
		if logErr != nil {
			return logger, fmt.Errorf("enable file logging failed: %w", logErr)
		}
		if fileLogger != nil {
			logger = fileLogger
		}
	}

	// OpenTelemetry Provider 초기화
	otelProvider, err := telemetry.NewProvider(ctx, telemetry.Config{
		Enabled:        telemetryCfg.Enabled,
		ServiceName:    telemetryCfg.ServiceName,
		ServiceVersion: telemetryCfg.ServiceVersion,
		Environment:    telemetryCfg.Environment,
		OTLPEndpoint:   telemetryCfg.OTLPEndpoint,
		OTLPInsecure:   telemetryCfg.OTLPInsecure,
		SampleRate:     telemetryCfg.SampleRate,
	})
	if err != nil {
		return logger, fmt.Errorf("otel init failed: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := otelProvider.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("otel_shutdown_failed", "err", shutdownErr)
		}
	}()

	if otelProvider.IsEnabled() {
		logger.Info("otel_enabled",
			"service", telemetryCfg.ServiceName,
			"endpoint", telemetryCfg.OTLPEndpoint,
			"sample_rate", telemetryCfg.SampleRate,
		)
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
