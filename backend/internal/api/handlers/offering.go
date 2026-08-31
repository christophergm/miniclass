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

type OfferingResponse struct {
	ID                      string    `json:"id" doc:"Opaque offering identifier."`
	OrganizationID          string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID            string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID               string    `json:"program_id" doc:"Opaque program identifier."`
	SessionID               string    `json:"session_id" doc:"Opaque session identifier."`
	Name                    string    `json:"name"`
	Description             string    `json:"description"`
	MinimumViableEnrollment *int      `json:"minimum_viable_enrollment,omitempty" nullable:"true"`
	Capacity                int       `json:"capacity"`
	MinGradeLevelID         string    `json:"min_grade_level_id" doc:"Opaque minimum grade-level identifier."`
	MaxGradeLevelID         string    `json:"max_grade_level_id" doc:"Opaque maximum grade-level identifier."`
	Location                string    `json:"location"`
	MeetingPoint            string    `json:"meeting_point"`
	MeetingInstructions     string    `json:"meeting_instructions"`
	InterestAreaID          *string   `json:"interest_area_id,omitempty" nullable:"true" doc:"Optional opaque interest-area identifier."`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type OfferingListOutput struct{ Body []OfferingResponse }
type OfferingOutput struct{ Body OfferingResponse }
type OfferingPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	ProgramID    string `path:"programID" minLength:"1" doc:"Opaque programme identifier."`
	SessionID    string `path:"sessionID" minLength:"1" doc:"Opaque session identifier."`
	OfferingID   string `path:"offeringID" minLength:"1" doc:"Opaque offering identifier."`
}
type OfferingCollectionInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	ProgramID    string `path:"programID" minLength:"1" doc:"Opaque programme identifier."`
	SessionID    string `path:"sessionID" minLength:"1" doc:"Opaque session identifier."`
}
type CreateOfferingInput struct {
	OfferingCollectionInput
	Body struct {
		Name                    string  `json:"name" minLength:"1"`
		Description             string  `json:"description,omitempty"`
		MinimumViableEnrollment *int    `json:"minimum_viable_enrollment,omitempty" minimum:"0" nullable:"true"`
		Capacity                int     `json:"capacity" minimum:"1"`
		MinGradeLevelID         string  `json:"min_grade_level_id" minLength:"1" doc:"Opaque minimum grade-level identifier."`
		MaxGradeLevelID         string  `json:"max_grade_level_id" minLength:"1" doc:"Opaque maximum grade-level identifier."`
		Location                string  `json:"location,omitempty"`
		MeetingPoint            string  `json:"meeting_point,omitempty"`
		MeetingInstructions     string  `json:"meeting_instructions,omitempty"`
		InterestAreaID          *string `json:"interest_area_id,omitempty" nullable:"true" doc:"Optional opaque interest-area identifier."`
	}
}
type UpdateOfferingInput struct {
	OfferingPathInput
	Body struct {
		Name                    *string `json:"name,omitempty" minLength:"1"`
		Description             *string `json:"description,omitempty"`
		MinimumViableEnrollment *int    `json:"minimum_viable_enrollment,omitempty" minimum:"0" nullable:"true"`
		Capacity                *int    `json:"capacity,omitempty" minimum:"1"`
		MinGradeLevelID         *string `json:"min_grade_level_id,omitempty" minLength:"1" doc:"Opaque minimum grade-level identifier."`
		MaxGradeLevelID         *string `json:"max_grade_level_id,omitempty" minLength:"1" doc:"Opaque maximum grade-level identifier."`
		Location                *string `json:"location,omitempty"`
		MeetingPoint            *string `json:"meeting_point,omitempty"`
		MeetingInstructions     *string `json:"meeting_instructions,omitempty"`
		InterestAreaID          *string `json:"interest_area_id,omitempty" nullable:"true" doc:"Optional opaque interest-area identifier."`
	}
}

func (h *ProgramHandler) ListOfferings(ctx context.Context, input *OfferingCollectionInput) (*OfferingListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, offeringNotFound()
	}
	rows, err := h.service.ListOfferings(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, offeringProblem(err)
	}
	result := make([]OfferingResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, offeringResponse(row))
	}
	return &OfferingListOutput{Body: result}, nil
}

func (h *ProgramHandler) CreateOffering(ctx context.Context, input *CreateOfferingInput) (*OfferingOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, offeringNotFound()
	}
	var interestAreaID *ids.XID
	if input.Body.InterestAreaID != nil {
		value := ids.XID(*input.Body.InterestAreaID)
		interestAreaID = &value
	}
	row, err := h.service.CreateOffering(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), input.Body.Name, input.Body.Description, input.Body.MinimumViableEnrollment, input.Body.Capacity, ids.XID(input.Body.MinGradeLevelID), ids.XID(input.Body.MaxGradeLevelID), input.Body.Location, input.Body.MeetingPoint, input.Body.MeetingInstructions, interestAreaID)
	if err != nil {
		return nil, offeringProblem(err)
	}
	return &OfferingOutput{Body: offeringResponse(row)}, nil
}

func (h *ProgramHandler) GetOffering(ctx context.Context, input *OfferingPathInput) (*OfferingOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, offeringNotFound()
	}
	row, err := h.service.GetOffering(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.OfferingID))
	if err != nil {
		return nil, offeringProblem(err)
	}
	return &OfferingOutput{Body: offeringResponse(row)}, nil
}

func (h *ProgramHandler) UpdateOffering(ctx context.Context, input *UpdateOfferingInput) (*OfferingOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, offeringNotFound()
	}
	var minGrade, maxGrade, interestArea *ids.XID
	if input.Body.MinGradeLevelID != nil {
		value := ids.XID(*input.Body.MinGradeLevelID)
		minGrade = &value
	}
	if input.Body.MaxGradeLevelID != nil {
		value := ids.XID(*input.Body.MaxGradeLevelID)
		maxGrade = &value
	}
	if input.Body.InterestAreaID != nil {
		value := ids.XID(*input.Body.InterestAreaID)
		interestArea = &value
	}
	row, err := h.service.UpdateOffering(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.OfferingID), programservice.OfferingUpdate{Name: input.Body.Name, Description: input.Body.Description, MinimumViableEnrollment: input.Body.MinimumViableEnrollment, Capacity: input.Body.Capacity, MinGradeLevelID: minGrade, MaxGradeLevelID: maxGrade, Location: input.Body.Location, MeetingPoint: input.Body.MeetingPoint, MeetingInstructions: input.Body.MeetingInstructions, InterestAreaID: interestArea})
	if err != nil {
		return nil, offeringProblem(err)
	}
	return &OfferingOutput{Body: offeringResponse(row)}, nil
}

func (h *ProgramHandler) DeleteOffering(ctx context.Context, input *OfferingPathInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, offeringNotFound()
	}
	if err := h.service.DeleteOffering(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.OfferingID)); err != nil {
		return nil, offeringProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func offeringResponse(row data.Offering) OfferingResponse {
	var interestAreaID *string
	if row.InterestAreaID != nil {
		value := string(*row.InterestAreaID)
		interestAreaID = &value
	}
	return OfferingResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), SessionID: string(row.SessionID), Name: row.Name, Description: row.Description, MinimumViableEnrollment: row.MinimumViableEnrollment, Capacity: row.Capacity, MinGradeLevelID: string(row.MinGradeLevelID), MaxGradeLevelID: string(row.MaxGradeLevelID), Location: row.Location, MeetingPoint: row.MeetingPoint, MeetingInstructions: row.MeetingInstructions, InterestAreaID: interestAreaID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func offeringNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "offering not found")
}

func offeringProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), strings.Contains(err.Error(), "offering not found"):
		return offeringNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.Is(err, programservice.ErrSessionReadOnly):
		return problems.New(http.StatusConflict, problems.SessionReadOnly, err.Error())
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the offering already exists in this session")
	case errors.Is(err, programservice.ErrOfferingGradeOrder), errors.Is(err, programservice.ErrOfferingNoChanges), strings.Contains(err.Error(), "name is required"), strings.Contains(err.Error(), "capacity must be positive"), strings.Contains(err.Error(), "minimum viable enrollment"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change offering data")
	}
}
