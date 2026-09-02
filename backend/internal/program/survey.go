package program

import (
	"context"

	"github.com/chrismott/miniclass/internal/audit"
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
