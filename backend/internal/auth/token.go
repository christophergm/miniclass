package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TokenInput contains the claims minted by cmd/devtoken. It is intentionally
// explicit so local tokens exercise the same subject and verified-email path
// as provider tokens.
type TokenInput struct {
	Subject       string
	Email         string
	EmailVerified bool
	Issuer        string
	Audience      string
	KeyID         string
	Now           time.Time
	ExpiresAt     time.Time
	NotBefore     time.Time
	Lifetime      time.Duration
}

// MintLocalToken signs an ES256 or RS256 local-development token.
func MintLocalToken(input TokenInput, privateKey any) (string, error) {
	if strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.Issuer) == "" || strings.TrimSpace(input.Audience) == "" {
		return "", errors.New("mint token: subject, issuer, and audience are required")
	}
	if strings.TrimSpace(input.Email) == "" {
		return "", errors.New("mint token: email is required")
	}
	method, err := signingMethod(privateKey)
	if err != nil {
		return "", err
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if input.Lifetime == 0 {
		input.Lifetime = 5 * time.Minute
	}
	expiresAt := input.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = now.Add(input.Lifetime)
	}
	notBefore := input.NotBefore
	if notBefore.IsZero() {
		notBefore = now
	}
	claims := jwt.MapClaims{
		"sub":            input.Subject,
		"email":          strings.ToLower(strings.TrimSpace(input.Email)),
		"email_verified": input.EmailVerified,
		"iss":            strings.TrimRight(strings.TrimSpace(input.Issuer), "/"),
		"aud":            input.Audience,
		"exp":            expiresAt.Unix(),
		"nbf":            notBefore.Unix(),
		"iat":            now.Unix(),
	}
	token := jwt.NewWithClaims(method, claims)
	if strings.TrimSpace(input.KeyID) != "" {
		token.Header["kid"] = strings.TrimSpace(input.KeyID)
	}
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("mint token: sign: %w", err)
	}
	return signed, nil
}

func signingMethod(key any) (jwt.SigningMethod, error) {
	switch value := key.(type) {
	case *ecdsa.PrivateKey:
		if value == nil || value.Curve != elliptic.P256() {
			return nil, errors.New("mint token: ES256 requires an ECDSA P-256 private key")
		}
		return jwt.SigningMethodES256, nil
	case *rsa.PrivateKey:
		if value == nil || value.N == nil || value.N.BitLen() < 2048 {
			return nil, errors.New("mint token: RS256 requires an RSA private key of at least 2048 bits")
		}
		return jwt.SigningMethodRS256, nil
	default:
		return nil, fmt.Errorf("mint token: key type %T is not ES256 or RS256", key)
	}
}

// ParsePrivateKeyPEM parses PKCS#8, SEC1, and PKCS#1 PEM private keys.
func ParsePrivateKeyPEM(value string) (any, error) {
	block, _ := pem.Decode([]byte(strings.ReplaceAll(value, `\n`, "\n")))
	if block == nil {
		return nil, errors.New("parse private key: PEM block is missing")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if _, err := signingMethod(key); err != nil {
			return nil, err
		}
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		if _, err := signingMethod(key); err != nil {
			return nil, err
		}
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		if _, err := signingMethod(key); err != nil {
			return nil, err
		}
		return key, nil
	}
	return nil, errors.New("parse private key: unsupported key encoding")
}
