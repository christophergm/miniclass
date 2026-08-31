package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

// ImportPreviewService is the read-only application boundary for the first
// phase of an import. It deliberately has no write method.
type ImportPreviewService interface {
	Preview(context.Context, string, ids.XID, string, []byte) (ingest.Preview, error)
}

// ImportCommitService is the mutating application boundary for the second
// phase. The actor comes from the authenticated principal and is persisted by
// the same transaction as the import changes.
type ImportCommitService interface {
	Commit(context.Context, string, ids.XID, string, []byte, string, audit.Actor) (ingest.Preview, error)
}

// ImportPreviewInput accepts the exact submitted source document as a raw
// body. The target year is a query parameter because the URL kind is shared
// by sources with different canonical records.
type ImportPreviewInput struct {
	Kind         string `path:"kind" minLength:"1" doc:"Registered import kind, such as roster_json."`
	SchoolYearID string `query:"school_year_id" minLength:"1" doc:"Opaque target school-year identifier."`
	RawBody      []byte
}

type ImportCommitInput struct {
	Kind              string `path:"kind" minLength:"1" doc:"Registered import kind, such as roster_json."`
	SchoolYearID      string `query:"school_year_id" minLength:"1" doc:"Opaque target school-year identifier."`
	ContentHash       string `query:"content_hash" minLength:"1" doc:"SHA-256 hash returned by the reviewed preview."`
	ContentHashHeader string `header:"X-Import-Content-Hash" doc:"Alternative hash header returned by the reviewed preview."`
	RawBody           []byte
}

type ImportPreviewOutput struct {
	Body ingest.Preview
}

type ImportCommitOutput struct {
	Body ingest.Preview
}

// ImportHandler exposes the stateless preview endpoint. No actor is passed to
// the service because a read-only preview records no audit entry.
type ImportHandler struct {
	service ImportPreviewService
	commit  ImportCommitService
	logger  *slog.Logger
}

func NewImportHandler(service ImportPreviewService, commit ...ImportCommitService) *ImportHandler {
	var commitService ImportCommitService
	if len(commit) > 0 {
		commitService = commit[0]
	} else if candidate, ok := service.(ImportCommitService); ok {
		commitService = candidate
	}
	return &ImportHandler{service: service, commit: commitService}
}

// WithLogger supplies the destination for the causes behind a 500. An import
// reads and writes the whole roster, so "unable to preview import" with no
// recorded cause leaves an operator with a failing upload and no next step.
func (h *ImportHandler) WithLogger(logger *slog.Logger) *ImportHandler {
	if h != nil {
		h.logger = logger
	}
	return h
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
		return nil, h.logged(ctx, "preview", input.Kind, err, importPreviewProblem(err))
	}
	return &ImportPreviewOutput{Body: preview}, nil
}

func (h *ImportHandler) Commit(ctx context.Context, input *ImportCommitInput) (*ImportCommitOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.commit == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "import commit service is not configured")
	}
	if input == nil || strings.TrimSpace(input.Kind) == "" || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ImportInvalid, "import kind and school year are required")
	}
	contentHash := strings.TrimSpace(input.ContentHash)
	if contentHash == "" {
		contentHash = strings.TrimSpace(input.ContentHashHeader)
	}
	if contentHash == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ImportInvalid, "the reviewed preview content hash is required")
	}
	preview, err := h.commit.Commit(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), strings.TrimSpace(input.Kind), input.RawBody, contentHash, importActor(account))
	if err != nil {
		return nil, h.logged(ctx, "commit", input.Kind, err, importCommitProblem(err))
	}
	return &ImportCommitOutput{Body: preview}, nil
}

// logged records the cause of an unclassified import failure and returns the
// problem unchanged. Only the 500 case is logged: every other status already
// carries an actionable reason in its own response.
func (h *ImportHandler) logged(ctx context.Context, phase, kind string, cause error, problem error) error {
	var model *huma.ErrorModel
	if !errors.As(problem, &model) || model.Status != http.StatusInternalServerError {
		return problem
	}
	logger := h.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.ErrorContext(ctx, "import failed",
		slog.String("phase", phase),
		slog.String("kind", strings.TrimSpace(kind)),
		slog.Any("error", cause),
	)
	return problem
}

// invalidSourceDetail names the kind that refused the document and why. The
// reason is the parser's own message about the document the caller just
// uploaded, so it discloses nothing the caller did not send.
func invalidSourceDetail(err *ingest.InvalidSourceError) string {
	reason := strings.TrimSpace(err.Reason)
	if reason == "" {
		return "the submitted import document is invalid"
	}
	return fmt.Sprintf("the submitted %s document is invalid: %s", err.Kind, reason)
}

func importActor(account auth.AccountPrincipal) audit.Actor {
	userID := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: account.Email}
}

func importPreviewProblem(err error) error {
	var invalid *ingest.InvalidSourceError
	switch {
	case errors.As(err, &invalid):
		return problems.New(http.StatusBadRequest, problems.ImportInvalid, invalidSourceDetail(invalid))
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

func importCommitProblem(err error) error {
	var invalid *ingest.InvalidSourceError
	switch {
	case errors.As(err, &invalid):
		return problems.New(http.StatusBadRequest, problems.ImportInvalid, invalidSourceDetail(invalid))
	case errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	case errors.Is(err, ingest.ErrUnknownKind):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "import kind not found")
	case errors.Is(err, ingest.ErrContentHashMismatch):
		return problems.New(http.StatusConflict, problems.ImportInvalid, "the submitted document does not match the reviewed preview content hash")
	case errors.Is(err, ingest.ErrCommitHasErrors):
		return problems.New(http.StatusConflict, problems.ImportInvalid, "the import cannot be committed while the preview contains error records")
	case errors.Is(err, ingest.ErrSchoolYearClosed):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot receive an import commit")
	case errors.Is(err, ingest.ErrInvalidSource):
		return problems.New(http.StatusBadRequest, problems.ImportInvalid, "the submitted import document is invalid")
	case errors.Is(err, ingest.ErrUnsupportedKind):
		return problems.New(http.StatusBadRequest, problems.ImportInvalid, "this import kind is registered but not supported in the current phase")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to commit import")
	}
}
