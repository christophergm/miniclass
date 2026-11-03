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

type studentPreferenceContextService struct {
	handlers.ProgramService
	context program.StudentPreferenceContext
	err     error
}

func (s *studentPreferenceContextService) GetStudentPreferenceContext(_ context.Context, _ string, _, _, _, _ ids.XID) (program.StudentPreferenceContext, error) {
	return s.context, s.err
}

func TestStudentPreferenceContextRouteFiltersSensitiveCommentsAndDeclaresCapability(t *testing.T) {
	context := program.StudentPreferenceContext{
		Student:         program.StudentPreferenceStudent{ID: "student", DisplayName: "Synthetic Student", LegalName: "Legal Student", LinkedAdults: []program.StudentLinkedAdult{}},
		CurrentComments: []data.PlacementComment{{ID: "internal", Body: "Routine", Sensitivity: "internal"}, {ID: "sensitive", Body: "Restricted", Sensitivity: "sensitive"}},
		Rankings:        program.StudentRankings{Rows: []program.StudentRanking{}},
		PriorPlacements: []program.StudentPriorPlacement{{SessionID: "prior", Comments: []data.PlacementComment{{ID: "prior-sensitive", Body: "Prior restricted", Sensitivity: "sensitive"}}}},
		InterestProfile: program.StudentInterestProfile{VeryInterested: []program.StudentInterestArea{}, Interested: []program.StudentInterestArea{}, NotInterested: []program.StudentInterestArea{}, NotRated: []program.StudentInterestArea{}},
	}
	path := "/api/school-years/year/programs/program/sessions/session/students/student/preference-context"
	for _, test := range []struct {
		name, role              string
		comments, priorComments int
	}{
		{name: "coordinator", role: string(auth.RoleCoordinator), comments: 1, priorComments: 0},
		{name: "owner", role: string(auth.RoleOwner), comments: 2, priorComments: 1},
		{name: "administrator", role: string(auth.RoleAdministrator), comments: 2, priorComments: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifier, resolver, token := testAuthRole(t, test.role)
			router := NewRouter(RouterOptions{Programs: &studentPreferenceContextService{context: context}, Verifier: verifier, Identity: resolver})
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
			var body handlers.StudentPreferenceContextResponse
			require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
			require.Len(t, body.Comments, test.comments)
			require.Len(t, body.PriorPlacements[0].Comments, test.priorComments)
		})
	}

	operation := NewOpenAPI(RouterOptions{}).Paths["/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/students/{studentID}/preference-context"].Get
	require.Equal(t, string(auth.CapabilityManageAssignments), operation.Extensions[auth.RequiredCapabilityExtension])
	verifier, resolver, token := testAuth(t)
	router := NewRouter(RouterOptions{Programs: &studentPreferenceContextService{err: fmt.Errorf("get student preference context: %w", pgx.ErrNoRows)}, Verifier: verifier, Identity: resolver})
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusNotFound, recording.Code)
}
