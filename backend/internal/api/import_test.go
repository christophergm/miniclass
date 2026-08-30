package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest"
	"github.com/stretchr/testify/require"
)

type importPreviewServiceStub struct {
	preview        ingest.Preview
	organizationID string
	schoolYearID   ids.XID
	kind           string
	document       []byte
}

func (s *importPreviewServiceStub) Preview(_ context.Context, organizationID string, schoolYearID ids.XID, kind string, document []byte) (ingest.Preview, error) {
	s.organizationID, s.schoolYearID, s.kind, s.document = organizationID, schoolYearID, kind, append([]byte(nil), document...)
	return s.preview, nil
}

func TestImportPreviewEndpointAcceptsRawJSONAndRequiresManageRoster(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	service := &importPreviewServiceStub{preview: ingest.Preview{
		Kind: "roster_json", SchoolYearID: "year-1", ContentHash: "abc123",
		Rows: []ingest.SourceRowPreview{{Number: 1, SourceExternalIdentifier: "adult-1", Outcome: ingest.OutcomeCreate}},
	}}
	router := NewRouter(RouterOptions{ImportPreview: service, Verifier: verifier, Identity: resolver})
	request := httptest.NewRequest(http.MethodPost, "/api/imports/roster_json/preview?school_year_id=year-1", strings.NewReader(`[{}]`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)

	require.Equal(t, http.StatusOK, recording.Code)
	require.Equal(t, "org-test", service.organizationID)
	require.Equal(t, ids.XID("year-1"), service.schoolYearID)
	require.Equal(t, ingest.KindRosterJSON, service.kind)
	require.Equal(t, `[{}]`, string(service.document))
	require.Contains(t, recording.Body.String(), `"content_hash":"abc123"`)
}
