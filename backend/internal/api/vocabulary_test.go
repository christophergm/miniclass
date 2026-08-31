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
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/stretchr/testify/require"
)

func TestVocabularyRoutesUseTenantPrincipalAndManageRosterCapability(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &fakeVocabularyService{}
	router := NewRouter(RouterOptions{Vocabularies: service, Verifier: verifier, Identity: resolver})

	request := httptest.NewRequest(http.MethodGet, "/api/school-years/year-test/vocabularies", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, "year-test", service.schoolYearID)

	request = httptest.NewRequest(http.MethodPost, "/api/school-years/year-test/grade-levels", strings.NewReader(`{"code":"K","label":"Kindergarten"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording = httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, "year-test", service.schoolYearID)
	require.Equal(t, "K", service.createdCode)
}

type fakeVocabularyService struct {
	organizationID string
	schoolYearID   string
	createdCode    string
}

func (f *fakeVocabularyService) List(_ context.Context, organizationID string, schoolYearID ids.XID, _ bool) (vocabulary.Snapshot, error) {
	f.organizationID = organizationID
	f.schoolYearID = string(schoolYearID)
	return vocabulary.Snapshot{SchoolYearID: schoolYearID, Settings: data.VocabularySettings{OrganizationID: "org-test", HomeroomLabel: "homeroom"}}, nil
}

func (f *fakeVocabularyService) GetGrade(context.Context, string, ids.XID, ids.XID) (data.GradeLevel, error) {
	return data.GradeLevel{}, nil
}

func (f *fakeVocabularyService) CreateGrade(_ context.Context, organizationID string, _ ids.XID, _ audit.Actor, code, _ string) (data.GradeLevel, error) {
	f.organizationID = organizationID
	f.schoolYearID = "year-test"
	f.createdCode = code
	now := time.Unix(1, 0)
	return data.GradeLevel{ID: "grade-test", OrganizationID: ids.XID(organizationID), SchoolYearID: "year-test", Code: code, Label: "Kindergarten", Ordinal: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (f *fakeVocabularyService) UpdateGrade(context.Context, string, ids.XID, ids.XID, audit.Actor, vocabulary.GradeLevelUpdate) (data.GradeLevel, error) {
	return data.GradeLevel{}, nil
}

func (f *fakeVocabularyService) ReorderGrades(context.Context, string, ids.XID, audit.Actor, []ids.XID) ([]data.GradeLevel, error) {
	return nil, nil
}

func (f *fakeVocabularyService) GetHomeroom(context.Context, string, ids.XID, ids.XID) (data.Homeroom, error) {
	return data.Homeroom{}, nil
}

func (f *fakeVocabularyService) CreateHomeroom(context.Context, string, ids.XID, audit.Actor, string, *string) (data.Homeroom, error) {
	return data.Homeroom{}, nil
}

func (f *fakeVocabularyService) UpdateHomeroom(context.Context, string, ids.XID, ids.XID, audit.Actor, vocabulary.HomeroomUpdate) (data.Homeroom, error) {
	return data.Homeroom{}, nil
}

func (f *fakeVocabularyService) UpdateHomeroomLabel(context.Context, string, audit.Actor, string) (data.VocabularySettings, error) {
	return data.VocabularySettings{}, nil
}
