package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/stretchr/testify/require"
)

type artifactService struct {
	handlers.ProgramService
	called                       bool
	organizationID               string
	yearID, programID, sessionID ids.XID
	kind                         program.ArtifactKind
}

func (s *artifactService) GenerateArtifact(_ context.Context, org string, year, prog, session ids.XID, kind program.ArtifactKind) (program.ArtifactDocument, error) {
	s.called = true
	s.organizationID = org
	s.yearID = year
	s.programID = prog
	s.sessionID = session
	s.kind = kind
	return program.ArtifactDocument{Kind: kind, SessionName: "Synthetic session", GeneratedAt: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC), Sections: []program.ArtifactSection{{ID: "o", Title: "Synthetic", Blocks: []program.ArtifactBlock{{Kind: "heading", Text: "Students"}}}}, Warnings: []program.ArtifactWarning{}}, nil
}
func TestArtifactRouteRoleMatrix(t *testing.T) {
	for _, role := range []auth.OrganizationRole{auth.RoleOwner, auth.RoleAdministrator, auth.RoleCoordinator} {
		t.Run(string(role), func(t *testing.T) {
			verifier, resolver, token := testAuthRole(t, string(role))
			service := &artifactService{}
			router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
			for _, kind := range []string{"class_list", "homeroom_dismissal"} {
				service.called = false
				request := httptest.NewRequest(http.MethodGet, "/api/school-years/year-a/programs/program-a/sessions/session-a/artifacts/"+kind, nil)
				request.Header.Set("Authorization", "Bearer "+token)
				recording := httptest.NewRecorder()
				router.ServeHTTP(recording, request)
				if role == auth.RoleCoordinator {
					require.Equal(t, http.StatusForbidden, recording.Code, recording.Body.String())
					require.False(t, service.called)
				} else {
					require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
					require.True(t, service.called)
				}
			}
		})
	}
}

func TestArtifactRouteAdminOnlyAndContract(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &artifactService{}
	router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
	path := "/api/school-years/year-a/programs/program-a/sessions/session-a/artifacts/"
	for _, kind := range []string{"class_list", "homeroom_dismissal"} {
		request := httptest.NewRequest(http.MethodGet, path+kind, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		recording := httptest.NewRecorder()
		router.ServeHTTP(recording, request)
		require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
		require.Equal(t, "org-test", service.organizationID)
		require.Equal(t, ids.XID("year-a"), service.yearID)
		require.Equal(t, ids.XID("program-a"), service.programID)
		require.Equal(t, ids.XID("session-a"), service.sessionID)
		require.Equal(t, program.ArtifactKind(kind), service.kind)
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
		// Huma adds its standard schema metadata to every object response.
		require.JSONEq(t, `"https://example.com/ArtifactDocumentResponse.json"`, string(body["$schema"]))
		delete(body, "$schema")
		require.Len(t, body, 5)
		require.JSONEq(t, `"2026-10-09T12:00:00Z"`, string(body["generated_at"]))
		require.JSONEq(t, `[{"id":"o","title":"Synthetic","blocks":[{"kind":"heading","label":"","text":"Students"}]}]`, string(body["sections"]))
		require.JSONEq(t, `[]`, string(body["warnings"]))
	}
	service.called = false
	request := httptest.NewRequest(http.MethodGet, path+"invalid", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusUnprocessableEntity, recording.Code)
	require.False(t, service.called)
	unauth := httptest.NewRecorder()
	router.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, path+"class_list", nil))
	require.Equal(t, http.StatusUnauthorized, unauth.Code)
	require.False(t, service.called)
	guardianRouter := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver, Sessions: fixedPreferenceSession{principal: auth.GuardianPrincipal{OrganizationID: "org-test", SchoolYearID: "year-a", AdultID: "guardian-a"}}})
	request = httptest.NewRequest(http.MethodGet, path+"class_list", nil)
	request.Header.Set("Authorization", "Bearer guardian-session")
	recording = httptest.NewRecorder()
	guardianRouter.ServeHTTP(recording, request)
	require.Equal(t, http.StatusForbidden, recording.Code)
	require.False(t, service.called)
	operation := NewOpenAPI(RouterOptions{}).Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/artifacts/{kind}"].Get
	require.Equal(t, string(auth.CapabilityManagePublishing), operation.Extensions[auth.RequiredCapabilityExtension])
}
