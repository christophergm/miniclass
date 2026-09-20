package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/guardianrecords"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/jackc/pgx/v5"
)

type GuardianRecordsService interface {
	List(context.Context, auth.GuardianPrincipal) ([]guardianrecords.Student, error)
	FindCandidates(context.Context, auth.GuardianPrincipal, guardianrecords.CandidateInput) ([]guardianrecords.Student, error)
	Select(context.Context, auth.GuardianPrincipal, ids.XID, data.GuardianRelationshipType, audit.Actor) (guardianrecords.Student, error)
	Create(context.Context, auth.GuardianPrincipal, guardianrecords.CreateInput, audit.Actor) (guardianrecords.Student, error)
	Update(context.Context, auth.GuardianPrincipal, ids.XID, guardianrecords.UpdateInput, audit.Actor) (guardianrecords.Student, error)
	UpdateProfile(context.Context, auth.GuardianPrincipal, guardianrecords.ProfileInput, audit.Actor) error
	Detach(context.Context, auth.GuardianPrincipal, ids.XID, bool, audit.Actor) error
	DeleteSelf(context.Context, auth.GuardianPrincipal, bool, audit.Actor) error
}

type GuardianRecordsHandler struct{ service GuardianRecordsService }

func NewGuardianRecordsHandler(service GuardianRecordsService) *GuardianRecordsHandler {
	return &GuardianRecordsHandler{service: service}
}

type GuardianStudentResponse struct {
	ID                 string                         `json:"id" doc:"Opaque student identifier."`
	LegalGivenName     string                         `json:"legal_given_name"`
	LegalFamilyName    string                         `json:"legal_family_name"`
	PreferredGivenName *string                        `json:"preferred_given_name,omitempty"`
	GradeLevelID       *string                        `json:"grade_level_id,omitempty" doc:"Opaque grade identifier."`
	GradeLabel         string                         `json:"grade_label"`
	HomeroomID         string                         `json:"homeroom_id" doc:"Opaque homeroom identifier."`
	HomeroomLabel      string                         `json:"homeroom_label"`
	Warnings           []GuardianStudentReviewWarning `json:"warnings"`
}

type GuardianStudentReviewWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GuardianStudentListOutput struct{ Body []GuardianStudentResponse }
type GuardianStudentOutput struct{ Body GuardianStudentResponse }

// GuardianCandidateResponse is deliberately narrower than the normal scoped
// student response. Matching may show only the fields needed to recognize a
// possible student; the full guardian record, preferred name, and vocabulary
// identifiers are not part of this render surface.
type GuardianCandidateResponse struct {
	ID              string `json:"id" doc:"Opaque selection handle for the candidate."`
	LegalGivenName  string `json:"legal_given_name"`
	LegalFamilyName string `json:"legal_family_name"`
	GradeLabel      string `json:"grade_label"`
	HomeroomLabel   string `json:"homeroom_label"`
}

type GuardianCandidatesInput struct {
	Body struct {
		GivenName  string `json:"given_name" minLength:"1"`
		FamilyName string `json:"family_name" minLength:"1"`
	}
}
type GuardianCandidatesQueryInput struct {
	GivenName  string `query:"given_name" minLength:"1"`
	FamilyName string `query:"family_name" minLength:"1"`
}
type GuardianCandidatesOutput struct{ Body []GuardianCandidateResponse }

type GuardianStudentCreateInput struct {
	Body struct {
		StudentID          string  `json:"student_id,omitempty" doc:"Opaque existing student identifier; omit to create a new record."`
		LegalGivenName     string  `json:"legal_given_name,omitempty"`
		LegalFamilyName    string  `json:"legal_family_name,omitempty"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		GradeLevelID       string  `json:"grade_level_id,omitempty" doc:"Opaque grade identifier."`
		HomeroomID         string  `json:"homeroom_id,omitempty" doc:"Opaque homeroom identifier."`
		RelationshipType   string  `json:"relationship_type" enum:"parent,guardian,grandparent,other"`
	}
}

type GuardianStudentPathInput struct {
	StudentID string `path:"studentID" minLength:"1" doc:"Opaque student identifier."`
}
type GuardianStudentUpdateInput struct {
	GuardianStudentPathInput
	Body struct {
		LegalGivenName     *string `json:"legal_given_name,omitempty" minLength:"1"`
		LegalFamilyName    *string `json:"legal_family_name,omitempty" minLength:"1"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		GradeLevelID       *string `json:"grade_level_id,omitempty" doc:"Opaque grade identifier."`
		HomeroomID         *string `json:"homeroom_id,omitempty" doc:"Opaque homeroom identifier."`
	}
}

type GuardianProfileInput struct {
	Body struct {
		LegalGivenName     *string `json:"legal_given_name,omitempty" minLength:"1"`
		LegalFamilyName    *string `json:"legal_family_name,omitempty" minLength:"1"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		Email              *string `json:"email,omitempty" format:"email"`
		Phone              *string `json:"phone,omitempty"`
	}
}
type GuardianProfileOutput struct {
	Body struct {
		Updated bool `json:"updated"`
	}
}
type GuardianConfirmationInput struct {
	Body struct {
		Confirm bool `json:"confirm"`
	}
}
type GuardianDeleteOutput struct{}
type GuardianStudentDeleteInput struct {
	GuardianStudentPathInput
	Body struct {
		Confirm bool `json:"confirm"`
	}
}

func (h *GuardianRecordsHandler) List(ctx context.Context, _ *struct{}) (*GuardianStudentListOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.service.List(ctx, principal)
	if err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianStudentListOutput{Body: guardianStudentResponses(rows)}, nil
}

func (h *GuardianRecordsHandler) Candidates(ctx context.Context, input *GuardianCandidatesInput) (*GuardianCandidatesOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "student names are required")
	}
	rows, err := h.service.FindCandidates(ctx, principal, guardianrecords.CandidateInput{GivenName: input.Body.GivenName, FamilyName: input.Body.FamilyName})
	if err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianCandidatesOutput{Body: guardianCandidateResponses(rows)}, nil
}

func (h *GuardianRecordsHandler) CandidatesQuery(ctx context.Context, input *GuardianCandidatesQueryInput) (*GuardianCandidatesOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "student names are required")
	}
	rows, err := h.service.FindCandidates(ctx, principal, guardianrecords.CandidateInput{GivenName: input.GivenName, FamilyName: input.FamilyName})
	if err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianCandidatesOutput{Body: guardianCandidateResponses(rows)}, nil
}

func (h *GuardianRecordsHandler) Create(ctx context.Context, input *GuardianStudentCreateInput) (*GuardianStudentOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "student input is required")
	}
	actor := guardianRecordsActor(principal)
	typeValue := data.GuardianRelationshipType(strings.TrimSpace(input.Body.RelationshipType))
	if input.Body.StudentID != "" {
		row, err := h.service.Select(ctx, principal, ids.XID(strings.TrimSpace(input.Body.StudentID)), typeValue, actor)
		if err != nil {
			return nil, guardianRecordsProblem(err)
		}
		return &GuardianStudentOutput{Body: guardianStudentResponse(row)}, nil
	}
	row, err := h.service.Create(ctx, principal, guardianrecords.CreateInput{LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName, PreferredGivenName: input.Body.PreferredGivenName, GradeLevelID: ids.XID(strings.TrimSpace(input.Body.GradeLevelID)), HomeroomID: ids.XID(strings.TrimSpace(input.Body.HomeroomID)), RelationshipType: typeValue}, actor)
	if err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianStudentOutput{Body: guardianStudentResponse(row)}, nil
}

func (h *GuardianRecordsHandler) Update(ctx context.Context, input *GuardianStudentUpdateInput) (*GuardianStudentOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, guardianRecordsProblem(guardianrecords.ErrOutOfScope)
	}
	var grade, room *ids.XID
	if input.Body.GradeLevelID != nil {
		value := ids.XID(strings.TrimSpace(*input.Body.GradeLevelID))
		grade = &value
	}
	if input.Body.HomeroomID != nil {
		value := ids.XID(strings.TrimSpace(*input.Body.HomeroomID))
		room = &value
	}
	row, err := h.service.Update(ctx, principal, ids.XID(input.StudentID), guardianrecords.UpdateInput{LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName, PreferredGivenName: input.Body.PreferredGivenName, GradeLevelID: grade, HomeroomID: room}, guardianRecordsActor(principal))
	if err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianStudentOutput{Body: guardianStudentResponse(row)}, nil
}

func (h *GuardianRecordsHandler) Detach(ctx context.Context, input *GuardianStudentDeleteInput) (*GuardianDeleteOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.StudentID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "student not found")
	}
	if err := h.service.Detach(ctx, principal, ids.XID(input.StudentID), input.Body.Confirm, guardianRecordsActor(principal)); err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianDeleteOutput{}, nil
}

func (h *GuardianRecordsHandler) DeleteSelf(ctx context.Context, input *GuardianConfirmationInput) (*GuardianDeleteOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "confirmation is required")
	}
	if err := h.service.DeleteSelf(ctx, principal, input.Body.Confirm, guardianRecordsActor(principal)); err != nil {
		return nil, guardianRecordsProblem(err)
	}
	return &GuardianDeleteOutput{}, nil
}

func (h *GuardianRecordsHandler) UpdateProfile(ctx context.Context, input *GuardianProfileInput) (*GuardianProfileOutput, error) {
	principal, err := guardianRecordsPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, guardianRecordsProblem(guardianrecords.ErrNoChanges)
	}
	if err := h.service.UpdateProfile(ctx, principal, guardianrecords.ProfileInput{LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName, PreferredGivenName: input.Body.PreferredGivenName, Email: input.Body.Email, Phone: input.Body.Phone}, guardianRecordsActor(principal)); err != nil {
		return nil, guardianRecordsProblem(err)
	}
	output := &GuardianProfileOutput{}
	output.Body.Updated = true
	return output, nil
}

func guardianRecordsPrincipal(ctx context.Context) (auth.GuardianPrincipal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.GuardianPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	guardian, ok := principal.(auth.GuardianPrincipal)
	if !ok {
		return auth.GuardianPrincipal{}, problems.New(http.StatusForbidden, problems.CapabilityRequired, "guardian access is required")
	}
	return guardian, nil
}

func guardianRecordsActor(principal auth.GuardianPrincipal) audit.Actor {
	return audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(principal.AdultID)}
}

func guardianStudentResponses(rows []guardianrecords.Student) []GuardianStudentResponse {
	result := make([]GuardianStudentResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, guardianStudentResponse(row))
	}
	return result
}

func guardianCandidateResponses(rows []guardianrecords.Student) []GuardianCandidateResponse {
	result := make([]GuardianCandidateResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, GuardianCandidateResponse{
			ID:              string(row.ID),
			LegalGivenName:  row.LegalGivenName,
			LegalFamilyName: row.LegalFamilyName,
			GradeLabel:      row.GradeLabel,
			HomeroomLabel:   row.HomeroomLabel,
		})
	}
	return result
}

func guardianStudentResponse(row guardianrecords.Student) GuardianStudentResponse {
	var gradeID *string
	if row.GradeLevelID != nil {
		value := string(*row.GradeLevelID)
		gradeID = &value
	}
	warnings := make([]GuardianStudentReviewWarning, 0, len(row.Warnings))
	for _, warning := range row.Warnings {
		warnings = append(warnings, GuardianStudentReviewWarning{Code: warning.Code, Message: warning.Message})
	}
	return GuardianStudentResponse{ID: string(row.ID), LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName, PreferredGivenName: row.PreferredGivenName, GradeLevelID: gradeID, GradeLabel: row.GradeLabel, HomeroomID: string(row.HomeroomID), HomeroomLabel: row.HomeroomLabel, Warnings: warnings}
}

func guardianRecordsProblem(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, guardianrecords.ErrOutOfScope), errors.Is(err, guardianrecords.ErrCandidateInvalid), errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "student not found")
	case errors.Is(err, guardianrecords.ErrGradeRequired):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, "grade is required")
	case errors.Is(err, guardianrecords.ErrConfirmationRequired):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, "confirmation is required")
	case errors.Is(err, guardianrecords.ErrNoChanges), errors.Is(err, people.ErrNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, "no permitted changes were supplied")
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case data.IsSchoolYearPurged(err):
		return schoolYearPurgedProblem()
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change guardian records")
	}
}
