// Command api starts the MiniClass HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chrismott/miniclass/internal/api"
	"github.com/chrismott/miniclass/internal/config"
	"github.com/chrismott/miniclass/internal/data"
)

const shutdownTimeout = 10 * time.Second

type httpServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(context.Background(), logger); err != nil {
		logger.Error("API server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	signalContext, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load API configuration: %w", err)
	}

	database, err := data.New(signalContext, cfg)
	if err != nil {
		return fmt.Errorf("start database: %w", err)
	}
	defer database.Close()

	server := api.NewServerWithConfig(
		*cfg,
		api.WithDatabase(database),
		api.WithLogger(logger),
	)
	return serve(signalContext, cfg, server.HTTPServer, logger)
}

func serve(ctx context.Context, cfg *config.Config, server httpServer, logger *slog.Logger) error {
	if cfg == nil {
		return errors.New("serve API: configuration is nil")
	}
	if server == nil {
		return errors.New("serve API: HTTP server is nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	address := ":" + cfg.Port
	logger.Info("API server listening",
		slog.String("address", address),
		slog.String("environment", cfg.AppEnv),
		slog.String("version", cfg.AppVersion),
	)

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve API: %w", err)
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		logger.Info("shutting down API server", slog.String("reason", ctx.Err().Error()))
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown API server: %w", err)
		}
		return nil
	}
}
