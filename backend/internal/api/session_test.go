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
