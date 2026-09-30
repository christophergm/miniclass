package solverclient_test

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/solverclient"
	"github.com/chrismott/miniclass/internal/solvercontract"
	"github.com/stretchr/testify/require"
)

const (
	fullSolveBudget        = 10 * time.Second
	pinnedResolveBudget    = 2 * time.Second
	sidecarRequestTimeout  = fullSolveBudget + 2*time.Second
	expectedScaleStudents  = 140
	expectedScaleOfferings = 8
)

func TestExpectedScaleScenarioBuildsContract(t *testing.T) {
	request := expectedScaleRequest(t)
	_, err := solvercontract.CanonicalJSON(request)
	require.NoError(t, err)
}

// TestExpectedScaleSidecarPerformance exercises the deployed solver image
// through the production Go HTTP client. It is deliberately opt-in for normal
// backend tests: CI's Solver image job supplies SOLVER_BASE_URL after it has
// started and health-checked that image.
func TestExpectedScaleSidecarPerformance(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("SOLVER_BASE_URL"))
	if baseURL == "" {
		t.Skip("SOLVER_BASE_URL is required for the sidecar performance check")
	}

	request := expectedScaleRequest(t)
	client, err := solverclient.New(baseURL, sidecarRequestTimeout)
	require.NoError(t, err)

	started := time.Now()
	first, err := client.Solve(context.Background(), request)
	fullElapsed := time.Since(started)
	require.NoError(t, err)
	require.Less(t, fullElapsed, fullSolveBudget,
		"full expected-scale solve took %s; budget is %s", fullElapsed, fullSolveBudget)
	require.Equal(t, "optimal", first.Status)
	require.Len(t, first.Assignments, len(request.Participants))

	second, err := client.Solve(context.Background(), request)
	require.NoError(t, err)
	firstJSON := canonicalResponse(t, first)
	require.Equal(t, firstJSON, canonicalResponse(t, second),
		"identical expected-scale request and seed must produce byte-identical canonical results")

	pinnedRequest := request
	pinnedRequest.Pins = []solvercontract.PinnedPlacement{{
		ParticipantID: first.Assignments[0].ParticipantID,
		OfferingID:    first.Assignments[0].OfferingID,
	}}
	started = time.Now()
	pinned, err := client.Solve(context.Background(), pinnedRequest)
	pinnedElapsed := time.Since(started)
	require.NoError(t, err)
	require.Less(t, pinnedElapsed, pinnedResolveBudget,
		"pinned expected-scale re-solve took %s; budget is %s", pinnedElapsed, pinnedResolveBudget)
	require.Equal(t, "optimal", pinned.Status)
	require.Len(t, pinned.Assignments, len(request.Participants))

	t.Logf("expected-scale sidecar timings: full=%s pinned_re_solve=%s", fullElapsed, pinnedElapsed)
}

func canonicalResponse(t *testing.T, response solvercontract.Response) []byte {
	t.Helper()
	encoded, err := solvercontract.CanonicalResponseJSON(response)
	require.NoError(t, err)
	return encoded
}

func expectedScaleRequest(t *testing.T) solvercontract.Request {
	t.Helper()
	directory := expectedScaleScenarioDirectory(t)
	participants := readExpectedScaleParticipants(t, filepath.Join(directory, "students.csv"))
	offerings := readExpectedScaleOfferings(t, filepath.Join(directory, "offerings.csv"))
	require.Len(t, participants, expectedScaleStudents, "expected-scale fixture must retain its specified student count")
	require.Len(t, offerings, expectedScaleOfferings, "expected-scale fixture must retain its specified offering count")

	return solvercontract.Request{
		Version:              solvercontract.Version,
		Seed:                 19,
		MaxDeterministicTime: 1,
		QualityConfig:        solvercontract.QualityConfig{HighRankMax: 3},
		Participants:         participants,
		Offerings:            offerings,
		Pins:                 []solvercontract.PinnedPlacement{},
	}
}

func expectedScaleScenarioDirectory(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(source), "..", "..", "..", "solver", "tests", "scenarios", "expected-scale")
}

func readExpectedScaleParticipants(t *testing.T, path string) []solvercontract.Participant {
	t.Helper()
	records := readCSV(t, path, []string{"id", "grade_ordinal"})
	participants := make([]solvercontract.Participant, 0, len(records))
	for _, record := range records {
		participants = append(participants, solvercontract.Participant{
			ID:           record[0],
			GradeOrdinal: parseCSVInteger(t, path, "grade_ordinal", record[1]),
		})
	}
	return participants
}

func readExpectedScaleOfferings(t *testing.T, path string) []solvercontract.Offering {
	t.Helper()
	records := readCSV(t, path, []string{"id", "capacity", "min_grade_ordinal", "max_grade_ordinal", "interest_area_id"})
	offerings := make([]solvercontract.Offering, 0, len(records))
	for _, record := range records {
		offering := solvercontract.Offering{
			ID:              record[0],
			Capacity:        parseCSVInteger(t, path, "capacity", record[1]),
			MinGradeOrdinal: parseCSVInteger(t, path, "min_grade_ordinal", record[2]),
			MaxGradeOrdinal: parseCSVInteger(t, path, "max_grade_ordinal", record[3]),
		}
		if record[4] != "" {
			offering.InterestAreaID = &record[4]
		}
		offerings = append(offerings, offering)
	}
	return offerings
}

func readCSV(t *testing.T, path string, headers []string) [][]string {
	t.Helper()
	source, err := os.Open(path)
	require.NoError(t, err)
	defer source.Close()

	records, err := csv.NewReader(source).ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records)
	require.Equal(t, headers, records[0], "unexpected headers in %s", path)
	for _, record := range records[1:] {
		require.Len(t, record, len(headers), "unexpected row in %s", path)
	}
	return records[1:]
}

func parseCSVInteger(t *testing.T, path, field, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(value)
	require.NoErrorf(t, err, "%s %s must be an integer", path, field)
	return parsed
}
