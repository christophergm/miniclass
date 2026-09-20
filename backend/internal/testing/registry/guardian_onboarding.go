package registry

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "guardian_invitation_contacts", YearScoped: true, Factory: createGuardianInvitationContact,
		ReadIDs: readGuardianInvitationContactIDs, FetchByID: fetchGuardianInvitationContactByID,
		UpdateByID: updateGuardianInvitationContactByID, DeleteByID: deleteGuardianInvitationContactByID,
		InsertWithForeignParent: insertGuardianInvitationContactWithForeignParent})
	Register(Entity{TableName: "guardian_onboarding_consents", YearScoped: true, Immutable: true, Factory: createGuardianOnboardingConsent,
		ReadIDs: readGuardianOnboardingConsentIDs, FetchByID: fetchGuardianOnboardingConsentByID,
		UpdateByID: updateGuardianOnboardingConsentByID, DeleteByID: deleteGuardianOnboardingConsentByID,
		InsertWithForeignParent: insertGuardianOnboardingConsentWithForeignParent})
}

func createGuardianInvitationContact(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create guardian invitation contact fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 guardian invitation factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic guardian invitation year %s", organizationID))
	if err != nil {
		return "", err
	}
	var contact data.GuardianInvitationContact
	err = harness.Database.InTenant(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 guardian invitation factory"}, func(ctx context.Context, tx *data.Tx) error {
		hash := sha256.Sum256([]byte("guardian-invitation-" + string(organizationID)))
		token, err := tx.CreateGuardianInvitationToken(ctx, hash[:], time.Now().UTC().Add(time.Hour), organizationID, year.ID)
		if err != nil {
			return err
		}
		contact, err = tx.CreateGuardianInvitationContact(ctx, year.ID, token.ID, "guardian-"+string(organizationID)+"@example.test")
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianInvitationImport, ObjectType: "guardian_invitation_contact", ObjectID: &contact.ID, SchoolYearID: &year.ID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return contact.ID, err
}

func createGuardianOnboardingConsent(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create guardian consent fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 guardian consent factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic guardian consent year %s", organizationID))
	if err != nil {
		return "", err
	}
	var consent data.GuardianOnboardingConsent
	err = harness.Database.InTenant(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 guardian consent factory"}, func(ctx context.Context, tx *data.Tx) error {
		hash := sha256.Sum256([]byte("guardian-session-" + string(organizationID)))
		parent, err := tx.CreateGuardianOnboardingSession(ctx, hash[:], time.Now().UTC().Add(time.Hour), &organizationID, &year.ID, nil, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		if err != nil {
			return err
		}
		consent, err = tx.CreateGuardianOnboardingConsent(ctx, year.ID, parent.ID, "consent-"+string(organizationID)+"@example.test", "terms-v1", "privacy-v1", nil, nil, time.Now().UTC(), "layer2-fixture")
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianTermsAccepted, ObjectType: "guardian_onboarding_consent", ObjectID: &consent.ID, SchoolYearID: &year.ID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return consent.ID, err
}

func readGuardianInvitationContactIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllGuardianInvitationContactsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchGuardianInvitationContactByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindGuardianInvitationContactForRegistry(ctx, id)
	return row.ID != "", err
}

func updateGuardianInvitationContactByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchGuardianInvitationContactForRegistry(ctx, id)
}

func deleteGuardianInvitationContactByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindGuardianInvitationContactForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.DeleteGuardianInvitationContact(ctx, row.SchoolYearID, id)
}

func readGuardianOnboardingConsentIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllGuardianOnboardingConsentsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchGuardianOnboardingConsentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindGuardianOnboardingConsentForRegistry(ctx, id)
	return row.ID != "", err
}

func updateGuardianOnboardingConsentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchGuardianOnboardingConsentForRegistry(ctx, id)
}

func deleteGuardianOnboardingConsentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteGuardianOnboardingConsentForRegistry(ctx, id)
}

func insertGuardianInvitationContactWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert guardian invitation contact fixture: app pool is nil")
	}
	factory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign guardian invitation fixture"})
	year, err := factory.CreateSchoolYear(ctx, "Synthetic Foreign Guardian Invitation Year")
	if err != nil {
		return err
	}
	var tokenID ids.XID
	err = harness.Database.InTenant(ctx, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign guardian invitation fixture"}, func(ctx context.Context, tx *data.Tx) error {
		hash := sha256.Sum256([]byte("foreign-guardian-invitation-" + string(foreignOrganizationID)))
		token, err := tx.CreateGuardianInvitationToken(ctx, hash[:], time.Now().UTC().Add(time.Hour), foreignOrganizationID, year.ID)
		if err != nil {
			return err
		}
		tokenID = token.ID
		tx.NoAuditRequired("foreign parent fixture")
		return nil
	})
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into guardian_invitation_contacts (organization_id, school_year_id, invitation_token_id, email) values ($1, $2, $3, $4)`, string(foreignOrganizationID), string(year.ID), string(tokenID), "foreign@example.test")
	return err
}

func insertGuardianOnboardingConsentWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert guardian consent fixture: app pool is nil")
	}
	factory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign guardian consent fixture"})
	year, err := factory.CreateSchoolYear(ctx, "Synthetic Foreign Guardian Consent Year")
	if err != nil {
		return err
	}
	var sessionID ids.XID
	err = harness.Database.InTenant(ctx, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign guardian consent fixture"}, func(ctx context.Context, tx *data.Tx) error {
		hash := sha256.Sum256([]byte("foreign-guardian-session-" + string(foreignOrganizationID)))
		session, err := tx.CreateGuardianOnboardingSession(ctx, hash[:], time.Now().UTC().Add(time.Hour), &foreignOrganizationID, &year.ID, nil, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		if err != nil {
			return err
		}
		sessionID = session.ID
		tx.NoAuditRequired("foreign parent fixture")
		return nil
	})
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into guardian_onboarding_consents (organization_id, school_year_id, session_token_id, verified_email, terms_version, privacy_version, accepted_at, source_surface) values ($1, $2, $3, $4, $5, $6, now(), $7)`, string(foreignOrganizationID), string(year.ID), string(sessionID), "foreign@example.test", "terms-v1", "privacy-v1", "foreign-fixture")
	return err
}
