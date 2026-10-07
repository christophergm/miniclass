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
	workspace                          *program.AssignmentWorkspace
}

func (s *assignmentWorkspaceService) GetAssignmentWorkspace(_ context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (program.AssignmentWorkspace, error) {
	s.organizationID, s.schoolYearID, s.programID, s.sessionID = organizationID, schoolYearID, programID, sessionID
	s.called = true
	if s.workspace != nil {
		return *s.workspace, nil
	}
	return program.AssignmentWorkspace{Session: data.Session{ID: sessionID, OrganizationID: ids.XID(organizationID), SchoolYearID: schoolYearID, ProgramID: programID, Name: "Empty draft", State: data.SessionAssigning}, Participants: []program.AssignmentParticipant{}, Offerings: []data.Offering{}, Assignments: []data.Assignment{}, Exclusions: []data.AssignmentExclusion{}, Overrides: []data.AssignmentOverride{}, Comments: []data.PlacementComment{}, RankedChoiceAnswers: []data.SessionResultAnswer{}}, nil
}

func TestAssignmentWorkspaceRouteIncludesParticipantContextAndOverrideTimestamp(t *testing.T) {
	createdAt := time.Date(2026, time.October, 6, 10, 11, 12, 123456000, time.UTC)
	gradeID := ids.XID("grade-a")
	gradeIDString := string(gradeID)
	ordinal := 0
	membership := data.ProgramMembership{
		ID: "membership-a", OrganizationID: "org-test", SchoolYearID: "year-a", ProgramID: "program-a", StudentID: "student-a",
		LegalGivenName: "Synthetic Legal", LegalFamilyName: "Student", GradeLevelID: &gradeID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	workspace := program.AssignmentWorkspace{
		Session: data.Session{ID: "session-a"},
		Participants: []program.AssignmentParticipant{
			{ProgramMembership: membership, DisplayName: "Synthetic Preferred Student", GradeLabel: "Synthetic Kindergarten", GradeOrdinal: &ordinal, HomeroomName: "Synthetic Room"},
			{ProgramMembership: data.ProgramMembership{ID: "membership-b", StudentID: "student-b", LegalGivenName: "Synthetic", LegalFamilyName: "Unknown", GradeMissing: true}, DisplayName: "Synthetic Unknown"},
		},
		Overrides: []data.AssignmentOverride{{ID: "override-a", AssignmentID: "assignment-a", Rule: "grade-out-of-range", Reason: "Synthetic reason", RecordedBy: "user-a", CreatedAt: createdAt}},
	}
	verifier, resolver, token := testAuth(t)
	router := NewRouter(RouterOptions{Programs: &assignmentWorkspaceService{workspace: &workspace}, Verifier: verifier, Identity: resolver})
	request := httptest.NewRequest(http.MethodGet, "/api/school-years/year-a/programs/program-a/sessions/session-a/assignment-workspace", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
	var body handlers.AssignmentWorkspaceResponse
	require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
	require.Len(t, body.Participants, 2)
	participant := body.Participants[0]
	require.Equal(t, handlers.ProgramMembershipResponse{
		ID: "membership-a", OrganizationID: "org-test", SchoolYearID: "year-a", ProgramID: "program-a", StudentID: "student-a",
		LegalGivenName: "Synthetic Legal", LegalFamilyName: "Student", GradeLevelID: &gradeIDString, CreatedAt: createdAt, UpdatedAt: createdAt,
	}, participant.ProgramMembershipResponse)
	require.Equal(t, "Synthetic Preferred Student", participant.DisplayName)
	require.Equal(t, "Synthetic Kindergarten", participant.GradeLabel)
	require.NotNil(t, participant.GradeOrdinal)
	require.Zero(t, *participant.GradeOrdinal)
	require.Equal(t, "Synthetic Room", participant.HomeroomName)
	require.Nil(t, body.Participants[1].GradeOrdinal)
	require.Empty(t, body.Participants[1].GradeLabel)
	require.Empty(t, body.Participants[1].HomeroomName)
	require.True(t, body.Participants[1].GradeMissing)
	require.Equal(t, []handlers.AssignmentOverrideResponse{{ID: "override-a", AssignmentID: "assignment-a", Rule: "grade-out-of-range", Reason: "Synthetic reason", RecordedBy: "user-a", CreatedAt: createdAt}}, body.Overrides)
	var raw struct {
		Participants []map[string]json.RawMessage `json:"participants"`
		Overrides    []map[string]json.RawMessage `json:"overrides"`
	}
	require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &raw))
	require.Equal(t, "null", string(raw.Participants[1]["grade_ordinal"]))
	require.Equal(t, `"2026-10-06T10:11:12.123456Z"`, string(raw.Overrides[0]["created_at"]))

	schemas := NewOpenAPI(RouterOptions{}).Components.Schemas.Map()
	participantSchema := schemas["AssignmentParticipantResponse"]
	require.NotNil(t, participantSchema)
	for _, field := range []string{"id", "student_id", "legal_given_name", "grade_level_id", "display_name", "grade_label", "grade_ordinal", "homeroom_name"} {
		require.Contains(t, participantSchema.Properties, field)
	}
	require.True(t, participantSchema.Properties["grade_ordinal"].Nullable)
	require.NotContains(t, schemas["ProgramMembershipResponse"].Properties, "display_name")
	require.Equal(t, "date-time", schemas["AssignmentOverrideResponse"].Properties["created_at"].Format)
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
	require.NotNil(t, body.Comments)
	require.NotNil(t, body.RankedChoiceAnswers)

	document := NewOpenAPI(RouterOptions{})
	operation := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/assignment-workspace"].Get
	require.Equal(t, string(auth.CapabilityManageAssignments), operation.Extensions[auth.RequiredCapabilityExtension])
	createExclusion := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/assignment-exclusions"].Post
	require.Equal(t, string(auth.CapabilityManageAssignments), createExclusion.Extensions[auth.RequiredCapabilityExtension])
	deleteExclusion := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/assignment-exclusions/{exclusionID}"].Delete
	require.Equal(t, string(auth.CapabilityManageAssignments), deleteExclusion.Extensions[auth.RequiredCapabilityExtension])
	createComment := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/placement-comments"].Post
	require.Equal(t, string(auth.CapabilityManageAssignments), createComment.Extensions[auth.RequiredCapabilityExtension])

	guardianRouter := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver, Sessions: fixedPreferenceSession{principal: auth.GuardianPrincipal{OrganizationID: "org-test", SchoolYearID: "year-a", AdultID: "guardian-a"}}})
	guardianRequest := httptest.NewRequest(http.MethodGet, path, nil)
	guardianRequest.Header.Set("Authorization", "Bearer guardian-session")
	guardianRecording := httptest.NewRecorder()
	guardianRouter.ServeHTTP(guardianRecording, guardianRequest)
	require.Equal(t, http.StatusForbidden, guardianRecording.Code)
}
