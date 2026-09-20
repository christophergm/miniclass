package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/danielgtaylor/huma/v2"
)

func TestGuardianOnboardingRateLimitRejectsBurstBeforeHandler(t *testing.T) {
	now := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC)
	limiter := newGuardianOnboardingLimiter(func() time.Time { return now }, nil, GuardianOnboardingRateLimitSettings{})
	var calls atomic.Int32
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))

	for range guardianOnboardingBurst {
		recording := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, apiBasePath+"/guardian/onboarding/begin", nil)
		request.RemoteAddr = "203.0.113.11:4321"
		handler.ServeHTTP(recording, request)
		if recording.Code != http.StatusNoContent {
			t.Fatalf("allowed request status = %d, want %d", recording.Code, http.StatusNoContent)
		}
	}

	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, apiBasePath+"/guardian/onboarding/begin", nil)
	request.RemoteAddr = "203.0.113.11:4321"
	handler.ServeHTTP(recording, request)
	if recording.Code != http.StatusTooManyRequests {
		t.Fatalf("limited request status = %d, want %d", recording.Code, http.StatusTooManyRequests)
	}
	if recording.Header().Get("Retry-After") != "3" {
		t.Fatalf("Retry-After = %q, want 3", recording.Header().Get("Retry-After"))
	}
	if calls.Load() != guardianOnboardingBurst {
		t.Fatalf("handler calls = %d, want %d", calls.Load(), guardianOnboardingBurst)
	}
	var response huma.ErrorModel
	if err := json.Unmarshal(recording.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rate-limit problem: %v", err)
	}
	if response.Type != string(problems.RateLimited) {
		t.Fatalf("problem type = %q, want %q", response.Type, problems.RateLimited)
	}

	now = now.Add(guardianOnboardingRefillWindow / guardianOnboardingRefill)
	recording = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, apiBasePath+"/guardian/onboarding/begin", nil)
	request.RemoteAddr = "203.0.113.11:4321"
	handler.ServeHTTP(recording, request)
	if recording.Code != http.StatusNoContent {
		t.Fatalf("refilled request status = %d, want %d", recording.Code, http.StatusNoContent)
	}
}

func TestGuardianOnboardingRateLimitUsesClientAndRouteKeys(t *testing.T) {
	limiter := newGuardianOnboardingLimiter(time.Now, nil, GuardianOnboardingRateLimitSettings{})
	for route := range map[string]bool{
		apiBasePath + "/guardian/onboarding/begin":             true,
		apiBasePath + "/guardian/onboarding/invitation/redeem": true,
		apiBasePath + "/guardian/onboarding/otp/request":       true,
	} {
		request := httptest.NewRequest(http.MethodPost, route, nil)
		request.RemoteAddr = "203.0.113.12:4321"
		if allowed, _ := limiter.allow(guardianOnboardingRateLimitKey(request)); !allowed {
			t.Fatalf("first request for %s was unexpectedly limited", route)
		}
	}

	for range guardianOnboardingBurst - 1 {
		request := httptest.NewRequest(http.MethodPost, apiBasePath+"/guardian/onboarding/begin", nil)
		request.RemoteAddr = "203.0.113.12:4321"
		if allowed, _ := limiter.allow(guardianOnboardingRateLimitKey(request)); !allowed {
			t.Fatal("begin route was unexpectedly limited")
		}
	}
	request := httptest.NewRequest(http.MethodPost, apiBasePath+"/guardian/onboarding/invitation/redeem", nil)
	request.RemoteAddr = "203.0.113.12:4321"
	if allowed, _ := limiter.allow(guardianOnboardingRateLimitKey(request)); !allowed {
		t.Fatal("invitation route shared the begin route bucket")
	}

	request = httptest.NewRequest(http.MethodPost, apiBasePath+"/guardian/onboarding/begin", nil)
	request.RemoteAddr = "203.0.113.13:4321"
	if allowed, _ := limiter.allow(guardianOnboardingRateLimitKey(request)); !allowed {
		t.Fatal("separate client shared a rate-limit bucket")
	}
}

func TestGuardianOnboardingRateLimitLeavesOtherRequestsAlone(t *testing.T) {
	if isGuardianOnboardingRateLimitedRoute(httptest.NewRequest(http.MethodGet, apiBasePath+"/guardian/onboarding/begin", nil)) {
		t.Fatal("GET request was rate limited")
	}
	if isGuardianOnboardingRateLimitedRoute(httptest.NewRequest(http.MethodPost, apiBasePath+"/health", nil)) {
		t.Fatal("unrelated POST route was rate limited")
	}
}
