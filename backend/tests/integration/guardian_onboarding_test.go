package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/guardian"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

type onboardingOTPDelivery struct {
	mu    sync.Mutex
	email string
	code  string
	count int
}

func (d *onboardingOTPDelivery) DeliverOTP(_ context.Context, email, code string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.email, d.code, d.count = email, code, d.count+1
	return nil
}

func (d *onboardingOTPDelivery) latest() (string, string, int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.email, d.code, d.count
}

func activateGuardianOnboardingYear(t *testing.T, database *data.DB, ctx context.Context, organizationID ids.XID, actor audit.Actor, schoolYearID ids.XID) {
	t.Helper()
	active := data.SchoolYearActive
	_, err := schoolyear.New(database).Update(ctx, string(organizationID), schoolYearID, auth.RoleAdministrator, actor, schoolyear.UpdateInput{State: &active})
	require.NoError(t, err)
}

func TestGuardianRegistrationLandingOnlyDisclosesActiveLinkMetadata(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian registration landing integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic landing year")
	require.NoError(t, err)
	activateGuardianOnboardingYear(t, harness.Database, ctx, organizationID, actor, year.ID)

	store := identity.NewStoreWithAuth(harness.Database, nil, nil)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)

	landing, err := store.RegistrationLanding(ctx, string(entry.ID), now)
	require.NoError(t, err)
	require.Equal(t, "Synthetic landing year", landing.SchoolYearLabel)
	require.NotEmpty(t, landing.OrganizationName)

	_, err = store.RegistrationLanding(ctx, string(entry.ID), entry.ExpiresAt)
	require.ErrorIs(t, err, guardian.ErrRegistrationInvalid)
	err = store.RevokeRegistrationEntryByID(ctx, organizationID, year.ID, entry.ID, actor, now)
	require.NoError(t, err)
	_, err = store.RegistrationLanding(ctx, string(entry.ID), now)
	require.ErrorIs(t, err, guardian.ErrRegistrationInvalid)

	active, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)
	closed := data.SchoolYearClosed
	_, err = schoolyear.New(harness.Database).Update(ctx, string(organizationID), year.ID, auth.RoleAdministrator, actor, schoolyear.UpdateInput{State: &closed})
	require.NoError(t, err)
	_, err = store.RegistrationLanding(ctx, string(active.ID), now)
	require.ErrorIs(t, err, guardian.ErrRegistrationInvalid)
	_, err = store.RegistrationLanding(ctx, "unknown-registration-link", now)
	require.ErrorIs(t, err, guardian.ErrRegistrationInvalid)
}

func TestGuardianOnboardingRequiresProofAndConsentBeforeAdultCreation(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian onboarding integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic onboarding year")
	require.NoError(t, err)
	activateGuardianOnboardingYear(t, harness.Database, ctx, organizationID, actor, year.ID)

	delivery := &onboardingOTPDelivery{}
	store := identity.NewStoreWithAuth(harness.Database, nil, delivery)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)
	session, err := store.Begin(ctx, guardian.BeginInput{EntryToken: string(entry.ID), Now: now})
	require.NoError(t, err)

	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", Now: now})
	require.ErrorIs(t, err, guardian.ErrMailboxUnverified)
	assertOnboardingRosterEmpty(t, harness, organizationID, year.ID)

	otp, err := store.RequestOTP(ctx, guardian.OTPRequestInput{SessionToken: session.Token, Email: "guardian@example.test", Now: now})
	require.NoError(t, err)
	email, code, count := delivery.latest()
	require.Equal(t, "guardian@example.test", email)
	require.Equal(t, 1, count)
	require.NotEmpty(t, otp.ChallengeID)
	verified, err := store.VerifyOTP(ctx, guardian.OTPVerifyInput{SessionToken: session.Token, ChallengeID: otp.ChallengeID, Code: code, Now: now})
	require.NoError(t, err)
	require.True(t, verified.Verified)

	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", Now: now})
	require.ErrorIs(t, err, guardian.ErrConsentRequired)
	assertOnboardingRosterEmpty(t, harness, organizationID, year.ID)

	consented, err := store.AcceptConsent(ctx, guardian.ConsentInput{SessionToken: session.Token, Email: "guardian@example.test", TermsVersion: guardian.TermsVersion, PrivacyVersion: guardian.PrivacyVersion, SourceSurface: "integration", Now: now})
	require.NoError(t, err)
	require.True(t, consented.Consented)
	completion, err := store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", Now: now})
	require.NoError(t, err)
	require.NotEmpty(t, completion.AdultID)
	require.Equal(t, organizationID, completion.OrganizationID)
	require.Equal(t, year.ID, completion.SchoolYearID)
	require.NotEmpty(t, completion.SessionToken)
	require.NotEmpty(t, completion.SessionID)
	require.False(t, completion.ExpiresAt.IsZero())
	require.False(t, completion.IdleExpiresAt.IsZero())
	require.Empty(t, completion.StudentIDs)
	assertOnboardingAdultOnly(t, harness, organizationID, year.ID, completion.AdultID)
	assertGuardianOnboardingCompletionAudit(t, harness, organizationID, completion.AdultID, completion.SessionID)

	principal, err := store.ResolveSession(ctx, completion.SessionToken)
	require.NoError(t, err)
	guardianPrincipal, ok := principal.(auth.GuardianPrincipal)
	require.True(t, ok)
	require.Equal(t, completion.SessionID, guardianPrincipal.SessionID)
	require.Equal(t, completion.AdultID, guardianPrincipal.AdultID)
	require.Equal(t, organizationID, guardianPrincipal.OrganizationID)
	require.Equal(t, year.ID, guardianPrincipal.SchoolYearID)
	require.Empty(t, guardianPrincipal.StudentIDs)

	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Replay", AdultFamilyName: "Attempt", Now: now})
	require.ErrorIs(t, err, guardian.ErrOnboardingInvalid)
}

func TestGuardianInvitationRedemptionIsSingleUseAndExportsMetadataOnly(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian invitation integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic invitation year")
	require.NoError(t, err)
	store := identity.NewStoreWithAuth(harness.Database, nil, nil)

	result, err := store.ImportInvitationContacts(ctx, organizationID, year.ID, []byte("email\nGuardian@Example.TEST\nGuardian@Example.TEST\ninvalid\n"), actor, time.Now().UTC())
	require.NoError(t, err)
	require.Len(t, result.Rows, 3)
	require.Equal(t, "issued", result.Rows[0].Status)
	require.Equal(t, "duplicate", result.Rows[1].Status)
	require.Equal(t, "error", result.Rows[2].Status)
	require.NotEmpty(t, result.Rows[0].Token)

	exported, err := store.ListInvitationContacts(ctx, organizationID, year.ID, actor)
	require.NoError(t, err)
	require.Len(t, exported, 1)
	require.Equal(t, "guardian@example.test", exported[0].Email)
	require.NotContains(t, exported[0].Status, result.Rows[0].Token)

	session, err := store.Redeem(ctx, guardian.RedeemInput{InvitationToken: result.Rows[0].Token, Now: time.Now().UTC()})
	require.NoError(t, err)
	require.True(t, session.Verified)
	_, err = store.Redeem(ctx, guardian.RedeemInput{InvitationToken: result.Rows[0].Token, Now: time.Now().UTC()})
	require.ErrorIs(t, err, guardian.ErrInvitationInvalid)
}

func TestGuardianSignupNoticeReadIsOrganizationScoped(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian signup notice integration"}
	organizationID := harness.MintOrganization(t)
	otherOrganizationID := harness.MintOrganization(t)
	store := identity.NewStoreWithAuth(harness.Database, nil, nil)
	content := "Bring the confirmation message to registration."

	updated, err := store.UpdateSignupNotice(ctx, organizationID, &content, actor, time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, updated.SignupNotice)
	require.Equal(t, content, updated.SignupNotice.Content)

	read, err := store.GetSignupNotice(ctx, organizationID)
	require.NoError(t, err)
	require.NotNil(t, read.SignupNotice)
	require.Equal(t, content, read.SignupNotice.Content)
	require.Equal(t, updated.SignupNotice.Version, read.SignupNotice.Version)

	other, err := store.GetSignupNotice(ctx, otherOrganizationID)
	require.NoError(t, err)
	require.Nil(t, other.SignupNotice)
}

func TestGuardianInvitedEmailOTPConsumesOutstandingInvitation(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian invited OTP integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic invited OTP year")
	require.NoError(t, err)
	activateGuardianOnboardingYear(t, harness.Database, ctx, organizationID, actor, year.ID)
	delivery := &onboardingOTPDelivery{}
	store := identity.NewStoreWithAuth(harness.Database, nil, delivery)
	imported, err := store.ImportInvitationContacts(ctx, organizationID, year.ID, []byte("email\ninvited@example.test\n"), actor, time.Now().UTC())
	require.NoError(t, err)
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, time.Now().UTC())
	require.NoError(t, err)
	session, err := store.Begin(ctx, guardian.BeginInput{EntryToken: string(entry.ID), Now: time.Now().UTC()})
	require.NoError(t, err)
	challenge, err := store.RequestOTP(ctx, guardian.OTPRequestInput{SessionToken: session.Token, Email: "invited@example.test", Now: time.Now().UTC()})
	require.NoError(t, err)
	_, code, _ := delivery.latest()
	verified, err := store.VerifyOTP(ctx, guardian.OTPVerifyInput{SessionToken: session.Token, ChallengeID: challenge.ChallengeID, Code: code, Now: time.Now().UTC()})
	require.NoError(t, err)
	require.True(t, verified.Verified)
	_, err = store.AcceptConsent(ctx, guardian.ConsentInput{SessionToken: session.Token, Email: "invited@example.test", TermsVersion: guardian.TermsVersion, PrivacyVersion: guardian.PrivacyVersion, SourceSurface: "integration", Now: time.Now().UTC()})
	require.NoError(t, err)
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		contact, err := tx.GetGuardianInvitationContactByEmail(ctx, year.ID, "invited@example.test")
		require.NoError(t, err)
		require.NotNil(t, contact.AcceptedConsentID)
		return nil
	})
	require.NoError(t, err)
	_, err = store.Redeem(ctx, guardian.RedeemInput{InvitationToken: imported.Rows[0].Token, Now: time.Now().UTC()})
	require.ErrorIs(t, err, guardian.ErrInvitationInvalid)
}

func TestGuardianRegistrationLinkUsesItsRecordXIDAndIsTenantIsolated(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian link isolation"}
	firstOrganizationID := harness.MintOrganization(t)
	secondOrganizationID := harness.MintOrganization(t)
	firstFactory := factories.New(harness.Database, string(firstOrganizationID), actor)
	secondFactory := factories.New(harness.Database, string(secondOrganizationID), actor)
	firstYear, err := firstFactory.CreateSchoolYear(ctx, "Synthetic first link year")
	require.NoError(t, err)
	secondYear, err := secondFactory.CreateSchoolYear(ctx, "Synthetic second link year")
	require.NoError(t, err)
	store := identity.NewStoreWithAuth(harness.Database, nil, nil)
	first, err := store.CreateRegistrationEntry(ctx, firstOrganizationID, firstYear.ID, actor, time.Now().UTC())
	require.NoError(t, err)
	_, err = store.Begin(ctx, guardian.BeginInput{EntryToken: string(first.ID), Now: time.Now().UTC()})
	require.ErrorIs(t, err, guardian.ErrRegistrationInvalid)
	activateGuardianOnboardingYear(t, harness.Database, ctx, firstOrganizationID, actor, firstYear.ID)
	activateGuardianOnboardingYear(t, harness.Database, ctx, secondOrganizationID, actor, secondYear.ID)
	second, err := store.CreateRegistrationEntry(ctx, secondOrganizationID, secondYear.ID, actor, time.Now().UTC())
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	firstSession, err := store.Begin(ctx, guardian.BeginInput{EntryToken: string(first.ID), Now: time.Now().UTC()})
	require.NoError(t, err)
	require.Equal(t, firstOrganizationID, firstSession.OrganizationID)
	secondSession, err := store.Begin(ctx, guardian.BeginInput{EntryToken: string(second.ID), Now: time.Now().UTC()})
	require.NoError(t, err)
	require.Equal(t, secondOrganizationID, secondSession.OrganizationID)
	_, err = store.Begin(ctx, guardian.BeginInput{EntryToken: "not-a-registration-xid", Now: time.Now().UTC()})
	require.ErrorIs(t, err, guardian.ErrRegistrationInvalid)
}

func TestGuardianRegistrationLinkHistoryPersistsRevocationKinds(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian link provenance"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic registration-link history year")
	require.NoError(t, err)
	store := identity.NewStoreWithAuth(harness.Database, nil, nil)
	now := time.Now().UTC()
	first, err := store.IssueRegistrationEntry(ctx, organizationID, year.ID, now.Add(24*time.Hour), actor, now)
	require.NoError(t, err)
	_, err = store.IssueRegistrationEntry(ctx, organizationID, year.ID, now.Add(48*time.Hour), actor, now.Add(time.Second))
	require.NoError(t, err)
	page, err := store.ListRegistrationEntries(ctx, organizationID, year.ID, nil, 10, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Len(t, page.Entries, 2)
	var replaced guardian.RegistrationEntry
	for _, entry := range page.Entries {
		if entry.ID == first.ID {
			replaced = entry
			break
		}
	}
	require.Equal(t, "revoked", replaced.Status)
	require.Equal(t, "replaced", replaced.RevocationKind)
	require.NotNil(t, replaced.RevokedAt)
	active := page.Entries[0]
	require.NoError(t, store.RevokeRegistrationEntryByID(ctx, organizationID, year.ID, active.ID, actor, now.Add(3*time.Second)))
	page, err = store.ListRegistrationEntries(ctx, organizationID, year.ID, nil, 10, now.Add(4*time.Second))
	require.NoError(t, err)
	for _, entry := range page.Entries {
		if entry.ID == active.ID {
			require.Equal(t, "manual", entry.RevocationKind)
			require.Equal(t, "revoked", entry.Status)
		}
	}
}

func TestGuardianRegistrationEntryRateLimit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian onboarding rate limit"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic rate-limit year")
	require.NoError(t, err)
	activateGuardianOnboardingYear(t, harness.Database, ctx, organizationID, actor, year.ID)
	store := identity.NewStoreWithAuth(harness.Database, nil, nil)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)
	for range 10 {
		_, err = store.Begin(ctx, guardian.BeginInput{EntryToken: string(entry.ID), Now: now})
		require.NoError(t, err)
	}
	_, err = store.Begin(ctx, guardian.BeginInput{EntryToken: string(entry.ID), Now: now})
	require.ErrorIs(t, err, guardian.ErrOnboardingRateLimit)
}

func TestGuardianOnboardingReusesSingleVerifiedEmailAdult(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian onboarding email reuse"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic email reuse year")
	require.NoError(t, err)
	activateGuardianOnboardingYear(t, harness.Database, ctx, organizationID, actor, year.ID)
	delivery := &onboardingOTPDelivery{}
	store := identity.NewStoreWithAuth(harness.Database, nil, delivery)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)

	complete := func() guardian.Completion {
		t.Helper()
		session, err := store.Begin(ctx, guardian.BeginInput{EntryToken: string(entry.ID), Now: now})
		require.NoError(t, err)
		challenge, err := store.RequestOTP(ctx, guardian.OTPRequestInput{SessionToken: session.Token, Email: "guardian@example.test", Now: now})
		require.NoError(t, err)
		_, code, _ := delivery.latest()
		_, err = store.VerifyOTP(ctx, guardian.OTPVerifyInput{SessionToken: session.Token, ChallengeID: challenge.ChallengeID, Code: code, Now: now})
		require.NoError(t, err)
		_, err = store.AcceptConsent(ctx, guardian.ConsentInput{SessionToken: session.Token, Email: "guardian@example.test", TermsVersion: guardian.TermsVersion, PrivacyVersion: guardian.PrivacyVersion, SourceSurface: "integration", Now: now})
		require.NoError(t, err)
		result, err := store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", Now: now})
		require.NoError(t, err)
		return result
	}

	first := complete()
	second := complete()
	require.Equal(t, first.AdultID, second.AdultID)
	assertOnboardingAdultOnly(t, harness, organizationID, year.ID, first.AdultID)
}

func assertOnboardingRosterEmpty(t *testing.T, harness *testharness.Harness, organizationID, schoolYearID ids.XID) {
	t.Helper()
	err := harness.Database.InTenantRead(harness.Context, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		adults, err := tx.ListAdults(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		students, err := tx.ListStudents(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		relationships, err := tx.ListGuardianRelationships(ctx, schoolYearID, data.GuardianRelationshipFilter{})
		if err != nil {
			return err
		}
		require.Empty(t, adults)
		require.Empty(t, students)
		require.Empty(t, relationships)
		return nil
	})
	require.NoError(t, err)
}

func assertOnboardingAdultOnly(t *testing.T, harness *testharness.Harness, organizationID, schoolYearID, adultID ids.XID) {
	t.Helper()
	err := harness.Database.InTenantRead(harness.Context, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		adults, err := tx.ListAdults(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		students, err := tx.ListStudents(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		relationships, err := tx.ListGuardianRelationships(ctx, schoolYearID, data.GuardianRelationshipFilter{})
		if err != nil {
			return err
		}
		require.Len(t, adults, 1)
		require.Equal(t, adultID, adults[0].ID)
		require.Empty(t, students)
		require.Empty(t, relationships)
		return nil
	})
	require.NoError(t, err)
}

func assertGuardianOnboardingCompletionAudit(t *testing.T, harness *testharness.Harness, organizationID, adultID, sessionID ids.XID) {
	t.Helper()
	entries, err := harness.Database.ListAuditLog(harness.Context, string(organizationID), data.AuditLogFilter{PageSize: 100})
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Action != string(audit.ActionGuardianOnboardingCompleted) {
			continue
		}
		var summary struct {
			AdultID           ids.XID `json:"adult_id"`
			ConsentID         ids.XID `json:"consent_id"`
			GuardianSessionID ids.XID `json:"guardian_session_id"`
			MailboxVerified   bool    `json:"mailbox_verified"`
		}
		require.NoError(t, json.Unmarshal(entry.ChangeSummary, &summary))
		require.Equal(t, adultID, summary.AdultID)
		require.NotEmpty(t, summary.ConsentID)
		require.Equal(t, sessionID, summary.GuardianSessionID)
		require.True(t, summary.MailboxVerified)
		return
	}
	require.Fail(t, "guardian onboarding completion audit entry not found")
}
