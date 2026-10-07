package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type fakeSessionLifecycleService struct {
	handlers.ProgramService
	organizationID string
	schoolYearID   ids.XID
	programID      ids.XID
	sessionID      ids.XID
	input          program.SessionTransitionInput
	err            error
}

func (f *fakeSessionLifecycleService) TransitionSession(_ context.Context, organizationID string, _ audit.Actor, schoolYearID, programID, sessionID ids.XID, input program.SessionTransitionInput) (program.SessionTransitionResult, error) {
	f.organizationID = organizationID
	f.schoolYearID = schoolYearID
	f.programID = programID
	f.sessionID = sessionID
	f.input = input
	if f.err != nil {
		return program.SessionTransitionResult{}, f.err
	}
	now := time.Unix(1, 0).UTC()
	return program.SessionTransitionResult{
		Session:   data.Session{ID: sessionID, OrganizationID: ids.XID(organizationID), SchoolYearID: schoolYearID, ProgramID: programID, Name: "Synthetic session", State: input.NextState, MeetingDates: []time.Time{now}},
		FromState: data.SessionPlanning, ToState: input.NextState, Applied: true,
	}, nil
}

type fakeSessionUpdateService struct {
	handlers.ProgramService
	organizationID                     string
	schoolYearID, programID, sessionID ids.XID
	input                              program.SessionUpdate
	err                                error
}

func (f *fakeSessionUpdateService) UpdateSession(_ context.Context, organizationID string, _ audit.Actor, schoolYearID, programID, sessionID ids.XID, input program.SessionUpdate) (data.Session, error) {
	f.organizationID, f.schoolYearID, f.programID, f.sessionID = organizationID, schoolYearID, programID, sessionID
	f.input = input
	if f.err != nil {
		return data.Session{}, f.err
	}
	return data.Session{ID: sessionID, OrganizationID: ids.XID(organizationID), SchoolYearID: schoolYearID, ProgramID: programID, Name: *input.Name, State: data.SessionVotingOpen, MeetingDates: *input.Dates, RankedChoice: input.RankedChoice}, nil
}

func (f *fakeSessionUpdateService) GetCatalogFeasibility(context.Context, string, ids.XID, ids.XID, ids.XID) (program.CatalogFeasibility, error) {
	return program.CatalogFeasibility{}, nil
}

func TestSessionUpdateRoutePassesConfigurationAndMapsErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		status  int
		problem problems.Slug
	}{
		{"metadata with expired unchanged configuration", nil, http.StatusOK, ""},
		{"locked configuration", program.ErrRankedChoiceConfigurationLocked, http.StatusConflict, problems.ProgramConflict},
		{"expired replacement", program.ErrRankedChoiceDeadlineInvalid, http.StatusBadRequest, problems.ProgramConflict},
		{"complete", program.ErrSessionReadOnly, http.StatusConflict, problems.SessionReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifier, resolver, token := testAuth(t)
			service := &fakeSessionUpdateService{err: test.err}
			router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
			request := httptest.NewRequest(http.MethodPatch, "/api/school-years/year-test/programs/program-test/sessions/session-test", strings.NewReader(`{"name":"Synthetic renamed session","meeting_dates":["2026-10-23"],"ranked_choice":{"rank_depth":3,"deadline":"2020-01-01T00:00:00Z"}}`))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, test.status, recording.Code, recording.Body.String())
			require.Equal(t, "org-test", service.organizationID)
			require.Equal(t, ids.XID("year-test"), service.schoolYearID)
			require.Equal(t, ids.XID("program-test"), service.programID)
			require.Equal(t, ids.XID("session-test"), service.sessionID)
			require.Equal(t, "Synthetic renamed session", *service.input.Name)
			require.Equal(t, []time.Time{time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)}, *service.input.Dates)
			require.Equal(t, 3, service.input.RankedChoice.RankDepth)
			require.True(t, service.input.RankedChoice.Deadline.Equal(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
			var response map[string]any
			require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
			if test.err != nil {
				require.Equal(t, string(test.problem), response["type"])
			} else {
				require.Equal(t, "voting_open", response["state"])
				require.Equal(t, "Synthetic renamed session", response["name"])
			}
		})
	}
}

func TestSessionTransitionRouteUsesCatalogCapabilityAndTypedPayload(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &fakeSessionLifecycleService{}
	router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})

	request := httptest.NewRequest(http.MethodPost, "/api/school-years/year-test/programs/program-test/sessions/session-test/transition", strings.NewReader(`{"state":"catalog_published"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)

	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, ids.XID("year-test"), service.schoolYearID)
	require.Equal(t, ids.XID("program-test"), service.programID)
	require.Equal(t, ids.XID("session-test"), service.sessionID)
	require.Equal(t, data.SessionCatalogPublished, service.input.NextState)
	require.False(t, service.input.Confirm)
	var response map[string]any
	require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
	require.Equal(t, true, response["applied"])
	require.Equal(t, "catalog_published", response["to_state"])
	sessionResponse, ok := response["session"].(map[string]any)
	require.True(t, ok)
	_, hasOrdinal := sessionResponse["ordinal"]
	require.False(t, hasOrdinal)
}

func TestSessionTransitionRouteLogsFailedLookupContext(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	var logs bytes.Buffer
	service := &fakeSessionLifecycleService{err: fmt.Errorf("transition session: get session for update: %w", pgx.ErrNoRows)}
	router := NewRouter(RouterOptions{
		Programs: service, Verifier: verifier, Identity: resolver,
		Logger: slog.New(slog.NewTextHandler(&logs, nil)),
	})

	request := httptest.NewRequest(http.MethodPost, "/api/school-years/year-test/programs/program-test/sessions/session-test/transition", strings.NewReader(`{"state":"voting_open","confirm":false}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)

	require.Equal(t, http.StatusNotFound, recording.Code)
	require.Contains(t, logs.String(), "session transition failed")
	require.Contains(t, logs.String(), "organization_id=org-test")
	require.Contains(t, logs.String(), "school_year_id=year-test")
	require.Contains(t, logs.String(), "program_id=program-test")
	require.Contains(t, logs.String(), "session_id=session-test")
	require.Contains(t, logs.String(), "requested_state=voting_open")
	require.Contains(t, logs.String(), "confirm=false")
	require.Contains(t, logs.String(), "get session for update")
}

func TestSessionTransitionRouteReturnsClearProblemForIllegalEdge(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &fakeSessionLifecycleService{err: fmt.Errorf("apply: %w: Planning cannot move to Complete", program.ErrSessionTransitionInvalid)}
	router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})

	request := httptest.NewRequest(http.MethodPost, "/api/school-years/year-test/programs/program-test/sessions/session-test/transition", strings.NewReader(`{"state":"complete"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)

	require.Equal(t, http.StatusConflict, recording.Code)
	var response map[string]any
	require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
	require.Equal(t, string(problems.SessionTransitionInvalid), response["type"])
	require.Contains(t, response["detail"], "Planning cannot move to Complete")
}
