package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type offeringPreferenceContextService struct {
	handlers.ProgramService
	context program.OfferingPreferenceContext
	err     error
}

func (s *offeringPreferenceContextService) GetOfferingPreferenceContext(_ context.Context, _ string, _, _, _, _ ids.XID) (program.OfferingPreferenceContext, error) {
	return s.context, s.err
}

func TestOfferingPreferenceContextRouteFiltersSensitiveCommentsByRole(t *testing.T) {
	assignmentID := ids.XID("assignment-synthetic")
	context := program.OfferingPreferenceContext{
		Offering: data.Offering{ID: "offering-synthetic", Name: "Synthetic Offering", Capacity: 2},
		Comments: []data.PlacementComment{
			{ID: "comment-public", HostType: data.CommentHostOffering, HostID: "offering-synthetic", Body: "Synthetic public", Sensitivity: "internal"},
			{ID: "comment-sensitive", HostType: data.CommentHostOffering, HostID: "offering-synthetic", Body: "Synthetic sensitive", Sensitivity: "sensitive"},
		},
		AssignmentComments: []data.PlacementComment{{ID: "assignment-sensitive", HostType: data.CommentHostAssignment, HostID: assignmentID, Body: "Synthetic assignment note", Sensitivity: "sensitive"}},
		Groups: []program.OfferingPreferenceGroup{{Category: program.OfferingPreferenceUnknown, Students: []program.OfferingPreferenceStudent{{
			Participant: dataToAssignmentParticipant("student-synthetic", "Synthetic Student"), Preference: program.OfferingPreference{Category: program.OfferingPreferenceUnknown, PotentialQuality: program.PotentialQualityNeutral},
			Placement: program.OfferingPreferencePlacement{Status: "here", AssignmentID: &assignmentID, RealizedQuality: "neutral"},
		}}}},
	}
	path := "/api/school-years/year-synthetic/programs/program-synthetic/sessions/session-synthetic/offerings/offering-synthetic/preference-context"
	for _, test := range []struct {
		name     string
		role     string
		comments int
		hasNotes bool
	}{
		{name: "owner", role: string(auth.RoleOwner), comments: 2, hasNotes: true},
		{name: "coordinator", role: string(auth.RoleCoordinator), comments: 1, hasNotes: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifier, resolver, token := testAuthRole(t, test.role)
			router := NewRouter(RouterOptions{Programs: &offeringPreferenceContextService{context: context}, Verifier: verifier, Identity: resolver})
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
			var body handlers.OfferingPreferenceContextResponse
			require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
			require.Len(t, body.Comments, test.comments)
			require.Equal(t, test.hasNotes, body.Groups[0].Students[0].HasNotes)
		})
	}

	document := NewOpenAPI(RouterOptions{})
	operation := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/offerings/{offeringID}/preference-context"].Get
	require.Equal(t, string(auth.CapabilityManageAssignments), operation.Extensions[auth.RequiredCapabilityExtension])

	verifier, resolver, token := testAuth(t)
	guardianRouter := NewRouter(RouterOptions{Programs: &offeringPreferenceContextService{context: context}, Verifier: verifier, Identity: resolver, Sessions: fixedPreferenceSession{principal: auth.GuardianPrincipal{OrganizationID: "org-test", SchoolYearID: "year-synthetic", AdultID: "guardian-synthetic"}}})
	guardianRequest := httptest.NewRequest(http.MethodGet, path, nil)
	guardianRequest.Header.Set("Authorization", "Bearer guardian-session")
	guardianRecording := httptest.NewRecorder()
	guardianRouter.ServeHTTP(guardianRecording, guardianRequest)
	require.Equal(t, http.StatusForbidden, guardianRecording.Code)

	notFoundRouter := NewRouter(RouterOptions{Programs: &offeringPreferenceContextService{err: fmt.Errorf("get offering preference context: %w", pgx.ErrNoRows)}, Verifier: verifier, Identity: resolver})
	notFoundRequest := httptest.NewRequest(http.MethodGet, path, nil)
	notFoundRequest.Header.Set("Authorization", "Bearer "+token)
	notFoundRecording := httptest.NewRecorder()
	notFoundRouter.ServeHTTP(notFoundRecording, notFoundRequest)
	require.Equal(t, http.StatusNotFound, notFoundRecording.Code)
}

func dataToAssignmentParticipant(studentID ids.XID, displayName string) program.AssignmentParticipant {
	return program.AssignmentParticipant{ProgramMembership: data.ProgramMembership{StudentID: studentID}, DisplayName: displayName}
}
