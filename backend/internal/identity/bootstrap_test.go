package identity

import (
	"net/url"
	"testing"
)

func TestAddTokenToURLReplacesTokenAndPreservesQuery(t *testing.T) {
	claimURL, err := addTokenToURL("https://planner.example/claim?next=%2Fwelcome&token=old", "new-token")
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(claimURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("token") != "new-token" {
		t.Fatalf("token query = %q, want new-token", parsed.Query().Get("token"))
	}
	if parsed.Query().Get("next") != "/welcome" {
		t.Fatalf("next query = %q, want /welcome", parsed.Query().Get("next"))
	}
}

func TestAddTokenToURLRejectsRelativeURL(t *testing.T) {
	if _, err := addTokenToURL("/claim", "token"); err == nil {
		t.Fatal("relative claim URL unexpectedly succeeded")
	}
}

// THE CLAIM URL CONTRACT, backend half. The token is the `token` query parameter
// and the path is whatever INVITATION_CLAIM_BASE_URL supplied, unchanged. The
// frontend half is pinned by the "invitation claim route" suite in
// frontend/src/App.test.tsx, which renders the router at this exact shape.
//
// A shared literal asserted from both sides is the best available check here:
// nothing generates this URL, so neither side can be derived from the other. The
// two halves therefore reference each other by name, and moving one without the
// other fails the opposite test rather than only production.
//
// This is the regression test for the defect where the builder emitted
// /claim?token=… and the router matched /claim/:token, so every administrator
// invitation link in every environment resolved to a not-found page.
func TestClaimURLShapeMatchesTheFrontendRoute(t *testing.T) {
	const base = "http://localhost:5173/claim"

	claimURL, err := addTokenToURL(base, "invitation-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if claimURL != base+"?token=invitation-token-value" {
		t.Fatalf("claim URL = %q, want %q", claimURL, base+"?token=invitation-token-value")
	}

	parsed, err := url.Parse(claimURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/claim" {
		t.Fatalf("claim path = %q, want /claim; the token must not become a path segment", parsed.Path)
	}
	if parsed.Query().Get("token") != "invitation-token-value" {
		t.Fatalf("token query = %q, want invitation-token-value", parsed.Query().Get("token"))
	}
}
