package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	dbtesting "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestAdultOTPGuardianScopeAndMFAStepUp(t *testing.T) {
	harness := dbtesting.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "adult authentication integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic adult authentication year")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic room")
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Student", HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	email := "guardian@example.test"
	adult, err := factory.CreateAdult(ctx, year.ID, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Guardian", Email: &email,
	})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, year.ID, people.GuardianRelationshipCreateInput{
		AdultID: adult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)

	providerSubject := "adult-auth-owner"
	userEmail := "owner@example.test"
	var userID string
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, providerSubject, userEmail).Scan(&userID))
	_, err = harness.Migrator.Exec(ctx, `
		insert into organization_members (organization_id, user_id, role)
		values ($1, $2, 'owner')`, organizationID, userID)
	require.NoError(t, err)

	deliveredCodes := map[string]string{}
	deliveryCount := 0
	store := identity.NewStoreWithOTPDelivery(harness.Database, []byte("integration-auth-key"), func(_ context.Context, deliveredEmail, code string) error {
		deliveryCount++
		deliveredCodes[deliveredEmail] = code
		return nil
	})

	now := time.Now().UTC().Truncate(time.Second)
	requested, err := store.RequestAdultOTP(ctx, auth.OTPRequest{
		OrganizationID: organizationID, SchoolYearID: year.ID, Email: strings.ToUpper(email), Now: now,
	})
	require.NoError(t, err)
	require.True(t, requested.Accepted)
	require.NotEmpty(t, requested.ChallengeID)
	require.Equal(t, 1, deliveryCount)
	deliveredCode := deliveredCodes[email]
	require.Len(t, deliveredCode, 6)
	require.NotEqual(t, deliveredCode, requested.ChallengeID)

	persisted, err := store.GetAccessTokenByBearer(ctx, requested.ChallengeID)
	require.NoError(t, err)
	require.Equal(t, "adult_otp", persisted.Purpose)
	require.Len(t, persisted.VerifierHash, 32)
	require.NotEqual(t, []byte(deliveredCode), persisted.VerifierHash)

	session, err := store.VerifyAdultOTP(ctx, auth.OTPVerification{ChallengeID: requested.ChallengeID, Code: deliveredCode, Now: now})
	require.NoError(t, err)
	require.Equal(t, []ids.XID{student.ID}, session.StudentIDs)
	principal, err := store.ResolveSession(ctx, session.Bearer)
	require.NoError(t, err)
	guardian, ok := principal.(auth.GuardianPrincipal)
	require.True(t, ok)
	require.Equal(t, []ids.XID{student.ID}, guardian.StudentIDs)
	require.True(t, guardian.HasCapability(auth.CapabilityGuardianAccess))
	require.False(t, guardian.HasCapability(auth.CapabilityManageRoster))
	_, err = store.VerifyAdultOTP(ctx, auth.OTPVerification{ChallengeID: requested.ChallengeID, Code: deliveredCode, Now: now})
	require.ErrorIs(t, err, auth.ErrOTPInvalid)

	duplicate, err := factory.CreateAdult(ctx, year.ID, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Duplicate Guardian", Email: &email,
	})
	require.NoError(t, err)
	require.NotEqual(t, adult.ID, duplicate.ID)
	duplicateRequest, err := store.RequestAdultOTP(ctx, auth.OTPRequest{
		OrganizationID: organizationID, SchoolYearID: year.ID, Email: email, Now: now,
	})
	require.NoError(t, err)
	require.True(t, duplicateRequest.Accepted)
	require.NotEmpty(t, duplicateRequest.ChallengeID)
	require.Equal(t, 1, deliveryCount, "duplicate email must not trigger delivery")
	_, err = store.VerifyAdultOTP(ctx, auth.OTPVerification{ChallengeID: duplicateRequest.ChallengeID, Code: deliveredCode, Now: now})
	require.ErrorIs(t, err, auth.ErrOTPInvalid, "duplicate email must not create a usable challenge")

	expiredEmail := "expired-guardian@example.test"
	expiredAdult, err := factory.CreateAdult(ctx, year.ID, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Expired Guardian", Email: &expiredEmail,
	})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, year.ID, people.GuardianRelationshipCreateInput{
		AdultID: expiredAdult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	expiredRequest, err := store.RequestAdultOTP(ctx, auth.OTPRequest{
		OrganizationID: organizationID, SchoolYearID: year.ID, Email: expiredEmail, Now: now,
	})
	require.NoError(t, err)
	require.True(t, expiredRequest.Accepted)
	require.Equal(t, 2, deliveryCount)
	_, err = store.VerifyAdultOTP(ctx, auth.OTPVerification{
		ChallengeID: expiredRequest.ChallengeID, Code: deliveredCodes[expiredEmail], Now: now.Add(11 * time.Minute),
	})
	require.ErrorIs(t, err, auth.ErrOTPInvalid, "expired OTP must not be usable")

	unknown, err := store.RequestAdultOTP(ctx, auth.OTPRequest{
		OrganizationID: organizationID, SchoolYearID: year.ID, Email: "unknown@example.test", Now: now,
	})
	require.NoError(t, err)
	require.True(t, unknown.Accepted)
	require.NotEmpty(t, unknown.ChallengeID)
	require.Equal(t, 2, deliveryCount)

	link, err := store.CreateAdultAccountLink(ctx, auth.AdultAccountLinkInput{
		OrganizationID: organizationID, SchoolYearID: year.ID, AdultID: adult.ID, UserID: ids.XID(userID), Actor: actor,
	})
	require.NoError(t, err)
	require.Equal(t, adult.ID, link.AdultID)

	enrollment, err := store.EnrollMFA(ctx, ids.XID(userID), organizationID, actor, now)
	require.NoError(t, err)
	require.Len(t, enrollment.RecoveryCodes, 8)
	code := totpForTest(t, enrollment.Secret, now)
	adminSession, err := store.VerifyMFA(ctx, auth.MFAVerification{
		UserID: ids.XID(userID), OrganizationID: organizationID, Code: code, Now: now,
	})
	require.NoError(t, err)
	adminPrincipal, err := store.ResolveSession(ctx, adminSession.Bearer)
	require.NoError(t, err)
	admin, ok := adminPrincipal.(auth.AccountPrincipal)
	require.True(t, ok)
	require.True(t, admin.MFAAuthenticated)
	require.True(t, admin.HasCapability(auth.CapabilityManageRoster))

	guardianSession, err := store.CreateGuardianSessionForAccount(ctx, ids.XID(userID), organizationID, year.ID, now)
	require.NoError(t, err)
	require.Equal(t, session.StudentIDs, guardianSession.StudentIDs)
	stepUp, err := store.VerifyMFAForGuardian(ctx, guardian, code, "", now)
	require.NoError(t, err)
	require.Equal(t, ids.XID(userID), stepUp.UserID)

	idleExpiredSession, err := store.CreateGuardianSessionForAccount(ctx, ids.XID(userID), organizationID, year.ID, now)
	require.NoError(t, err)
	_, err = harness.Migrator.Exec(ctx, `
		update access_tokens
		set idle_expires_at = $1
		where id = $2`, now.Add(-time.Minute), idleExpiredSession.SessionID)
	require.NoError(t, err)
	_, err = store.ResolveSession(ctx, idleExpiredSession.Bearer)
	require.ErrorIs(t, err, auth.ErrSessionInvalid, "idle-expired guardian session must be rejected")

	absoluteExpiredSession, err := store.CreateGuardianSessionForAccount(ctx, ids.XID(userID), organizationID, year.ID, now)
	require.NoError(t, err)
	_, err = harness.Migrator.Exec(ctx, `
		update access_tokens
		set expires_at = $1
		where id = $2`, now.Add(-time.Minute), absoluteExpiredSession.SessionID)
	require.NoError(t, err)
	_, err = store.ResolveSession(ctx, absoluteExpiredSession.Bearer)
	require.ErrorIs(t, err, auth.ErrSessionInvalid, "absolutely expired guardian session must be rejected")

	peopleService := people.New(harness.Database)
	relationships, err := peopleService.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{AdultID: adult.ID})
	require.NoError(t, err)
	require.Len(t, relationships, 1)
	require.NoError(t, peopleService.DeleteGuardianRelationship(ctx, string(organizationID), year.ID, relationships[0].ID, actor))
	resolvedAfterRemoval, err := store.ResolveSession(ctx, session.Bearer)
	require.NoError(t, err)
	guardianAfterRemoval, ok := resolvedAfterRemoval.(auth.GuardianPrincipal)
	require.True(t, ok)
	require.Empty(t, guardianAfterRemoval.StudentIDs, "guardian session scope must be resolved from current relationships")
	require.NoError(t, store.RevokeSession(ctx, guardianSession.Bearer))
	_, err = store.ResolveSession(ctx, guardianSession.Bearer)
	require.ErrorIs(t, err, auth.ErrSessionInvalid, "revoked guardian session must be rejected")

	recoverySession, err := store.VerifyMFA(ctx, auth.MFAVerification{
		UserID: ids.XID(userID), OrganizationID: organizationID, RecoveryCode: enrollment.RecoveryCodes[0], Now: now,
	})
	require.NoError(t, err)
	require.NotEmpty(t, recoverySession.Bearer)
	_, err = store.VerifyMFA(ctx, auth.MFAVerification{
		UserID: ids.XID(userID), OrganizationID: organizationID, RecoveryCode: enrollment.RecoveryCodes[0], Now: now,
	})
	require.ErrorIs(t, err, auth.ErrMFAInvalid)

	require.NoError(t, store.ResetMFA(ctx, auth.MFAReset{
		OrganizationID: organizationID, Actor: actor, TargetUserID: ids.XID(userID), Reason: "lost authenticator", Now: now,
	}))
	_, err = store.ResolveSession(ctx, adminSession.Bearer)
	require.ErrorIs(t, err, auth.ErrSessionInvalid)
	_, err = store.VerifyMFA(ctx, auth.MFAVerification{
		UserID: ids.XID(userID), OrganizationID: organizationID, Code: code, Now: now,
	})
	require.ErrorIs(t, err, auth.ErrMFANotEnrolled)
}

func totpForTest(t *testing.T, secret string, now time.Time) string {
	t.Helper()
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	require.NoError(t, err)
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(now.Unix()/30))
	mac := hmac.New(sha1.New, raw)
	_, err = mac.Write(counter[:])
	require.NoError(t, err)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}
