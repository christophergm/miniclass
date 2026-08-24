package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestLocalVerifierValidatesAsymmetricClaimsAndClockSkew(t *testing.T) {
	now := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	verifier, err := NewLocalVerifier(LocalVerifierOptions{
		Issuer: "https://issuer.test", Audience: "authenticated", PublicKey: &key.PublicKey, Now: func() time.Time { return now },
	})
	require.NoError(t, err)

	token, err := MintLocalToken(TokenInput{
		Subject: "subject-1", Email: "owner@example.test", EmailVerified: true,
		Issuer: "https://issuer.test", Audience: "authenticated", Now: now,
		ExpiresAt: now.Add(ClockSkew), NotBefore: now.Add(-ClockSkew),
	}, key)
	require.NoError(t, err)
	identity, err := verifier.Verify(t.Context(), token)
	require.NoError(t, err)
	require.Equal(t, "subject-1", identity.Subject)
	require.True(t, identity.EmailVerified)

	for _, test := range []struct {
		name      string
		expiresAt time.Time
		notBefore time.Time
		wantErr   bool
	}{
		{name: "expired beyond skew", expiresAt: now.Add(-ClockSkew - time.Second), notBefore: now.Add(-time.Minute), wantErr: true},
		{name: "not before beyond skew", expiresAt: now.Add(time.Minute), notBefore: now.Add(ClockSkew + time.Second), wantErr: true},
		{name: "expired within skew", expiresAt: now.Add(-ClockSkew + time.Second), notBefore: now.Add(-time.Minute)},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := MintLocalToken(TokenInput{
				Subject: "subject-1", Email: "owner@example.test", EmailVerified: true,
				Issuer: "https://issuer.test", Audience: "authenticated", Now: now,
				ExpiresAt: test.expiresAt, NotBefore: test.notBefore,
			}, key)
			require.NoError(t, err)
			_, err = verifier.Verify(t.Context(), raw)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLocalVerifierRejectsUnsupportedAlgorithmAndProduction(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	_, err = NewLocalVerifierForEnvironment("production", LocalVerifierOptions{
		Issuer: "https://issuer.test", Audience: "authenticated", PublicKey: &key.PublicKey,
	})
	require.ErrorIs(t, err, ErrLocalProduction)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "subject-1", "email": "owner@example.test", "iss": "https://issuer.test",
		"aud": "authenticated", "exp": time.Now().Add(time.Minute).Unix(),
	})
	raw, err := token.SignedString([]byte("not-an-asymmetric-key"))
	require.NoError(t, err)
	verifier, err := NewLocalVerifier(LocalVerifierOptions{
		Issuer: "https://issuer.test", Audience: "authenticated", PublicKey: &key.PublicKey,
	})
	require.NoError(t, err)
	_, err = verifier.Verify(t.Context(), raw)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnsupportedAlg))
}

func TestSupabaseVerifierCachesJWKSAndRateLimitsUnknownKids(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/.well-known/jwks.json", r.URL.Path)
		requests++
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "EC", "kid": "known", "alg": "ES256", "crv": "P-256",
			"x": base64URL(key.PublicKey.X.Bytes()), "y": base64URL(key.PublicKey.Y.Bytes()),
		}}})
	}))
	defer server.Close()
	now := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	verifier, err := NewSupabaseVerifier(SupabaseVerifierOptions{
		Issuer: server.URL, Audience: "authenticated", Now: func() time.Time { return now }, RefreshEvery: time.Minute,
	})
	require.NoError(t, err)
	known, err := MintLocalToken(TokenInput{
		Subject: "subject-1", Email: "owner@example.test", EmailVerified: true,
		Issuer: server.URL, Audience: "authenticated", KeyID: "known", Now: now, Lifetime: time.Minute,
	}, key)
	require.NoError(t, err)
	_, err = verifier.Verify(t.Context(), known)
	require.NoError(t, err)
	require.Equal(t, 1, requests)
	unknown, err := MintLocalToken(TokenInput{
		Subject: "subject-1", Email: "owner@example.test", EmailVerified: true,
		Issuer: server.URL, Audience: "authenticated", KeyID: "unknown", Now: now, Lifetime: time.Minute,
	}, key)
	require.NoError(t, err)
	_, err = verifier.Verify(t.Context(), unknown)
	require.Error(t, err)
	require.Equal(t, 1, requests, "unknown kid triggered an unbounded JWKS refresh")
}

func TestCapabilityMatrixContainsEverySpecCell(t *testing.T) {
	want := map[OrganizationRole]map[Capability]bool{
		RoleOwner:         {CapabilityManageAdministrators: true, CapabilityDeletePersonalData: true, CapabilityManageSchoolYear: true, CapabilityManageRoster: true, CapabilityManageCatalog: true, CapabilityManageAssignments: true, CapabilityManagePublishing: true, CapabilityReadAuditLog: true},
		RoleAdministrator: {CapabilityManageAdministrators: false, CapabilityDeletePersonalData: false, CapabilityManageSchoolYear: true, CapabilityManageRoster: true, CapabilityManageCatalog: true, CapabilityManageAssignments: true, CapabilityManagePublishing: true, CapabilityReadAuditLog: true},
		RoleCoordinator:   {CapabilityManageAdministrators: false, CapabilityDeletePersonalData: false, CapabilityManageSchoolYear: false, CapabilityManageRoster: true, CapabilityManageCatalog: true, CapabilityManageAssignments: true, CapabilityManagePublishing: false, CapabilityReadAuditLog: false},
	}
	for _, cell := range MatrixCells() {
		require.Equal(t, want[cell.Role][cell.Capability], cell.Allowed, "%s/%s", cell.Role, cell.Capability)
	}
}

func base64URL(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}
