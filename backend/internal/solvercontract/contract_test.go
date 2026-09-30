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
	first := Request{Version: Version, Seed: 41, MaxDeterministicTime: 1, Offerings: []Offering{{ID: "offering-b", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 2}, {ID: "offering-a", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}}, Participants: []Participant{{ID: "student-b", GradeOrdinal: 2}, {ID: "student-a", GradeOrdinal: 1}}}
	second := Request{Version: Version, Seed: 41, MaxDeterministicTime: 1, Offerings: []Offering{{ID: "offering-a", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}, {ID: "offering-b", Capacity: 1, MinGradeOrdinal: 1, MaxGradeOrdinal: 2}}, Participants: []Participant{{ID: "student-a", GradeOrdinal: 1}, {ID: "student-b", GradeOrdinal: 2}}}

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
	_, err := CanonicalJSON(Request{Version: Version, Seed: 1, MaxDeterministicTime: 1, Offerings: []Offering{{ID: "offering-1", Capacity: 1, MinGradeOrdinal: 2, MaxGradeOrdinal: 1}}, Participants: []Participant{{ID: "student-1", GradeOrdinal: 1}}})
	require.Error(t, err)
}

func TestCanonicalResponsePreservesFutureConflictDiagnosticField(t *testing.T) {
	encoded, err := CanonicalResponseJSON(Response{Version: Version, Seed: 1, Status: "infeasible", Assignments: []Assignment{}, ConflictDiagnostics: []ConflictDiagnostic{}})
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"v1","seed":1,"status":"infeasible","assignments":[],"conflict_diagnostics":[]}`, string(encoded))
}
