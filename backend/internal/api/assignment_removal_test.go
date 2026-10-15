package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type assignmentRemovalService struct {
	handlers.ProgramService
	organizationID string
	actor          audit.Actor
	input          program.AssignmentOperationInput
	studentID      ids.XID
	operation      string
	result         program.AssignmentOperationResult
	err            error
}

func (s *assignmentRemovalService) RemoveAssignment(_ context.Context, organizationID string, actor audit.Actor, input program.AssignmentOperationInput, studentID ids.XID) (program.AssignmentOperationResult, error) {
	s.organizationID, s.actor, s.input, s.studentID, s.operation = organizationID, actor, input, studentID, "remove-assignment"
	return s.result, s.err
}

func (s *assignmentRemovalService) RemoveUnpinnedAssignments(_ context.Context, organizationID string, actor audit.Actor, input program.AssignmentOperationInput) (program.AssignmentOperationResult, error) {
	s.organizationID, s.actor, s.input, s.operation = organizationID, actor, input, "remove-unpinned-assignments"
	return s.result, s.err
}

func TestAssignmentRemovalRoutesContractAndAuthorization(t *testing.T) {
	for _, test := range []struct{ suffix, template, operation string }{
		{"/student-a/remove", "/{studentID}/remove", "remove-assignment"},
		{"/remove-all", "/remove-all", "remove-unpinned-assignments"},
	} {
		t.Run(test.operation, func(t *testing.T) {
			verifier, resolver, token := testAuth(t)
			service := &assignmentRemovalService{result: program.AssignmentOperationResult{DraftRevision: 8, Assignments: []data.Assignment{{ID: "assignment-pinned", StudentID: "student-pinned", OfferingID: "offering-a", Origin: "manual", Pinned: true, RealizedQuality: "high"}}}}
			router := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver})
			path := "/api/school-years/year-a/programs/program-a/sessions/session-a/assignments" + test.suffix
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"expected_revision":7,"reason":"Synthetic removal reason"}`))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
			require.Equal(t, test.operation, service.operation)
			require.Equal(t, "org-test", service.organizationID)
			require.Equal(t, program.AssignmentOperationInput{SchoolYearID: "year-a", ProgramID: "program-a", SessionID: "session-a", ExpectedRevision: 7, Reason: "Synthetic removal reason"}, service.input)
			require.Equal(t, audit.ActorTypeUser, service.actor.Type)
			require.Equal(t, ids.XID("user-test"), *service.actor.UserID)
			require.Equal(t, "organizer@example.test", service.actor.Label)
			if test.operation == "remove-assignment" {
				require.Equal(t, ids.XID("student-a"), service.studentID)
			}
			var body handlers.AssignmentOperationResponse
			require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
			require.Equal(t, int64(8), body.DraftRevision)
			require.Equal(t, []handlers.AssignmentResponse{{ID: "assignment-pinned", StudentID: "student-pinned", OfferingID: "offering-a", Origin: "manual", Pinned: true, RealizedQuality: "high"}}, body.Assignments)

			document := NewOpenAPI(RouterOptions{})
			operation := document.Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/assignments"+test.template].Post
			require.Equal(t, test.operation, operation.OperationID)
			require.Equal(t, string(auth.CapabilityManageAssignments), operation.Extensions[auth.RequiredCapabilityExtension])
			schemas := document.Components.Schemas.Map()
			requestSchema := operation.RequestBody.Content["application/json"].Schema
			if requestSchema.Ref != "" {
				requestSchema = schemas[strings.TrimPrefix(requestSchema.Ref, "#/components/schemas/")]
			}
			require.Len(t, requestSchema.Properties, 3) // Huma adds the optional $schema field.
			require.Contains(t, requestSchema.Properties, "$schema")
			require.ElementsMatch(t, []string{"expected_revision", "reason"}, requestSchema.Required)
			require.Equal(t, "#/components/schemas/AssignmentOperationResponse", operation.Responses["200"].Content["application/json"].Schema.Ref)

			for _, failure := range []struct {
				name, body string
				err        error
				status     int
			}{
				{"stale", `{"expected_revision":7,"reason":"Synthetic reason"}`, program.ErrDraftRevisionConflict, http.StatusConflict},
				{"not-found", `{"expected_revision":7,"reason":"Synthetic reason"}`, fmt.Errorf("get session for update: %w", pgx.ErrNoRows), http.StatusNotFound},
				{"negative-revision", `{"expected_revision":-1,"reason":"Synthetic reason"}`, nil, http.StatusUnprocessableEntity},
				{"missing-revision", `{"reason":"Synthetic reason"}`, nil, http.StatusUnprocessableEntity},
				{"missing-reason", `{"expected_revision":7}`, nil, http.StatusUnprocessableEntity},
			} {
				t.Run(failure.name, func(t *testing.T) {
					service.err, service.operation = failure.err, ""
					request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(failure.body))
					request.Header.Set("Authorization", "Bearer "+token)
					request.Header.Set("Content-Type", "application/json")
					recording := httptest.NewRecorder()
					router.ServeHTTP(recording, request)
					require.Equal(t, failure.status, recording.Code, recording.Body.String())
					if failure.status == http.StatusUnprocessableEntity {
						require.Empty(t, service.operation)
					}
				})
			}
			service.err, service.operation = nil, ""
			service.result = program.AssignmentOperationResult{DraftRevision: 7}
			request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"expected_revision":7,"reason":""}`))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			recording = httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
			require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
			require.NotNil(t, body.Assignments)
			require.Empty(t, body.Assignments)
			require.Equal(t, int64(7), body.DraftRevision)

			service.operation = ""
			unauthenticated := httptest.NewRecorder()
			router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"expected_revision":7,"reason":""}`)))
			require.Equal(t, http.StatusUnauthorized, unauthenticated.Code)
			guardianRouter := NewRouter(RouterOptions{Programs: service, Verifier: verifier, Identity: resolver, Sessions: fixedPreferenceSession{principal: auth.GuardianPrincipal{OrganizationID: "org-test", SchoolYearID: "year-a", AdultID: "guardian-a"}}})
			request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"expected_revision":7,"reason":""}`))
			request.Header.Set("Authorization", "Bearer guardian-session")
			request.Header.Set("Content-Type", "application/json")
			forbidden := httptest.NewRecorder()
			guardianRouter.ServeHTTP(forbidden, request)
			require.Equal(t, http.StatusForbidden, forbidden.Code)
			require.Empty(t, service.operation)
		})
	}
}
