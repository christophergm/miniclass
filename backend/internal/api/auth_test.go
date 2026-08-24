package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestAuthenticatedMeEndpointReturnsResolvedPrincipal(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	router := NewRouter(RouterOptions{Verifier: verifier, Identity: resolver})
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)

	require.Equal(t, http.StatusOK, recording.Code)
	var response struct {
		Principal struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"principal"`
		Organization struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"organization"`
		Role string `json:"role"`
	}
	require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
	require.Equal(t, "user-test", response.Principal.ID)
	require.Equal(t, "organizer@example.test", response.Principal.Email)
	require.Equal(t, "org-test", response.Organization.ID)
	require.Equal(t, "Synthetic Academy", response.Organization.Name)
	require.Equal(t, "owner", response.Role)
}

func TestAuthenticationAndMembershipFailuresUseDistinctProblems(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	router := NewRouter(RouterOptions{Verifier: verifier, Identity: resolver})
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	require.Equal(t, http.StatusUnauthorized, recording.Code)
	var unauth huma.ErrorModel
	require.NoError(t, json.NewDecoder(recording.Body).Decode(&unauth))
	require.Equal(t, string(problems.AuthenticationRequired), unauth.Type)

	for _, test := range []struct {
		name     string
		resolver auth.AccountResolver
		status   int
		problem  problems.Slug
	}{
		{name: "no organization", resolver: errorResolver{err: auth.ErrNoOrganization}, status: http.StatusForbidden, problem: problems.NoOrganization},
		{name: "multiple organizations", resolver: errorResolver{err: auth.ErrMultipleOrganizations}, status: http.StatusConflict, problem: problems.MultipleOrganizations},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter(RouterOptions{Verifier: verifier, Identity: test.resolver})
			request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)
			require.Equal(t, test.status, recording.Code)
			var problemResponse huma.ErrorModel
			require.NoError(t, json.NewDecoder(recording.Body).Decode(&problemResponse))
			require.Equal(t, string(test.problem), problemResponse.Type)
		})
	}
}

type errorResolver struct{ err error }

func (r errorResolver) ResolveAccount(context.Context, string) (auth.Account, error) {
	return auth.Account{}, r.err
}

type testClaimer struct {
	called bool
	input  auth.InvitationClaimInput
}

func (c *testClaimer) ClaimAdminInvitation(_ context.Context, input auth.InvitationClaimInput) (auth.Account, error) {
	c.called = true
	c.input = input
	return auth.Account{
		User:       auth.AccountUser{ID: "claimed-user", ProviderSubject: input.ProviderSubject, Email: input.Email},
		Membership: auth.AccountMembership{OrganizationID: "claimed-org", OrganizationName: "Claimed Academy", Role: "owner"},
	}, nil
}

func TestInvitationClaimAllowsAnUnresolvedVerifiedSubject(t *testing.T) {
	verifier, _, token := testAuth(t)
	claimer := &testClaimer{}
	router := NewRouter(RouterOptions{
		Verifier: verifier, Identity: errorResolver{err: auth.ErrNoOrganization}, Claimer: claimer,
	})
	request := httptest.NewRequest(http.MethodPost, "/api/auth/claim", strings.NewReader(`{"token":"invitation-token"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, request)

	require.Equal(t, http.StatusOK, recording.Code)
	require.True(t, claimer.called)
	require.Equal(t, "invitation-token", claimer.input.Bearer)
	require.True(t, claimer.input.EmailVerified)
}

func TestEveryRegisteredOperationDeclaresCapabilityMetadata(t *testing.T) {
	document := NewOpenAPI(RouterOptions{})
	encoded, err := json.Marshal(document)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(encoded, &raw))
	paths := raw["paths"].(map[string]any)
	for path, value := range paths {
		operations := value.(map[string]any)
		for method, operation := range operations {
			if method == "parameters" {
				continue
			}
			op := operation.(map[string]any)
			require.NotEmpty(t, op[auth.RequiredCapabilityExtension], "%s %s has no required capability", method, path)
			require.NotEmpty(t, op["security"], "%s %s has no bearer security declaration", method, path)
		}
	}
}

func TestAdministratorManagementRejectsAdministratorAndCoordinator(t *testing.T) {
	for _, role := range []string{"administrator", "coordinator"} {
		t.Run(role, func(t *testing.T) {
			verifier, resolver, token := testAuthRole(t, role)
			router := NewRouter(RouterOptions{Verifier: verifier, Identity: resolver})
			request := httptest.NewRequest(http.MethodGet, "/api/administrators", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			recording := httptest.NewRecorder()
			router.ServeHTTP(recording, request)

			require.Equal(t, http.StatusForbidden, recording.Code)
			var response huma.ErrorModel
			require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
			require.Equal(t, string(problems.CapabilityRequired), response.Type)
		})
	}
}

func TestCrossOrganizationResourceAccessMapsToNotFound(t *testing.T) {
	verifier, resolver, token := testAuth(t)
	type resourceInput struct {
		OrganizationID string `path:"organizationID"`
	}
	type resourceBody struct {
		ID string `json:"id"`
	}
	type resourceOutput struct {
		Body resourceBody
	}
	resourceRouter := chi.NewRouter()
	resourceAPI := humachi.New(resourceRouter, huma.DefaultConfig("resource test", "test"))
	resourceAPI.UseMiddleware(auth.Middleware(verifier, resolver, writeAuthError))
	registerOperation(resourceAPI, huma.Operation{
		OperationID: "get-test-resource-real",
		Method:      http.MethodGet,
		Path:        "/api/resources/{organizationID}",
	}, auth.CapabilityAuthenticated, false, func(ctx context.Context, input *resourceInput) (*resourceOutput, error) {
		principal, ok := auth.PrincipalFromContext(ctx)
		if !ok {
			return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "principal missing")
		}
		account := principal.(auth.AccountPrincipal)
		if input.OrganizationID != string(account.OrganizationID) {
			return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "resource not found")
		}
		return &resourceOutput{Body: resourceBody{ID: input.OrganizationID}}, nil
	})
	request := httptest.NewRequest(http.MethodGet, "/api/resources/foreign-org", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	resourceRouter.ServeHTTP(recording, request)
	require.Equal(t, http.StatusNotFound, recording.Code)
	var response huma.ErrorModel
	require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
	require.Equal(t, string(problems.ResourceNotFound), response.Type)
}
