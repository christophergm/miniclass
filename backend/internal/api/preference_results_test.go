package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/stretchr/testify/require"
)

type preferenceResultsService struct {
	handlers.ProgramService
	org                       string
	year, program, instrument ids.XID
	filter                    data.PreferenceResultsFilter
}

func (s *preferenceResultsService) record(org string, year, program, instrument ids.XID, filter data.PreferenceResultsFilter) {
	s.org, s.year, s.program, s.instrument, s.filter = org, year, program, instrument, filter
}
func (s *preferenceResultsService) GetInterestProfileResults(_ context.Context, org string, year, program, survey ids.XID, filter data.PreferenceResultsFilter) (preference.InterestProfileResults, error) {
	s.record(org, year, program, survey, filter)
	return preference.InterestProfileResults{ResponseTrackingSummary: preference.ResponseTrackingSummary{InstrumentType: preference.ResponseTrackingInterestProfile, InstrumentID: survey, State: "closed", TotalStudents: 2, RespondedStudents: 1, CompletionPercentage: 50}, ScaleVersion: "custom", ScaleOptions: []preference.PreferenceFormScaleOption{{Value: "not_interested", Label: "No", Ordinal: 1}}, Items: []preference.InterestProfileResultItem{{ID: "area", Label: "Area", ExplicitAnswers: 1, RatingCounts: []preference.RatingCount{{Value: "not_interested", Label: "No", Ordinal: 1, Count: 1}}}}}, nil
}
func (s *preferenceResultsService) GetRankedChoiceResults(_ context.Context, org string, year, program, session ids.XID, filter data.PreferenceResultsFilter) (preference.RankedChoiceResults, error) {
	s.record(org, year, program, session, filter)
	return preference.RankedChoiceResults{ResponseTrackingSummary: preference.ResponseTrackingSummary{InstrumentType: preference.ResponseTrackingRankedChoice, InstrumentID: session, State: "voting_open", TotalStudents: 2, RespondedStudents: 1}, RankDepth: 2, Items: []preference.RankedChoiceResultItem{{ID: "offering", Label: "Offering", ExplicitAnswers: 1, NotInterested: 1, RankCounts: []preference.RankCount{{Rank: 1}, {Rank: 2}}}}}, nil
}
func (s *preferenceResultsService) GetInterestProfileResponseTracking(_ context.Context, org string, year, program, survey ids.XID, filters ...data.PreferenceResultsFilter) (preference.ResponseTracking, error) {
	s.record(org, year, program, survey, filters[0])
	return preference.ResponseTracking{}, nil
}
func (s *preferenceResultsService) GetRankedChoiceResponseTracking(_ context.Context, org string, year, program, session ids.XID, filters ...data.PreferenceResultsFilter) (preference.ResponseTracking, error) {
	s.record(org, year, program, session, filters[0])
	return preference.ResponseTracking{}, nil
}
func (s *preferenceResultsService) ListResponseTrackingSummaries(_ context.Context, org string, year, program ids.XID, filters ...data.PreferenceResultsFilter) ([]preference.ResponseTrackingSummary, error) {
	s.record(org, year, program, "", filters[0])
	return []preference.ResponseTrackingSummary{}, nil
}

func TestPreferenceResultsAndCompletionFiltersContract(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &preferenceResultsService{}
	router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
	prefix := "/api/school-years/year/programs/program"
	for _, path := range []string{
		prefix + "/interest-profile-surveys/survey/results",
		prefix + "/sessions/session/results",
		prefix + "/interest-profile-surveys/survey/response-tracking",
		prefix + "/sessions/session/response-tracking",
		prefix + "/response-tracking/summary",
	} {
		t.Run(path, func(t *testing.T) {
			for _, query := range []string{"?grade_level_ids=g1,%20g2,,&homeroom_ids=h1,h2", "?grade_level_ids=&homeroom_ids="} {
				request := httptest.NewRequest(http.MethodGet, path+query, nil)
				request.Header.Set("Authorization", "Bearer "+token)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				require.Equal(t, "org-test", service.org)
				require.Equal(t, ids.XID("year"), service.year)
				require.Equal(t, ids.XID("program"), service.program)
				if query == "?grade_level_ids=&homeroom_ids=" {
					require.Empty(t, service.filter.GradeLevelIDs)
					require.Empty(t, service.filter.HomeroomIDs)
				} else {
					require.Equal(t, []string{"g1", "g2"}, service.filter.GradeLevelIDs)
					require.Equal(t, []string{"h1", "h2"}, service.filter.HomeroomIDs)
				}
				if path == prefix+"/interest-profile-surveys/survey/results" {
					var body handlers.InterestProfileResultsResponse
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
					require.Equal(t, "closed", body.State)
					require.Equal(t, 2, body.TotalStudents)
					require.Equal(t, "custom", body.ScaleVersion)
					require.Equal(t, 1, body.Items[0].RatingCounts[0].Count)
					require.NotContains(t, response.Body.String(), "student_id")
				}
				if path == prefix+"/sessions/session/results" {
					var body handlers.RankedChoiceResultsResponse
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
					require.Equal(t, "voting_open", body.State)
					require.Equal(t, 2, body.RankDepth)
					require.Equal(t, 1, body.Items[0].NotInterested)
					require.Equal(t, []handlers.PreferenceRankCountResponse{{Rank: 1}, {Rank: 2}}, body.Items[0].RankCounts)
					require.NotContains(t, response.Body.String(), "student_id")
				}
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusUnauthorized, response.Code)
		})
	}
	document := NewOpenAPI(RouterOptions{})
	for path, capability := range map[string]auth.Capability{
		"/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}/results": auth.CapabilityManageRoster,
		"/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/results":                auth.CapabilityManageCatalog,
	} {
		operation := document.Paths[path].Get
		require.NotNil(t, operation)
		require.Equal(t, string(capability), operation.Extensions[auth.RequiredCapabilityExtension])
		require.Len(t, operation.Parameters, 5)
	}
}
