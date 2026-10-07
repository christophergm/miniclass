package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	solverservice "github.com/chrismott/miniclass/internal/solver"
	"github.com/chrismott/miniclass/internal/solverclient"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestSolveRunLogsInternalErrorCause(t *testing.T) {
	for _, operation := range []string{"start", "get", "rerun"} {
		t.Run(operation, func(t *testing.T) {
			var recorded bytes.Buffer
			handler := NewSolveRunHandler(nil).WithLogger(slog.New(slog.NewTextHandler(&recorded, nil)))
			ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "test-request")
			cause := fmt.Errorf("compile solver snapshot: %w", errors.New("participant references an unknown grade level"))
			input := SessionPathInput{SchoolYearID: "test-year", ProgramID: "test-program", SessionID: "test-session"}

			problem := handler.problem(ctx, operation, "test-organization", input, cause)

			var model *huma.ErrorModel
			require.True(t, errors.As(problem, &model))
			require.Equal(t, http.StatusInternalServerError, model.Status)
			require.Equal(t, "unable to process solve run", model.Detail)
			require.NotContains(t, model.Detail, "grade level")
			require.Contains(t, recorded.String(), "solve run failed")
			require.Contains(t, recorded.String(), cause.Error())
			require.Contains(t, recorded.String(), "operation="+operation)
			require.Contains(t, recorded.String(), "request_id=test-request")
			require.Contains(t, recorded.String(), "organization_id=test-organization")
			require.Contains(t, recorded.String(), "school_year_id=test-year")
			require.Contains(t, recorded.String(), "program_id=test-program")
			require.Contains(t, recorded.String(), "session_id=test-session")
		})
	}
}

func TestSolveRunDoesNotLogClassifiedErrors(t *testing.T) {
	for _, cause := range []error{
		solverclient.ErrUnavailable,
		pgx.ErrNoRows,
		solverservice.ErrInputFingerprintMismatch,
		solverservice.ErrDraftRevisionChanged,
	} {
		t.Run(cause.Error(), func(t *testing.T) {
			var recorded bytes.Buffer
			handler := NewSolveRunHandler(nil).WithLogger(slog.New(slog.NewTextHandler(&recorded, nil)))

			problem := handler.problem(context.Background(), "start", "test-organization", SessionPathInput{}, cause)

			var model *huma.ErrorModel
			require.True(t, errors.As(problem, &model))
			require.NotEqual(t, http.StatusInternalServerError, model.Status)
			require.Empty(t, recorded.String())
		})
	}
}
