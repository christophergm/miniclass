package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SessionNonParticipationResponse struct {
	ID             string    `json:"id" doc:"Opaque non-participation identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID      string    `json:"program_id" doc:"Opaque program identifier."`
	SessionID      string    `json:"session_id" doc:"Opaque session identifier."`
	StudentID      string    `json:"student_id" doc:"Opaque student identifier."`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SessionNonParticipationListOutput struct {
	Body []SessionNonParticipationResponse
}
type SessionNonParticipationOutput struct {
	Body SessionNonParticipationResponse
}
type SessionNonParticipationPathInput struct {
	SessionPathInput
	NonParticipationID string `path:"nonParticipationID" minLength:"1" doc:"Opaque non-participation identifier."`
}
type ListSessionNonParticipationsInput struct{ SessionPathInput }
type CreateSessionNonParticipationInput struct {
	SessionPathInput
	Body struct {
		StudentID string `json:"student_id" minLength:"1" doc:"Opaque student identifier."`
		Reason    string `json:"reason" minLength:"1"`
	}
}
type UpdateSessionNonParticipationInput struct {
	SessionNonParticipationPathInput
	Body struct {
		Reason *string `json:"reason,omitempty" minLength:"1"`
	}
}
type GetSessionNonParticipationInput struct {
	SessionNonParticipationPathInput
}
type DeleteSessionNonParticipationInput struct {
	SessionNonParticipationPathInput
}

func (h *ProgramHandler) ListSessionNonParticipations(ctx context.Context, input *ListSessionNonParticipationsInput) (*SessionNonParticipationListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNonParticipationNotFound()
	}
	rows, err := h.service.ListSessionNonParticipations(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, sessionNonParticipationProblem(err)
	}
	result := make([]SessionNonParticipationResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, sessionNonParticipationResponse(row))
	}
	return &SessionNonParticipationListOutput{Body: result}, nil
}

func (h *ProgramHandler) CreateSessionNonParticipation(ctx context.Context, input *CreateSessionNonParticipationInput) (*SessionNonParticipationOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNonParticipationNotFound()
	}
	row, err := h.service.CreateSessionNonParticipation(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.Body.StudentID), input.Body.Reason)
	if err != nil {
		return nil, sessionNonParticipationProblem(err)
	}
	return &SessionNonParticipationOutput{Body: sessionNonParticipationResponse(row)}, nil
}

func (h *ProgramHandler) GetSessionNonParticipation(ctx context.Context, input *GetSessionNonParticipationInput) (*SessionNonParticipationOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNonParticipationNotFound()
	}
	row, err := h.service.GetSessionNonParticipation(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.NonParticipationID))
	if err != nil {
		return nil, sessionNonParticipationProblem(err)
	}
	return &SessionNonParticipationOutput{Body: sessionNonParticipationResponse(row)}, nil
}

func (h *ProgramHandler) UpdateSessionNonParticipation(ctx context.Context, input *UpdateSessionNonParticipationInput) (*SessionNonParticipationOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNonParticipationNotFound()
	}
	row, err := h.service.UpdateSessionNonParticipation(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.NonParticipationID), programservice.SessionNonParticipationUpdate{Reason: input.Body.Reason})
	if err != nil {
		return nil, sessionNonParticipationProblem(err)
	}
	return &SessionNonParticipationOutput{Body: sessionNonParticipationResponse(row)}, nil
}

func (h *ProgramHandler) DeleteSessionNonParticipation(ctx context.Context, input *DeleteSessionNonParticipationInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNonParticipationNotFound()
	}
	if err := h.service.DeleteSessionNonParticipation(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.NonParticipationID)); err != nil {
		return nil, sessionNonParticipationProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func sessionNonParticipationResponse(row data.SessionNonParticipation) SessionNonParticipationResponse {
	return SessionNonParticipationResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), SessionID: string(row.SessionID), StudentID: string(row.StudentID), Reason: row.Reason, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func sessionNonParticipationNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "session non-participation not found")
}

func sessionNonParticipationProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), strings.Contains(err.Error(), "session non-participation not found"):
		return sessionNonParticipationNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the student is already marked as not participating in this session")
	case errors.Is(err, programservice.ErrSessionNonParticipationNoChanges), errors.Is(err, programservice.ErrStudentNotProgramMember), strings.Contains(err.Error(), "reason is required"), strings.Contains(err.Error(), "student id"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change session non-participation data")
	}
}
