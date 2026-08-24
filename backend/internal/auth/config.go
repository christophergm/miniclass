package auth

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/config"
)

// NewFromConfig builds the configured verifier. The local provider accepts a
// public key directly or derives it from the configured private key, which is
// useful for a single-file development environment. Production still refuses
// the local provider before key parsing.
func NewFromConfig(cfg config.Config) (Verifier, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.AuthProvider))
	switch provider {
	case "local":
		if strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "production") {
			return nil, ErrLocalProduction
		}
		var publicKey any
		if strings.TrimSpace(cfg.AuthLocalPublicKey) != "" {
			key, err := ParsePublicKeyPEM(cfg.AuthLocalPublicKey)
			if err != nil {
				return nil, fmt.Errorf("configure local verifier: %w", err)
			}
			publicKey = key
		} else if strings.TrimSpace(cfg.AuthLocalPrivateKey) != "" {
			privateKey, err := ParsePrivateKeyPEM(cfg.AuthLocalPrivateKey)
			if err != nil {
				return nil, fmt.Errorf("configure local verifier: %w", err)
			}
			switch key := privateKey.(type) {
			case *ecdsa.PrivateKey:
				publicKey = &key.PublicKey
			case *rsa.PrivateKey:
				publicKey = &key.PublicKey
			default:
				return nil, errors.New("configure local verifier: private key is not asymmetric")
			}
		} else {
			return nil, ErrMissingVerifierKey
		}
		return NewLocalVerifierForEnvironment(cfg.AppEnv, LocalVerifierOptions{
			Issuer: cfg.AuthIssuer, Audience: cfg.AuthAudience, PublicKey: publicKey,
		})
	case "supabase":
		return NewSupabaseVerifier(SupabaseVerifierOptions{Issuer: cfg.AuthIssuer, Audience: cfg.AuthAudience})
	default:
		return nil, fmt.Errorf("configure verifier: unsupported provider %q", cfg.AuthProvider)
	}
}
