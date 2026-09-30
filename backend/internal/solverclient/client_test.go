package solverclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/solvercontract"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestSolveUsesCanonicalContractAndValidatesResponse(t *testing.T) {
	client := testClient(t, func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "/v1/solve", request.URL.Path)
		require.Equal(t, "application/json", request.Header.Get("Content-Type"))
		return response(http.StatusOK, `{"version":"v1","seed":41,"status":"optimal","assignments":[{"participant_id":"student-b","offering_id":"offering-b"},{"participant_id":"student-a","offering_id":"offering-a"}]}`), nil
	})

	result, err := client.Solve(context.Background(), testRequest())
	require.NoError(t, err)
	require.Equal(t, "student-a", result.Assignments[0].ParticipantID)
}

func TestSolveMapsSidecarFiveHundredToUnavailable(t *testing.T) {
	client := testClient(t, func(*http.Request) (*http.Response, error) { return response(http.StatusServiceUnavailable, ""), nil })
	_, err := client.Solve(context.Background(), testRequest())
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSolveRejectsMismatchedSeed(t *testing.T) {
	client := testClient(t, func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, `{"version":"v1","seed":42,"status":"optimal","assignments":[]}`), nil
	})
	_, err := client.Solve(context.Background(), testRequest())
	require.True(t, errors.Is(err, ErrInvalidResponse))
}

func testClient(t *testing.T, transport roundTripperFunc) *HTTPClient {
	t.Helper()
	client, err := New("http://solver.test", time.Second)
	require.NoError(t, err)
	client.client.Transport = transport
	return client
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func testRequest() solvercontract.Request {
	return solvercontract.Request{Version: solvercontract.Version, Seed: 41, MaxDeterministicTime: 1, Offerings: []solvercontract.Offering{{ID: "offering-a", Capacity: 1}, {ID: "offering-b", Capacity: 1}}, Participants: []solvercontract.Participant{{ID: "student-a", AcceptableOfferingIDs: []string{"offering-a"}}, {ID: "student-b", AcceptableOfferingIDs: []string{"offering-b"}}}}
}
