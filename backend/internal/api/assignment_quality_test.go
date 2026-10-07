package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/stretchr/testify/require"
)

type assignmentQualityService struct {
	handlers.ProgramService
	quality program.AssignmentQuality
}

func (s *assignmentQualityService) GetAssignmentQuality(_ context.Context, _ string, _, _, _ ids.XID) (program.AssignmentQuality, error) {
	return s.quality, nil
}

// SPEC §§16.5–16.6: context identifies occurrences without suppressing any.
func TestAssignmentQualityRoutePreservesCatalogAreaGapContextAndOccurrences(t *testing.T) {
	quality := program.EvaluateAssignmentQuality(program.AssignmentQualitySnapshot{
		Session: data.Session{ID: "session-a", DraftAssignmentsStale: true},
		InterestAreas: []data.InterestArea{
			{ID: "area-one", Label: "Synthetic Arts"},
			{ID: "area-two", Label: "Synthetic Science"},
		},
		Profiles: map[ids.XID]map[ids.XID]data.InterestProfileRating{
			"student-one": {"area-one": data.InterestProfileVeryInterested, "area-two": data.InterestProfileVeryInterested},
		},
	})
	// Even identical occurrences must survive response mapping and serialization.
	quality.Warnings = append(quality.Warnings, quality.Warnings[0])
	verifier, resolver, token := testAuth(t)
	router := NewRouter(RouterOptions{Programs: &assignmentQualityService{quality: quality}, Verifier: verifier, Identity: resolver})
	request := httptest.NewRequest(http.MethodGet, "/api/school-years/year-a/programs/program-a/sessions/session-a/assignment-quality", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())

	var body handlers.AssignmentQualityResponse
	require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &body))
	require.Len(t, body.Warnings, 4)
	for i, area := range []handlers.CatalogAreaGapResponse{
		{ID: "area-one", Label: "Synthetic Arts", HighRatingCount: 1},
		{ID: "area-two", Label: "Synthetic Science", HighRatingCount: 1},
	} {
		warning := body.Warnings[i]
		require.Equal(t, "catalog-area-gap", warning.ID)
		require.Equal(t, "info", warning.Severity)
		require.Equal(t, "session", warning.HostType)
		require.Equal(t, "session-a", warning.HostID)
		require.Equal(t, []handlers.CatalogAreaGapResponse{area}, warning.AffectedAreas)
		require.Equal(t, quality.Warnings[i].Message, warning.Message)
		require.Contains(t, warning.Message, area.Label)
		require.Contains(t, warning.Message, "No offering")
		require.Contains(t, warning.Message, "very interested")
	}
	require.Equal(t, body.Warnings[0], body.Warnings[3])
	require.Equal(t, "stale-draft", body.Warnings[2].ID)
	var raw struct {
		Warnings []map[string]json.RawMessage `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &raw))
	require.NotContains(t, raw.Warnings[2], "message")
	require.NotContains(t, raw.Warnings[2], "affected_areas")
}

func TestDraftWarningSchemaPublishesOptionalAreaGapContext(t *testing.T) {
	schemas := NewOpenAPI(RouterOptions{}).Components.Schemas.Map()
	warning := schemas["DraftWarningResponse"]
	require.NotNil(t, warning)
	require.NotNil(t, warning.Properties["message"])
	require.Equal(t, "string", warning.Properties["message"].Type)
	require.NotContains(t, warning.Required, "message")
	areas := warning.Properties["affected_areas"]
	require.NotNil(t, areas)
	require.Equal(t, "array", areas.Type)
	require.Equal(t, "#/components/schemas/CatalogAreaGapResponse", areas.Items.Ref)
	require.NotContains(t, warning.Required, "affected_areas")
	area := schemas["CatalogAreaGapResponse"]
	require.NotNil(t, area)
	require.ElementsMatch(t, []string{"id", "label", "high_rating_count"}, area.Required)
}
