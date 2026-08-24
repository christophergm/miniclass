package identity

import (
	"encoding/base64"
	"testing"
)

func TestGenerateAccessTokenIs256BitAndHashable(t *testing.T) {
	first, err := GenerateAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	if first.Value == second.Value {
		t.Fatal("two generated access tokens were identical")
	}
	raw, err := base64.RawURLEncoding.DecodeString(first.Value)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != AccessTokenBytes {
		t.Fatalf("token entropy = %d bytes, want %d", len(raw), AccessTokenBytes)
	}
	if len(first.Hash) != 32 {
		t.Fatalf("hash length = %d, want 32", len(first.Hash))
	}
	hash, err := HashAccessToken(first.Value)
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) != string(first.Hash) {
		t.Fatal("hashing the bearer value did not reproduce its stored digest")
	}
}

func TestHashAccessTokenRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"", "short", "!!!!", base64.RawURLEncoding.EncodeToString(make([]byte, 31))} {
		if _, err := HashAccessToken(value); err == nil {
			t.Fatalf("HashAccessToken(%q) unexpectedly succeeded", value)
		}
	}
}
