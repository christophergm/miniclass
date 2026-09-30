package solvercontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedRequestFixtureValidates(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "solver", "tests", "fixtures", "solve-request-v1.json"))
	require.NoError(t, err)
	var request Request
	require.NoError(t, json.Unmarshal(fixture, &request))
	_, err = CanonicalJSON(request)
	require.NoError(t, err)
}

func TestCanonicalJSONIgnoresInputCollectionOrder(t *testing.T) {
	first := Request{
		Version:              Version,
		Seed:                 41,
		MaxDeterministicTime: 1,
		QualityConfig:        QualityConfig{HighRankMax: 3},
		Offerings: []Offering{
			{ID: "offering-b", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 2},
			{ID: "offering-a", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 1},
		},
		Participants: []Participant{
			{ID: "student-b", GradeOrdinal: 2, InterestProfile: []InterestRating{}},
			{ID: "student-a", GradeOrdinal: 1, InterestProfile: []InterestRating{}},
		},
	}
	second := Request{
		Version:              Version,
		Seed:                 41,
		MaxDeterministicTime: 1,
		QualityConfig:        QualityConfig{HighRankMax: 3},
		Offerings: []Offering{
			{ID: "offering-a", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 1},
			{ID: "offering-b", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 2},
		},
		Participants: []Participant{
			{ID: "student-a", GradeOrdinal: 1, InterestProfile: []InterestRating{}},
			{ID: "student-b", GradeOrdinal: 2, InterestProfile: []InterestRating{}},
		},
	}

	firstBytes, err := CanonicalJSON(first)
	require.NoError(t, err)
	secondBytes, err := CanonicalJSON(second)
	require.NoError(t, err)
	require.Equal(t, string(firstBytes), string(secondBytes))
	firstFingerprint, err := Fingerprint(first)
	require.NoError(t, err)
	secondFingerprint, err := Fingerprint(second)
	require.NoError(t, err)
	require.Equal(t, firstFingerprint, secondFingerprint)
}

func TestCanonicalJSONRejectsInvalidGradeWindow(t *testing.T) {
	_, err := CanonicalJSON(Request{
		Version:              Version,
		Seed:                 1,
		MaxDeterministicTime: 1,
		QualityConfig:        QualityConfig{HighRankMax: 3},
		Offerings:            []Offering{{ID: "offering-1", Capacity: 1, MinGradeOrdinal: 2, MaxGradeOrdinal: 1}},
		Participants:         []Participant{{ID: "student-1", GradeOrdinal: 1}},
	})
	require.Error(t, err)
}

func TestCanonicalJSONValidatesAndOrdersPreferenceInputs(t *testing.T) {
	request := Request{
		Version:              Version,
		Seed:                 1,
		MaxDeterministicTime: 1,
		QualityConfig:        QualityConfig{HighRankMax: 3},
		Offerings: []Offering{
			{ID: "offering-a", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 1, InterestAreaID: stringPointer("area-a")},
			{ID: "offering-b", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 1},
		},
		Participants: []Participant{{
			ID:           "student-a",
			GradeOrdinal: 1,
			RankedChoices: &RankedChoices{Choices: []RankedChoice{
				{OfferingID: "offering-b", Response: InterestedResponse},
				{OfferingID: "offering-a", Response: RankedResponse, Rank: 1},
			}},
			InterestProfile: []InterestRating{
				{InterestAreaID: "area-b", Rating: InterestedResponse},
				{InterestAreaID: "area-a", Rating: VeryInterestedRating},
			},
		}},
	}

	encoded, err := CanonicalJSON(request)
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"v1","seed":1,"max_deterministic_time":1,"quality_config":{"high_rank_max":3},"participants":[{"id":"student-a","grade_ordinal":1,"ranked_choices":{"choices":[{"offering_id":"offering-a","response":"ranked","rank":1},{"offering_id":"offering-b","response":"interested"}]},"interest_profile":[{"interest_area_id":"area-a","rating":"very_interested"},{"interest_area_id":"area-b","rating":"interested"}]}],"offerings":[{"id":"offering-a","capacity":1,"min_grade_ordinal":1,"max_grade_ordinal":1,"interest_area_id":"area-a"},{"id":"offering-b","capacity":1,"min_grade_ordinal":1,"max_grade_ordinal":1}]}`, string(encoded))
}

func TestCanonicalResponsePreservesFutureConflictDiagnosticField(t *testing.T) {
	encoded, err := CanonicalResponseJSON(Response{Version: Version, Seed: 1, Status: "infeasible", Assignments: []Assignment{}, ConflictDiagnostics: []ConflictDiagnostic{}})
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"v1","seed":1,"status":"infeasible","assignments":[],"conflict_diagnostics":[]}`, string(encoded))
}

func TestCanonicalResponseRequiresRealizedQuality(t *testing.T) {
	_, err := CanonicalResponseJSON(Response{Version: Version, Seed: 1, Status: "optimal", Assignments: []Assignment{{ParticipantID: "student-a", OfferingID: "offering-a"}}})
	require.Error(t, err)
}

func stringPointer(value string) *string { return &value }
