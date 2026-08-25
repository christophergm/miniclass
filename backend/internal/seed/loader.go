package seed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
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
	ClaimBaseURL     string
	InvitationTTL    time.Duration
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

	corpus := Generate()
	if err := corpus.Validate(); err != nil {
		return Result{}, fmt.Errorf("seed: validate corpus: %w", err)
	}
	bootstrap, err := identity.Bootstrap(ctx, identity.NewStore(database), identity.BootstrapInput{
		OrganizationName: options.OrganizationName,
		HomeroomLabel:    options.HomeroomLabel,
		OwnerEmail:       options.OwnerEmail,
		ClaimBaseURL:     options.ClaimBaseURL,
		InvitationTTL:    options.InvitationTTL,
	})
	if err != nil {
		return Result{}, fmt.Errorf("seed: bootstrap organization: %w", err)
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
	}, nil
}

func mustXID(value string) ids.XID { return ids.XID(value) }
