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
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/jackc/pgx/v5"
)

type PreferenceHandler struct{ service ProgramService }

func NewPreferenceHandler(service ProgramService) *PreferenceHandler {
	return &PreferenceHandler{service: service}
}

type PreferenceFormQuestionResponse struct {
	InterestAreaID string `json:"interest_area_id" doc:"Opaque interest-area identifier."`
	Label          string `json:"label"`
	Ordinal        int    `json:"ordinal"`
}

type PreferenceFormScaleOptionResponse struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	Ordinal int    `json:"ordinal"`
}

type PreferenceFormOfferingResponse struct {
	ID                  string   `json:"id" doc:"Opaque offering identifier."`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	MinGradeLevelID     string   `json:"min_grade_level_id" doc:"Opaque grade-level identifier."`
	MaxGradeLevelID     string   `json:"max_grade_level_id" doc:"Opaque grade-level identifier."`
	Location            string   `json:"location"`
	MeetingPoint        string   `json:"meeting_point"`
	MeetingInstructions string   `json:"meeting_instructions"`
	MeetingDates        []string `json:"meeting_dates" format:"date"`
}

type PreferenceFormInterestAnswerResponse struct {
	InterestAreaID string  `json:"interest_area_id" doc:"Opaque interest-area identifier."`
	Rating         *string `json:"rating,omitempty" enum:"very_interested,interested,not_interested,unrated"`
}

type PreferenceFormRankedAnswerResponse struct {
	OfferingID string `json:"offering_id" doc:"Opaque offering identifier."`
	Answer     string `json:"answer" enum:"ranked,interested,not_interested,no_response"`
	Rank       *int   `json:"rank,omitempty"`
}

type PreferenceFormResponse struct {
	Type            string                                 `json:"type" enum:"interest_profile,ranked_choice"`
	ID              string                                 `json:"id" doc:"Opaque survey or session identifier."`
	SchoolYearID    string                                 `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID       string                                 `json:"program_id" doc:"Opaque program identifier."`
	SessionID       *string                                `json:"session_id,omitempty" doc:"Opaque session identifier."`
	ProgramName     string                                 `json:"program_name"`
	SessionName     string                                 `json:"session_name,omitempty"`
	Name            string                                 `json:"name"`
	Introduction    string                                 `json:"introduction,omitempty"`
	StudentID       string                                 `json:"student_id" doc:"Opaque student identifier bound to this form."`
	StudentName     string                                 `json:"student_name,omitempty" doc:"Shown to the student identified by a ranked-choice code, guardians and administrators."`
	ClosesAt        *time.Time                             `json:"closes_at,omitempty" format:"date-time"`
	RankDepth       int                                    `json:"rank_depth,omitempty"`
	Questions       []PreferenceFormQuestionResponse       `json:"questions,omitempty"`
	ScaleOptions    []PreferenceFormScaleOptionResponse    `json:"scale_options,omitempty"`
	Offerings       []PreferenceFormOfferingResponse       `json:"offerings,omitempty"`
	InterestAnswers []PreferenceFormInterestAnswerResponse `json:"interest_answers,omitempty"`
	RankedAnswers   []PreferenceFormRankedAnswerResponse   `json:"ranked_answers,omitempty"`
	SubmittedAt     *time.Time                             `json:"submitted_at,omitempty" format:"date-time"`
}

type GuardianPreferenceStudentResponse struct {
	StudentID   string                   `json:"student_id" doc:"Opaque student identifier."`
	DisplayName string                   `json:"display_name"`
	Forms       []PreferenceFormResponse `json:"forms"`
}

type GuardianPreferenceFormsResponse struct {
	SchoolYearID string                              `json:"school_year_id" doc:"Opaque school-year identifier."`
	Students     []GuardianPreferenceStudentResponse `json:"students"`
}

type ResponseTrackingBreakdownResponse struct {
	ID                   string  `json:"id" doc:"Opaque grade-level or homeroom identifier; empty means no grade is assigned."`
	Label                string  `json:"label"`
	TotalStudents        int     `json:"total_students"`
	RespondedStudents    int     `json:"responded_students"`
	CompletionPercentage float64 `json:"completion_percentage" minimum:"0" maximum:"100"`
}

type ResponseTrackingNonResponderResponse struct {
	StudentID     string  `json:"student_id" doc:"Opaque student identifier."`
	DisplayName   string  `json:"display_name"`
	GradeLevelID  *string `json:"grade_level_id" nullable:"true"`
	GradeLabel    string  `json:"grade_label"`
	HomeroomID    string  `json:"homeroom_id" doc:"Opaque homeroom identifier."`
	HomeroomName  string  `json:"homeroom_name"`
	ContactStatus string  `json:"contact_status" enum:"unreachable,guardian_follow_up"`
}

type ResponseTrackingGuardianFollowUpResponse struct {
	AdultID       string  `json:"adult_id" doc:"Opaque adult identifier."`
	AdultName     string  `json:"adult_name"`
	Email         *string `json:"email" nullable:"true"`
	StudentID     string  `json:"student_id" doc:"Opaque student identifier."`
	StudentName   string  `json:"student_name"`
	ContactStatus string  `json:"contact_status" enum:"no_email,not_responded"`
}

type ResponseTrackingSummaryResponse struct {
	InstrumentType       string  `json:"instrument_type" enum:"interest_profile_survey,ranked_choice_session"`
	InstrumentID         string  `json:"instrument_id" doc:"Opaque survey or session identifier."`
	InstrumentName       string  `json:"instrument_name"`
	State                string  `json:"state"`
	SchoolYearID         string  `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID            string  `json:"program_id" doc:"Opaque program identifier."`
	TotalStudents        int     `json:"total_students"`
	RespondedStudents    int     `json:"responded_students"`
	CompletionPercentage float64 `json:"completion_percentage" minimum:"0" maximum:"100"`
}

type ResponseTrackingSummaryOutput struct {
	Body []ResponseTrackingSummaryResponse
}

type ResponseTrackingResponse struct {
	InstrumentType       string                                     `json:"instrument_type" enum:"interest_profile_survey,ranked_choice_session"`
	InstrumentID         string                                     `json:"instrument_id" doc:"Opaque survey or session identifier."`
	InstrumentName       string                                     `json:"instrument_name"`
	SchoolYearID         string                                     `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID            string                                     `json:"program_id" doc:"Opaque program identifier."`
	TotalStudents        int                                        `json:"total_students"`
	RespondedStudents    int                                        `json:"responded_students"`
	CompletionPercentage float64                                    `json:"completion_percentage" minimum:"0" maximum:"100"`
	GradeBreakdown       []ResponseTrackingBreakdownResponse        `json:"grade_breakdown"`
	HomeroomBreakdown    []ResponseTrackingBreakdownResponse        `json:"homeroom_breakdown"`
	NonResponders        []ResponseTrackingNonResponderResponse     `json:"non_responders"`
	GuardianFollowUp     []ResponseTrackingGuardianFollowUpResponse `json:"guardian_follow_up"`
}

type ResponseTrackingOutput struct{ Body ResponseTrackingResponse }

type PreferenceFormOutput struct{ Body PreferenceFormResponse }
type GuardianPreferenceFormsOutput struct {
	Body GuardianPreferenceFormsResponse
}

type PreferenceFormPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	ProgramID    string `path:"programID" minLength:"1" doc:"Opaque program identifier."`
}

type InterestProfilePreferenceFormPathInput struct {
	PreferenceFormPathInput
	SurveyID string `path:"surveyID" minLength:"1" doc:"Opaque survey identifier."`
}

type RankedChoicePreferenceFormPathInput struct {
	PreferenceFormPathInput
	SessionID string `path:"sessionID" minLength:"1" doc:"Opaque session identifier."`
}

type PreferenceStudentPathInput struct {
	InterestProfilePreferenceFormPathInput
	StudentID string `path:"studentID" minLength:"1" doc:"Opaque student identifier."`
}

type RankedChoiceStudentPathInput struct {
	RankedChoicePreferenceFormPathInput
	StudentID string `path:"studentID" minLength:"1" doc:"Opaque student identifier."`
}

type InterestProfileAnswerInput struct {
	InterestAreaID string `json:"interest_area_id" minLength:"1" doc:"Opaque interest-area identifier."`
	Rating         string `json:"rating" enum:"very_interested,interested,not_interested,unrated"`
}

type RankedChoiceAnswerInput struct {
	OfferingID string `json:"offering_id" minLength:"1" doc:"Opaque offering identifier."`
	Answer     string `json:"answer" enum:"ranked,interested,not_interested,no_response"`
	Rank       *int   `json:"rank,omitempty" minimum:"1"`
}

type InterestProfileSubmitBody struct {
	OrganizationID string                       `json:"organization_id" minLength:"1" doc:"Opaque organization identifier from the respondent link."`
	Code           string                       `json:"code" minLength:"1" doc:"High-entropy instrument-bound access code."`
	Answers        []InterestProfileAnswerInput `json:"answers" minItems:"1"`
}

type RankedChoiceSubmitBody struct {
	OrganizationID string                    `json:"organization_id" minLength:"1" doc:"Opaque organization identifier from the respondent link."`
	Code           string                    `json:"code" minLength:"1" doc:"High-entropy instrument-bound access code."`
	Responses      []RankedChoiceAnswerInput `json:"responses" minItems:"1"`
}

type InterestProfileStudentCodeInput struct {
	InterestProfilePreferenceFormPathInput
	Body struct {
		OrganizationID string `json:"organization_id" minLength:"1"`
		Code           string `json:"code" minLength:"1"`
	}
}

type InterestProfileStudentCodeSubmitInput struct {
	InterestProfilePreferenceFormPathInput
	Body InterestProfileSubmitBody
}

type RankedChoiceStudentCodeInput struct {
	RankedChoicePreferenceFormPathInput
	Body struct {
		OrganizationID string `json:"organization_id" minLength:"1"`
		Code           string `json:"code" minLength:"1"`
	}
}

type RankedChoiceStudentCodeSubmitInput struct {
	RankedChoicePreferenceFormPathInput
	Body RankedChoiceSubmitBody
}

type InterestProfileGuardianSubmitInput struct {
	PreferenceStudentPathInput
	Body struct {
		Answers []InterestProfileAnswerInput `json:"answers" minItems:"1"`
	}
}

type RankedChoiceGuardianSubmitInput struct {
	RankedChoiceStudentPathInput
	Body struct {
		Responses []RankedChoiceAnswerInput `json:"responses" minItems:"1"`
	}
}

type AdministratorPreferenceFormInput struct {
	Body struct {
		Type         string `json:"type" enum:"interest_profile,ranked_choice"`
		SchoolYearID string `json:"school_year_id" minLength:"1"`
		ProgramID    string `json:"program_id" minLength:"1"`
		InstrumentID string `json:"instrument_id" minLength:"1" doc:"Opaque survey or session identifier."`
		StudentID    string `json:"student_id" minLength:"1" doc:"Opaque selected student identifier."`
	}
}

type InterestProfileAdministratorSubmitInput struct {
	PreferenceStudentPathInput
	Body struct {
		Answers []InterestProfileAnswerInput `json:"answers" minItems:"1"`
	}
}

type RankedChoiceAdministratorSubmitInput struct {
	RankedChoiceStudentPathInput
	Body struct {
		Responses []RankedChoiceAnswerInput `json:"responses" minItems:"1"`
	}
}

type InterestProfileResponseTrackingInput struct {
	InterestProfilePreferenceFormPathInput
}

type RankedChoiceResponseTrackingInput struct {
	RankedChoicePreferenceFormPathInput
}

func (h *PreferenceHandler) GuardianForms(ctx context.Context, _ *struct{}) (*GuardianPreferenceFormsOutput, error) {
	guardian, err := guardianPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, preferenceServiceUnavailable()
	}
	forms, err := h.service.ListGuardianPreferenceForms(ctx, string(guardian.OrganizationID), guardian.SchoolYearID, guardian.AdultID)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	students := make([]GuardianPreferenceStudentResponse, 0, len(forms.Students))
	for _, student := range forms.Students {
		studentForms := make([]PreferenceFormResponse, 0, len(student.Forms))
		for _, form := range student.Forms {
			studentForms = append(studentForms, preferenceFormResponse(form, true))
		}
		students = append(students, GuardianPreferenceStudentResponse{StudentID: string(student.StudentID), DisplayName: student.DisplayName, Forms: studentForms})
	}
	return &GuardianPreferenceFormsOutput{Body: GuardianPreferenceFormsResponse{SchoolYearID: string(forms.SchoolYearID), Students: students}}, nil
}

func (h *PreferenceHandler) StudentCodeInterestForm(ctx context.Context, input *InterestProfileStudentCodeInput) (*PreferenceFormOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	form, err := h.service.GetInterestProfileFormByCode(ctx, input.Body.OrganizationID, ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), input.Body.Code)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, false)}, nil
}

func (h *PreferenceHandler) StudentCodeInterestSubmit(ctx context.Context, input *InterestProfileStudentCodeSubmitInput) (*PreferenceFormOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: "student-code respondent"}
	_, err := h.service.SubmitInterestProfileSurvey(ctx, input.Body.OrganizationID, actor, preference.InterestProfileSurveySubmissionInput{SchoolYearID: ids.XID(input.SchoolYearID), ProgramID: ids.XID(input.ProgramID), SurveyID: ids.XID(input.SurveyID), Code: input.Body.Code, Channel: data.PreferenceChannelStudentCode, Answers: interestAnswers(input.Body.Answers)})
	if err != nil {
		return nil, preferenceProblem(err)
	}
	form, err := h.service.GetInterestProfileFormByCode(ctx, input.Body.OrganizationID, ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), input.Body.Code)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, false)}, nil
}

func (h *PreferenceHandler) StudentCodeRankedForm(ctx context.Context, input *RankedChoiceStudentCodeInput) (*PreferenceFormOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	form, err := h.service.GetRankedChoiceFormByCode(ctx, input.Body.OrganizationID, ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), input.Body.Code)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) StudentCodeRankedSubmit(ctx context.Context, input *RankedChoiceStudentCodeSubmitInput) (*PreferenceFormOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: "student-code respondent"}
	_, err := h.service.SubmitRankedChoices(ctx, input.Body.OrganizationID, actor, preference.RankedChoiceSubmissionInput{SchoolYearID: ids.XID(input.SchoolYearID), ProgramID: ids.XID(input.ProgramID), SessionID: ids.XID(input.SessionID), Code: input.Body.Code, Channel: data.PreferenceChannelStudentCode, Responses: rankedAnswers(input.Body.Responses)})
	if err != nil {
		return nil, preferenceProblem(err)
	}
	form, err := h.service.GetRankedChoiceFormByCode(ctx, input.Body.OrganizationID, ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), input.Body.Code)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) GuardianInterestSubmit(ctx context.Context, input *InterestProfileGuardianSubmitInput) (*PreferenceFormOutput, error) {
	guardian, err := guardianPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: guardian.Email}
	adultID := guardian.AdultID
	result, err := h.service.SubmitInterestProfileSurvey(ctx, string(guardian.OrganizationID), actor, preference.InterestProfileSurveySubmissionInput{SchoolYearID: ids.XID(input.SchoolYearID), ProgramID: ids.XID(input.ProgramID), SurveyID: ids.XID(input.SurveyID), StudentID: ids.XID(input.StudentID), Channel: data.PreferenceChannelGuardian, ActorAdultID: &adultID, GuardianAdultID: &adultID, Answers: interestAnswers(input.Body.Answers)})
	if err != nil {
		return nil, preferenceProblem(err)
	}
	form, err := h.service.GetInterestProfileForm(ctx, string(guardian.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), result.StudentID)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) GuardianRankedSubmit(ctx context.Context, input *RankedChoiceGuardianSubmitInput) (*PreferenceFormOutput, error) {
	guardian, err := guardianPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: guardian.Email}
	adultID := guardian.AdultID
	result, err := h.service.SubmitRankedChoices(ctx, string(guardian.OrganizationID), actor, preference.RankedChoiceSubmissionInput{SchoolYearID: ids.XID(input.SchoolYearID), ProgramID: ids.XID(input.ProgramID), SessionID: ids.XID(input.SessionID), StudentID: ids.XID(input.StudentID), Channel: data.PreferenceChannelGuardian, ActorAdultID: &adultID, GuardianAdultID: &adultID, Responses: rankedAnswers(input.Body.Responses)})
	if err != nil {
		return nil, preferenceProblem(err)
	}
	form, err := h.service.GetRankedChoiceForm(ctx, string(guardian.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), result.StudentID)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) AdministratorForm(ctx context.Context, input *AdministratorPreferenceFormInput) (*PreferenceFormOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	var form preference.PreferenceForm
	switch strings.TrimSpace(input.Body.Type) {
	case string(preference.FormTypeInterestProfile):
		form, err = h.service.GetInterestProfileForm(ctx, string(account.OrganizationID), ids.XID(input.Body.SchoolYearID), ids.XID(input.Body.ProgramID), ids.XID(input.Body.InstrumentID), ids.XID(input.Body.StudentID))
	case string(preference.FormTypeRankedChoice):
		form, err = h.service.GetRankedChoiceForm(ctx, string(account.OrganizationID), ids.XID(input.Body.SchoolYearID), ids.XID(input.Body.ProgramID), ids.XID(input.Body.InstrumentID), ids.XID(input.Body.StudentID))
	default:
		return nil, problems.New(http.StatusBadRequest, problems.ProgramConflict, "preference form type is invalid")
	}
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) AdministratorInterestSubmit(ctx context.Context, input *InterestProfileAdministratorSubmitInput) (*PreferenceFormOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	result, err := h.service.SubmitInterestProfileSurvey(ctx, string(account.OrganizationID), programActor(account), preference.InterestProfileSurveySubmissionInput{SchoolYearID: ids.XID(input.SchoolYearID), ProgramID: ids.XID(input.ProgramID), SurveyID: ids.XID(input.SurveyID), StudentID: ids.XID(input.StudentID), Channel: data.PreferenceChannelAdministratorOnBehalf, Answers: interestAnswers(input.Body.Answers)})
	if err != nil {
		return nil, preferenceProblem(err)
	}
	form, err := h.service.GetInterestProfileForm(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID), result.StudentID)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) AdministratorRankedSubmit(ctx context.Context, input *RankedChoiceAdministratorSubmitInput) (*PreferenceFormOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	result, err := h.service.SubmitRankedChoices(ctx, string(account.OrganizationID), programActor(account), preference.RankedChoiceSubmissionInput{SchoolYearID: ids.XID(input.SchoolYearID), ProgramID: ids.XID(input.ProgramID), SessionID: ids.XID(input.SessionID), StudentID: ids.XID(input.StudentID), Channel: data.PreferenceChannelAdministratorOnBehalf, Responses: rankedAnswers(input.Body.Responses)})
	if err != nil {
		return nil, preferenceProblem(err)
	}
	form, err := h.service.GetRankedChoiceForm(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), result.StudentID)
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &PreferenceFormOutput{Body: preferenceFormResponse(form, true)}, nil
}

func (h *PreferenceHandler) ResponseTrackingSummaries(ctx context.Context, input *PreferenceFormPathInput) (*ResponseTrackingSummaryOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	summaries, err := h.service.ListResponseTrackingSummaries(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID))
	if err != nil {
		return nil, preferenceProblem(err)
	}
	result := make([]ResponseTrackingSummaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		result = append(result, ResponseTrackingSummaryResponse{
			InstrumentType: string(summary.InstrumentType), InstrumentID: string(summary.InstrumentID),
			InstrumentName: summary.InstrumentName, State: summary.State, SchoolYearID: string(summary.SchoolYearID),
			ProgramID: string(summary.ProgramID), TotalStudents: summary.TotalStudents,
			RespondedStudents: summary.RespondedStudents, CompletionPercentage: summary.CompletionPercentage,
		})
	}
	return &ResponseTrackingSummaryOutput{Body: result}, nil
}

func (h *PreferenceHandler) InterestProfileResponseTracking(ctx context.Context, input *InterestProfileResponseTrackingInput) (*ResponseTrackingOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	tracking, err := h.service.GetInterestProfileResponseTracking(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SurveyID))
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &ResponseTrackingOutput{Body: responseTrackingResponse(tracking)}, nil
}

func (h *PreferenceHandler) RankedChoiceResponseTracking(ctx context.Context, input *RankedChoiceResponseTrackingInput) (*ResponseTrackingOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, preferenceServiceUnavailable()
	}
	tracking, err := h.service.GetRankedChoiceResponseTracking(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, preferenceProblem(err)
	}
	return &ResponseTrackingOutput{Body: responseTrackingResponse(tracking)}, nil
}

func preferenceFormResponse(form preference.PreferenceForm, includeStudentName bool) PreferenceFormResponse {
	var sessionID *string
	if form.SessionID != "" {
		value := string(form.SessionID)
		sessionID = &value
	}
	result := PreferenceFormResponse{Type: string(form.Type), ID: string(form.ID), SchoolYearID: string(form.SchoolYearID), ProgramID: string(form.ProgramID), SessionID: sessionID, ProgramName: form.ProgramName, SessionName: form.SessionName, Name: form.Name, Introduction: form.Introduction, StudentID: string(form.StudentID), ClosesAt: form.ClosesAt, RankDepth: form.RankDepth, Questions: []PreferenceFormQuestionResponse{}, ScaleOptions: []PreferenceFormScaleOptionResponse{}, Offerings: []PreferenceFormOfferingResponse{}, InterestAnswers: []PreferenceFormInterestAnswerResponse{}, RankedAnswers: []PreferenceFormRankedAnswerResponse{}, SubmittedAt: form.SubmittedAt}
	if includeStudentName {
		result.StudentName = form.StudentName
	}
	for _, question := range form.Questions {
		result.Questions = append(result.Questions, PreferenceFormQuestionResponse{InterestAreaID: string(question.InterestAreaID), Label: question.Label, Ordinal: question.Ordinal})
	}
	for _, option := range form.ScaleOptions {
		result.ScaleOptions = append(result.ScaleOptions, PreferenceFormScaleOptionResponse{Value: option.Value, Label: option.Label, Ordinal: option.Ordinal})
	}
	for _, offering := range form.Offerings {
		dates := make([]string, 0, len(offering.MeetingDates))
		for _, date := range offering.MeetingDates {
			dates = append(dates, date.Format("2006-01-02"))
		}
		result.Offerings = append(result.Offerings, PreferenceFormOfferingResponse{ID: string(offering.ID), Name: offering.Name, Description: offering.Description, MinGradeLevelID: string(offering.MinGradeLevelID), MaxGradeLevelID: string(offering.MaxGradeLevelID), Location: offering.Location, MeetingPoint: offering.MeetingPoint, MeetingInstructions: offering.MeetingInstructions, MeetingDates: dates})
	}
	for _, answer := range form.InterestAnswers {
		var rating *string
		if answer.Rating != nil {
			value := string(*answer.Rating)
			rating = &value
		}
		result.InterestAnswers = append(result.InterestAnswers, PreferenceFormInterestAnswerResponse{InterestAreaID: string(answer.InterestAreaID), Rating: rating})
	}
	for _, answer := range form.RankedAnswers {
		result.RankedAnswers = append(result.RankedAnswers, PreferenceFormRankedAnswerResponse{OfferingID: string(answer.OfferingID), Answer: string(answer.Answer), Rank: answer.Rank})
	}
	return result
}

func responseTrackingResponse(value preference.ResponseTracking) ResponseTrackingResponse {
	result := ResponseTrackingResponse{
		InstrumentType: string(value.InstrumentType), InstrumentID: string(value.InstrumentID), InstrumentName: value.InstrumentName,
		SchoolYearID: string(value.SchoolYearID), ProgramID: string(value.ProgramID), TotalStudents: value.TotalStudents,
		RespondedStudents: value.RespondedStudents, CompletionPercentage: value.CompletionPercentage,
		GradeBreakdown:    make([]ResponseTrackingBreakdownResponse, 0, len(value.GradeBreakdown)),
		HomeroomBreakdown: make([]ResponseTrackingBreakdownResponse, 0, len(value.HomeroomBreakdown)),
		NonResponders:     make([]ResponseTrackingNonResponderResponse, 0, len(value.NonResponders)),
		GuardianFollowUp:  make([]ResponseTrackingGuardianFollowUpResponse, 0, len(value.GuardianFollowUp)),
	}
	for _, row := range value.GradeBreakdown {
		result.GradeBreakdown = append(result.GradeBreakdown, responseTrackingBreakdownResponse(row))
	}
	for _, row := range value.HomeroomBreakdown {
		result.HomeroomBreakdown = append(result.HomeroomBreakdown, responseTrackingBreakdownResponse(row))
	}
	for _, row := range value.NonResponders {
		result.NonResponders = append(result.NonResponders, ResponseTrackingNonResponderResponse{
			StudentID: string(row.StudentID), DisplayName: row.DisplayName, GradeLevelID: optionalXIDString(row.GradeLevelID),
			GradeLabel: row.GradeLabel, HomeroomID: string(row.HomeroomID), HomeroomName: row.HomeroomName, ContactStatus: row.ContactStatus,
		})
	}
	for _, row := range value.GuardianFollowUp {
		result.GuardianFollowUp = append(result.GuardianFollowUp, ResponseTrackingGuardianFollowUpResponse{
			AdultID: string(row.AdultID), AdultName: row.AdultName, Email: row.Email, StudentID: string(row.StudentID),
			StudentName: row.StudentName, ContactStatus: row.ContactStatus,
		})
	}
	return result
}

func responseTrackingBreakdownResponse(value preference.ResponseTrackingBreakdown) ResponseTrackingBreakdownResponse {
	return ResponseTrackingBreakdownResponse{ID: value.ID, Label: value.Label, TotalStudents: value.TotalStudents, RespondedStudents: value.RespondedStudents, CompletionPercentage: value.CompletionPercentage}
}

func interestAnswers(values []InterestProfileAnswerInput) []data.InterestProfileAnswer {
	result := make([]data.InterestProfileAnswer, 0, len(values))
	for _, value := range values {
		result = append(result, data.InterestProfileAnswer{InterestAreaID: ids.XID(value.InterestAreaID), Rating: data.InterestProfileRating(value.Rating)})
	}
	return result
}

func rankedAnswers(values []RankedChoiceAnswerInput) []data.RankedChoiceResponseInput {
	result := make([]data.RankedChoiceResponseInput, 0, len(values))
	for _, value := range values {
		result = append(result, data.RankedChoiceResponseInput{OfferingID: ids.XID(value.OfferingID), Answer: data.RankedChoiceAnswer(value.Answer), Rank: value.Rank})
	}
	return result
}

func preferenceServiceUnavailable() error {
	return problems.New(http.StatusInternalServerError, problems.InternalError, "preference service is not configured")
}

func preferenceProblem(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, preference.ErrSurveyCodeInvalid), errors.Is(err, preference.ErrRankedChoiceCodeInvalid):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "the preference access code is invalid or revoked")
	case errors.Is(err, preference.ErrPreferenceStudentNotProgramMember):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, "the selected student is not included in the selected program")
	case errors.Is(err, preference.ErrPreferenceFormNotAvailable), errors.Is(err, preference.ErrSurveyNotAcceptingSubmissions), errors.Is(err, preference.ErrRankedChoiceNotAccepting), errors.Is(err, preference.ErrRankedChoiceDeadlinePassed):
		return problems.New(http.StatusConflict, problems.ProgramConflict, "this preference form is no longer accepting submissions")
	case errors.Is(err, preference.ErrPreferenceStudentOutOfScope), errors.Is(err, preference.ErrRankedChoiceGuardianScope), errors.Is(err, preference.ErrSurveyStudentExcluded), errors.Is(err, preference.ErrRankedChoiceStudentExcluded):
		return problems.New(http.StatusForbidden, problems.CapabilityRequired, "the selected student is outside the respondent scope")
	case errors.Is(err, preference.ErrRankedChoiceNotConfigured):
		return problems.New(http.StatusConflict, problems.ProgramConflict, "ranked-choice voting is not configured for this session")
	case errors.Is(err, preference.ErrRankedChoiceCodeRequired):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, "a student access code is required")
	case errors.Is(err, preference.ErrRankedChoiceStudentMismatch):
		return problems.New(http.StatusForbidden, problems.CapabilityRequired, "the access code is not bound to the selected student")
	case errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "the preference instrument or student was not found")
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.Is(err, preference.ErrRankedChoiceNotComplete), errors.Is(err, preference.ErrRankedChoiceInvalid), errors.Is(err, preference.ErrInterestAreaNotInProgram), strings.Contains(err.Error(), "required"), strings.Contains(err.Error(), "invalid"), strings.Contains(err.Error(), "repeated"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to complete preference submission")
	}
}
