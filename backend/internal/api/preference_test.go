package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/stretchr/testify/require"
)

func TestStudentCodeOperationsAndDTOsAreRemoved(t *testing.T) {
	encoded, err := json.Marshal(NewOpenAPI(RouterOptions{}))
	require.NoError(t, err)
	for _, retired := range []string{"/respondent/", "regenerate-codes", "revoke-codes", "access_codes", "active_codes", "regenerate_codes", "StudentCode", "AccessCode"} {
		require.NotContains(t, string(encoded), retired)
	}
	// Historical audit actions remain readable even though issuance is gone.
	require.Contains(t, string(encoded), "survey_code_change")
	require.Contains(t, string(encoded), "ranked_choice_code_change")
	router := NewRouter(RouterOptions{})
	for _, path := range []string{
		"/api/respondent/interest-profile-surveys/year/program/survey/form",
		"/api/respondent/interest-profile-surveys/year/program/survey/submit",
		"/api/respondent/sessions/year/program/session/form",
		"/api/respondent/sessions/year/program/session/submit",
		"/api/school-years/year/programs/program/interest-profile-surveys/survey/regenerate-codes",
		"/api/school-years/year/programs/program/interest-profile-surveys/survey/revoke-codes",
		"/api/school-years/year/programs/program/sessions/session/regenerate-codes",
		"/api/school-years/year/programs/program/sessions/session/revoke-codes",
	} {
		for _, bearer := range []string{"", "legacy-student-code", "guardian-invitation", "onboarding-otp"} {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"organization_id":"org","code":"legacy-student-code"}`))
			request.Header.Set("Content-Type", "application/json")
			if bearer != "" {
				request.Header.Set("Authorization", "Bearer "+bearer)
			}
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, http.StatusNotFound, recording.Code, path)
		}
	}
}

type preferenceKioskService struct {
	handlers.ProgramService
	organizationID string
	actor          audit.Actor
	input          preference.RankedChoiceSubmissionInput
}

func (s *preferenceKioskService) GetRankedChoiceForm(_ context.Context, org string, year, program, session, student ids.XID) (preference.PreferenceForm, error) {
	s.organizationID = org
	return preference.PreferenceForm{Type: preference.FormTypeRankedChoice, ID: session, SessionID: session, SchoolYearID: year, ProgramID: program, StudentID: student, StudentName: "Synthetic Kiosk Student"}, nil
}

func (s *preferenceKioskService) SubmitRankedChoices(_ context.Context, org string, actor audit.Actor, input preference.RankedChoiceSubmissionInput) (data.RankedChoiceSubmission, error) {
	s.organizationID, s.actor, s.input = org, actor, input
	return data.RankedChoiceSubmission{StudentID: input.StudentID}, nil
}

func TestRankedChoiceAdministratorKioskRemainsAuthenticatedAndAttributed(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &preferenceKioskService{}
	router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
	for _, test := range []struct{ path, body string }{
		{"/api/administrator/preference-form", `{"type":"ranked_choice","school_year_id":"year","program_id":"program","instrument_id":"session","student_id":"student"}`},
		{"/api/administrator/sessions/year/program/session/students/student", `{"responses":[{"offering_id":"offering","answer":"interested"}]}`},
	} {
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		recording := httptest.NewRecorder()
		router.ServeHTTP(recording, request)
		require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
		require.Contains(t, recording.Body.String(), "Synthetic Kiosk Student")
	}
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, data.PreferenceChannelAdministratorOnBehalf, service.input.Channel)
	require.Equal(t, ids.XID("student"), service.input.StudentID)
	require.Equal(t, audit.ActorTypeUser, service.actor.Type)
	require.Equal(t, ids.XID("user-test"), *service.actor.UserID)
	require.Nil(t, service.input.ActorAdultID)
	encoded, err := json.Marshal(NewOpenAPI(RouterOptions{}))
	require.NoError(t, err)
	var document struct {
		Paths map[string]map[string]map[string]any
	}
	require.NoError(t, json.Unmarshal(encoded, &document))
	for path, capability := range map[string]auth.Capability{
		"/api/guardian/preference-forms": auth.CapabilityGuardianAccess,
		"/api/guardian/interest-profile-surveys/{schoolYearID}/{programID}/{surveyID}/students/{studentID}": auth.CapabilityGuardianAccess,
		"/api/guardian/sessions/{schoolYearID}/{programID}/{sessionID}/students/{studentID}":                auth.CapabilityGuardianAccess,
		"/api/administrator/preference-form":                                                      auth.CapabilityManageRoster,
		"/api/administrator/sessions/{schoolYearID}/{programID}/{sessionID}/students/{studentID}": auth.CapabilityManageRoster,
	} {
		for _, operation := range document.Paths[path] {
			require.Equal(t, string(capability), operation[auth.RequiredCapabilityExtension], path)
		}
		require.NotEmpty(t, document.Paths[path], path)
	}
}

type fixedPreferenceSession struct{ principal auth.Principal }

func (s fixedPreferenceSession) ResolveSession(_ context.Context, bearer string) (auth.Principal, error) {
	if bearer != "guardian-session" {
		return nil, auth.ErrSessionInvalid
	}
	return s.principal, nil
}

func TestPreferenceEndpointsRejectAnonymousAndWrongCredentialPurposes(t *testing.T) {
	verifier, resolver, accountToken := testAuth(t)
	router := NewRouter(RouterOptions{Verifier: verifier, Identity: resolver, Sessions: fixedPreferenceSession{principal: auth.GuardianPrincipal{OrganizationID: "org-test", SchoolYearID: "year", AdultID: "adult"}}})
	paths := []string{
		"/api/guardian/interest-profile-surveys/year/program/survey/students/student",
		"/api/guardian/sessions/year/program/session/students/student",
		"/api/administrator/preference-form",
		"/api/administrator/sessions/year/program/session/students/student",
	}
	for _, path := range paths {
		for _, bearer := range []string{"", "legacy-student-code", "guardian-invitation", "onboarding-otp"} {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
			request.Header.Set("Content-Type", "application/json")
			if bearer != "" {
				request.Header.Set("Authorization", "Bearer "+bearer)
			}
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, http.StatusUnauthorized, recording.Code, path+" "+bearer)
		}
		wrongMode := accountToken
		if strings.Contains(path, "/administrator/") {
			wrongMode = "guardian-session"
		}
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+wrongMode)
		recording := httptest.NewRecorder()
		router.ServeHTTP(recording, request)
		require.Equal(t, http.StatusForbidden, recording.Code, "credentials must not cross guardian/administrator modes")
	}
}
