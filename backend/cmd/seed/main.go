// Command seed creates a fresh synthetic organization and roster in PostgreSQL.
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

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/seed"
)

const defaultClaimBaseURL = "http://localhost:5173/claim"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Getenv("DATABASE_URL"), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, databaseURL string, output io.Writer) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("seed failed: DATABASE_URL is required")
	}
	if output == nil {
		return errors.New("seed failed: output is nil")
	}
	flags := flag.NewFlagSet("seed", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	defaults := seed.DefaultOptions()
	organizationName := flags.String("organization-name", defaults.OrganizationName, "fresh organization name")
	ownerEmail := flags.String("owner-email", defaults.OwnerEmail, "Owner invitation email")
	ownerSubject := flags.String("owner-subject", defaults.OwnerSubject, "verified provider subject to bind to the Owner invitation; empty leaves the invitation to be claimed by hand")
	homeroomLabel := flags.String("homeroom-label", defaults.HomeroomLabel, "organization homeroom label")
	claimBaseURL := flags.String("claim-base-url", defaultClaimBaseURL, "absolute invitation claim page URL")
	invitationTTL := flags.Duration("invitation-ttl", 48*time.Hour, "Owner invitation lifetime")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}
	database, err := data.NewFromURL(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("seed failed: connect database: %w", err)
	}
	defer database.Close()
	result, err := seed.Load(ctx, database, seed.Options{
		OrganizationName: *organizationName, OwnerEmail: *ownerEmail, OwnerSubject: *ownerSubject,
		HomeroomLabel: *homeroomLabel, ClaimBaseURL: *claimBaseURL, InvitationTTL: *invitationTTL,
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(output, "Organization: %s (%s)\n", result.OrganizationName, result.OrganizationID)
	_, _ = fmt.Fprintf(output, "School year: %s\n", result.SchoolYearID)
	_, _ = fmt.Fprintf(output, "Roster: %d students, %d adults, %d households\n", result.Students, result.Adults, result.Households)
	if bound := result.BoundOwner; bound != nil {
		_, _ = fmt.Fprintf(output, "Bound provider subject: %s\n", bound.ProviderSubject)
		_, _ = fmt.Fprintf(output, "Bound email: %s\n", bound.Email)
		_, _ = fmt.Fprintf(output, "Bound organization: %s (%s)\n", bound.OrganizationName, bound.OrganizationID)
		_, _ = fmt.Fprintf(output, "Bound role: %s\n", bound.Role)
		// Deliberately no claim URL. The binding above consumed the invitation, so
		// the URL is a dead link, and now that claim links resolve again someone
		// would click it and be told the invitation is invalid.
		_, _ = fmt.Fprintln(output, "Owner invitation: consumed by the binding above; it cannot be claimed again")
		return nil
	}
	_, _ = fmt.Fprintf(output, "Owner invitation claim URL: %s\n", result.ClaimURL)
	_, _ = fmt.Fprintf(output, "Expires: %s\n", result.ExpiresAt.UTC().Format(time.RFC3339))
	return nil
}
