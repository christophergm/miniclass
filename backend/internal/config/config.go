// Package config loads the API configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv       = "development"
	defaultAppVersion   = "0.1.0"
	defaultPort         = "8080"
	defaultAPIBaseURL   = "http://localhost:8080"
	defaultAuthIssuer   = "http://localhost:8080"
	defaultAuthAudience = "authenticated"
	defaultAuthProvider = "local"
)

// Config contains the settings used by the API and its dependencies.
// Values are intentionally kept as strings because they originate in the
// environment and are passed to the HTTP and database clients unchanged.
type Config struct {
	AppEnv              string
	AppVersion          string
	Port                string
	APIBaseURL          string
	TrustedProxyCIDRs   []string
	DatabaseURL         string
	TestDatabaseURL     string
	SupabaseURL         string
	SupabaseAnonKey     string
	SupabaseJWTSecret   string
	AuthProvider        string
	AuthIssuer          string
	AuthAudience        string
	AuthLocalPublicKey  string
	AuthLocalPrivateKey string
	AuthLocalKeyID      string
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
	cfg := &Config{
		AppEnv:              getEnv("APP_ENV", defaultAppEnv),
		AppVersion:          getEnv("APP_VERSION", defaultAppVersion),
		Port:                port,
		APIBaseURL:          getEnv("API_BASE_URL", defaultAPIBaseURL),
		TrustedProxyCIDRs:   getListEnv("TRUSTED_PROXY_CIDRS"),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		TestDatabaseURL:     strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")),
		SupabaseURL:         supabaseURL,
		SupabaseAnonKey:     strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		SupabaseJWTSecret:   strings.TrimSpace(os.Getenv("SUPABASE_JWT_SECRET")),
		AuthProvider:        getEnv("AUTH_PROVIDER", defaultAuthProvider),
		AuthIssuer:          authIssuer,
		AuthAudience:        getEnv("AUTH_AUDIENCE", defaultAuthAudience),
		AuthLocalPublicKey:  strings.TrimSpace(os.Getenv("AUTH_LOCAL_PUBLIC_KEY")),
		AuthLocalPrivateKey: strings.TrimSpace(os.Getenv("AUTH_LOCAL_PRIVATE_KEY")),
		AuthLocalKeyID:      strings.TrimSpace(os.Getenv("AUTH_LOCAL_KEY_ID")),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("configuration error: DATABASE_URL is required")
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
