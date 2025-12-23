package app

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
)

// ProvideFetchProfilesLogger - fetch_profiles 전용 로거
func ProvideFetchProfilesLogger() (*zap.Logger, func(), error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, nil, fmt.Errorf("fetch profiles logger 초기화 실패: %w", err)
	}
	cleanup := func() { _ = logger.Sync() }
	return logger, cleanup, nil
}

// ProvideFetchProfilesHTTPClient - fetch_profiles 전용 HTTP 클라이언트
func ProvideFetchProfilesHTTPClient() *http.Client {
	return &http.Client{Timeout: constants.OfficialProfileConfig.RequestTimeout}
}
