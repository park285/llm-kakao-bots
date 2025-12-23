package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// FetchProfilesRuntime 는 타입이다.
type FetchProfilesRuntime struct {
	Logger     *zap.Logger
	HTTPClient *http.Client

	cleanup func()
}

// Close 는 동작을 수행한다.
func (r *FetchProfilesRuntime) Close() {
	if r != nil && r.cleanup != nil {
		r.cleanup()
	}
}

// BuildFetchProfilesRuntime 는 동작을 수행한다.
func BuildFetchProfilesRuntime(ctx context.Context) (*FetchProfilesRuntime, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	runtime, cleanup, err := InitializeFetchProfilesRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch profiles 런타임 초기화 실패: %w", err)
	}
	runtime.cleanup = cleanup

	return runtime, nil
}
