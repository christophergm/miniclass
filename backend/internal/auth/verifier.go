// Package auth contains authentication, principal and capability primitives
// shared by the HTTP API and local development tools.
package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// ClockSkew is the permitted clock difference between the API and the
	// identity provider when checking temporal JWT claims.
	ClockSkew = 30 * time.Second
	algES256  = "ES256"
	algRS256  = "RS256"
)

var (
	ErrInvalidToken       = errors.New("invalid access token")
	ErrUnsupportedAlg     = errors.New("unsupported JWT signing algorithm")
	ErrLocalProduction    = errors.New("local token verification is not allowed in production")
	ErrMissingVerifierKey = errors.New("token verifier key is not configured")
)

// Verifier verifies a bearer token and returns only claims that have passed
// signature and registered-claim validation.
type Verifier interface {
	Verify(context.Context, string) (VerifiedIdentity, error)
}

// VerifiedIdentity is the identity-provider assertion carried by a verified
// access token. Authorization data is deliberately not read from the token.
type VerifiedIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Issuer        string
	Claims        map[string]any
}

type verifierOptions struct {
	Issuer   string
	Audience string
	Now      func() time.Time
	Skew     time.Duration
}

func (o verifierOptions) normalized() (verifierOptions, error) {
	o.Issuer = strings.TrimRight(strings.TrimSpace(o.Issuer), "/")
	o.Audience = strings.TrimSpace(o.Audience)
	if o.Issuer == "" {
		return o, errors.New("token verifier issuer is required")
	}
	if o.Audience == "" {
		return o, errors.New("token verifier audience is required")
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Skew == 0 {
		o.Skew = ClockSkew
	}
	if o.Skew < 0 {
		return o, errors.New("token verifier clock skew must not be negative")
	}
	return o, nil
}

// LocalVerifier validates tokens against one statically configured public key.
// It is intended for development and tests; production construction is
// rejected by NewLocalVerifierForEnvironment.
type LocalVerifier struct {
	options verifierOptions
	key     any
}

// LocalVerifierOptions configures a local verifier.
type LocalVerifierOptions struct {
	Issuer    string
	Audience  string
	PublicKey any
	Now       func() time.Time
	Skew      time.Duration
}

// NewLocalVerifier constructs a verifier from an asymmetric public key.
func NewLocalVerifier(options LocalVerifierOptions) (*LocalVerifier, error) {
	base, err := (verifierOptions{
		Issuer:   options.Issuer,
		Audience: options.Audience,
		Now:      options.Now,
		Skew:     options.Skew,
	}).normalized()
	if err != nil {
		return nil, err
	}
	if err := validateVerificationKey(options.PublicKey); err != nil {
		return nil, err
	}
	return &LocalVerifier{options: base, key: options.PublicKey}, nil
}

// NewLocalVerifierForEnvironment applies the production safety rule before
// constructing a local verifier.
func NewLocalVerifierForEnvironment(environment string, options LocalVerifierOptions) (*LocalVerifier, error) {
	if strings.EqualFold(strings.TrimSpace(environment), "production") {
		return nil, ErrLocalProduction
	}
	return NewLocalVerifier(options)
}

// NewLocalVerifierFromPEM parses a PKIX, PKCS#1, or SEC1 public key.
func NewLocalVerifierFromPEM(environment string, options LocalPEMOptions) (*LocalVerifier, error) {
	key, err := ParsePublicKeyPEM(options.PublicKeyPEM)
	if err != nil {
		return nil, err
	}
	return NewLocalVerifierForEnvironment(environment, LocalVerifierOptions{
		Issuer:    options.Issuer,
		Audience:  options.Audience,
		PublicKey: key,
		Now:       options.Now,
		Skew:      options.Skew,
	})
}

// LocalPEMOptions is the PEM-backed form used by API configuration.
type LocalPEMOptions struct {
	Issuer       string
	Audience     string
	PublicKeyPEM string
	Now          func() time.Time
	Skew         time.Duration
}

// Verify verifies a locally signed bearer token.
func (v *LocalVerifier) Verify(_ context.Context, bearer string) (VerifiedIdentity, error) {
	if v == nil {
		return VerifiedIdentity{}, ErrMissingVerifierKey
	}
	return verifyJWT(bearer, v.options, func(_ *jwt.Token) (any, error) {
		return v.key, nil
	})
}

// SupabaseVerifier validates JWTs against a cached Supabase JWKS document.
// Unknown key refreshes are rate-limited so attacker-controlled kids cannot
// turn verification into an outbound request primitive.
type SupabaseVerifier struct {
	options       verifierOptions
	issuer        string
	client        *http.Client
	refreshEvery  time.Duration
	mu            sync.Mutex
	keys          map[string]any
	lastRefreshAt time.Time
}

// SupabaseVerifierOptions configures a Supabase JWKS verifier.
type SupabaseVerifierOptions struct {
	Issuer       string
	Audience     string
	Client       *http.Client
	Now          func() time.Time
	Skew         time.Duration
	RefreshEvery time.Duration
}

// NewSupabaseVerifier creates a cached JWKS verifier. The first verification
// performs the initial fetch lazily, keeping construction side-effect free.
func NewSupabaseVerifier(options SupabaseVerifierOptions) (*SupabaseVerifier, error) {
	base, err := (verifierOptions{
		Issuer:   options.Issuer,
		Audience: options.Audience,
		Now:      options.Now,
		Skew:     options.Skew,
	}).normalized()
	if err != nil {
		return nil, err
	}
	refreshEvery := options.RefreshEvery
	if refreshEvery == 0 {
		refreshEvery = time.Minute
	}
	if refreshEvery < 0 {
		return nil, errors.New("JWKS refresh interval must not be negative")
	}
	client := options.Client
	if client == nil {
		client = http.DefaultClient
	}
	return &SupabaseVerifier{
		options:      base,
		issuer:       base.Issuer,
		client:       client,
		refreshEvery: refreshEvery,
		keys:         make(map[string]any),
	}, nil
}

// Verify verifies a Supabase bearer token using its kid-selected JWKS key.
func (v *SupabaseVerifier) Verify(ctx context.Context, bearer string) (VerifiedIdentity, error) {
	if v == nil {
		return VerifiedIdentity{}, ErrMissingVerifierKey
	}
	return verifyJWT(bearer, v.options, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if strings.TrimSpace(kid) == "" {
			return nil, errors.New("JWT kid is required")
		}
		return v.key(ctx, kid)
	})
}

func (v *SupabaseVerifier) key(ctx context.Context, kid string) (any, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	now := v.options.Now().UTC()
	if !v.lastRefreshAt.IsZero() && now.Sub(v.lastRefreshAt) < v.refreshEvery {
		return nil, fmt.Errorf("JWKS key %q is unavailable and refresh is rate-limited", kid)
	}
	if err := v.refreshLocked(ctx); err != nil {
		return nil, err
	}
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("JWKS key %q is not published", kid)
	}
	return key, nil
}

type jwksDocument struct {
	Keys []json.RawMessage `json:"keys"`
}

func (v *SupabaseVerifier) refreshLocked(ctx context.Context) error {
	// Record attempts, not only successful refreshes. A failing JWKS endpoint
	// must not become an attacker-controlled outbound request loop.
	v.lastRefreshAt = v.options.Now().UTC()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}
	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: unexpected status %d", response.StatusCode)
	}
	var document jwksDocument
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}
	keys := make(map[string]any, len(document.Keys))
	for _, raw := range document.Keys {
		key, keyID, err := parseJWK(raw)
		if err != nil {
			return err
		}
		keys[keyID] = key
	}
	if len(keys) == 0 {
		return errors.New("decode JWKS: no usable keys")
	}
	v.keys = keys
	return nil
}

func parseJWK(raw json.RawMessage) (any, string, error) {
	var jwk struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	if err := json.Unmarshal(raw, &jwk); err != nil {
		return nil, "", fmt.Errorf("decode JWK: %w", err)
	}
	if jwk.Kid == "" {
		return nil, "", errors.New("decode JWK: kid is required")
	}
	if jwk.Alg != "" && jwk.Alg != algES256 && jwk.Alg != algRS256 {
		return nil, "", fmt.Errorf("decode JWK %q: unsupported alg %q", jwk.Kid, jwk.Alg)
	}
	switch jwk.Kty {
	case "RSA":
		modulus, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			return nil, "", fmt.Errorf("decode JWK %q modulus: %w", jwk.Kid, err)
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil || len(exponentBytes) == 0 {
			return nil, "", fmt.Errorf("decode JWK %q exponent", jwk.Kid)
		}
		exponent := 0
		for _, value := range exponentBytes {
			exponent = exponent<<8 | int(value)
		}
		if exponent < 2 {
			return nil, "", fmt.Errorf("decode JWK %q exponent is invalid", jwk.Kid)
		}
		key := &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponent}
		if err := validateVerificationKey(key); err != nil {
			return nil, "", fmt.Errorf("decode JWK %q: %w", jwk.Kid, err)
		}
		return key, jwk.Kid, nil
	case "EC":
		if jwk.Crv != "P-256" {
			return nil, "", fmt.Errorf("decode JWK %q curve %q is unsupported", jwk.Kid, jwk.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return nil, "", fmt.Errorf("decode JWK %q x: %w", jwk.Kid, err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
		if err != nil {
			return nil, "", fmt.Errorf("decode JWK %q y: %w", jwk.Kid, err)
		}
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		if !elliptic.P256().IsOnCurve(x, y) {
			return nil, "", fmt.Errorf("decode JWK %q point is not on P-256", jwk.Kid)
		}
		return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, jwk.Kid, nil
	default:
		return nil, "", fmt.Errorf("decode JWK %q type %q is unsupported", jwk.Kid, jwk.Kty)
	}
}

func validateVerificationKey(key any) error {
	switch value := key.(type) {
	case *ecdsa.PublicKey:
		if value == nil || value.Curve != elliptic.P256() {
			return errors.New("local verifier requires an ECDSA P-256 public key")
		}
	case *rsa.PublicKey:
		if value == nil || value.N == nil || value.N.BitLen() < 2048 {
			return errors.New("local verifier requires an RSA public key of at least 2048 bits")
		}
	default:
		return fmt.Errorf("local verifier key type %T is not ES256 or RS256", key)
	}
	return nil
}

func verifyJWT(raw string, options verifierOptions, keyFunc jwt.Keyfunc) (VerifiedIdentity, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return VerifiedIdentity{}, ErrInvalidToken
	}
	claims := jwt.MapClaims{}
	parser := &jwt.Parser{ValidMethods: []string{algES256, algRS256}, UseJSONNumber: true, SkipClaimsValidation: true}
	unverified, _, parseErr := parser.ParseUnverified(raw, jwt.MapClaims{})
	if parseErr != nil || unverified == nil {
		return VerifiedIdentity{}, fmt.Errorf("%w: malformed JWT", ErrInvalidToken)
	}
	if unverified.Method.Alg() != algES256 && unverified.Method.Alg() != algRS256 {
		return VerifiedIdentity{}, ErrUnsupportedAlg
	}
	token, err := parser.ParseWithClaims(raw, claims, keyFunc)
	if err != nil || token == nil || !token.Valid {
		if errors.Is(err, ErrUnsupportedAlg) {
			return VerifiedIdentity{}, err
		}
		return VerifiedIdentity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if err := validateClaims(claims, options); err != nil {
		return VerifiedIdentity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	subject, _ := claims["sub"].(string)
	issuer, _ := claims["iss"].(string)
	email, _ := claims["email"].(string)
	emailVerified, _ := claims["email_verified"].(bool)
	if !emailVerified {
		if confirmedAt, ok := claims["email_confirmed_at"].(string); ok && strings.TrimSpace(confirmedAt) != "" {
			emailVerified = true
		}
	}
	return VerifiedIdentity{
		Subject:       strings.TrimSpace(subject),
		Email:         strings.ToLower(strings.TrimSpace(email)),
		EmailVerified: emailVerified,
		Issuer:        issuer,
		Claims:        claims,
	}, nil
}

func validateClaims(claims jwt.MapClaims, options verifierOptions) error {
	issuer, ok := claims["iss"].(string)
	if !ok || strings.TrimRight(issuer, "/") != options.Issuer {
		return errors.New("issuer claim is invalid")
	}
	if !claims.VerifyAudience(options.Audience, true) {
		return errors.New("audience claim is invalid")
	}
	subject, ok := claims["sub"].(string)
	if !ok || strings.TrimSpace(subject) == "" {
		return errors.New("subject claim is required")
	}
	expiresAt, ok, err := numericClaim(claims, "exp")
	if err != nil || !ok {
		return errors.New("exp claim is required and must be numeric")
	}
	now := options.Now().UTC()
	if now.After(time.Unix(int64(expiresAt), 0).Add(options.Skew)) {
		return errors.New("token is expired")
	}
	if notBefore, ok, err := numericClaim(claims, "nbf"); err != nil {
		return errors.New("nbf claim must be numeric")
	} else if ok && now.Before(time.Unix(int64(notBefore), 0).Add(-options.Skew)) {
		return errors.New("token is not valid yet")
	}
	return nil
}

func numericClaim(claims jwt.MapClaims, name string) (float64, bool, error) {
	value, ok := claims[name]
	if !ok {
		return 0, false, nil
	}
	switch value := value.(type) {
	case json.Number:
		result, err := value.Float64()
		return result, true, err
	case float64:
		return value, true, nil
	default:
		return 0, true, errors.New("not numeric")
	}
}

// ParsePublicKeyPEM parses common public-key encodings accepted by the local
// verifier.
func ParsePublicKeyPEM(value string) (any, error) {
	block, _ := pem.Decode([]byte(strings.ReplaceAll(value, `\n`, "\n")))
	if block == nil {
		return nil, errors.New("parse public key: PEM block is missing")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if err := validateVerificationKey(key); err != nil {
			return nil, err
		}
		return key, nil
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		if err := validateVerificationKey(key); err != nil {
			return nil, err
		}
		return key, nil
	}
	return nil, errors.New("parse public key: unsupported key encoding")
}
