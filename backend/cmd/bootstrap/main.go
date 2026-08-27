// Command bootstrap creates the first organization and Owner invitation.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/config"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
)

const defaultClaimBaseURL = "http://localhost:5173/claim"

// main bootstraps application data through miniclass_app. Schema changes are
// the separate responsibility of cmd/migrate and DATABASE_URL.
func main() {
	if err := run(context.Background(), os.Args[1:], os.Getenv("APP_DATABASE_URL"), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, databaseURL string, output io.Writer) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("bootstrap failed: APP_DATABASE_URL is required")
	}
	if output == nil {
		return errors.New("bootstrap failed: output is nil")
	}

	flags := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	organizationName := flags.String("organization-name", "", "organization name")
	ownerEmail := flags.String("owner-email", "", "first Owner email address")
	homeroomLabel := flags.String("homeroom-label", "homeroom", "label used for the organization's homeroom")
	claimBaseURL := flags.String("claim-base-url", defaultClaimBaseURL, "absolute URL to the invitation claim page")
	invitationTTL := flags.Duration("invitation-ttl", 48*time.Hour, "invitation lifetime")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	if strings.TrimSpace(*organizationName) == "" {
		return errors.New("bootstrap failed: -organization-name is required")
	}
	if strings.TrimSpace(*ownerEmail) == "" {
		return errors.New("bootstrap failed: -owner-email is required")
	}

	// Validate the application configuration shape without creating an API
	// server or acquiring any special database role.
	cfg := config.Config{AppDatabaseURL: databaseURL, Port: "8080"}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	database, err := data.NewApplicationFromURL(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	defer database.Close()
	store := identity.NewStore(database)

	result, err := identity.Bootstrap(ctx, store, identity.BootstrapInput{
		OrganizationName: *organizationName,
		HomeroomLabel:    *homeroomLabel,
		OwnerEmail:       *ownerEmail,
		ClaimBaseURL:     *claimBaseURL,
		InvitationTTL:    *invitationTTL,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(output, "Organization: %s (%s)\n", result.Organization.Name, result.Organization.ID)
	_, _ = fmt.Fprintf(output, "Owner invitation claim URL: %s\n", result.ClaimURL)
	_, _ = fmt.Fprintf(output, "Expires: %s\n", result.Token.ExpiresAt.UTC().Format(time.RFC3339))
	return nil
}
