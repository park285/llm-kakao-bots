package app

import (
	"net/http"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
)

// ProvideFetchProfilesLogger - fetch_profiles 전용 로거
func ProvideFetchProfilesLogger() (*slog.Logger, func(), error) {
	logger := slog.Default()
	cleanup := func() {} // slog는 Sync 필요 없음
	return logger, cleanup, nil
}

// ProvideFetchProfilesHTTPClient - fetch_profiles 전용 HTTP 클라이언트
func ProvideFetchProfilesHTTPClient() *http.Client {
	return &http.Client{Timeout: constants.OfficialProfileConfig.RequestTimeout}
}
