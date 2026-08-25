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

	request := httptest.NewRequest(http.MethodGet, "/api/vocabularies", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)

	request = httptest.NewRequest(http.MethodPost, "/api/grade-levels", strings.NewReader(`{"code":"K","label":"Kindergarten"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording = httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, "K", service.createdCode)
}

type fakeVocabularyService struct {
	organizationID string
	createdCode    string
}

func (f *fakeVocabularyService) List(_ context.Context, organizationID string, _ bool) (vocabulary.Snapshot, error) {
	f.organizationID = organizationID
	return vocabulary.Snapshot{Settings: data.VocabularySettings{OrganizationID: "org-test", HomeroomLabel: "homeroom"}}, nil
}

func (f *fakeVocabularyService) GetGrade(context.Context, string, ids.XID) (data.GradeLevel, error) {
	return data.GradeLevel{}, nil
}

func (f *fakeVocabularyService) CreateGrade(_ context.Context, organizationID string, _ audit.Actor, code, _ string) (data.GradeLevel, error) {
	f.organizationID = organizationID
	f.createdCode = code
	now := time.Unix(1, 0)
	return data.GradeLevel{ID: "grade-test", OrganizationID: ids.XID(organizationID), Code: code, Label: "Kindergarten", Ordinal: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (f *fakeVocabularyService) UpdateGrade(context.Context, string, ids.XID, audit.Actor, vocabulary.GradeLevelUpdate) (data.GradeLevel, error) {
	return data.GradeLevel{}, nil
}

func (f *fakeVocabularyService) ReorderGrades(context.Context, string, audit.Actor, []ids.XID) ([]data.GradeLevel, error) {
	return nil, nil
}

func (f *fakeVocabularyService) GetHomeroom(context.Context, string, ids.XID) (data.Homeroom, error) {
	return data.Homeroom{}, nil
}

func (f *fakeVocabularyService) CreateHomeroom(context.Context, string, audit.Actor, string) (data.Homeroom, error) {
	return data.Homeroom{}, nil
}

func (f *fakeVocabularyService) UpdateHomeroom(context.Context, string, ids.XID, audit.Actor, vocabulary.HomeroomUpdate) (data.Homeroom, error) {
	return data.Homeroom{}, nil
}

func (f *fakeVocabularyService) UpdateHomeroomLabel(context.Context, string, audit.Actor, string) (data.VocabularySettings, error) {
	return data.VocabularySettings{}, nil
}
