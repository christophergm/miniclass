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
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ProgramService interface {
	Create(context.Context, string, audit.Actor, ids.XID, string) (data.Program, error)
	List(context.Context, string, ids.XID) ([]data.Program, error)
	ListMemberships(context.Context, string, ids.XID, ids.XID) ([]data.ProgramMembership, error)
	AddMembership(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) (data.ProgramMembership, error)
	DeleteMembership(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) error
	CountStudentsWithoutGrade(context.Context, string, ids.XID) (int64, error)
}

type ProgramResponse struct {
	ID             string    `json:"id" doc:"Opaque program identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProgramMembershipResponse struct {
	ID              string    `json:"id" doc:"Opaque membership identifier."`
	OrganizationID  string    `json:"organization_id"`
	SchoolYearID    string    `json:"school_year_id"`
	ProgramID       string    `json:"program_id"`
	StudentID       string    `json:"student_id"`
	LegalGivenName  string    `json:"legal_given_name"`
	LegalFamilyName string    `json:"legal_family_name"`
	GradeLevelID    *string   `json:"grade_level_id" nullable:"true"`
	GradeMissing    bool      `json:"grade_missing" doc:"True when the member's current roster grade is unknown."`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProgramRosterSummaryResponse struct {
	MissingGradeCount int64 `json:"missing_grade_count"`
}
type ProgramListOutput struct{ Body []ProgramResponse }
type ProgramOutput struct{ Body ProgramResponse }
type ProgramMembershipListOutput struct{ Body []ProgramMembershipResponse }
type ProgramMembershipOutput struct{ Body ProgramMembershipResponse }
type ProgramRosterSummaryOutput struct{ Body ProgramRosterSummaryResponse }
type ProgramYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
}
type ProgramPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ProgramID    string `path:"programID" minLength:"1"`
}
type ProgramMembershipPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ProgramID    string `path:"programID" minLength:"1"`
	MembershipID string `path:"membershipID" minLength:"1"`
}
type ListProgramsInput struct{ ProgramYearPathInput }
type CreateProgramInput struct {
	ProgramYearPathInput
	Body struct {
		Name string `json:"name" minLength:"1"`
	}
}
type ListProgramMembershipsInput struct{ ProgramPathInput }
type CreateProgramMembershipInput struct {
	ProgramPathInput
	Body struct {
		StudentID string `json:"student_id" minLength:"1" doc:"Opaque student identifier."`
	}
}
type DeleteProgramMembershipInput struct{ ProgramMembershipPathInput }
type ProgramDeleteOutput struct{}
type MissingGradeSummaryInput struct{ ProgramYearPathInput }

type ProgramHandler struct{ service ProgramService }

func NewProgramHandler(service ProgramService) *ProgramHandler {
	return &ProgramHandler{service: service}
}

func (h *ProgramHandler) List(ctx context.Context, input *ListProgramsInput) (*ProgramListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	rows, err := h.service.List(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, programProblem(err)
	}
	result := make([]ProgramResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, programResponse(row))
	}
	return &ProgramListOutput{Body: result}, nil
}

func (h *ProgramHandler) Create(ctx context.Context, input *CreateProgramInput) (*ProgramOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	row, err := h.service.Create(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), input.Body.Name)
	if err != nil {
		return nil, programProblem(err)
	}
	return &ProgramOutput{Body: programResponse(row)}, nil
}

func (h *ProgramHandler) ListMemberships(ctx context.Context, input *ListProgramMembershipsInput) (*ProgramMembershipListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	rows, err := h.service.ListMemberships(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID))
	if err != nil {
		return nil, programProblem(err)
	}
	result := make([]ProgramMembershipResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, programMembershipResponse(row))
	}
	return &ProgramMembershipListOutput{Body: result}, nil
}

func (h *ProgramHandler) AddMembership(ctx context.Context, input *CreateProgramMembershipInput) (*ProgramMembershipOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	row, err := h.service.AddMembership(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.Body.StudentID))
	if err != nil {
		return nil, programProblem(err)
	}
	return &ProgramMembershipOutput{Body: programMembershipResponse(row)}, nil
}

func (h *ProgramHandler) DeleteMembership(ctx context.Context, input *DeleteProgramMembershipInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	if err := h.service.DeleteMembership(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.MembershipID)); err != nil {
		return nil, programProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func (h *ProgramHandler) MissingGradeSummary(ctx context.Context, input *MissingGradeSummaryInput) (*ProgramRosterSummaryOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	count, err := h.service.CountStudentsWithoutGrade(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, programProblem(err)
	}
	return &ProgramRosterSummaryOutput{Body: ProgramRosterSummaryResponse{MissingGradeCount: count}}, nil
}

func programAccount(ctx context.Context) (auth.AccountPrincipal, error) {
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
func programActor(account auth.AccountPrincipal) audit.Actor {
	id := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &id, Label: account.Email}
}
func programResponse(row data.Program) ProgramResponse {
	return ProgramResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func programMembershipResponse(row data.ProgramMembership) ProgramMembershipResponse {
	var grade *string
	if row.GradeLevelID != nil {
		value := string(*row.GradeLevelID)
		grade = &value
	}
	return ProgramMembershipResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), StudentID: string(row.StudentID), LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName, GradeLevelID: grade, GradeMissing: row.GradeMissing, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func programNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "program not found")
}
func programProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), strings.Contains(err.Error(), "program membership not found"):
		return programNotFound()
	case errors.Is(err, programservice.ErrStudentGradeRequired):
		return problems.New(http.StatusBadRequest, problems.ProgramStudentGradeRequired, err.Error())
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the program or membership already exists")
	case strings.Contains(err.Error(), "name is required") || strings.Contains(err.Error(), "student id"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change program data")
	}
}
