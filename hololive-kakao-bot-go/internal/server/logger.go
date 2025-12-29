package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware: slog 기반 HTTP 접속 로깅 미들웨어 (고성능 최적화)
func LoggerMiddleware(ctx context.Context, logger *slog.Logger, skipPaths ...string) gin.HandlerFunc {
	// 스킵 경로를 맵으로 변환 (O(1) 조회)
	skipMap := make(map[string]bool, len(skipPaths))
	for _, path := range skipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 스킵 경로는 로깅 제외
		if skipMap[path] {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		status := c.Writer.Status()

		// 레벨 결정: 정상 요청은 DEBUG, 4xx는 WARN, 5xx는 ERROR
		level := slog.LevelDebug
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		// 효율화: 해당 레벨이 활성화 상태인지 먼저 확인
		if !logger.Enabled(ctx, level) {
			return
		}

		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		// 기본 필드
		attrs := []slog.Attr{
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.String("ip", clientIP),
			slog.String("ua", truncateUA(userAgent)),
		}

		// 느린 요청(100ms+)만 레이턴시 포함
		if latency >= 100*time.Millisecond {
			attrs = append(attrs, slog.Duration("latency", latency))
		}

		logger.LogAttrs(ctx, level, "HTTP", attrs...)
	}
}

// truncateUA: User-Agent를 적절한 길이로 자름 (로그 가독성)
func truncateUA(ua string) string {
	const maxLen = 80
	if len(ua) > maxLen {
		return ua[:maxLen] + "..."
	}
	return ua
}

// LogDebugf: Debug 레벨 로그를 조건부로 출력 (지연 평가)
func LogDebugf(ctx context.Context, logger *slog.Logger, msg string, attrs ...any) {
	if logger.Enabled(ctx, slog.LevelDebug) {
		logger.Debug(msg, attrs...)
	}
}
