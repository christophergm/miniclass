package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/schoolyear"
	"github.com/jackc/pgx/v5"
)

// SchoolYearService is the application boundary used by the Huma handlers.
// It keeps endpoint tests independent from PostgreSQL while the production
// implementation remains the schoolyear package over internal/data.
type SchoolYearService interface {
	Create(context.Context, string, audit.Actor, string) (data.SchoolYear, error)
	List(context.Context, string) ([]data.SchoolYear, error)
	Get(context.Context, string, ids.XID) (data.SchoolYear, error)
	Update(context.Context, string, ids.XID, auth.OrganizationRole, audit.Actor, schoolyear.UpdateInput) (data.SchoolYear, error)
	Delete(context.Context, string, ids.XID, audit.Actor) error
}

// SchoolYearResponse is the administrator-facing school-year resource.
type SchoolYearResponse struct {
	ID             string    `json:"id" doc:"Opaque school-year identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	Label          string    `json:"label" doc:"School-year display label."`
	State          string    `json:"state" enum:"setup,active,closed" doc:"School-year lifecycle state."`
	CreatedAt      time.Time `json:"created_at" doc:"Creation timestamp."`
	UpdatedAt      time.Time `json:"updated_at" doc:"Last update timestamp."`
}

type SchoolYearListInput struct{}

type SchoolYearListOutput struct {
	Body []SchoolYearResponse
}

type CreateSchoolYearInput struct {
	Body struct {
		Label string `json:"label" minLength:"1" doc:"Display label for the new school year."`
	}
}

type CreateSchoolYearOutput struct {
	Body SchoolYearResponse
}

type SchoolYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
}

type GetSchoolYearOutput struct {
	Body SchoolYearResponse
}

type UpdateSchoolYearInput struct {
	SchoolYearPathInput
	Body struct {
		Label  *string `json:"label,omitempty" minLength:"1" doc:"Replacement display label."`
		State  *string `json:"state,omitempty" enum:"setup,active,closed" doc:"Requested lifecycle state."`
		Reason string  `json:"reason,omitempty" doc:"Required reason when an Owner reopens a closed year."`
	}
}

type UpdateSchoolYearOutput struct {
	Body SchoolYearResponse
}

type DeleteSchoolYearOutput struct{}

// SchoolYearHandler exposes CRUD and lifecycle operations for one tenant.
type SchoolYearHandler struct {
	service SchoolYearService
}

func NewSchoolYearHandler(service SchoolYearService) *SchoolYearHandler {
	return &SchoolYearHandler{service: service}
}

func (h *SchoolYearHandler) List(ctx context.Context, _ *SchoolYearListInput) (*SchoolYearListOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "school-year service is not configured")
	}
	rows, err := h.service.List(ctx, string(account.OrganizationID))
	if err != nil {
		return nil, schoolYearProblem(err)
	}
	response := make([]SchoolYearResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, schoolYearResponse(row))
	}
	return &SchoolYearListOutput{Body: response}, nil
}

func (h *SchoolYearHandler) Create(ctx context.Context, input *CreateSchoolYearInput) (*CreateSchoolYearOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "school-year service is not configured")
	}
	if input == nil || strings.TrimSpace(input.Body.Label) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.SchoolYearTransitionInvalid, "school-year label is required")
	}
	row, err := h.service.Create(ctx, string(account.OrganizationID), schoolYearActor(account), input.Body.Label)
	if err != nil {
		return nil, schoolYearProblem(err)
	}
	return &CreateSchoolYearOutput{Body: schoolYearResponse(row)}, nil
}

func (h *SchoolYearHandler) Get(ctx context.Context, input *SchoolYearPathInput) (*GetSchoolYearOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "school-year service is not configured")
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	row, err := h.service.Get(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, schoolYearProblem(err)
	}
	return &GetSchoolYearOutput{Body: schoolYearResponse(row)}, nil
}

func (h *SchoolYearHandler) Update(ctx context.Context, input *UpdateSchoolYearInput) (*UpdateSchoolYearOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "school-year service is not configured")
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	var state *data.SchoolYearState
	if input.Body.State != nil {
		value := data.SchoolYearState(strings.TrimSpace(*input.Body.State))
		state = &value
	}
	row, err := h.service.Update(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), account.Role, schoolYearActor(account), schoolyear.UpdateInput{
		Label: input.Body.Label, State: state, Reason: input.Body.Reason,
	})
	if err != nil {
		return nil, schoolYearProblem(err)
	}
	return &UpdateSchoolYearOutput{Body: schoolYearResponse(row)}, nil
}

func (h *SchoolYearHandler) Delete(ctx context.Context, input *SchoolYearPathInput) (*DeleteSchoolYearOutput, error) {
	account, err := schoolYearAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "school-year service is not configured")
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	if err := h.service.Delete(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), schoolYearActor(account)); err != nil {
		return nil, schoolYearProblem(err)
	}
	return &DeleteSchoolYearOutput{}, nil
}

func schoolYearAccount(ctx context.Context) (auth.AccountPrincipal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	account, ok := principal.(auth.AccountPrincipal)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "account principal has an unsupported type")
	}
	return account, nil
}

func schoolYearActor(account auth.AccountPrincipal) audit.Actor {
	userID := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: account.Email}
}

func schoolYearResponse(row data.SchoolYear) SchoolYearResponse {
	return SchoolYearResponse{
		ID: string(row.ID), OrganizationID: string(row.OrganizationID), Label: row.Label,
		State: string(row.State), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func schoolYearProblem(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.Is(err, schoolyear.ErrReasonRequired):
		return problems.New(http.StatusBadRequest, problems.SchoolYearReasonRequired, "a reason is required to reopen a closed school year")
	case errors.Is(err, schoolyear.ErrOwnerRequired):
		return problems.New(http.StatusForbidden, problems.CapabilityRequired, "only the Owner can reopen a closed school year")
	case errors.Is(err, schoolyear.ErrRoleRequired):
		return problems.New(http.StatusForbidden, problems.CapabilityRequired, "Owner or Administrator role is required for this transition")
	case errors.Is(err, schoolyear.ErrInvalidTransition), errors.Is(err, schoolyear.ErrNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change school year")
	}
}
