package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapLoggerMiddleware Zap 기반 HTTP 접속 로깅 미들웨어
func ZapLoggerMiddleware(logger *zap.Logger, skipPaths ...string) gin.HandlerFunc {
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
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		// 상태 코드에 따른 로그 레벨 결정
		logFunc := logger.Info
		if status >= 500 {
			logFunc = logger.Error
		} else if status >= 400 {
			logFunc = logger.Warn
		}

		// 기본 필드
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.String("ip", clientIP),
			zap.String("ua", truncateUA(userAgent)),
		}

		// 느린 요청(100ms+)만 레이턴시 포함
		if latency >= 100*time.Millisecond {
			fields = append(fields, zap.Duration("latency", latency))
		}

		logFunc("HTTP", fields...)
	}
}

// truncateUA User-Agent를 적절한 길이로 자름 (로그 가독성)
func truncateUA(ua string) string {
	const maxLen = 80
	if len(ua) > maxLen {
		return ua[:maxLen] + "..."
	}
	return ua
}
