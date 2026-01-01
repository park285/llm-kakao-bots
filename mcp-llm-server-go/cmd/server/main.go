package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/di"
)

func main() {
	app, err := di.InitializeApp()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer func() {
		app.Close()
	}()

	config.LogEnvStatus(app.Config, app.Logger)
	app.Logger.Info(
		"http_server_start",
		"host", app.Config.HTTP.Host,
		"port", app.Config.HTTP.Port,
		"http2", app.Config.HTTP.HTTP2Enabled,
	)

	httpServerErr := make(chan error, 1)
	go func() {
		httpServerErr <- app.Server.ListenAndServe()
	}()

	grpcEnabled := app.GRPCServer != nil && app.GRPCListener != nil
	var grpcServerErr chan error
	if grpcEnabled {
		app.Logger.Info(
			"grpc_server_start",
			"addr", app.GRPCListener.Addr().String(),
			"tls_enabled", false,
		)

		grpcServerErr = make(chan error, 1)
		go func() {
			grpcServerErr <- app.GRPCServer.Serve(app.GRPCListener)
		}()

		// UDS listener가 있으면 별도 goroutine으로 serve함
		if app.GRPCUDSListener != nil {
			app.Logger.Info(
				"grpc_uds_server_start",
				"addr", app.GRPCUDSListener.Addr().String(),
			)
			go func() {
				// UDS serve 에러는 TCP와 동일한 서버이므로 별도 채널 불필요
				// 서버 종료 시 자동으로 에러 반환됨
				_ = app.GRPCServer.Serve(app.GRPCUDSListener)
			}()
		}
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	stopGRPC := func(ctx context.Context) {
		if !grpcEnabled {
			return
		}

		done := make(chan struct{})
		go func() {
			app.GRPCServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			app.GRPCServer.Stop()
		}
	}

	select {
	case err = <-httpServerErr:
		if grpcEnabled {
			app.Logger.Error("http_server_failed", "err", err)
			app.GRPCServer.Stop()
			<-grpcServerErr
		}
	case sig := <-signalCh:
		app.Logger.Info("http_server_shutdown_signal", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stopGRPC(shutdownCtx)
		if shutdownErr := app.Server.Shutdown(shutdownCtx); shutdownErr != nil {
			app.Logger.Error("http_server_shutdown_failed", "err", shutdownErr)
			_ = app.Server.Close()
		}

		err = <-httpServerErr
		if grpcEnabled {
			<-grpcServerErr
		}
	case err = <-grpcServerErr:
		app.Logger.Error("grpc_server_failed", "err", err)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if shutdownErr := app.Server.Shutdown(shutdownCtx); shutdownErr != nil {
			app.Logger.Error("http_server_shutdown_failed", "err", shutdownErr)
			_ = app.Server.Close()
		}

		httpErr := <-httpServerErr
		if err == nil {
			err = httpErr
		}
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		os.Exit(1)
	}
}
