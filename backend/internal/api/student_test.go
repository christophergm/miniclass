package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/stretchr/testify/require"
)

func TestStudentRoutesUseTenantPrincipalAndCapability(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &fakeStudentService{}
	router := NewRouter(RouterOptions{Students: service, Verifier: verifier, Identity: resolver})
	request := httptest.NewRequest(http.MethodPost, "/api/school-years/year-test/students", strings.NewReader(`{"legal_given_name":"Alex","legal_family_name":"Rivera","grade_level_id":"grade-test","homeroom_id":"room-test"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, ids.XID("year-test"), service.schoolYearID)
	require.Equal(t, ids.XID("grade-test"), service.input.GradeLevelID)
	require.Equal(t, ids.XID("room-test"), service.input.HomeroomID)
}

type fakeStudentService struct {
	organizationID string
	schoolYearID   ids.XID
	input          people.StudentCreateInput
}

func (f *fakeStudentService) CreateStudent(_ context.Context, organizationID string, schoolYearID ids.XID, _ audit.Actor, input people.StudentCreateInput) (data.Student, error) {
	f.organizationID, f.schoolYearID, f.input = organizationID, schoolYearID, input
	return data.Student{
		ID: "student-test", OrganizationID: ids.XID(organizationID), SchoolYearID: schoolYearID,
		LegalGivenName: input.LegalGivenName, LegalFamilyName: input.LegalFamilyName,
		GradeLevelID: input.GradeLevelID, HomeroomID: input.HomeroomID,
		CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0),
	}, nil
}

func (f *fakeStudentService) ListStudents(context.Context, string, ids.XID) ([]data.Student, error) {
	return []data.Student{}, nil
}

func (f *fakeStudentService) GetStudent(context.Context, string, ids.XID, ids.XID) (data.Student, error) {
	return data.Student{}, nil
}

func (f *fakeStudentService) UpdateStudent(context.Context, string, ids.XID, ids.XID, audit.Actor, people.StudentUpdateInput) (data.Student, error) {
	return data.Student{}, nil
}

func (f *fakeStudentService) DeleteStudent(context.Context, string, ids.XID, ids.XID, audit.Actor) error {
	return nil
}
