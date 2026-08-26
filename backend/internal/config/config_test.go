package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testPublicKeyPEM  = "-----BEGIN PUBLIC KEY-----\nTUlOSUNMQVNTIFRFU1QgUFVCTElDIEtFWQ==\n-----END PUBLIC KEY-----"
	testPrivateKeyPEM = "-----BEGIN EC PRIVATE KEY-----\nTUlOSUNMQVNTIFRFU1QgUFJJVkFURSBLRVk=\n-----END EC PRIVATE KEY-----"
)

func TestLoadFromDotEnv(t *testing.T) {
	for _, key := range []string{"APP_ENV", "APP_VERSION", "PORT", "API_BASE_URL", "INVITATION_CLAIM_BASE_URL", "TRUSTED_PROXY_CIDRS", "DATABASE_URL", "TEST_DATABASE_URL", "AUTH_PROVIDER", "AUTH_ISSUER", "AUTH_AUDIENCE", "AUTH_LOCAL_PUBLIC_KEY", "AUTH_LOCAL_PRIVATE_KEY", "AUTH_LOCAL_PUBLIC_KEY_FILE", "AUTH_LOCAL_PRIVATE_KEY_FILE", "AUTH_LOCAL_KEY_ID"} {
		unsetEnv(t, key)
	}

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("APP_ENV=test\nPORT=9090\nDATABASE_URL=postgres://example\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.AppEnv != "test" || cfg.Port != "9090" || cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("LoadFrom() = %#v", cfg)
	}
	if cfg.AppVersion != defaultAppVersion || cfg.APIBaseURL != defaultAPIBaseURL || cfg.InvitationClaimBaseURL != defaultInvitationClaimBaseURL {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
}

func TestEnvironmentOverridesDotEnv(t *testing.T) {
	unsetLocalAuthKeyFileEnv(t)
	t.Setenv("PORT", "7070")
	t.Setenv("DATABASE_URL", "postgres://environment")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("PORT=9090\nDATABASE_URL=postgres://file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Port != "7070" || cfg.DatabaseURL != "postgres://environment" {
		t.Fatalf("environment did not override dotenv: %#v", cfg)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	unsetLocalAuthKeyFileEnv(t)
	unsetEnv(t, "DATABASE_URL")

	_, err := LoadFrom(filepath.Join(t.TempDir(), "missing.env"))
	if err == nil || err.Error() != "configuration error: DATABASE_URL is required" {
		t.Fatalf("LoadFrom() error = %v", err)
	}
}

func TestLoadReadsLocalAuthKeysFromFiles(t *testing.T) {
	for _, key := range []string{"AUTH_LOCAL_PUBLIC_KEY", "AUTH_LOCAL_PRIVATE_KEY", "AUTH_LOCAL_PUBLIC_KEY_FILE", "AUTH_LOCAL_PRIVATE_KEY_FILE"} {
		unsetEnv(t, key)
	}

	dir := t.TempDir()
	publicKeyPath := writeKeyFile(t, dir, "local_auth_public.pem", testPublicKeyPEM)
	privateKeyPath := writeKeyFile(t, dir, "local_auth_private.pem", testPrivateKeyPEM)

	path := filepath.Join(dir, ".env")
	contents := "DATABASE_URL=postgres://example\n" +
		"AUTH_LOCAL_PUBLIC_KEY_FILE=" + publicKeyPath + "\n" +
		"AUTH_LOCAL_PRIVATE_KEY_FILE=" + privateKeyPath + "\n"
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.AuthLocalPublicKey != testPublicKeyPEM {
		t.Fatalf("AuthLocalPublicKey = %q, want %q", cfg.AuthLocalPublicKey, testPublicKeyPEM)
	}
	if cfg.AuthLocalPrivateKey != testPrivateKeyPEM {
		t.Fatalf("AuthLocalPrivateKey = %q, want %q", cfg.AuthLocalPrivateKey, testPrivateKeyPEM)
	}
}

func TestLoadFallsBackToInlineLocalAuthKeys(t *testing.T) {
	for _, key := range []string{"AUTH_LOCAL_PUBLIC_KEY_FILE", "AUTH_LOCAL_PRIVATE_KEY_FILE"} {
		unsetEnv(t, key)
	}
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("AUTH_LOCAL_PUBLIC_KEY", testPublicKeyPEM)
	t.Setenv("AUTH_LOCAL_PRIVATE_KEY", testPrivateKeyPEM)

	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.AuthLocalPublicKey != testPublicKeyPEM {
		t.Fatalf("AuthLocalPublicKey = %q, want %q", cfg.AuthLocalPublicKey, testPublicKeyPEM)
	}
	if cfg.AuthLocalPrivateKey != testPrivateKeyPEM {
		t.Fatalf("AuthLocalPrivateKey = %q, want %q", cfg.AuthLocalPrivateKey, testPrivateKeyPEM)
	}
}

func TestLoadPrefersLocalAuthKeyFileOverInlineKey(t *testing.T) {
	dir := t.TempDir()
	publicKeyPath := writeKeyFile(t, dir, "local_auth_public.pem", testPublicKeyPEM)
	privateKeyPath := writeKeyFile(t, dir, "local_auth_private.pem", testPrivateKeyPEM)

	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("AUTH_LOCAL_PUBLIC_KEY", "-----BEGIN PUBLIC KEY-----\naW5saW5l\n-----END PUBLIC KEY-----")
	t.Setenv("AUTH_LOCAL_PRIVATE_KEY", "-----BEGIN EC PRIVATE KEY-----\naW5saW5l\n-----END EC PRIVATE KEY-----")
	t.Setenv("AUTH_LOCAL_PUBLIC_KEY_FILE", publicKeyPath)
	t.Setenv("AUTH_LOCAL_PRIVATE_KEY_FILE", privateKeyPath)

	cfg, err := LoadFrom(filepath.Join(dir, "missing.env"))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.AuthLocalPublicKey != testPublicKeyPEM {
		t.Fatalf("AuthLocalPublicKey = %q, want %q", cfg.AuthLocalPublicKey, testPublicKeyPEM)
	}
	if cfg.AuthLocalPrivateKey != testPrivateKeyPEM {
		t.Fatalf("AuthLocalPrivateKey = %q, want %q", cfg.AuthLocalPrivateKey, testPrivateKeyPEM)
	}
}

func TestLoadRejectsUnreadableLocalAuthKeyFile(t *testing.T) {
	unsetEnv(t, "AUTH_LOCAL_PUBLIC_KEY_FILE")
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.pem")

	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("AUTH_LOCAL_PRIVATE_KEY", testPrivateKeyPEM)
	t.Setenv("AUTH_LOCAL_PRIVATE_KEY_FILE", missing)

	_, err := LoadFrom(filepath.Join(dir, "missing.env"))
	if err == nil {
		t.Fatal("LoadFrom() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "AUTH_LOCAL_PRIVATE_KEY_FILE") {
		t.Fatalf("LoadFrom() error = %v, want it to name AUTH_LOCAL_PRIVATE_KEY_FILE", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("LoadFrom() error = %v, want it to name %q", err, missing)
	}
}

// unsetLocalAuthKeyFileEnv keeps a test hermetic: a *_FILE variable inherited
// from the developer's environment is read before Validate runs, so a path that
// does not resolve from the package directory would fail an unrelated test.
func unsetLocalAuthKeyFileEnv(t *testing.T) {
	t.Helper()
	unsetEnv(t, "AUTH_LOCAL_PUBLIC_KEY_FILE")
	unsetEnv(t, "AUTH_LOCAL_PRIVATE_KEY_FILE")
}

func writeKeyFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestConfigValidateRejectsEmptyPort(t *testing.T) {
	err := (Config{DatabaseURL: "postgres://example"}).Validate()
	if err == nil || err.Error() != "configuration error: PORT must not be empty" {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadReadsTrustedProxyCIDRs(t *testing.T) {
	unsetLocalAuthKeyFileEnv(t)
	unsetEnv(t, "TRUSTED_PROXY_CIDRS")
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DATABASE_URL=postgres://example\nTRUSTED_PROXY_CIDRS=10.0.0.0/8, 192.0.2.10/32\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if got, want := cfg.TrustedProxyCIDRs, []string{"10.0.0.0/8", "192.0.2.10/32"}; !equalStrings(got, want) {
		t.Fatalf("TrustedProxyCIDRs = %#v, want %#v", got, want)
	}
}

func TestConfigValidateRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	err := (Config{DatabaseURL: "postgres://example", Port: "8080", TrustedProxyCIDRs: []string{"not-a-cidr"}}).Validate()
	if err == nil || err.Error() != `configuration error: TRUSTED_PROXY_CIDRS contains invalid CIDR "not-a-cidr"` {
		t.Fatalf("Validate() error = %v", err)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
