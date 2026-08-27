package seed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

const defaultInvitationTTL = 48 * time.Hour

// Options controls one fresh organization and its Owner invitation.
type Options struct {
	OrganizationName string
	HomeroomLabel    string
	OwnerEmail       string
	// OwnerSubject, when set, is the verified provider subject the seed binds
	// to the Owner invitation it just created, so a non-interactive caller
	// reaches a usable login without opening ClaimURL. The claim reuses
	// OwnerEmail from this same value, which is why no separate claim email
	// exists to disagree with the invited one.
	OwnerSubject  string
	ClaimBaseURL  string
	InvitationTTL time.Duration
}

// Result is the operator-facing outcome of a successful seed.
type Result struct {
	OrganizationID   string
	OrganizationName string
	SchoolYearID     string
	ClaimURL         string
	ExpiresAt        time.Time
	Students         int
	Adults           int
	Households       int
	// BoundOwner is nil unless Options.OwnerSubject was set.
	BoundOwner *BoundAccount
}

// BoundAccount describes the login the seed bound to the Owner invitation. It
// is read back from the claim rather than copied from Options, so what is
// reported is what a later token exchange will actually resolve to.
type BoundAccount struct {
	ProviderSubject  string
	Email            string
	UserID           string
	MembershipID     string
	OrganizationID   string
	OrganizationName string
	Role             string
}

// DefaultOptions provides a safe, synthetic local-development configuration.
func DefaultOptions() Options {
	return Options{
		OrganizationName: "Synthetic Academy",
		HomeroomLabel:    "homeroom",
		OwnerEmail:       "owner@example.test",
		ClaimBaseURL:     "http://localhost:5173/claim",
		InvitationTTL:    defaultInvitationTTL,
	}
}

// Load creates a new organization, its Owner invitation, and the complete
// synthetic corpus. It intentionally never reuses an existing organization.
// With Options.OwnerSubject set it also claims that invitation, so a
// non-interactive caller ends with a usable login rather than a URL to click.
func Load(ctx context.Context, database *data.DB, options Options) (Result, error) {
	if database == nil {
		return Result{}, errors.New("seed: database is nil")
	}
	defaults := DefaultOptions()
	if options.OrganizationName == "" {
		options.OrganizationName = defaults.OrganizationName
	}
	if options.HomeroomLabel == "" {
		options.HomeroomLabel = defaults.HomeroomLabel
	}
	if options.OwnerEmail == "" {
		options.OwnerEmail = defaults.OwnerEmail
	}
	if options.ClaimBaseURL == "" {
		options.ClaimBaseURL = defaults.ClaimBaseURL
	}
	if options.InvitationTTL == 0 {
		options.InvitationTTL = defaults.InvitationTTL
	}
	options.OwnerSubject = strings.TrimSpace(options.OwnerSubject)

	store := identity.NewStore(database)
	if options.OwnerSubject != "" {
		// Refuse before the organization is created: Load commits it on its own,
		// so a refusal afterwards would leave an orphan organization and a full
		// roster behind.
		if err := CheckOwnerSubjectAvailable(ctx, database, options.OwnerSubject); err != nil {
			return Result{}, err
		}
	}

	corpus := Generate()
	if err := corpus.Validate(); err != nil {
		return Result{}, fmt.Errorf("seed: validate corpus: %w", err)
	}
	bootstrap, err := identity.Bootstrap(ctx, store, identity.BootstrapInput{
		OrganizationName: options.OrganizationName,
		HomeroomLabel:    options.HomeroomLabel,
		OwnerEmail:       options.OwnerEmail,
		ClaimBaseURL:     options.ClaimBaseURL,
		InvitationTTL:    options.InvitationTTL,
	})
	if err != nil {
		return Result{}, fmt.Errorf("seed: bootstrap organization: %w", err)
	}

	// Claim before the corpus so a failure to bind does not leave a roster
	// behind either.
	var boundOwner *BoundAccount
	if options.OwnerSubject != "" {
		// The invited email and the claimed email are the same field of the same
		// Options value, so the pair cannot disagree; the claim still compares
		// them exactly as it does for a human. The seed is both inviter and
		// claimant, so the verified-email precondition holds by construction
		// rather than by trusting a caller.
		account, err := store.ClaimAdminInvitation(ctx, auth.InvitationClaimInput{
			Bearer:          bootstrap.TokenValue,
			ProviderSubject: options.OwnerSubject,
			Email:           options.OwnerEmail,
			EmailVerified:   true,
		})
		if err != nil {
			return Result{}, fmt.Errorf("seed: claim owner invitation for %q: %w", options.OwnerSubject, err)
		}
		boundOwner = &BoundAccount{
			ProviderSubject: account.User.ProviderSubject, Email: account.User.Email,
			UserID: string(account.User.ID), MembershipID: string(account.Membership.ID),
			OrganizationID:   string(account.Membership.OrganizationID),
			OrganizationName: account.Membership.OrganizationName, Role: account.Membership.Role,
		}
	}

	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "deterministic seed corpus"}
	factory := factories.New(database, string(bootstrap.Organization.ID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic School Year")
	if err != nil {
		return Result{}, fmt.Errorf("seed: create school year: %w", err)
	}

	gradeIDs := make([]string, len(corpus.Grades))
	for index, grade := range corpus.Grades {
		row, err := factory.CreateGradeLevel(ctx, "grade-"+grade, "Grade "+grade)
		if err != nil {
			return Result{}, fmt.Errorf("seed: create grade %s: %w", grade, err)
		}
		gradeIDs[index] = string(row.ID)
	}
	homeroomIDs := make([]string, len(corpus.Homerooms))
	for index, name := range corpus.Homerooms {
		row, err := factory.CreateHomeroom(ctx, name)
		if err != nil {
			return Result{}, fmt.Errorf("seed: create homeroom %s: %w", name, err)
		}
		homeroomIDs[index] = string(row.ID)
	}

	studentIDs := make([]string, len(corpus.Students))
	for index, spec := range corpus.Students {
		row, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{
			LegalGivenName: spec.LegalGivenName, LegalFamilyName: spec.LegalFamilyName,
			PreferredGivenName: spec.PreferredGivenName, GradeLevelID: mustXID(gradeIDs[spec.Grade-1]),
			HomeroomID: mustXID(homeroomIDs[spec.Homeroom]), ExternalIdentifier: spec.ExternalIdentifier,
		})
		if err != nil {
			return Result{}, fmt.Errorf("seed: create student %d: %w", index+1, err)
		}
		studentIDs[index] = string(row.ID)
	}

	adultIDs := make([]string, len(corpus.Adults))
	for index, spec := range corpus.Adults {
		row, err := factory.CreateAdult(ctx, year.ID, people.AdultCreateInput{
			LegalGivenName: spec.LegalGivenName, LegalFamilyName: spec.LegalFamilyName,
			PreferredGivenName: spec.PreferredGivenName, Email: spec.Email, Phone: spec.Phone,
			ExternalIdentifier: spec.ExternalIdentifier, ParticipationIntent: spec.ParticipationIntent,
		})
		if err != nil {
			return Result{}, fmt.Errorf("seed: create adult %d: %w", index+1, err)
		}
		adultIDs[index] = string(row.ID)
	}

	householdIDs := make([]string, len(corpus.Households))
	for index, spec := range corpus.Households {
		row, err := factory.CreateHousehold(ctx, year.ID, people.HouseholdCreateInput{DisplayName: spec.DisplayName})
		if err != nil {
			return Result{}, fmt.Errorf("seed: create household %d: %w", index+1, err)
		}
		householdIDs[index] = string(row.ID)
	}
	for index, membership := range corpus.StudentMemberships {
		if _, err := factory.AddStudentToHousehold(ctx, year.ID, mustXID(householdIDs[membership.HouseholdIndex]), mustXID(studentIDs[membership.StudentIndex])); err != nil {
			return Result{}, fmt.Errorf("seed: create student membership %d: %w", index+1, err)
		}
	}
	for index, membership := range corpus.AdultMemberships {
		if _, err := factory.AddAdultToHousehold(ctx, year.ID, mustXID(householdIDs[membership.HouseholdIndex]), mustXID(adultIDs[membership.AdultIndex])); err != nil {
			return Result{}, fmt.Errorf("seed: create adult membership %d: %w", index+1, err)
		}
	}
	for index, relationship := range corpus.Guardians {
		if _, err := factory.CreateGuardianRelationship(ctx, year.ID, people.GuardianRelationshipCreateInput{
			AdultID: mustXID(adultIDs[relationship.AdultIndex]), StudentID: mustXID(studentIDs[relationship.StudentIndex]),
			RelationshipType: relationship.RelationshipType,
		}); err != nil {
			return Result{}, fmt.Errorf("seed: create guardian relationship %d: %w", index+1, err)
		}
	}

	return Result{
		OrganizationID: string(bootstrap.Organization.ID), OrganizationName: bootstrap.Organization.Name,
		SchoolYearID: string(year.ID), ClaimURL: bootstrap.ClaimURL, ExpiresAt: bootstrap.Token.ExpiresAt,
		Students: len(studentIDs), Adults: len(adultIDs), Households: len(householdIDs),
		BoundOwner: boundOwner,
	}, nil
}

// CheckOwnerSubjectAvailable refuses a seed whose owner subject already holds a
// membership. Load never reuses an existing organization, so binding an
// already-bound subject would commit a second organization and then leave that
// subject with two memberships, which ResolveAccount treats as a hard error:
// the working login would break rather than gain a tenant. Callers that seed
// without going through Load should run this first.
func CheckOwnerSubjectAvailable(ctx context.Context, database *data.DB, providerSubject string) error {
	if database == nil {
		return errors.New("seed: database is nil")
	}
	providerSubject = strings.TrimSpace(providerSubject)
	if providerSubject == "" {
		return errors.New("seed: owner subject is empty")
	}
	_, err := identity.NewStore(database).ResolveAccount(ctx, providerSubject)
	switch {
	case errors.Is(err, auth.ErrNoOrganization):
		return nil
	case err == nil, errors.Is(err, auth.ErrMultipleOrganizations):
		return fmt.Errorf("seed: provider subject %q already has an organization membership; the seed only ever creates a new organization, so it needs an empty database: run 'make db-reset CONFIRM=1', which drops, migrates and seeds in one step", providerSubject)
	default:
		return fmt.Errorf("seed: resolve owner subject: %w", err)
	}
}

func mustXID(value string) ids.XID { return ids.XID(value) }
