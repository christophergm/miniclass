// Package config loads the API configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv                 = "development"
	defaultAppVersion             = "0.1.0"
	defaultPort                   = "8080"
	defaultAPIBaseURL             = "http://localhost:8080"
	defaultInvitationClaimBaseURL = "http://localhost:5173/claim"
	defaultAuthIssuer             = "http://localhost:8080"
	defaultAuthAudience           = "authenticated"
	defaultAuthProvider           = "local"
)

// Config contains the settings used by the API and its dependencies.
// Values are intentionally kept as strings because they originate in the
// environment and are passed to the HTTP and database clients unchanged.
type Config struct {
	AppEnv                 string
	AppVersion             string
	Port                   string
	APIBaseURL             string
	InvitationClaimBaseURL string
	TrustedProxyCIDRs      []string
	AppDatabaseURL         string
	TestDatabaseURL        string
	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseJWTSecret      string
	AuthProvider           string
	AuthIssuer             string
	AuthAudience           string
	AuthLocalPublicKey     string
	AuthLocalPrivateKey    string
	AuthLocalKeyID         string
}

// Load reads .env when it exists, then builds and validates the application
// configuration. Existing environment variables take precedence over .env.
func Load() (*Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}

	return fromEnvironment()
}

// LoadFrom reads configuration from the supplied dotenv file. It is useful
// for commands that run from a directory other than the repository root and
// for callers that need to select a specific environment file.
func LoadFrom(path string) (*Config, error) {
	if err := loadDotEnv(path); err != nil {
		return nil, err
	}

	return fromEnvironment()
}

func loadDotEnv(path string) error {
	err := godotenv.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func fromEnvironment() (*Config, error) {
	port := getEnv("PORT", defaultPort)

	supabaseURL := strings.TrimSpace(os.Getenv("SUPABASE_URL"))
	authIssuer := getEnv("AUTH_ISSUER", defaultAuthIssuer)
	if supabaseURL != "" && strings.TrimSpace(os.Getenv("AUTH_ISSUER")) == "" {
		authIssuer = supabaseURL
	}

	authLocalPublicKey, err := keyMaterial("AUTH_LOCAL_PUBLIC_KEY_FILE", "AUTH_LOCAL_PUBLIC_KEY")
	if err != nil {
		return nil, err
	}
	authLocalPrivateKey, err := keyMaterial("AUTH_LOCAL_PRIVATE_KEY_FILE", "AUTH_LOCAL_PRIVATE_KEY")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", defaultAppEnv),
		AppVersion:             getEnv("APP_VERSION", defaultAppVersion),
		Port:                   port,
		APIBaseURL:             getEnv("API_BASE_URL", defaultAPIBaseURL),
		InvitationClaimBaseURL: getEnv("INVITATION_CLAIM_BASE_URL", defaultInvitationClaimBaseURL),
		TrustedProxyCIDRs:      getListEnv("TRUSTED_PROXY_CIDRS"),
		AppDatabaseURL:         strings.TrimSpace(os.Getenv("APP_DATABASE_URL")),
		TestDatabaseURL:        strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")),
		SupabaseURL:            supabaseURL,
		SupabaseAnonKey:        strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		SupabaseJWTSecret:      strings.TrimSpace(os.Getenv("SUPABASE_JWT_SECRET")),
		AuthProvider:           getEnv("AUTH_PROVIDER", defaultAuthProvider),
		AuthIssuer:             authIssuer,
		AuthAudience:           getEnv("AUTH_AUDIENCE", defaultAuthAudience),
		AuthLocalPublicKey:     authLocalPublicKey,
		AuthLocalPrivateKey:    authLocalPrivateKey,
		AuthLocalKeyID:         strings.TrimSpace(os.Getenv("AUTH_LOCAL_KEY_ID")),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// keyMaterial returns the PEM text for a local auth signing key, preferring the
// file named by fileKey and falling back to the inline value in inlineKey.
//
// Key material lives in a file because a PEM header contains spaces, which no
// .env value may; ADR 0011 records why. A relative path resolves against the
// process working directory, and no searching is attempted, so the absolute path
// is reported on failure: a path that resolved against an unexpected directory
// then says so rather than looking like a missing key.
func keyMaterial(fileKey, inlineKey string) (string, error) {
	path := strings.TrimSpace(os.Getenv(fileKey))
	if path == "" {
		return strings.TrimSpace(os.Getenv(inlineKey)), nil
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	pem, err := os.ReadFile(absolute)
	if err != nil {
		return "", fmt.Errorf("read %s %q: %w", fileKey, absolute, err)
	}
	return strings.TrimSpace(string(pem)), nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func getListEnv(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	values := strings.Split(value, ",")
	result := make([]string, 0, len(values))
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// Validate checks the configuration required to start the API.
func (c Config) Validate() error {
	if strings.TrimSpace(c.AppDatabaseURL) == "" {
		return fmt.Errorf("configuration error: APP_DATABASE_URL is required")
	}
	if strings.TrimSpace(c.Port) == "" {
		return fmt.Errorf("configuration error: PORT must not be empty")
	}
	provider := strings.ToLower(strings.TrimSpace(c.AuthProvider))
	if provider != "" && provider != "local" && provider != "supabase" {
		return fmt.Errorf("configuration error: AUTH_PROVIDER must be local or supabase")
	}
	if strings.TrimSpace(c.AuthIssuer) != "" && strings.TrimSpace(c.AuthAudience) == "" {
		return fmt.Errorf("configuration error: AUTH_AUDIENCE must not be empty")
	}
	if strings.TrimSpace(c.AuthIssuer) == "" && strings.TrimSpace(c.AuthAudience) != "" {
		return fmt.Errorf("configuration error: AUTH_ISSUER must not be empty")
	}
	for _, cidr := range c.TrustedProxyCIDRs {
		if _, err := netip.ParsePrefix(cidr); err != nil {
			return fmt.Errorf("configuration error: TRUSTED_PROXY_CIDRS contains invalid CIDR %q", cidr)
		}
	}
	return nil
}
