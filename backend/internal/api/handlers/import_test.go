package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/chrismott/miniclass/internal/ingest"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/require"
)

func TestImportProblemCarriesTheParserReason(t *testing.T) {
	// The service wraps the parser's message, and the response is the only
	// place an organiser can read it: a refused upload they cannot diagnose is
	// the same as no importer at all.
	cause := fmt.Errorf("preview import: %w", &ingest.InvalidSourceError{
		Kind: ingest.KindRosterJSON, Reason: "roster: adult record 1 has no opaque id",
	})

	for name, problem := range map[string]error{
		"preview": importPreviewProblem(cause),
		"commit":  importCommitProblem(cause),
	} {
		var model *huma.ErrorModel
		require.True(t, errors.As(problem, &model), name)
		require.Equal(t, http.StatusBadRequest, model.Status, name)
		require.Equal(t, "import-invalid", model.Type, name)
		require.Equal(t, "the submitted roster_json document is invalid: roster: adult record 1 has no opaque id", model.Detail, name)
	}

	require.True(t, errors.Is(cause, ingest.ErrInvalidSource), "classification by sentinel must still work")
}

func TestImportLogsTheCauseBehindAnInternalError(t *testing.T) {
	var recorded bytes.Buffer
	handler := NewImportHandler(nil).WithLogger(slog.New(slog.NewTextHandler(&recorded, nil)))
	cause := errors.New("list homerooms: column \"external_identifier\" does not exist")

	problem := handler.logged(context.Background(), "preview", ingest.KindRosterJSON, cause, importPreviewProblem(cause))

	var model *huma.ErrorModel
	require.True(t, errors.As(problem, &model))
	require.Equal(t, http.StatusInternalServerError, model.Status)
	require.Equal(t, "unable to preview import", model.Detail, "the response stays generic")
	require.Contains(t, recorded.String(), "import failed")
	require.Contains(t, recorded.String(), "external_identifier")
	require.Contains(t, recorded.String(), "phase=preview")
}

func TestImportDoesNotLogAnActionableProblem(t *testing.T) {
	var recorded bytes.Buffer
	handler := NewImportHandler(nil).WithLogger(slog.New(slog.NewTextHandler(&recorded, nil)))
	cause := &ingest.InvalidSourceError{Kind: ingest.KindGradesCSV, Reason: "grades: no student_name column"}

	handler.logged(context.Background(), "preview", ingest.KindGradesCSV, cause, importPreviewProblem(cause))

	require.Empty(t, recorded.String(), "a 400 already tells the caller what to fix")
}
