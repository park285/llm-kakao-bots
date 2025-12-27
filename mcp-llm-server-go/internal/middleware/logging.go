package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger 는 HTTP 요청 로그 미들웨어다.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return func(c *gin.Context) {
		startedAt := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path

		defer func() {
			status := c.Writer.Status()
			if status < http.StatusBadRequest && len(c.Errors) == 0 && isNoisyInfoPath(path) {
				return
			}

			latency := time.Since(startedAt)
			fields := []any{
				"request_id", GetRequestID(c),
				"method", method,
				"path", path,
				"status", status,
				"latency", latency,
				"bytes", c.Writer.Size(),
			}
			if len(c.Errors) > 0 {
				fields = append(fields, "errors", c.Errors.String())
			}

			switch {
			case status >= 500:
				logger.Error("http_request", fields...)
			case status >= 400:
				logger.Warn("http_request", fields...)
			default:
				logger.Debug("http_request", fields...)
			}
		}()

		c.Next()
	}
}

func isNoisyInfoPath(path string) bool {
	switch path {
	case "/health", "/health/ready", "/health/models":
		return true
	default:
		return false
	}
}
