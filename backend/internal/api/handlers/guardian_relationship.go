package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
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
	ListGuardianRelationships(context.Context, string, ids.XID) ([]data.GuardianRelationship, error)
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
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	rows, err := h.service.ListGuardianRelationships(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, householdProblem(err)
	}
	result := make([]GuardianRelationshipResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, guardianRelationshipResponse(row))
	}
	return &GuardianRelationshipListOutput{Body: result}, nil
}

func (h *GuardianRelationshipHandler) Create(ctx context.Context, input *CreateGuardianRelationshipInput) (*GuardianRelationshipOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.CreateGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), householdActor(account), people.GuardianRelationshipCreateInput{AdultID: ids.XID(input.Body.AdultID), StudentID: ids.XID(input.Body.StudentID), RelationshipType: data.GuardianRelationshipType(strings.TrimSpace(input.Body.RelationshipType))})
	if err != nil {
		return nil, householdProblem(err)
	}
	return &GuardianRelationshipOutput{Body: guardianRelationshipResponse(row)}, nil
}

func (h *GuardianRelationshipHandler) Get(ctx context.Context, input *GetGuardianRelationshipInput) (*GuardianRelationshipOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.GetGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.RelationshipID))
	if err != nil {
		return nil, householdProblem(err)
	}
	return &GuardianRelationshipOutput{Body: guardianRelationshipResponse(row)}, nil
}

func (h *GuardianRelationshipHandler) Update(ctx context.Context, input *UpdateGuardianRelationshipInput) (*GuardianRelationshipOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	var relationshipType *data.GuardianRelationshipType
	if input.Body.RelationshipType != nil {
		value := data.GuardianRelationshipType(strings.TrimSpace(*input.Body.RelationshipType))
		relationshipType = &value
	}
	row, err := h.service.UpdateGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.RelationshipID), householdActor(account), people.GuardianRelationshipUpdateInput{RelationshipType: relationshipType})
	if err != nil {
		return nil, householdProblem(err)
	}
	return &GuardianRelationshipOutput{Body: guardianRelationshipResponse(row)}, nil
}

func (h *GuardianRelationshipHandler) Delete(ctx context.Context, input *DeleteGuardianRelationshipInput) (*GuardianRelationshipDeleteOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	if err := h.service.DeleteGuardianRelationship(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.RelationshipID), householdActor(account)); err != nil {
		return nil, householdProblem(err)
	}
	return &GuardianRelationshipDeleteOutput{}, nil
}

func guardianRelationshipResponse(row data.GuardianRelationship) GuardianRelationshipResponse {
	return GuardianRelationshipResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), AdultID: string(row.AdultID), StudentID: string(row.StudentID), RelationshipType: string(row.RelationshipType), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
