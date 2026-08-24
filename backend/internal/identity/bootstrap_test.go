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
