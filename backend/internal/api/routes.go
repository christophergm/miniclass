package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const apiBasePath = "/api"

// RouterOptions configures the HTTP router independently of process startup.
type RouterOptions struct {
	AllowedOrigins []string
	Logger         *slog.Logger
}

// NewRouter builds the complete API router and middleware chain.
//
// Middleware order, from outermost to innermost, is:
// request ID, real client IP, request logging, panic recovery, CORS, and JSON
// response content type. Keeping the order here makes it visible to callers
// and ensures error handlers receive the same cross-cutting behavior.
func NewRouter(options RouterOptions) chi.Router {
	router := chi.NewRouter()
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		RequestLogger(logger),
		Recoverer(logger),
		cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins(options.AllowedOrigins),
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			AllowCredentials: false,
			MaxAge:           300,
		}),
		JSONContentType,
	)

	router.NotFound(jsonNotFound)
	router.MethodNotAllowed(jsonMethodNotAllowed)

	router.Route(apiBasePath, func(apiRouter chi.Router) {
		apiRouter.NotFound(jsonNotFound)
		apiRouter.MethodNotAllowed(jsonMethodNotAllowed)
		apiRouter.Get("/", apiRoot)
	})
	router.Get(apiBasePath, apiRoot)

	return router
}

func allowedOrigins(origins []string) []string {
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func apiRoot(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(struct {
		Service string `json:"service"`
	}{Service: "miniclass-api"})
}

func jsonNotFound(w http.ResponseWriter, _ *http.Request) {
	JSONError(w, http.StatusNotFound, "route not found")
}

func jsonMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
