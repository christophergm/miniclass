package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/guardian"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
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

func TestGuardianOnboardingRequiresProofAndConsentBeforeRosterWrites(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian onboarding integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic onboarding year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-onboarding-grade", "Synthetic Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Onboarding Room")
	require.NoError(t, err)

	delivery := &onboardingOTPDelivery{}
	store := identity.NewStoreWithAuth(harness.Database, nil, delivery)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)
	session, err := store.Begin(ctx, guardian.BeginInput{EntryToken: entry.Token, Now: now})
	require.NoError(t, err)

	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", StudentGivenName: "Student", StudentFamilyName: "One", HomeroomID: homeroom.ID, RelationshipType: data.GuardianRelationshipParent, Now: now})
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

	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", StudentGivenName: "Student", StudentFamilyName: "One", HomeroomID: homeroom.ID, RelationshipType: data.GuardianRelationshipParent, Now: now})
	require.ErrorIs(t, err, guardian.ErrConsentRequired)
	assertOnboardingRosterEmpty(t, harness, organizationID, year.ID)

	consented, err := store.AcceptConsent(ctx, guardian.ConsentInput{SessionToken: session.Token, Email: "guardian@example.test", TermsVersion: guardian.TermsVersion, PrivacyVersion: guardian.PrivacyVersion, SourceSurface: "integration", Now: now})
	require.NoError(t, err)
	require.True(t, consented.Consented)
	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", StudentGivenName: "Student", StudentFamilyName: "One", HomeroomID: homeroom.ID, RelationshipType: data.GuardianRelationshipParent, Now: now})
	require.ErrorIs(t, err, guardian.ErrStudentAttributesRequired)
	assertOnboardingRosterEmpty(t, harness, organizationID, year.ID)
	completion, err := store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", StudentGivenName: "Student", StudentFamilyName: "One", GradeLevelID: grade.ID, HomeroomID: homeroom.ID, RelationshipType: data.GuardianRelationshipParent, Now: now})
	require.NoError(t, err)
	require.NotEmpty(t, completion.AdultID)
	require.NotEmpty(t, completion.StudentID)
	require.NotEmpty(t, completion.RelationshipID)

	_, err = store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Replay", AdultFamilyName: "Attempt", StudentGivenName: "Replay", StudentFamilyName: "Attempt", HomeroomID: homeroom.ID, RelationshipType: data.GuardianRelationshipParent, Now: now})
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
	delivery := &onboardingOTPDelivery{}
	store := identity.NewStoreWithAuth(harness.Database, nil, delivery)
	imported, err := store.ImportInvitationContacts(ctx, organizationID, year.ID, []byte("email\ninvited@example.test\n"), actor, time.Now().UTC())
	require.NoError(t, err)
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, time.Now().UTC())
	require.NoError(t, err)
	session, err := store.Begin(ctx, guardian.BeginInput{EntryToken: entry.Token, Now: time.Now().UTC()})
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

func TestGuardianRegistrationEntryRateLimit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian onboarding rate limit"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic rate-limit year")
	require.NoError(t, err)
	store := identity.NewStoreWithAuth(harness.Database, nil, nil)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)
	for range 10 {
		_, err = store.Begin(ctx, guardian.BeginInput{EntryToken: entry.Token, Now: now})
		require.NoError(t, err)
	}
	_, err = store.Begin(ctx, guardian.BeginInput{EntryToken: entry.Token, Now: now})
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
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-email-reuse-grade", "Synthetic Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Email Reuse Room")
	require.NoError(t, err)
	delivery := &onboardingOTPDelivery{}
	store := identity.NewStoreWithAuth(harness.Database, nil, delivery)
	now := time.Now().UTC()
	entry, err := store.CreateRegistrationEntry(ctx, organizationID, year.ID, actor, now)
	require.NoError(t, err)

	complete := func(studentGivenName string) guardian.Completion {
		t.Helper()
		session, err := store.Begin(ctx, guardian.BeginInput{EntryToken: entry.Token, Now: now})
		require.NoError(t, err)
		challenge, err := store.RequestOTP(ctx, guardian.OTPRequestInput{SessionToken: session.Token, Email: "guardian@example.test", Now: now})
		require.NoError(t, err)
		_, code, _ := delivery.latest()
		_, err = store.VerifyOTP(ctx, guardian.OTPVerifyInput{SessionToken: session.Token, ChallengeID: challenge.ChallengeID, Code: code, Now: now})
		require.NoError(t, err)
		_, err = store.AcceptConsent(ctx, guardian.ConsentInput{SessionToken: session.Token, Email: "guardian@example.test", TermsVersion: guardian.TermsVersion, PrivacyVersion: guardian.PrivacyVersion, SourceSurface: "integration", Now: now})
		require.NoError(t, err)
		result, err := store.Complete(ctx, guardian.CompleteInput{SessionToken: session.Token, AdultGivenName: "Guardian", AdultFamilyName: "One", StudentGivenName: studentGivenName, StudentFamilyName: "One", GradeLevelID: grade.ID, HomeroomID: homeroom.ID, RelationshipType: data.GuardianRelationshipParent, Now: now})
		require.NoError(t, err)
		return result
	}

	first := complete("Student")
	second := complete("Student Two")
	require.Equal(t, first.AdultID, second.AdultID)
	require.NotEqual(t, first.StudentID, second.StudentID)
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
