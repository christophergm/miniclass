package integration

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/identity"
	dbtesting "github.com/chrismott/miniclass/internal/testing"
	"github.com/stretchr/testify/require"
)

func TestAuthenticatedMeWithLocallySignedJWT(t *testing.T) {
	harness := dbtesting.Open(t)
	ctx := harness.Context
	userSubject := "local-auth-subject"
	userEmail := "local-organizer@example.test"
	var organizationID, userID string
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into organizations (name)
		values ('Synthetic Auth Academy')
		returning id`).Scan(&organizationID))
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, userSubject, userEmail).Scan(&userID))
	_, err := harness.Migrator.Exec(ctx, `
		insert into organization_members (organization_id, user_id, role)
		values ($1, $2, 'owner')`, organizationID, userID)
	require.NoError(t, err)

	verifier, token := localTestVerifier(t, userSubject, userEmail)
	server := api.NewServer(
		api.WithDatabase(harness.Database),
		api.WithIdentity(identity.NewStore(harness.Database)),
		api.WithVerifier(verifier),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recording := httptest.NewRecorder()
	server.Handler().ServeHTTP(recording, request)

	require.Equal(t, http.StatusOK, recording.Code)
	var response map[string]any
	require.NoError(t, json.NewDecoder(recording.Body).Decode(&response))
	require.Equal(t, "owner", response["role"])
	organization := response["organization"].(map[string]any)
	require.Equal(t, organizationID, organization["id"])
}

func TestAdminInvitationClaimUsesVerifiedEmailAndConsumesBearer(t *testing.T) {
	harness := dbtesting.Open(t)
	store := identity.NewStore(harness.Database)
	invitation, err := identity.Bootstrap(harness.Context, store, identity.BootstrapInput{
		OrganizationName: "Synthetic Claim Academy",
		HomeroomLabel:    "homeroom",
		OwnerEmail:       "claim-owner@example.test",
		ClaimBaseURL:     "https://planner.example/claim",
		InvitationTTL:    time.Hour,
	})
	require.NoError(t, err)
	verifier, token := localTestVerifier(t, "claim-subject", "claim-owner@example.test")
	server := api.NewServer(
		api.WithIdentity(store),
		api.WithVerifier(verifier),
		api.WithDatabase(harness.Database),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/auth/claim", jsonReader(map[string]string{"token": invitation.TokenValue}))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recording := httptest.NewRecorder()
	server.Handler().ServeHTTP(recording, request)
	require.Equal(t, http.StatusOK, recording.Code)

	second := httptest.NewRequest(http.MethodPost, "/api/auth/claim", jsonReader(map[string]string{"token": invitation.TokenValue}))
	second.Header.Set("Authorization", "Bearer "+token)
	second.Header.Set("Content-Type", "application/json")
	secondRecording := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRecording, second)
	require.Equal(t, http.StatusForbidden, secondRecording.Code)
}

func localTestVerifier(t *testing.T, subject, email string) (auth.Verifier, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	issuer := "https://local-issuer.example"
	verifier, err := auth.NewLocalVerifier(auth.LocalVerifierOptions{Issuer: issuer, Audience: "authenticated", PublicKey: &key.PublicKey})
	require.NoError(t, err)
	token, err := auth.MintLocalToken(auth.TokenInput{
		Subject: subject, Email: email, EmailVerified: true,
		Issuer: issuer, Audience: "authenticated", Lifetime: time.Minute,
	}, key)
	require.NoError(t, err)
	return verifier, token
}

func jsonReader(value any) *bytes.Reader {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(encoded)
}
