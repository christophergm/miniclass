package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgconn"
)

// RequestLogger records one structured log entry for every completed request.
// The logger is passed in so applications and tests can choose their output.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			failure := &requestFailure{}
			r = r.WithContext(context.WithValue(r.Context(), requestFailureKey{}, failure))
			recorder := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)

			path := ""
			if route := chi.RouteContext(r.Context()); route != nil {
				path = route.RoutePattern()
			}
			attributes := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", path),
				slog.String("operation_id", failure.operation),
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.Int("status", recorder.statusCode()),
				slog.Duration("duration", time.Since(started)),
			}
			level := slog.LevelInfo
			if recorder.statusCode() >= http.StatusInternalServerError {
				level = slog.LevelError
				if failure.cause != nil {
					attributes = append(attributes, diagnosticAttributes(failure.cause)...)
				}
			}
			logger.LogAttrs(r.Context(), level, "http request", attributes...)
		})
	}
}

// Recoverer turns unexpected handler panics into the same JSON error shape as
// other server errors. The panic is logged with request context and is not
// allowed to terminate the process serving other requests.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic serving request", slog.Any("panic", recovered))
					problems.Write(w, problems.New(http.StatusInternalServerError, problems.InternalError, "internal server error"))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type requestFailureKey struct{}

type requestFailure struct {
	operation string
	cause     error
}

func operationContext(ctx huma.Context, next func(huma.Context)) {
	if failure, ok := ctx.Context().Value(requestFailureKey{}).(*requestFailure); ok {
		failure.operation = ctx.Operation().OperationID
	}
	next(ctx)
}

func captureHandlerError(ctx context.Context, err error) {
	if failure, ok := ctx.Value(requestFailureKey{}).(*requestFailure); ok {
		failure.cause = err
	}
}

func diagnosticAttributes(err error) []slog.Attr {
	if cause := problems.Cause(err); cause != nil {
		err = cause
	} else {
		// Public problem details can contain validation input. A mapped problem
		// without a retained cause supplies only its stable type, not its detail.
		var problem *huma.ErrorModel
		if errors.As(err, &problem) {
			return []slog.Attr{slog.String("problem_type", problem.Type)}
		}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// PostgreSQL detail, hint, and internal query may contain roster values.
		// Keep the database diagnostic, but never serialize the whole PgError.
		return []slog.Attr{
			slog.String("error", pgErr.Error()),
			slog.String("sqlstate", pgErr.Code),
		}
	}
	return []slog.Attr{slog.String("error", err.Error())}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func (r *statusRecorder) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}
