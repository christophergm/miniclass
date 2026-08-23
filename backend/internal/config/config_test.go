package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDotEnv(t *testing.T) {
	for _, key := range []string{"APP_ENV", "APP_VERSION", "PORT", "API_BASE_URL", "DATABASE_URL", "TEST_DATABASE_URL"} {
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
	if cfg.AppVersion != defaultAppVersion || cfg.APIBaseURL != defaultAPIBaseURL {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
}

func TestEnvironmentOverridesDotEnv(t *testing.T) {
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
	unsetEnv(t, "DATABASE_URL")

	_, err := LoadFrom(filepath.Join(t.TempDir(), "missing.env"))
	if err == nil || err.Error() != "configuration error: DATABASE_URL is required" {
		t.Fatalf("LoadFrom() error = %v", err)
	}
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
