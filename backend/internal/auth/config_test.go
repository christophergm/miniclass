package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/chrismott/miniclass/internal/config"
)

// TestNewFromConfigBuildsLocalVerifierFromKeyFiles covers the local development
// contract in ADR 0011: .env carries only paths, because a PEM header contains
// spaces and no .env value may. The whole chain is exercised — config reads the
// files, the verifier is built from them, and a token minted with the matching
// private key verifies — because "the key was loaded" and "the API can
// authenticate" are different claims.
func TestNewFromConfigBuildsLocalVerifierFromKeyFiles(t *testing.T) {
	key, publicPath, privatePath := writeKeyPair(t)
	setLocalAuthEnv(t, map[string]string{
		"AUTH_LOCAL_PUBLIC_KEY_FILE":  publicPath,
		"AUTH_LOCAL_PRIVATE_KEY_FILE": privatePath,
	})

	cfg := loadConfig(t)
	require.Empty(t, os.Getenv("AUTH_LOCAL_PUBLIC_KEY"), "the inline form must not be what makes this pass")
	require.Contains(t, cfg.AuthLocalPublicKey, "BEGIN PUBLIC KEY")

	requireVerifies(t, cfg, key)
}

// TestNewFromConfigBuildsLocalVerifierFromPrivateKeyFileAlone pins the
// single-value case: cmd/devtoken needs the private key, and deriving the public
// key from it means a developer cannot get the pair out of step.
func TestNewFromConfigBuildsLocalVerifierFromPrivateKeyFileAlone(t *testing.T) {
	key, _, privatePath := writeKeyPair(t)
	setLocalAuthEnv(t, map[string]string{"AUTH_LOCAL_PRIVATE_KEY_FILE": privatePath})

	cfg := loadConfig(t)
	require.Empty(t, cfg.AuthLocalPublicKey)
	require.Contains(t, cfg.AuthLocalPrivateKey, "BEGIN EC PRIVATE KEY")

	requireVerifies(t, cfg, key)
}

// TestNewFromConfigBuildsLocalVerifierFromInlineKey proves the inline form is
// still a working fallback, for a deployment target that injects secrets as
// environment variables and has no writable filesystem.
func TestNewFromConfigBuildsLocalVerifierFromInlineKey(t *testing.T) {
	key, publicPath, _ := writeKeyPair(t)
	publicPEM, err := os.ReadFile(publicPath)
	require.NoError(t, err)
	setLocalAuthEnv(t, map[string]string{"AUTH_LOCAL_PUBLIC_KEY": string(publicPEM)})

	cfg := loadConfig(t)
	requireVerifies(t, cfg, key)
}

// TestNewFromConfigReportsMissingKeyFile checks the error a wrong working
// directory produces. Key paths in .env are relative to backend/, so this is the
// most likely way to misconfigure them, and the absolute path is what makes the
// mistake visible.
func TestNewFromConfigReportsMissingKeyFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.pem")
	setLocalAuthEnv(t, map[string]string{"AUTH_LOCAL_PRIVATE_KEY_FILE": missing})

	_, err := config.LoadFrom(filepath.Join(t.TempDir(), "absent.env"))
	require.Error(t, err)
	require.ErrorContains(t, err, "AUTH_LOCAL_PRIVATE_KEY_FILE")
	require.ErrorContains(t, err, missing)
}

func TestNewFromConfigRequiresALocalKey(t *testing.T) {
	setLocalAuthEnv(t, nil)

	cfg := loadConfig(t)
	_, err := NewFromConfig(*cfg)
	require.ErrorIs(t, err, ErrMissingVerifierKey)
}

// writeKeyPair generates an ES256 keypair and writes it in the PEM encodings
// openssl produces, so the fixtures match what scripts/setup.sh generates.
func writeKeyPair(t *testing.T) (key *ecdsa.PrivateKey, publicPath, privatePath string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	privateDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	directory := t.TempDir()
	privatePath = filepath.Join(directory, "local_auth_private.pem")
	publicPath = filepath.Join(directory, "local_auth_public.pem")
	require.NoError(t, os.WriteFile(privatePath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateDER}), 0o600))
	require.NoError(t, os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o644))

	return key, publicPath, privatePath
}

// setLocalAuthEnv clears every variable that can supply a local key and then
// applies the given ones, so a developer's exported AUTH_LOCAL_* values cannot
// decide whether these tests pass.
func setLocalAuthEnv(t *testing.T, values map[string]string) {
	t.Helper()

	for _, name := range []string{
		"AUTH_LOCAL_PUBLIC_KEY", "AUTH_LOCAL_PRIVATE_KEY",
		"AUTH_LOCAL_PUBLIC_KEY_FILE", "AUTH_LOCAL_PRIVATE_KEY_FILE",
		"SUPABASE_URL",
	} {
		t.Setenv(name, "")
	}
	for name, value := range values {
		t.Setenv(name, value)
	}

	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_PROVIDER", "local")
	t.Setenv("AUTH_ISSUER", "http://localhost:8080")
	t.Setenv("AUTH_AUDIENCE", "authenticated")
	t.Setenv("DATABASE_URL", "postgres://example")
}

func loadConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := config.LoadFrom(filepath.Join(t.TempDir(), "absent.env"))
	require.NoError(t, err)
	return cfg
}

func requireVerifies(t *testing.T, cfg *config.Config, key *ecdsa.PrivateKey) {
	t.Helper()

	verifier, err := NewFromConfig(*cfg)
	require.NoError(t, err)

	token, err := MintLocalToken(TokenInput{
		Subject: "local:dev", Email: "owner@example.test", EmailVerified: true,
		Issuer: cfg.AuthIssuer, Audience: cfg.AuthAudience, KeyID: "local", Lifetime: time.Minute,
	}, key)
	require.NoError(t, err)

	identity, err := verifier.Verify(t.Context(), token)
	require.NoError(t, err)
	require.Equal(t, "owner@example.test", identity.Email)
}
