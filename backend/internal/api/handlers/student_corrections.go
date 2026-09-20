package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type StudentCorrectionsService interface {
	CreateStudentCorrection(context.Context, string, ids.XID, audit.Actor, people.StudentCreateInput, string) (data.Student, error)
	UpdateStudentCorrection(context.Context, string, ids.XID, ids.XID, audit.Actor, people.StudentUpdateInput, string) (data.Student, error)
	DeleteStudentCorrection(context.Context, string, ids.XID, ids.XID, audit.Actor, string) error
	CreatePlaceholderStudent(context.Context, string, ids.XID, audit.Actor, people.PlaceholderStudentInput) (data.Student, error)
	ReconcilePlaceholderStudent(context.Context, string, ids.XID, ids.XID, ids.XID, audit.Actor, string) (people.StudentReconciliationResult, error)
	ListStudentReviewSignals(context.Context, string, ids.XID) ([]people.StudentReviewSignal, error)
}

type StudentCorrectionsHandler struct{ service StudentCorrectionsService }

func NewStudentCorrectionsHandler(service StudentCorrectionsService) *StudentCorrectionsHandler {
	return &StudentCorrectionsHandler{service: service}
}

type CreateStudentCorrectionInput struct {
	StudentYearPathInput
	Body struct {
		LegalGivenName     string  `json:"legal_given_name" minLength:"1"`
		LegalFamilyName    string  `json:"legal_family_name" minLength:"1"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		GradeLevelID       *string `json:"grade_level_id,omitempty" nullable:"true" minLength:"1"`
		HomeroomID         string  `json:"homeroom_id" minLength:"1"`
		ExternalIdentifier *string `json:"external_identifier,omitempty"`
		PriorYearStudentID *string `json:"prior_year_student_id,omitempty"`
		Reason             string  `json:"reason" minLength:"1" doc:"Authority and reason for this individual correction."`
	}
}

type UpdateStudentCorrectionInput struct {
	StudentPathInput
	Body struct {
		LegalGivenName     *string `json:"legal_given_name,omitempty" minLength:"1"`
		LegalFamilyName    *string `json:"legal_family_name,omitempty" minLength:"1"`
		PreferredGivenName *string `json:"preferred_given_name,omitempty"`
		GradeLevelID       *string `json:"grade_level_id,omitempty" nullable:"true" minLength:"1"`
		HomeroomID         *string `json:"homeroom_id,omitempty" minLength:"1"`
		ExternalIdentifier *string `json:"external_identifier,omitempty"`
		Reason             string  `json:"reason" minLength:"1" doc:"Authority and reason for this individual correction."`
	}
}

type DeleteStudentCorrectionInput struct {
	StudentPathInput
	Reason string `query:"reason" minLength:"1" doc:"Authority and reason for this individual correction."`
}

type CreatePlaceholderStudentInput struct {
	StudentYearPathInput
	Body struct {
		LegalGivenName  string `json:"legal_given_name" minLength:"1" maxLength:"80" doc:"Abbreviated placeholder label, such as Unknown A."`
		LegalFamilyName string `json:"legal_family_name" minLength:"1" maxLength:"80" doc:"Abbreviated placeholder label."`
		GradeLevelID    string `json:"grade_level_id" minLength:"1" doc:"Required opaque grade identifier."`
		HomeroomID      string `json:"homeroom_id" minLength:"1" doc:"Required opaque homeroom identifier."`
		Reason          string `json:"reason" minLength:"1" doc:"Why the placeholder is needed."`
	}
}

type ReconcilePlaceholderStudentInput struct {
	StudentYearPathInput
	Body struct {
		PlaceholderStudentID string `json:"placeholder_student_id" minLength:"1" doc:"Opaque placeholder student identifier."`
		TargetStudentID      string `json:"target_student_id" minLength:"1" doc:"Opaque consented student identifier."`
		Reason               string `json:"reason" minLength:"1" doc:"Review authority and reason for the reconciliation."`
	}
}

type ReconcilePlaceholderStudentResponse struct {
	SourceStudentID               string `json:"source_student_id" doc:"Opaque reconciled placeholder identifier."`
	TargetStudentID               string `json:"target_student_id" doc:"Opaque consented student identifier."`
	ProgramMembershipsMoved       int64  `json:"program_memberships_moved"`
	SessionNonParticipationsMoved int64  `json:"session_non_participations_moved"`
	ArtifactsRegenerated          int    `json:"artifacts_regenerated"`
}

type ReconcilePlaceholderStudentOutput struct {
	Body ReconcilePlaceholderStudentResponse
}

type StudentReviewSignalResponse struct {
	Code      string `json:"code" doc:"Stable review signal code."`
	Severity  string `json:"severity" enum:"info,warning"`
	StudentID string `json:"student_id" doc:"Opaque student identifier."`
	Detail    string `json:"detail"`
	Count     int64  `json:"count,omitempty"`
}

type StudentReviewSignalsResponse struct {
	SchoolYearID string                        `json:"school_year_id" doc:"Opaque school-year identifier."`
	Signals      []StudentReviewSignalResponse `json:"signals"`
}

type StudentReviewSignalsOutput struct{ Body StudentReviewSignalsResponse }

func (h *StudentCorrectionsHandler) Create(ctx context.Context, input *CreateStudentCorrectionInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, studentNotFound()
	}
	var gradeLevelID *ids.XID
	if input.Body.GradeLevelID != nil && strings.TrimSpace(*input.Body.GradeLevelID) != "" {
		value := ids.XID(strings.TrimSpace(*input.Body.GradeLevelID))
		gradeLevelID = &value
	}
	row, err := h.service.CreateStudentCorrection(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), adultActor(account), people.StudentCreateInput{
		LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName,
		PreferredGivenName: input.Body.PreferredGivenName, GradeLevelID: gradeLevelID,
		HomeroomID: ids.XID(input.Body.HomeroomID), ExternalIdentifier: input.Body.ExternalIdentifier,
	}, input.Body.Reason)
	if err != nil {
		return nil, studentCorrectionProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func (h *StudentCorrectionsHandler) Update(ctx context.Context, input *UpdateStudentCorrectionInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
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
	row, err := h.service.UpdateStudentCorrection(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.StudentID), adultActor(account), serviceInput, input.Body.Reason)
	if err != nil {
		return nil, studentCorrectionProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func (h *StudentCorrectionsHandler) Delete(ctx context.Context, input *DeleteStudentCorrectionInput) (*StudentDeleteOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, studentNotFound()
	}
	if err := h.service.DeleteStudentCorrection(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.StudentID), adultActor(account), input.Reason); err != nil {
		return nil, studentCorrectionProblem(err)
	}
	return &StudentDeleteOutput{}, nil
}

func (h *StudentCorrectionsHandler) CreatePlaceholder(ctx context.Context, input *CreatePlaceholderStudentInput) (*StudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, studentNotFound()
	}
	row, err := h.service.CreatePlaceholderStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), adultActor(account), people.PlaceholderStudentInput{
		LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName,
		GradeLevelID: ids.XID(input.Body.GradeLevelID), HomeroomID: ids.XID(input.Body.HomeroomID), Reason: input.Body.Reason,
	})
	if err != nil {
		return nil, studentCorrectionProblem(err)
	}
	return &StudentOutput{Body: studentResponse(row)}, nil
}

func (h *StudentCorrectionsHandler) Reconcile(ctx context.Context, input *ReconcilePlaceholderStudentInput) (*ReconcilePlaceholderStudentOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, studentNotFound()
	}
	result, err := h.service.ReconcilePlaceholderStudent(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.Body.PlaceholderStudentID), ids.XID(input.Body.TargetStudentID), adultActor(account), input.Body.Reason)
	if err != nil {
		return nil, studentCorrectionProblem(err)
	}
	return &ReconcilePlaceholderStudentOutput{Body: ReconcilePlaceholderStudentResponse{
		SourceStudentID: string(result.SourceStudentID), TargetStudentID: string(result.TargetStudentID),
		ProgramMembershipsMoved: result.ProgramMembershipsMoved, SessionNonParticipationsMoved: result.SessionNonParticipationsMoved,
		ArtifactsRegenerated: result.ArtifactsRegenerated,
	}}, nil
}

func (h *StudentCorrectionsHandler) ReviewSignals(ctx context.Context, input *StudentYearPathInput) (*StudentReviewSignalsOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, studentNotFound()
	}
	signals, err := h.service.ListStudentReviewSignals(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, studentCorrectionProblem(err)
	}
	response := make([]StudentReviewSignalResponse, 0, len(signals))
	for _, signal := range signals {
		response = append(response, StudentReviewSignalResponse{Code: signal.Code, Severity: signal.Severity, StudentID: string(signal.StudentID), Detail: signal.Detail, Count: signal.Count})
	}
	return &StudentReviewSignalsOutput{Body: StudentReviewSignalsResponse{SchoolYearID: input.SchoolYearID, Signals: response}}, nil
}

func studentCorrectionProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return studentNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.Is(err, people.ErrCorrectionReasonRequired), errors.Is(err, people.ErrPlaceholderGradeRequired), errors.Is(err, people.ErrPlaceholderLabelRequired):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, err.Error())
	case errors.Is(err, people.ErrStudentIsNotPlaceholder), errors.Is(err, people.ErrReconciliationTargetNotConsented), errors.Is(err, people.ErrStudentHasGuardianDependencies), errors.Is(err, people.ErrStudentHasPreferenceDependencies), errors.Is(err, people.ErrStudentDependentConflict), errors.Is(err, people.ErrStudentNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	case errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514"):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, "the correction conflicts with an existing student dependency")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.StudentExternalIdentifierConflict, "the external identifier is already used in this school year")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to apply student correction")
	}
}
