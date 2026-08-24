// Command devtoken mints a locally signed bearer token for development and
// integration tests. It never contacts Supabase.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/auth"
)

func main() {
	var (
		subject       = flag.String("subject", "", "provider subject")
		email         = flag.String("email", "", "verified email")
		issuer        = flag.String("issuer", envOr("AUTH_ISSUER", "http://localhost:8080"), "JWT issuer")
		audience      = flag.String("audience", envOr("AUTH_AUDIENCE", "authenticated"), "JWT audience")
		keyPEM        = flag.String("private-key", os.Getenv("AUTH_LOCAL_PRIVATE_KEY"), "PEM private key")
		keyPath       = flag.String("private-key-file", os.Getenv("AUTH_LOCAL_PRIVATE_KEY_FILE"), "file containing PEM private key")
		keyID         = flag.String("key-id", envOr("AUTH_LOCAL_KEY_ID", "local"), "JWT key id")
		lifetime      = flag.Duration("lifetime", 5*time.Minute, "token lifetime")
		emailVerified = flag.Bool("email-verified", true, "include email_verified=true")
	)
	flag.Parse()

	if strings.TrimSpace(*keyPath) != "" {
		value, err := os.ReadFile(*keyPath)
		if err != nil {
			fatal(err)
		}
		*keyPEM = string(value)
	}
	if strings.TrimSpace(*keyPEM) == "" {
		fatal(errors.New("a private key is required via --private-key or --private-key-file"))
	}
	key, err := auth.ParsePrivateKeyPEM(*keyPEM)
	if err != nil {
		fatal(err)
	}
	token, err := auth.MintLocalToken(auth.TokenInput{
		Subject: *subject, Email: *email, EmailVerified: *emailVerified,
		Issuer: *issuer, Audience: *audience, KeyID: *keyID, Lifetime: *lifetime,
	}, key)
	if err != nil {
		fatal(err)
	}
	fmt.Println(token)
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "devtoken: %v\n", err)
	os.Exit(1)
}
