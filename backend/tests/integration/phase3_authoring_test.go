package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api"
	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

// TestPhase3AuthoringAPIRoundTrip covers the route sequence used by the
// frontend authoring workspace. It deliberately crosses the API boundary so
// generated-client paths are backed by the same capability and tenant checks
// as a deployed organiser session (SPEC §§12, 14, 22.4).
func TestPhase3AuthoringAPIRoundTrip(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "phase 3 authoring integration test"}
	var userID string
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ('phase3-authoring-subject', 'phase3-authoring@example.test')
		returning id`).Scan(&userID))
	_, err := harness.Migrator.Exec(ctx, `
		insert into organization_members (organization_id, user_id, role)
		values ($1, $2, 'owner')`, organizationID, userID)
	require.NoError(t, err)

	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic Phase 3 year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Phase 3 programme")
	require.NoError(t, err)

	verifier, token := localTestVerifier(t, "phase3-authoring-subject", "phase3-authoring@example.test")
	server := api.NewServer(
		api.WithDatabase(harness.Database),
		api.WithIdentity(identity.NewStore(harness.Database)),
		api.WithPrograms(program.New(harness.Database)),
		api.WithVerifier(verifier),
	)
	path := "/api/school-years/" + string(year.ID) + "/programs/" + string(programRow.ID)

	created := phase3JSONRequest(t, server.Handler(), http.MethodPost, path+"/sessions", token, map[string]any{
		"name": "Synthetic authoring session", "ordinal": 1, "meeting_dates": []string{"2026-10-02", "2026-10-09"},
	})
	require.Contains(t, []int{http.StatusOK, http.StatusCreated}, created.Code)
	var session handlers.SessionResponse
	require.NoError(t, json.NewDecoder(created.Body).Decode(&session))
	require.Equal(t, "Synthetic authoring session", session.Name)
	require.Equal(t, []string{"2026-10-02", "2026-10-09"}, session.MeetingDates)

	fetched := phase3JSONRequest(t, server.Handler(), http.MethodGet, path+"/sessions/"+session.ID, token, nil)
	require.Equal(t, http.StatusOK, fetched.Code)

	transitioned := phase3JSONRequest(t, server.Handler(), http.MethodPost, path+"/sessions/"+session.ID+"/transition", token, map[string]any{"state": "catalog_published"})
	require.Equal(t, http.StatusOK, transitioned.Code)

	illegal := phase3JSONRequest(t, server.Handler(), http.MethodPost, path+"/sessions/"+session.ID+"/transition", token, map[string]any{"state": "complete"})
	require.Equal(t, http.StatusConflict, illegal.Code)
	require.Contains(t, illegal.Body.String(), "session-transition-invalid")

	yearService := schoolyear.New(harness.Database)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	closedCreate := phase3JSONRequest(t, server.Handler(), http.MethodPost, path+"/sessions", token, map[string]any{
		"name": "Should be refused", "ordinal": 2, "meeting_dates": []string{time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC).Format("2006-01-02")},
	})
	require.Equal(t, http.StatusConflict, closedCreate.Code)
	require.Contains(t, closedCreate.Body.String(), "school-year-closed")
}

func phase3JSONRequest(t *testing.T, handler http.Handler, method, path, token string, value any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if value == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		body = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	handler.ServeHTTP(recording, request)
	return recording
}
