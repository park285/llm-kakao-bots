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

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- app.Server.ListenAndServe()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case sig := <-signalCh:
		app.Logger.Info("http_server_shutdown_signal", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := app.Server.Shutdown(shutdownCtx); shutdownErr != nil {
			app.Logger.Error("http_server_shutdown_failed", "err", shutdownErr)
			_ = app.Server.Close()
		}

		err = <-serverErr
	case err = <-serverErr:
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Error("http_server_failed", "err", err)
		os.Exit(1)
	}
}
