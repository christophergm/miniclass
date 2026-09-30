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
	first := Request{Version: Version, Seed: 41, MaxDeterministicTime: 1, Offerings: []Offering{{ID: "offering-b", Capacity: 1}, {ID: "offering-a", Capacity: 1}}, Participants: []Participant{{ID: "student-b", AcceptableOfferingIDs: []string{"offering-b", "offering-a"}}, {ID: "student-a", AcceptableOfferingIDs: []string{"offering-a"}}}}
	second := Request{Version: Version, Seed: 41, MaxDeterministicTime: 1, Offerings: []Offering{{ID: "offering-a", Capacity: 1}, {ID: "offering-b", Capacity: 1}}, Participants: []Participant{{ID: "student-a", AcceptableOfferingIDs: []string{"offering-a"}}, {ID: "student-b", AcceptableOfferingIDs: []string{"offering-a", "offering-b"}}}}

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

func TestCanonicalJSONRejectsUnknownOffering(t *testing.T) {
	_, err := CanonicalJSON(Request{Version: Version, Seed: 1, MaxDeterministicTime: 1, Offerings: []Offering{{ID: "offering-1", Capacity: 1}}, Participants: []Participant{{ID: "student-1", AcceptableOfferingIDs: []string{"missing"}}}})
	require.Error(t, err)
}
