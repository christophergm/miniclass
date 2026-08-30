package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/config"
	"github.com/chrismott/miniclass/internal/identity"
)

const (
	defaultServerAddress          = ":8080"
	defaultServerVersion          = "0.1.0"
	defaultInvitationClaimBaseURL = "http://localhost:5173/claim"
)

// ServerOptions controls construction of a Server without starting a process.
type ServerOptions struct {
	Address                string
	AllowedOrigins         []string
	Database               handlers.DatabasePinger
	Identity               auth.AccountResolver
	Claimer                handlers.InvitationClaimer
	Administrators         handlers.AdministratorManager
	InvitationClaimBaseURL string
	SchoolYears            handlers.SchoolYearService
	AuditLog               handlers.AuditLogReader
	Vocabularies           handlers.VocabularyService
	Adults                 handlers.AdultService
	Students               handlers.StudentService
	GuardianRelationships  handlers.GuardianRelationshipService
	ImportPreview          handlers.ImportPreviewService
	ImportCommit           handlers.ImportCommitService
	Verifier               auth.Verifier
	Logger                 *slog.Logger
	TrustedProxyCIDRs      []string
	ReadTimeout            time.Duration
	ReadHeaderTimeout      time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	Version                string
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
		Address:                defaultServerAddress,
		Version:                defaultServerVersion,
		InvitationClaimBaseURL: defaultInvitationClaimBaseURL,
		ReadHeaderTimeout:      5 * time.Second,
		ReadTimeout:            15 * time.Second,
		WriteTimeout:           15 * time.Second,
		IdleTimeout:            60 * time.Second,
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
		AllowedOrigins:         settings.AllowedOrigins,
		Database:               settings.Database,
		Identity:               settings.Identity,
		Claimer:                settings.Claimer,
		Administrators:         settings.Administrators,
		InvitationClaimBaseURL: settings.InvitationClaimBaseURL,
		SchoolYears:            settings.SchoolYears,
		AuditLog:               settings.AuditLog,
		Vocabularies:           settings.Vocabularies,
		Adults:                 settings.Adults,
		Students:               settings.Students,
		GuardianRelationships:  settings.GuardianRelationships,
		ImportPreview:          settings.ImportPreview,
		ImportCommit:           settings.ImportCommit,
		Verifier:               settings.Verifier,
		Logger:                 settings.Logger,
		TrustedProxyCIDRs:      settings.TrustedProxyCIDRs,
		Version:                settings.Version,
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
	options = append([]ServerOption{
		WithAddress(":" + cfg.Port),
		WithTrustedProxyCIDRs(cfg.TrustedProxyCIDRs...),
		WithVersion(cfg.AppVersion),
	}, options...)
	if strings.TrimSpace(cfg.InvitationClaimBaseURL) != "" {
		options = append(options, WithInvitationClaimBaseURL(cfg.InvitationClaimBaseURL))
	}
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

// WithAuditLog supplies the tenant-scoped audit log reader.
func WithAuditLog(reader handlers.AuditLogReader) ServerOption {
	return func(options *ServerOptions) { options.AuditLog = reader }
}

// WithIdentity supplies the local identity resolver used by authentication.
func WithIdentity(store *identity.Store) ServerOption {
	return func(options *ServerOptions) {
		options.Identity = store
		options.Claimer = store
		options.Administrators = store
	}
}

// WithAccountResolver supplies a testable or alternate local membership
// resolver without exposing generated SQL to the API package.
func WithAccountResolver(resolver auth.AccountResolver) ServerOption {
	return func(options *ServerOptions) { options.Identity = resolver }
}

// WithInvitationClaimer supplies the invitation binding use case.
func WithInvitationClaimer(claimer handlers.InvitationClaimer) ServerOption {
	return func(options *ServerOptions) { options.Claimer = claimer }
}

// WithAdministratorManager supplies the owner-only administrator use case.
func WithAdministratorManager(manager handlers.AdministratorManager) ServerOption {
	return func(options *ServerOptions) { options.Administrators = manager }
}

// WithInvitationClaimBaseURL sets the absolute URL used in generated admin
// invitation links.
func WithInvitationClaimBaseURL(baseURL string) ServerOption {
	return func(options *ServerOptions) { options.InvitationClaimBaseURL = baseURL }
}

// WithSchoolYears supplies the school-year service used by lifecycle routes.
func WithSchoolYears(service handlers.SchoolYearService) ServerOption {
	return func(options *ServerOptions) { options.SchoolYears = service }
}

// WithVocabularies supplies the grade and homeroom vocabulary service.
func WithVocabularies(service handlers.VocabularyService) ServerOption {
	return func(options *ServerOptions) { options.Vocabularies = service }
}

// WithAdults supplies the adult roster service used by CRUD routes.
func WithAdults(service handlers.AdultService) ServerOption {
	return func(options *ServerOptions) { options.Adults = service }
}

// WithStudents supplies the student roster service used by CRUD routes.
func WithStudents(service handlers.StudentService) ServerOption {
	return func(options *ServerOptions) { options.Students = service }
}

// WithGuardianRelationships supplies the relationship service used by guardian routes.
func WithGuardianRelationships(service handlers.GuardianRelationshipService) ServerOption {
	return func(options *ServerOptions) { options.GuardianRelationships = service }
}

// WithImportPreview supplies the read-only import preview service.
func WithImportPreview(service handlers.ImportPreviewService) ServerOption {
	return func(options *ServerOptions) { options.ImportPreview = service }
}

// WithImportCommit supplies the mutating import service.
func WithImportCommit(service handlers.ImportCommitService) ServerOption {
	return func(options *ServerOptions) { options.ImportCommit = service }
}

// WithVerifier supplies the configured bearer-token verifier.
func WithVerifier(verifier auth.Verifier) ServerOption {
	return func(options *ServerOptions) { options.Verifier = verifier }
}

// WithVersion sets the application version reported by the health endpoint.
func WithVersion(version string) ServerOption {
	return func(options *ServerOptions) { options.Version = version }
}

// WithLogger sets the structured logger used by request and panic middleware.
func WithLogger(logger *slog.Logger) ServerOption {
	return func(options *ServerOptions) { options.Logger = logger }
}

// WithTrustedProxyCIDRs configures the proxy networks allowed to supply
// forwarding headers for the effective request address.
func WithTrustedProxyCIDRs(cidrs ...string) ServerOption {
	return func(options *ServerOptions) { options.TrustedProxyCIDRs = cidrs }
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
