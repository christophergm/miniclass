package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrismott/miniclass/internal/api"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/stretchr/testify/require"
)

func TestClosedYearVocabularyAPIReturnsConflictFromDatabaseTrigger(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed vocabulary API integration test"}
	var organizationID, userID string
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into organizations (name)
		values ('Synthetic Closed Vocabulary Academy')
		returning id`).Scan(&organizationID))
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ('closed-vocabulary-subject', 'closed-vocabulary@example.test')
		returning id`).Scan(&userID))
	_, err := harness.Migrator.Exec(ctx, `
		insert into organization_members (organization_id, user_id, role)
		values ($1, $2, 'owner')`, organizationID, userID)
	require.NoError(t, err)

	yearService := schoolyear.New(harness.Database)
	year, err := yearService.Create(ctx, organizationID, actor, "2026–2027")
	require.NoError(t, err)
	year, err = yearService.Update(ctx, organizationID, year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, organizationID, year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	verifier, token := localTestVerifier(t, "closed-vocabulary-subject", "closed-vocabulary@example.test")
	server := api.NewServer(
		api.WithIdentity(identity.NewStore(harness.Database)),
		api.WithVocabularies(vocabulary.New(harness.Database)),
		api.WithVerifier(verifier),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/school-years/"+string(year.ID)+"/grade-levels", bytes.NewBufferString(`{"code":"closed","label":"Closed"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	server.Handler().ServeHTTP(recording, request)

	require.Equal(t, http.StatusConflict, recording.Code)
	require.Contains(t, recording.Body.String(), "school-year-closed")
}
