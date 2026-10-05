package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type failedSurveyService struct {
	handlers.ProgramService
	cause error
}

func (s failedSurveyService) TransitionInterestProfileSurvey(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, preference.InterestProfileSurveyTransitionInput) (preference.InterestProfileSurveyTransitionResult, error) {
	return preference.InterestProfileSurveyTransitionResult{}, s.cause
}

func TestSurveyTransitionLogsUnexpectedCause(t *testing.T) {
	for _, test := range []struct {
		name       string
		cause      error
		status     int
		diagnostic string
	}{
		{"unexpected service error", errors.New("load survey audience: placeholder student dependency rejected"), 500, "placeholder student dependency rejected"},
		{"database error", fmt.Errorf("open survey: %w", &pgconn.PgError{Code: "23514", Message: "placeholder students cannot receive guardian, preference, or access records", Detail: "private student name", Hint: "private email", InternalQuery: "private SQL"}), 500, "placeholder students cannot receive guardian, preference, or access records"},
		{"ordinary conflict", preference.ErrSurveyTransitionInvalid, 409, ""},
		{"ordinary validation", preference.ErrSurveyClosingTimeRequired, 400, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			verifier, resolver, token := testAuth(t)
			router := NewRouter(RouterOptions{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Verifier: verifier, Identity: resolver, Programs: failedSurveyService{cause: test.cause}})
			request := httptest.NewRequest(http.MethodPost, "/api/school-years/year/programs/program/interest-profile-surveys/survey/transition?secret=private-query", strings.NewReader(`{"state":"open","reason":"private reason"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+token)
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, test.status, recording.Code, recording.Body.String())
			var entry map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry))
			require.Equal(t, "transition-interest-profile-survey", entry["operation_id"])
			require.Equal(t, "/api/school-years/{schoolYearID}/programs/{programID}/interest-profile-surveys/{surveyID}/transition", entry["path"])
			require.NotEmpty(t, entry["request_id"])
			if test.status == 500 {
				require.Equal(t, "ERROR", entry["level"])
				require.Contains(t, entry["error"], test.diagnostic)
				var problem huma.ErrorModel
				require.NoError(t, json.Unmarshal(recording.Body.Bytes(), &problem))
				require.Equal(t, string(problems.InternalError), problem.Type)
				require.Equal(t, "unable to change interest profile survey data", problem.Detail)
				require.NotContains(t, recording.Body.String(), test.diagnostic)
			} else {
				require.Equal(t, "INFO", entry["level"])
				require.NotContains(t, entry, "error")
			}
			for _, secret := range []string{token, "private reason", "private-query", "private student name", "private email", "private SQL"} {
				require.NotContains(t, logs.String(), secret)
				require.NotContains(t, recording.Body.String(), secret)
			}
		})
	}
}

func TestCentralErrorLogging(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"unmapped error", errors.New("unexpected service failure"), 500},
		{"mapped 500 without cause", problems.New(500, problems.InternalError, "private detail"), 500},
		{"ordinary not found", problems.New(404, problems.ResourceNotFound, "not found"), 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			router, api := newRouter(RouterOptions{Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
			registerOperation(api, huma.Operation{OperationID: "test-failure", Method: http.MethodGet, Path: "/api/test-failure"}, auth.CapabilityPublic, false, func(context.Context, *struct{}) (*struct{}, error) { return nil, test.err })
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/test-failure", nil))
			require.Equal(t, test.status, recording.Code)
			var entry map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry))
			if test.status == 500 {
				require.Equal(t, "ERROR", entry["level"])
			} else {
				require.Equal(t, "INFO", entry["level"])
			}
			require.NotContains(t, logs.String(), "private detail")
			require.NotContains(t, recording.Body.String(), "unexpected service failure")
		})
	}
}

func TestRequestLoggerCatchesNonHandler500(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		problems.Write(w, problems.New(500, problems.InternalError, "private detail"))
	}))
	recording := httptest.NewRecorder()
	handler.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/private-path?secret=private-query", nil))
	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry))
	require.Equal(t, "ERROR", entry["level"])
	require.Equal(t, float64(500), entry["status"])
	require.NotContains(t, logs.String(), "private")
}
