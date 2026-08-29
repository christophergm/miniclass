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
	"github.com/chrismott/miniclass/internal/people"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type GuardianRelationshipListOutput struct {
	Body []GuardianRelationshipResponse
}
type GuardianRelationshipOutput struct{ Body GuardianRelationshipResponse }

type GuardianRelationshipResponse struct {
	ID               string    `json:"id" doc:"Opaque guardian-relationship identifier."`
	OrganizationID   string    `json:"organization_id"`
	SchoolYearID     string    `json:"school_year_id"`
	AdultID          string    `json:"adult_id"`
	StudentID        string    `json:"student_id"`
	RelationshipType string    `json:"relationship_type" enum:"parent,guardian,grandparent,other"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type GuardianRelationshipYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
}
type GuardianRelationshipPathInput struct {
	SchoolYearID   string `path:"schoolYearID" minLength:"1"`
	RelationshipID string `path:"relationshipID" minLength:"1"`
}
type ListGuardianRelationshipsInput struct {
	GuardianRelationshipYearPathInput
	AdultID   string `query:"adult_id" doc:"Only return relationships for this adult."`
	StudentID string `query:"student_id" doc:"Only return relationships for this student."`
}
type CreateGuardianRelationshipInput struct {
	GuardianRelationshipYearPathInput
	Body struct {
		AdultID          string `json:"adult_id" minLength:"1"`
		StudentID        string `json:"student_id" minLength:"1"`
		RelationshipType string `json:"relationship_type" enum:"parent,guardian,grandparent,other"`
	}
}
type GetGuardianRelationshipInput struct{ GuardianRelationshipPathInput }
type UpdateGuardianRelationshipInput struct {
	GuardianRelationshipPathInput
	Body struct {
		RelationshipType *string `json:"relationship_type,omitempty" enum:"parent,guardian,grandparent,other"`
	}
}
type DeleteGuardianRelationshipInput struct{ GuardianRelationshipPathInput }
type GuardianRelationshipDeleteOutput struct{}

type GuardianRelationshipService interface {
	ListGuardianRelationships(context.Context, string, ids.XID, data.GuardianRelationshipFilter) ([]data.GuardianRelationship, error)
	CreateGuardianRelationship(context.Context, string, ids.XID, audit.Actor, people.GuardianRelationshipCreateInput) (data.GuardianRelationship, error)
	GetGuardianRelationship(context.Context, string, ids.XID, ids.XID) (data.GuardianRelationship, error)
	UpdateGuardianRelationship(context.Context, string, ids.XID, ids.XID, audit.Actor, people.GuardianRelationshipUpdateInput) (data.GuardianRelationship, error)
	DeleteGuardianRelationship(context.Context, string, ids.XID, ids.XID, audit.Actor) error
}

type GuardianRelationshipHandler struct{ service GuardianRelationshipService }

func NewGuardianRelationshipHandler(service GuardianRelationshipService) *GuardianRelationshipHandler {
	return &GuardianRelationshipHandler{service: service}
}

func (h *GuardianRelationshipHandler) List(ctx context.Context, input *ListGuardianRelationshipsInput) (*GuardianRelationshipListOutput, error) {
	account, err := guardianRelationshipAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, guardianRelationshipNotFound()
	}
	filter := data.GuardianRelationshipFilter{
		AdultID:   ids.XID(strings.TrimSpace(input.AdultID)),
		StudentID: ids.XID(strings.TrimSpace(input.StudentID)),
	}
	rows, err := h.service.ListGuardianRelationships(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), filter)
	if err != nil {
		return nil, guardianRelationshipProblem(err)
	}
	result := make([]GuardianRelationshipResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, guardianRelationshipResponse(row))
	}
	return &GuardianRelationshipListOutput{Body: result}, nil
}

func (h *GuardianRelationshipHandler) Create(ctx context.Context, input *CreateGuardianRelationshipInput) (*GuardianRelationshipOutput, error) {
	account, err := guardianRelationshipAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, guardianRelationshipNotFound()
	}
	row, err := h.service.CreateGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), guardianRelationshipActor(account), people.GuardianRelationshipCreateInput{AdultID: ids.XID(input.Body.AdultID), StudentID: ids.XID(input.Body.StudentID), RelationshipType: data.GuardianRelationshipType(strings.TrimSpace(input.Body.RelationshipType))})
	if err != nil {
		return nil, guardianRelationshipProblem(err)
	}
	return &GuardianRelationshipOutput{Body: guardianRelationshipResponse(row)}, nil
}

func (h *GuardianRelationshipHandler) Get(ctx context.Context, input *GetGuardianRelationshipInput) (*GuardianRelationshipOutput, error) {
	account, err := guardianRelationshipAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, guardianRelationshipNotFound()
	}
	row, err := h.service.GetGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.RelationshipID))
	if err != nil {
		return nil, guardianRelationshipProblem(err)
	}
	return &GuardianRelationshipOutput{Body: guardianRelationshipResponse(row)}, nil
}

func (h *GuardianRelationshipHandler) Update(ctx context.Context, input *UpdateGuardianRelationshipInput) (*GuardianRelationshipOutput, error) {
	account, err := guardianRelationshipAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, guardianRelationshipNotFound()
	}
	var relationshipType *data.GuardianRelationshipType
	if input.Body.RelationshipType != nil {
		value := data.GuardianRelationshipType(strings.TrimSpace(*input.Body.RelationshipType))
		relationshipType = &value
	}
	row, err := h.service.UpdateGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.RelationshipID), guardianRelationshipActor(account), people.GuardianRelationshipUpdateInput{RelationshipType: relationshipType})
	if err != nil {
		return nil, guardianRelationshipProblem(err)
	}
	return &GuardianRelationshipOutput{Body: guardianRelationshipResponse(row)}, nil
}

func (h *GuardianRelationshipHandler) Delete(ctx context.Context, input *DeleteGuardianRelationshipInput) (*GuardianRelationshipDeleteOutput, error) {
	account, err := guardianRelationshipAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, guardianRelationshipNotFound()
	}
	if err := h.service.DeleteGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.RelationshipID), guardianRelationshipActor(account)); err != nil {
		return nil, guardianRelationshipProblem(err)
	}
	return &GuardianRelationshipDeleteOutput{}, nil
}

func guardianRelationshipAccount(ctx context.Context) (auth.AccountPrincipal, error) {
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

func guardianRelationshipActor(account auth.AccountPrincipal) audit.Actor {
	userID := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: account.Email}
}

func guardianRelationshipResponse(row data.GuardianRelationship) GuardianRelationshipResponse {
	return GuardianRelationshipResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), AdultID: string(row.AdultID), StudentID: string(row.StudentID), RelationshipType: string(row.RelationshipType), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func guardianRelationshipNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "guardian relationship not found")
}

func guardianRelationshipProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return guardianRelationshipNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.GuardianRelationshipConflict, "the guardian relationship already exists")
	case strings.Contains(err.Error(), "invalid relationship type"):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, err.Error())
	case errors.Is(err, people.ErrGuardianRelationshipNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change guardian relationship data")
	}
}
