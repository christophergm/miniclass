package solver

import (
	"encoding/json"
	"testing"

	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/solvercontract"
	"github.com/stretchr/testify/require"
)

func TestValidateSuccessfulResponseRequiresCompleteKnownAssignments(t *testing.T) {
	request := solverRequest()
	response := solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "optimal", Assignments: []solvercontract.Assignment{{ParticipantID: "student-a", OfferingID: "offering-a", RealizedQuality: solvercontract.QualityTop}}}

	require.ErrorContains(t, validateSuccessfulResponse(request, response), "every participant")

	response.Assignments = append(response.Assignments, solvercontract.Assignment{ParticipantID: "student-b", OfferingID: "unknown", RealizedQuality: solvercontract.QualityNeutral})
	require.ErrorContains(t, validateSuccessfulResponse(request, response), "unknown offering")
}

func TestValidateSuccessfulResponseRetainsPins(t *testing.T) {
	request := solverRequest()
	request.Pins = []solvercontract.PinnedPlacement{{ParticipantID: "student-a", OfferingID: "offering-a"}}
	response := solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "optimal", Assignments: []solvercontract.Assignment{
		{ParticipantID: "student-a", OfferingID: "offering-b", RealizedQuality: solvercontract.QualityTop},
		{ParticipantID: "student-b", OfferingID: "offering-b", RealizedQuality: solvercontract.QualityNeutral},
	}}

	require.ErrorContains(t, validateSuccessfulResponse(request, response), "did not retain pin")
}

func TestAssignmentInputsAndMetricsCaptureHistoricalQuality(t *testing.T) {
	request := solverRequest()
	request.Pins = []solvercontract.PinnedPlacement{{ParticipantID: "student-a", OfferingID: "offering-a"}}
	response := solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "optimal", Assignments: []solvercontract.Assignment{
		{ParticipantID: "student-a", OfferingID: "offering-a", RealizedQuality: solvercontract.QualityTop},
		{ParticipantID: "student-b", OfferingID: "offering-b", RealizedQuality: solvercontract.QualityUnwanted},
	}}

	assignments := assignmentInputs(StartInput{SchoolYearID: "year", ProgramID: "program", SessionID: "session", Request: request}, ids.XID("run"), response)
	require.Equal(t, "top", assignments[0].RealizedQuality)
	require.True(t, assignments[0].Pinned)
	require.Equal(t, "unwanted", assignments[1].RealizedQuality)

	document, err := metricsDocument(response)
	require.NoError(t, err)
	var metrics map[string]map[string]int
	require.NoError(t, json.Unmarshal(document, &metrics))
	require.Equal(t, 1, metrics["quality_distribution"][solvercontract.QualityTop])
	require.Equal(t, 1, metrics["quality_distribution"][solvercontract.QualityUnwanted])
}

func solverRequest() solvercontract.Request {
	return solvercontract.Request{Version: solvercontract.Version, Seed: 17, MaxDeterministicTime: 1, QualityConfig: solvercontract.QualityConfig{HighRankMax: 3}, Participants: []solvercontract.Participant{{ID: "student-a", GradeOrdinal: 1}, {ID: "student-b", GradeOrdinal: 1}}, Offerings: []solvercontract.Offering{{ID: "offering-a", Capacity: 2, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}, {ID: "offering-b", Capacity: 2, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}}}
}
