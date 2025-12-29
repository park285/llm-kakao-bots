package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Serve: HTTP 서버를 시작하고 종료 시그널을 적절히 처리하여 우아하게 종료(Graceful Shutdown)합니다.
func Serve(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server listen failed: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown failed: %w", err)
		}

		err := <-errCh
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server stopped with error: %w", err)
	}
}
