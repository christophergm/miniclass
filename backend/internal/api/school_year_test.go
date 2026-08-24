package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/schoolyear"
	"github.com/stretchr/testify/require"
)

func TestSchoolYearRoutesUseTenantPrincipalAndCapability(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &fakeSchoolYearService{}
	router := NewRouter(RouterOptions{SchoolYears: service, Verifier: verifier, Identity: resolver})

	request := httptest.NewRequest(http.MethodPost, "/api/school-years", strings.NewReader(`{"label":"2026–2027"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)

	request = httptest.NewRequest(http.MethodPatch, "/api/school-years/year-test", strings.NewReader(`{"state":"active"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording = httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "year-test", string(service.updatedID))
	require.NotNil(t, service.updatedInput.State)
	require.Equal(t, data.SchoolYearActive, *service.updatedInput.State)
}

type fakeSchoolYearService struct {
	organizationID string
	updatedID      ids.XID
	updatedInput   schoolyear.UpdateInput
}

func (f *fakeSchoolYearService) Create(_ context.Context, organizationID string, _ audit.Actor, label string) (data.SchoolYear, error) {
	f.organizationID = organizationID
	return data.SchoolYear{ID: "year-test", OrganizationID: ids.XID(organizationID), Label: label, State: data.SchoolYearSetup, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)}, nil
}

func (f *fakeSchoolYearService) List(context.Context, string) ([]data.SchoolYear, error) {
	return []data.SchoolYear{}, nil
}

func (f *fakeSchoolYearService) Get(context.Context, string, ids.XID) (data.SchoolYear, error) {
	return data.SchoolYear{}, nil
}

func (f *fakeSchoolYearService) Update(_ context.Context, organizationID string, id ids.XID, _ auth.OrganizationRole, _ audit.Actor, input schoolyear.UpdateInput) (data.SchoolYear, error) {
	f.organizationID = organizationID
	f.updatedID = id
	f.updatedInput = input
	state := data.SchoolYearActive
	if input.State != nil {
		state = *input.State
	}
	return data.SchoolYear{ID: id, OrganizationID: ids.XID(organizationID), Label: "2026–2027", State: state, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)}, nil
}

func (f *fakeSchoolYearService) Delete(context.Context, string, ids.XID, audit.Actor) error {
	return nil
}
