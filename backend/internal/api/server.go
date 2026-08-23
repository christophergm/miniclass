package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/config"
)

const (
	defaultServerAddress = ":8080"
	defaultServerVersion = "0.1.0"
)

// ServerOptions controls construction of a Server without starting a process.
type ServerOptions struct {
	Address           string
	AllowedOrigins    []string
	Database          handlers.DatabasePinger
	Logger            *slog.Logger
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	Version           string
}

// Server owns the HTTP handler and server settings used by the API process.
// Callers decide when and how to start HTTPServer, which keeps construction
// straightforward to exercise in unit tests and to compose in main.go.
type Server struct {
	Router     http.Handler
	HTTPServer *http.Server
}

// NewServer constructs a server with safe local-development defaults.
func NewServer(options ...ServerOption) *Server {
	settings := ServerOptions{
		Address:           defaultServerAddress,
		Version:           defaultServerVersion,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	if settings.Address == "" {
		settings.Address = defaultServerAddress
	}
	if settings.Logger == nil {
		settings.Logger = slog.Default()
	}

	router := NewRouter(RouterOptions{
		AllowedOrigins: settings.AllowedOrigins,
		Database:       settings.Database,
		Logger:         settings.Logger,
		Version:        settings.Version,
	})
	httpServer := &http.Server{
		Addr:              settings.Address,
		Handler:           router,
		ReadTimeout:       settings.ReadTimeout,
		ReadHeaderTimeout: settings.ReadHeaderTimeout,
		WriteTimeout:      settings.WriteTimeout,
		IdleTimeout:       settings.IdleTimeout,
	}

	return &Server{Router: router, HTTPServer: httpServer}
}

// NewServerWithConfig constructs a server using the configured application
// port while preserving the same independent construction behavior.
func NewServerWithConfig(cfg config.Config, options ...ServerOption) *Server {
	options = append([]ServerOption{WithAddress(":" + cfg.Port), WithVersion(cfg.AppVersion)}, options...)
	return NewServer(options...)
}

// ServerOption customizes NewServer.
type ServerOption func(*ServerOptions)

// WithAddress sets the address used by the HTTP server.
func WithAddress(address string) ServerOption {
	return func(options *ServerOptions) { options.Address = address }
}

// WithAllowedOrigins sets the origins permitted by CORS. An empty list uses
// the development default of allowing all origins.
func WithAllowedOrigins(origins ...string) ServerOption {
	return func(options *ServerOptions) { options.AllowedOrigins = origins }
}

// WithDatabase sets the dependency checked by the health endpoint.
func WithDatabase(database handlers.DatabasePinger) ServerOption {
	return func(options *ServerOptions) { options.Database = database }
}

// WithVersion sets the application version reported by the health endpoint.
func WithVersion(version string) ServerOption {
	return func(options *ServerOptions) { options.Version = version }
}

// WithLogger sets the structured logger used by request and panic middleware.
func WithLogger(logger *slog.Logger) ServerOption {
	return func(options *ServerOptions) { options.Logger = logger }
}

// Handler returns the router for callers that need an http.Handler without
// reaching into the server's HTTPServer field.
func (s *Server) Handler() http.Handler {
	if s == nil {
		return http.NotFoundHandler()
	}
	return s.Router
}

// Shutdown forwards graceful shutdown to the underlying HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.HTTPServer == nil {
		return nil
	}
	return s.HTTPServer.Shutdown(ctx)
}
