package shared

import (
	"log/slog"
)

// LogError: 에러를 경고 레벨로 로깅합니다.
func LogError(logger *slog.Logger, domain string, err error) {
	if logger == nil || err == nil {
		return
	}
	logger.Warn(domain+"_error", "err", err)
}
