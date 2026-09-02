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
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type InterestProfileSurveyAudienceInput struct {
	Type          string   `json:"type,omitempty" enum:"all_members,explicit_students,grade_level,response_state"`
	StudentIDs    []string `json:"student_ids,omitempty" doc:"Opaque student identifiers for an explicit audience."`
	GradeLevelID  *string  `json:"grade_level_id,omitempty" doc:"Opaque grade-level identifier."`
	PriorSurveyID *string  `json:"prior_survey_id,omitempty" doc:"Opaque earlier survey identifier."`
	ResponseState *string  `json:"response_state,omitempty" enum:"responded,not_responded"`
}

type InterestProfileSurveyQuestionInput struct {
	InterestAreaID string `json:"interest_area_id" minLength:"1" doc:"Opaque interest-area identifier."`
}

type InterestProfileSurveyScaleOptionInput struct {
	Value   string `json:"value" minLength:"1"`
	Label   string `json:"label" minLength:"1"`
	Ordinal int    `json:"ordinal,omitempty" minimum:"1"`
}

type InterestProfileSurveyInputBody struct {
	Name         string                                  `json:"name" minLength:"1"`
	Introduction string                                  `json:"introduction,omitempty"`
	OpensAt      *time.Time                              `json:"opens_at,omitempty" format:"date-time"`
	ClosesAt     *time.Time                              `json:"closes_at,omitempty" format:"date-time"`
	Audience     InterestProfileSurveyAudienceInput      `json:"audience"`
	ScaleVersion string                                  `json:"scale_version,omitempty"`
	Questions    []InterestProfileSurveyQuestionInput    `json:"questions,omitempty"`
	ScaleOptions []InterestProfileSurveyScaleOptionInput `json:"scale_options,omitempty"`
}

type InterestProfileSurveyResponse struct {
	ID                    string                                     `json:"id" doc:"Opaque survey identifier."`
	OrganizationID        string                                     `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID          string                                     `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID             string                                     `json:"program_id" doc:"Opaque program identifier."`
	Name                  string                                     `json:"name"`
	Introduction          string                                     `json:"introduction"`
	State                 string                                     `json:"state" enum:"draft,open,closed"`
	OpensAt               *time.Time                                 `json:"opens_at,omitempty" format:"date-time"`
	ClosesAt              *time.Time                                 `json:"closes_at,omitempty" format:"date-time"`
	AudienceType          string                                     `json:"audience_type" enum:"all_members,explicit_students,grade_level,response_state"`
	AudienceStudentIDs    []string                                   `json:"audience_student_ids" doc:"Opaque student identifiers in an explicit audience."`
	AudienceGradeLevelID  *string                                    `json:"audience_grade_level_id,omitempty" doc:"Opaque grade-level identifier."`
	AudiencePriorSurveyID *string                                    `json:"audience_prior_survey_id,omitempty" doc:"Opaque earlier survey identifier."`
	AudienceResponseState *string                                    `json:"audience_response_state,omitempty" enum:"responded,not_responded"`
	ScaleVersion          string                                     `json:"scale_version"`
	OpenedAt              *time.Time                                 `json:"opened_at,omitempty" format:"date-time"`
	Questions             []InterestProfileSurveyQuestionResponse    `json:"questions"`
	ScaleOptions          []InterestProfileSurveyScaleOptionResponse `json:"scale_options"`
	AudienceSnapshot      []string                                   `json:"audience_snapshot" doc:"Opaque student identifiers frozen when the survey opened."`
	ActiveCodes           []InterestProfileSurveyCodeResponse        `json:"active_codes"`
	CreatedAt             time.Time                                  `json:"created_at"`
	UpdatedAt             time.Time                                  `json:"updated_at"`
}

type InterestProfileSurveyQuestionResponse struct {
	ID             string `json:"id" doc:"Opaque survey-question identifier."`
	InterestAreaID string `json:"interest_area_id" doc:"Opaque interest-area identifier."`
	Ordinal        int    `json:"ordinal"`
	Label          string `json:"label"`
}

type InterestProfileSurveyScaleOptionResponse struct {
	ID      string `json:"id" doc:"Opaque scale-option identifier."`
	Value   string `json:"value"`
	Label   string `json:"label"`
	Ordinal int    `json:"ordinal"`
}

type InterestProfileSurveyCodeResponse struct {
	StudentID string     `json:"student_id" doc:"Opaque student identifier."`
	Code      string     `json:"code,omitempty" doc:"Plaintext code is returned only when newly issued."`
	IssuedAt  *time.Time `json:"issued_at,omitempty" format:"date-time"`
}

type InterestProfileSurveyListOutput struct {
	Body []InterestProfileSurveyResponse
}
type InterestProfileSurveyOutput struct{ Body InterestProfileSurveyResponse }
type InterestProfileSurveyPathInput struct {
	ProgramPathInput
	SurveyID string `path:"surveyID" minLength:"1" doc:"Opaque survey identifier."`
}
type ListInterestProfileSurveysInput struct{ ProgramPathInput }
type CreateInterestProfileSurveyInput struct {
	ProgramPathInput
	Body InterestProfileSurveyInputBody
}
type GetInterestProfileSurveyInput struct{ InterestProfileSurveyPathInput }
type UpdateInterestProfileSurveyInput struct {
	InterestProfileSurveyPathInput
	Body InterestProfileSurveyInputBody
}
type DeleteInterestProfileSurveyInput struct{ InterestProfileSurveyPathInput }
type TransitionInterestProfileSurveyInput struct {
	InterestProfileSurveyPathInput
	Body struct {
		State           string     `json:"state" enum:"open,closed"`
		ClosingAt       *time.Time `json:"closing_at,omitempty" format:"date-time"`
		RegenerateCodes bool       `json:"regenerate_codes,omitempty"`
		Reason          string     `json:"reason,omitempty"`
	}
}
type InterestProfileSurveyTransitionResponse struct {
	Survey      InterestProfileSurveyResponse       `json:"survey"`
	Warnings    []string                            `json:"warnings"`
	AccessCodes []InterestProfileSurveyCodeResponse `json:"access_codes"`
}
type InterestProfileSurveyTransitionOutput struct {
	Body InterestProfileSurveyTransitionResponse
}
type RegenerateInterestProfileSurveyCodesInput struct {
	InterestProfileSurveyPathInput
	Body struct {
		Reason string `json:"reason,omitempty"`
	}
}
type InterestProfileSurveyCodesOutput struct {
	Body []InterestProfileSurveyCodeResponse
}

func (h *ProgramHandler) ListInterestProfileSurveys(ctx context.Context, input *ListInterestProfileSurveysInput) (*InterestProfileSurveyListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	rows, err := h.service.ListInterestProfileSurveys(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID))
	if err != nil {
		return nil, surveyProblem(err)
	}
	result := make([]InterestProfileSurveyResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, surveyResponse(row))
	}
	return &InterestProfileSurveyListOutput{Body: result}, nil
}

func (h *ProgramHandler) CreateInterestProfileSurvey(ctx context.Context, input *CreateInterestProfileSurveyInput) (*InterestProfileSurveyOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	row, err := h.service.CreateInterestProfileSurvey(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), surveyInput(input.Body))
	if err != nil {
		return nil, surveyProblem(err)
	}
	return &InterestProfileSurveyOutput{Body: surveyResponse(row)}, nil
}

func (h *ProgramHandler) GetInterestProfileSurvey(ctx context.Context, input *GetInterestProfileSurveyInput) (*InterestProfileSurveyOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	row, err := h.service.GetInterestProfileSurvey(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID))
	if err != nil {
		return nil, surveyProblem(err)
	}
	return &InterestProfileSurveyOutput{Body: surveyResponse(row)}, nil
}

func (h *ProgramHandler) UpdateInterestProfileSurvey(ctx context.Context, input *UpdateInterestProfileSurveyInput) (*InterestProfileSurveyOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	row, err := h.service.UpdateInterestProfileSurvey(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), preference.InterestProfileSurveyUpdate{InterestProfileSurveyInput: surveyInput(input.Body)})
	if err != nil {
		return nil, surveyProblem(err)
	}
	return &InterestProfileSurveyOutput{Body: surveyResponse(row)}, nil
}

func (h *ProgramHandler) DeleteInterestProfileSurvey(ctx context.Context, input *DeleteInterestProfileSurveyInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	if err := h.service.DeleteInterestProfileSurvey(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID)); err != nil {
		return nil, surveyProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func (h *ProgramHandler) TransitionInterestProfileSurvey(ctx context.Context, input *TransitionInterestProfileSurveyInput) (*InterestProfileSurveyTransitionOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	result, err := h.service.TransitionInterestProfileSurvey(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyState(input.Body.State), ClosingAt: input.Body.ClosingAt, RegenerateCodes: input.Body.RegenerateCodes, Reason: input.Body.Reason})
	if err != nil {
		return nil, surveyProblem(err)
	}
	codes := make([]InterestProfileSurveyCodeResponse, 0, len(result.AccessCodes))
	for _, code := range result.AccessCodes {
		codes = append(codes, InterestProfileSurveyCodeResponse{StudentID: string(code.StudentID), Code: code.Code})
	}
	return &InterestProfileSurveyTransitionOutput{Body: InterestProfileSurveyTransitionResponse{Survey: surveyResponse(result.Survey), Warnings: result.Warnings, AccessCodes: codes}}, nil
}

func (h *ProgramHandler) RegenerateInterestProfileSurveyCodes(ctx context.Context, input *RegenerateInterestProfileSurveyCodesInput) (*InterestProfileSurveyCodesOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, surveyNotFound()
	}
	result, err := h.service.RegenerateInterestProfileSurveyCodes(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), input.Body.Reason)
	if err != nil {
		return nil, surveyProblem(err)
	}
	codes := make([]InterestProfileSurveyCodeResponse, 0, len(result))
	for _, code := range result {
		codes = append(codes, InterestProfileSurveyCodeResponse{StudentID: string(code.StudentID), Code: code.Code})
	}
	return &InterestProfileSurveyCodesOutput{Body: codes}, nil
}

func surveyInput(input InterestProfileSurveyInputBody) preference.InterestProfileSurveyInput {
	questions := make([]preference.InterestProfileSurveyQuestionInput, 0, len(input.Questions))
	for _, question := range input.Questions {
		questions = append(questions, preference.InterestProfileSurveyQuestionInput{InterestAreaID: ids.XID(question.InterestAreaID)})
	}
	options := make([]preference.InterestProfileSurveyScaleOptionInput, 0, len(input.ScaleOptions))
	for _, option := range input.ScaleOptions {
		options = append(options, preference.InterestProfileSurveyScaleOptionInput{Value: option.Value, Label: option.Label, Ordinal: option.Ordinal})
	}
	audience := preference.InterestProfileSurveyAudienceInput{Type: data.InterestProfileSurveyAudienceType(input.Audience.Type), GradeLevelID: optionalXID(input.Audience.GradeLevelID), PriorSurveyID: optionalXID(input.Audience.PriorSurveyID)}
	for _, studentID := range input.Audience.StudentIDs {
		audience.StudentIDs = append(audience.StudentIDs, ids.XID(studentID))
	}
	if input.Audience.ResponseState != nil {
		value := data.InterestProfileSurveyResponseState(*input.Audience.ResponseState)
		audience.ResponseState = &value
	}
	return preference.InterestProfileSurveyInput{Name: input.Name, Introduction: input.Introduction, OpensAt: input.OpensAt, ClosesAt: input.ClosesAt, Audience: audience, ScaleVersion: input.ScaleVersion, Questions: questions, ScaleOptions: options}
}

func surveyResponse(view preference.InterestProfileSurveyView) InterestProfileSurveyResponse {
	survey := view.Survey
	response := InterestProfileSurveyResponse{ID: string(survey.ID), OrganizationID: string(survey.OrganizationID), SchoolYearID: string(survey.SchoolYearID), ProgramID: string(survey.ProgramID), Name: survey.Name, Introduction: survey.Introduction, State: string(survey.State), OpensAt: survey.OpensAt, ClosesAt: survey.ClosesAt, AudienceType: string(survey.AudienceType), AudienceGradeLevelID: optionalString(survey.AudienceGradeLevelID), AudiencePriorSurveyID: optionalString(survey.AudiencePriorSurveyID), ScaleVersion: survey.ScaleVersion, OpenedAt: survey.OpenedAt, CreatedAt: survey.CreatedAt, UpdatedAt: survey.UpdatedAt}
	if survey.AudienceResponseState != nil {
		value := string(*survey.AudienceResponseState)
		response.AudienceResponseState = &value
	}
	for _, student := range view.DefinitionStudents {
		response.AudienceStudentIDs = append(response.AudienceStudentIDs, string(student.StudentID))
	}
	for _, question := range view.Questions {
		response.Questions = append(response.Questions, InterestProfileSurveyQuestionResponse{ID: string(question.ID), InterestAreaID: string(question.InterestAreaID), Ordinal: question.Ordinal, Label: question.Label})
	}
	for _, option := range view.ScaleOptions {
		response.ScaleOptions = append(response.ScaleOptions, InterestProfileSurveyScaleOptionResponse{ID: string(option.ID), Value: option.Value, Label: option.Label, Ordinal: option.Ordinal})
	}
	for _, snapshot := range view.AudienceSnapshot {
		response.AudienceSnapshot = append(response.AudienceSnapshot, string(snapshot.StudentID))
	}
	for _, code := range view.ActiveCodes {
		response.ActiveCodes = append(response.ActiveCodes, InterestProfileSurveyCodeResponse{StudentID: string(code.StudentID), IssuedAt: &code.IssuedAt})
	}
	return response
}

func optionalXID(value *string) *ids.XID {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	result := ids.XID(strings.TrimSpace(*value))
	return &result
}

func optionalString(value *ids.XID) *string {
	if value == nil {
		return nil
	}
	result := string(*value)
	return &result
}

func surveyNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "interest profile survey not found")
}

func surveyProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), strings.Contains(err.Error(), "interest profile survey not found"):
		return surveyNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.Is(err, preference.ErrSurveyDefinitionLocked), errors.Is(err, preference.ErrSurveyHasSubmissions), errors.Is(err, preference.ErrSurveyTransitionInvalid), errors.Is(err, preference.ErrSurveyNotAcceptingSubmissions):
		return problems.New(http.StatusConflict, problems.ProgramConflict, err.Error())
	case errors.Is(err, preference.ErrSurveyClosingTimeRequired), errors.Is(err, preference.ErrSurveyClosingTimeInvalid), errors.Is(err, preference.ErrSurveyQuestionRequired), errors.Is(err, preference.ErrSurveyScaleRequired), errors.Is(err, preference.ErrSurveyAudienceInvalid), strings.Contains(err.Error(), "requires a reason"), strings.Contains(err.Error(), "is required"), strings.Contains(err.Error(), "is invalid"), strings.Contains(err.Error(), "is repeated"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the survey definition conflicts with existing data")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change interest profile survey data")
	}
}
