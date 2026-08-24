// Package identity contains identity-domain primitives that sit above the
// unscoped data accessor.
package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// AccessTokenBytes is the required 256-bit bearer-token size.
	AccessTokenBytes = 32

	PurposeAdminInvitation   = "admin_invitation"
	PurposeHousehold         = "household_submission"
	PurposeClassLeader       = "class_leader"
	PurposeHomeroomTeacher   = "homeroom_teacher"
	PurposePublishedArtifact = "published_artifact"
)

// Token is the one-time value returned to a caller that must distribute a
// link. Hash is the only part that belongs in the database.
type Token struct {
	Value string
	Hash  []byte
}

// GenerateAccessToken creates a URL-safe, unpadded bearer token from 256 bits
// of operating-system randomness.
func GenerateAccessToken() (Token, error) {
	raw := make([]byte, AccessTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, fmt.Errorf("generate access token: %w", err)
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	return Token{Value: value, Hash: hashBytes(raw)}, nil
}

// NewAccessToken is an alias with a concise name for callers that already
// operate in the identity package.
func NewAccessToken() (Token, error) {
	return GenerateAccessToken()
}

// HashAccessToken validates the encoded bearer value and returns its SHA-256
// digest. It does not hash arbitrary strings: malformed or non-256-bit values
// must never become valid database lookups.
func HashAccessToken(value string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("hash access token: value is not valid base64url")
	}
	if len(raw) != AccessTokenBytes {
		return nil, fmt.Errorf("hash access token: value decodes to %d bytes, want %d", len(raw), AccessTokenBytes)
	}
	return hashBytes(raw), nil
}

func hashBytes(raw []byte) []byte {
	digest := sha256.Sum256(raw)
	return append([]byte(nil), digest[:]...)
}
