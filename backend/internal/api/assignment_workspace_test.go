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
	"github.com/chrismott/miniclass/internal/program"
	"github.com/stretchr/testify/require"
)

type assignmentWorkspaceService struct {
	handlers.ProgramService
	organizationID                     string
	schoolYearID, programID, sessionID ids.XID
	called                             bool
}

func (s *assignmentWorkspaceService) GetAssignmentWorkspace(_ context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (program.AssignmentWorkspace, error) {
	s.organizationID, s.schoolYearID, s.programID, s.sessionID = organizationID, schoolYearID, programID, sessionID
	s.called = true
	return program.AssignmentWorkspace{Session: data.Session{ID: sessionID, OrganizationID: ids.XID(organizationID), SchoolYearID: schoolYearID, ProgramID: programID, Name: "Empty draft", State: data.SessionAssigning}, Participants: []data.ProgramMembership{}, Offerings: []data.Offering{}, Assignments: []data.Assignment{}, Exclusions: []data.AssignmentExclusion{}, Overrides: []data.AssignmentOverride{}, RankedChoiceAnswers: []data.SessionResultAnswer{}}, nil
}

func TestAssignmentWorkspaceRouteScopesEmptyDraftAndRequiresAssignmentsCapability(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &assignmentWorkspaceService{}
	router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
	path := "/api/school-years/year-a/programs/program-a/sessions/session-a/assignment-workspace"
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
	require.True(t, service.called)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, ids.XID("year-a"), service.schoolYearID)
	require.Equal(t, ids.XID("program-a"), service.programID)
	require.Equal(t, ids.XID("session-a"), service.sessionID)
	var body handlers.AssignmentWorkspaceResponse
	require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
	require.Equal(t, "session-a", body.Session.ID)
	require.Zero(t, body.DraftRevision)
	require.NotNil(t, body.Participants)
	require.NotNil(t, body.Offerings)
	require.NotNil(t, body.Assignments)
	require.NotNil(t, body.Exclusions)
	require.NotNil(t, body.Overrides)
	require.NotNil(t, body.RankedChoiceAnswers)

	document := NewOpenAPI(RouterOptions{})
	operation := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/assignment-workspace"].Get
	require.Equal(t, string(auth.CapabilityManageAssignments), operation.Extensions[auth.RequiredCapabilityExtension])

	guardianRouter := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver, Sessions: fixedPreferenceSession{principal: auth.GuardianPrincipal{OrganizationID: "org-test", SchoolYearID: "year-a", AdultID: "guardian-a"}}})
	guardianRequest := httptest.NewRequest(http.MethodGet, path, nil)
	guardianRequest.Header.Set("Authorization", "Bearer guardian-session")
	guardianRecording := httptest.NewRecorder()
	guardianRouter.ServeHTTP(guardianRecording, guardianRequest)
	require.Equal(t, http.StatusForbidden, guardianRecording.Code)
}
