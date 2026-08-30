package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest"
	"github.com/jackc/pgx/v5"
)

// ImportPreviewService is the read-only application boundary for the first
// phase of an import. It deliberately has no write method.
type ImportPreviewService interface {
	Preview(context.Context, string, ids.XID, string, []byte) (ingest.Preview, error)
}

// ImportPreviewInput accepts the exact submitted source document as a raw
// body. The target year is a query parameter because the URL kind is shared
// by sources with different canonical records.
type ImportPreviewInput struct {
	Kind         string `path:"kind" minLength:"1" doc:"Registered import kind, such as roster_json."`
	SchoolYearID string `query:"school_year_id" minLength:"1" doc:"Opaque target school-year identifier."`
	RawBody      []byte
}

type ImportPreviewOutput struct {
	Body ingest.Preview
}

// ImportHandler exposes the stateless preview endpoint. No actor is passed to
// the service because a read-only preview records no audit entry.
type ImportHandler struct {
	service ImportPreviewService
}

func NewImportHandler(service ImportPreviewService) *ImportHandler {
	return &ImportHandler{service: service}
}

func (h *ImportHandler) Preview(ctx context.Context, input *ImportPreviewInput) (*ImportPreviewOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "import preview service is not configured")
	}
	if input == nil || strings.TrimSpace(input.Kind) == "" || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ImportInvalid, "import kind and school year are required")
	}
	preview, err := h.service.Preview(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), strings.TrimSpace(input.Kind), input.RawBody)
	if err != nil {
		return nil, importPreviewProblem(err)
	}
	return &ImportPreviewOutput{Body: preview}, nil
}

func importPreviewProblem(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	case errors.Is(err, ingest.ErrUnknownKind):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "import kind not found")
	case errors.Is(err, ingest.ErrSchoolYearClosed):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot receive an import preview")
	case errors.Is(err, ingest.ErrInvalidSource):
		return problems.New(http.StatusBadRequest, problems.ImportInvalid, "the submitted import document is invalid")
	case errors.Is(err, ingest.ErrUnsupportedKind):
		return problems.New(http.StatusBadRequest, problems.ImportInvalid, "this import kind is registered but not supported in the current phase")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to preview import")
	}
}
