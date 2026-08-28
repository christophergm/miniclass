package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type StudentService interface {
	CreateStudent(context.Context, string, ids.XID, audit.Actor, people.StudentCreateInput) (data.Student, error)
	ListStudents(context.Context, string, ids.XID, bool) ([]data.Student, error)
	GetStudent(context.Context, string, ids.XID, ids.XID) (data.Student, error)
	UpdateStudent(context.Context, string, ids.XID, ids.XID, audit.Actor, people.StudentUpdateInput) (data.Student, error)
	DeleteStudent(context.Context, string, ids.XID, ids.XID, audit.Actor) error
	RestoreStudent(context.Context, string, ids.XID, ids.XID, audit.Actor, string) (data.Student, error)
}

type StudentHandler struct{ service StudentService }

func NewStudentHandler(service StudentService) *StudentHandler {
	return &StudentHandler{service: service}
}

type StudentResponse struct {
	ID                 string     `json:"id" doc:"Opaque student identifier."`
	OrganizationID     string     `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID       string     `json:"school_year_id" doc:"Opaque school-year identifier."`
	LegalGivenName     string     `json:"legal_given_name"`
	LegalFamilyName    string     `json:"legal_family_name"`
	PreferredGivenName *string    `json:"preferred_given_name,omitempty"`
	GradeLevelID       string     `json:"grade_level_id" doc:"Opaque grade-level identifier."`
	HomeroomID         string     `json:"homeroom_id" doc:"Opaque homeroom identifier."`
	ExternalIdentifier *string    `json:"external_identifier,omitempty"`
	PriorYearStudentID *string    `json:"prior_year_student_id,omitempty" doc:"Opaque prior-year student identifier."`
	DisplayName        string     `json:"display_name"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type StudentPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	StudentID    string `path:"studentID" minLength:"1" doc:"Opaque student identifier."`
}

type StudentYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
}

type StudentListOutput struct{ Body []StudentResponse }
type StudentOutput struct{ Body StudentResponse }
type StudentDeleteOutput struct{}

type CreateStudentInput struct {
	StudentYearPathInput
	Body struct {
		LegalGivenName     string  `json:"legal_given_name" minLength:"1"`
		LegalFamilyName    string  `json:"legal_family_name" minLength:"1"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		GradeLevelID       string  `json:"grade_level_id" minLength:"1" doc:"Opaque grade-level identifier."`
		HomeroomID         string  `json:"homeroom_id" minLength:"1" doc:"Opaque homeroom identifier."`
		ExternalIdentifier *string `json:"external_identifier,omitempty"`
		PriorYearStudentID *string `json:"prior_year_student_id,omitempty" doc:"Opaque prior-year student identifier."`
	}
}

type ListStudentsInput struct {
	StudentYearPathInput
	IncludeDeleted bool `query:"include_deleted" doc:"Include soft-deleted students."`
}
type GetStudentInput struct{ StudentPathInput }

type UpdateStudentInput struct {
	StudentPathInput
	Body struct {
		LegalGivenName     *string `json:"legal_given_name,omitempty" minLength:"1"`
		LegalFamilyName    *string `json:"legal_family_name,omitempty" minLength:"1"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		GradeLevelID       *string `json:"grade_level_id,omitempty" minLength:"1"`
		HomeroomID         *string `json:"homeroom_id,omitempty" minLength:"1"`
		ExternalIdentifier *string `json:"external_identifier,omitempty"`
		PriorYearStudentID *string `json:"prior_year_student_id,omitempty"`
	}
}

type DeleteStudentInput struct{ StudentPathInput }
type RestoreStudentInput struct {
	StudentPathInput
	Body struct {
		Reason string `json:"reason" minLength:"1" doc:"Why the deleted student is being restored."`
	}
}

func (h *StudentHandler) List(ctx context.Context, input *ListStudentsInput) (*StudentListOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, studentNotFound()
	}
	rows, err := h.service.ListStudents(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), input.IncludeDeleted)
	if err != nil {
		return nil, studentProblem(err)
	}
	response := make([]StudentResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, studentResponse(row))
	}
	return &StudentListOutput{Body: response}, nil
}

func (h *StudentHandler) Create(ctx context.Context, input *CreateStudentInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, studentNotFound()
	}
	var priorYearStudentID *ids.XID
	if input.Body.PriorYearStudentID != nil && strings.TrimSpace(*input.Body.PriorYearStudentID) != "" {
		value := ids.XID(strings.TrimSpace(*input.Body.PriorYearStudentID))
		priorYearStudentID = &value
	}
	row, err := h.service.CreateStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), adultActor(account), people.StudentCreateInput{
		LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName,
		PreferredGivenName: input.Body.PreferredGivenName, GradeLevelID: ids.XID(input.Body.GradeLevelID),
		HomeroomID: ids.XID(input.Body.HomeroomID), ExternalIdentifier: input.Body.ExternalIdentifier,
		PriorYearStudentID: priorYearStudentID,
	})
	if err != nil {
		return nil, studentProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func (h *StudentHandler) Get(ctx context.Context, input *GetStudentInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.StudentID) == "" {
		return nil, studentNotFound()
	}
	row, err := h.service.GetStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.StudentID))
	if err != nil {
		return nil, studentProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func (h *StudentHandler) Update(ctx context.Context, input *UpdateStudentInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.StudentID) == "" {
		return nil, studentNotFound()
	}
	serviceInput := people.StudentUpdateInput{LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName}
	if input.Body.PreferredGivenName != nil {
		value := input.Body.PreferredGivenName
		serviceInput.PreferredGivenName = &value
	}
	if input.Body.GradeLevelID != nil {
		value := ids.XID(strings.TrimSpace(*input.Body.GradeLevelID))
		serviceInput.GradeLevelID = &value
	}
	if input.Body.HomeroomID != nil {
		value := ids.XID(strings.TrimSpace(*input.Body.HomeroomID))
		serviceInput.HomeroomID = &value
	}
	if input.Body.ExternalIdentifier != nil {
		value := input.Body.ExternalIdentifier
		serviceInput.ExternalIdentifier = &value
	}
	if input.Body.PriorYearStudentID != nil {
		var value *ids.XID
		if strings.TrimSpace(*input.Body.PriorYearStudentID) != "" {
			converted := ids.XID(strings.TrimSpace(*input.Body.PriorYearStudentID))
			value = &converted
		}
		serviceInput.PriorYearStudentID = &value
	}
	row, err := h.service.UpdateStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.StudentID), adultActor(account), serviceInput)
	if err != nil {
		return nil, studentProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func (h *StudentHandler) Delete(ctx context.Context, input *DeleteStudentInput) (*StudentDeleteOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.StudentID) == "" {
		return nil, studentNotFound()
	}
	if err := h.service.DeleteStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.StudentID), adultActor(account)); err != nil {
		return nil, studentProblem(err)
	}
	return &StudentDeleteOutput{}, nil
}

func (h *StudentHandler) Restore(ctx context.Context, input *RestoreStudentInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.StudentID) == "" {
		return nil, studentNotFound()
	}
	row, err := h.service.RestoreStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.StudentID), adultActor(account), input.Body.Reason)
	if err != nil {
		return nil, studentProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func studentResponse(row data.Student) StudentResponse {
	preferred := row.PreferredGivenName
	legalGiven, legalFamily := row.LegalGivenName, row.LegalFamilyName
	var priorYearStudentID *string
	if row.PriorYearStudentID != nil {
		value := string(*row.PriorYearStudentID)
		priorYearStudentID = &value
	}
	return StudentResponse{
		ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID),
		LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName, PreferredGivenName: row.PreferredGivenName,
		GradeLevelID: string(row.GradeLevelID), HomeroomID: string(row.HomeroomID), ExternalIdentifier: row.ExternalIdentifier,
		PriorYearStudentID: priorYearStudentID, DisplayName: people.DisplayName(preferred, &legalGiven, &legalFamily),
		DeletedAt: row.DeletedAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func studentNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "student not found")
}

func studentProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return studentNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.StudentExternalIdentifierConflict, "the external identifier is already used in this school year")
	case errors.As(err, &pgErr) && pgErr.Code == "23503":
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, "the referenced grade, homeroom, or prior-year student is invalid")
	case strings.Contains(err.Error(), "legal names are required"), strings.Contains(err.Error(), "school year, grade, and homeroom are required"), strings.Contains(err.Error(), "grade and homeroom are required"):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, err.Error())
	case errors.Is(err, people.ErrStudentNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	case errors.Is(err, people.ErrRestoreReasonRequired):
		return problems.New(http.StatusBadRequest, problems.RestoreReasonRequired, "a reason is required to restore the student")
	case errors.Is(err, people.ErrRestoreNotDeleted):
		return problems.New(http.StatusConflict, problems.RosterRecordNotDeleted, "student is not deleted")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change student")
	}
}
