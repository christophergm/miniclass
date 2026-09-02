package program

import (
	"context"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/preference"
)

// Survey operations live in the preference domain while the API's program
// handler remains the route owner for the year/program URL hierarchy.
func (s *Service) CreateInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID ids.XID, input preference.InterestProfileSurveyInput) (preference.InterestProfileSurveyView, error) {
	return preference.New(s.database).CreateInterestProfileSurvey(ctx, organizationID, actor, schoolYearID, programID, input)
}

func (s *Service) ListInterestProfileSurveys(ctx context.Context, organizationID string, schoolYearID, programID ids.XID) ([]preference.InterestProfileSurveyView, error) {
	return preference.New(s.database).ListInterestProfileSurveys(ctx, organizationID, schoolYearID, programID)
}

func (s *Service) GetInterestProfileSurvey(ctx context.Context, organizationID string, schoolYearID, programID, surveyID ids.XID) (preference.InterestProfileSurveyView, error) {
	return preference.New(s.database).GetInterestProfileSurvey(ctx, organizationID, schoolYearID, programID, surveyID)
}

func (s *Service) UpdateInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, input preference.InterestProfileSurveyUpdate) (preference.InterestProfileSurveyView, error) {
	return preference.New(s.database).UpdateInterestProfileSurvey(ctx, organizationID, actor, schoolYearID, programID, surveyID, input)
}

func (s *Service) DeleteInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID) error {
	return preference.New(s.database).DeleteInterestProfileSurvey(ctx, organizationID, actor, schoolYearID, programID, surveyID)
}

func (s *Service) TransitionInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, input preference.InterestProfileSurveyTransitionInput) (preference.InterestProfileSurveyTransitionResult, error) {
	return preference.New(s.database).TransitionInterestProfileSurvey(ctx, organizationID, actor, schoolYearID, programID, surveyID, input)
}

func (s *Service) RegenerateInterestProfileSurveyCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, reason string) ([]preference.SurveyAccessCode, error) {
	return preference.New(s.database).RegenerateInterestProfileSurveyCodes(ctx, organizationID, actor, schoolYearID, programID, surveyID, reason)
}

func (s *Service) RevokeInterestProfileSurveyCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, reason string) error {
	return preference.New(s.database).RevokeInterestProfileSurveyCodes(ctx, organizationID, actor, schoolYearID, programID, surveyID, reason)
}

func (s *Service) GetInterestProfileForm(ctx context.Context, organizationID string, schoolYearID, programID, surveyID, studentID ids.XID) (preference.PreferenceForm, error) {
	return preference.New(s.database).GetInterestProfileForm(ctx, organizationID, schoolYearID, programID, surveyID, studentID)
}

func (s *Service) GetInterestProfileFormByCode(ctx context.Context, organizationID string, schoolYearID, programID, surveyID ids.XID, code string) (preference.PreferenceForm, error) {
	return preference.New(s.database).GetInterestProfileFormByCode(ctx, organizationID, schoolYearID, programID, surveyID, code)
}

func (s *Service) SubmitInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, input preference.InterestProfileSurveySubmissionInput) (data.InterestProfileSubmission, error) {
	return preference.New(s.database).SubmitInterestProfileSurvey(ctx, organizationID, actor, input)
}

func (s *Service) GetInterestProfileResponseTracking(ctx context.Context, organizationID string, schoolYearID, programID, surveyID ids.XID) (preference.ResponseTracking, error) {
	return preference.New(s.database).GetInterestProfileResponseTracking(ctx, organizationID, schoolYearID, programID, surveyID)
}

func (s *Service) ListGuardianPreferenceForms(ctx context.Context, organizationID string, schoolYearID, adultID ids.XID) (preference.GuardianPreferenceForms, error) {
	return preference.New(s.database).ListGuardianPreferenceForms(ctx, organizationID, schoolYearID, adultID)
}

func (s *Service) GetRankedChoiceForm(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, studentID ids.XID) (preference.PreferenceForm, error) {
	return preference.New(s.database).GetRankedChoiceForm(ctx, organizationID, schoolYearID, programID, sessionID, studentID)
}

func (s *Service) GetRankedChoiceFormByCode(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID, code string) (preference.PreferenceForm, error) {
	return preference.New(s.database).GetRankedChoiceFormByCode(ctx, organizationID, schoolYearID, programID, sessionID, code)
}

func (s *Service) SubmitRankedChoices(ctx context.Context, organizationID string, actor audit.Actor, input preference.RankedChoiceSubmissionInput) (data.RankedChoiceSubmission, error) {
	return preference.New(s.database).SubmitRankedChoices(ctx, organizationID, actor, input)
}

func (s *Service) GetRankedChoiceResponseTracking(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (preference.ResponseTracking, error) {
	return preference.New(s.database).GetRankedChoiceResponseTracking(ctx, organizationID, schoolYearID, programID, sessionID)
}
