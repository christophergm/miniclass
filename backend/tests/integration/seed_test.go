package integration

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/seed"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/stretchr/testify/require"
)

func TestSeedCorpusLoadsWithAppendixDistributionAndEdgeCases(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	options := seed.DefaultOptions()
	options.OrganizationName = "Synthetic Seed Test"
	options.OwnerEmail = "seed-owner@example.test"
	result, err := seed.Load(ctx, harness.Database, options)
	require.NoError(t, err)
	require.Equal(t, seed.StudentCount, result.Students)
	require.Equal(t, seed.AdultCount, result.Adults)
	require.Equal(t, seed.HouseholdCount, result.Households)
	require.NotEmpty(t, result.OrganizationID)
	require.Contains(t, result.ClaimURL, "http://localhost:5173/claim")
	require.Nil(t, result.BoundOwner, "a seed without an owner subject bound a login")

	students, err := people.New(harness.Database).ListStudents(ctx, result.OrganizationID, resultSchoolYearID(result))
	require.NoError(t, err)
	adults, err := people.New(harness.Database).List(ctx, result.OrganizationID, resultSchoolYearID(result))
	require.NoError(t, err)
	households, err := people.New(harness.Database).ListHouseholds(ctx, result.OrganizationID, resultSchoolYearID(result))
	require.NoError(t, err)
	require.Len(t, students, seed.StudentCount)
	require.Len(t, adults, seed.AdultCount)
	require.Len(t, households, seed.HouseholdCount)

	vocabularySnapshot, err := vocabulary.New(harness.Database).List(ctx, result.OrganizationID, false)
	require.NoError(t, err)
	require.Len(t, vocabularySnapshot.Grades, 6)
	require.Len(t, vocabularySnapshot.Homerooms, 6)

	gradeCounts := map[string]int{}
	for _, student := range students {
		gradeCounts[string(student.GradeLevelID)]++
	}
	for index, grade := range vocabularySnapshot.Grades {
		want := []int{20, 27, 22, 21, 30, 19}[index]
		require.Equal(t, want, gradeCounts[string(grade.ID)])
	}
	homeroomIDs := map[string]bool{}
	for _, student := range students {
		homeroomIDs[string(student.HomeroomID)] = true
	}
	require.Len(t, homeroomIDs, 6)

	participation := map[data.AdultParticipationIntent]int{}
	for _, adult := range adults {
		participation[adult.ParticipationIntent]++
	}
	require.Equal(t, 13, participation[data.AdultParticipationLead])
	require.Equal(t, 45, participation[data.AdultParticipationHelp])
	require.Equal(t, 44, participation[data.AdultParticipationUnavailable])

	studentMemberships := allStudentMemberships(t, harness, result.OrganizationID)
	studentMembershipCounts := countStudentMemberships(studentMemberships)
	studentsWithoutHousehold := 0
	studentsInMultipleHouseholds := 0
	for _, student := range students {
		switch studentMembershipCounts[string(student.ID)] {
		case 0:
			studentsWithoutHousehold++
		case 2:
			studentsInMultipleHouseholds++
		}
	}
	require.Equal(t, 2, studentsWithoutHousehold)
	require.GreaterOrEqual(t, studentsInMultipleHouseholds, 2)

	adultMemberships := allAdultMemberships(t, harness, result.OrganizationID)
	adultMembershipCounts := map[string]int{}
	for _, membership := range adultMemberships {
		adultMembershipCounts[string(membership.AdultID)]++
	}
	guardianRelationships := allGuardianRelationships(t, harness, result.OrganizationID)
	guardianCounts := map[string]int{}
	for _, relationship := range guardianRelationships {
		guardianCounts[string(relationship.AdultID)]++
	}
	adultsInMultipleHouseholds := 0
	adultsWithoutHouseholdOrGuardian := 0
	for _, adult := range adults {
		if adultMembershipCounts[string(adult.ID)] >= 2 {
			adultsInMultipleHouseholds++
		}
		if adultMembershipCounts[string(adult.ID)] == 0 && guardianCounts[string(adult.ID)] == 0 {
			adultsWithoutHouseholdOrGuardian++
		}
	}
	require.GreaterOrEqual(t, adultsInMultipleHouseholds, 1)
	require.GreaterOrEqual(t, adultsWithoutHouseholdOrGuardian, 2)

	noExternalIdentifier := 0
	familyCounts := map[string]int{}
	twoWordSurname := 0
	for _, student := range students {
		if student.ExternalIdentifier == nil {
			noExternalIdentifier++
		}
		familyCounts[student.LegalFamilyName]++
		if student.LegalFamilyName == "Synthetic De La Sample" {
			twoWordSurname++
		}
	}
	duplicateFamily := false
	for _, count := range familyCounts {
		if count > 1 {
			duplicateFamily = true
			break
		}
	}
	require.Equal(t, 1, noExternalIdentifier)
	require.True(t, duplicateFamily)
	require.Equal(t, 1, twoWordSurname)
}

func TestSeedBindsOwnerSubjectAndRefusesToRebindIt(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	store := identity.NewStore(harness.Database)
	subject := fmt.Sprintf("local|seed-owner-%d", time.Now().UnixNano())
	options := seed.DefaultOptions()
	options.OrganizationName = fmt.Sprintf("Synthetic Seed Bind %d", time.Now().UnixNano())
	options.OwnerEmail = "seed-bind-owner@example.test"
	options.OwnerSubject = subject

	result, err := seed.Load(ctx, harness.Database, options)
	require.NoError(t, err)
	require.NotNil(t, result.BoundOwner)
	require.Equal(t, subject, result.BoundOwner.ProviderSubject)
	require.Equal(t, options.OwnerEmail, result.BoundOwner.Email)
	require.Equal(t, "owner", result.BoundOwner.Role)
	require.Equal(t, result.OrganizationID, result.BoundOwner.OrganizationID)
	require.Equal(t, options.OrganizationName, result.BoundOwner.OrganizationName)

	// ResolveAccount refuses zero and multiple memberships outright, so a
	// successful resolve is the assertion that exactly one exists.
	account, err := store.ResolveAccount(ctx, subject)
	require.NoError(t, err)
	require.Equal(t, subject, account.User.ProviderSubject)
	require.Equal(t, options.OwnerEmail, account.User.Email)
	require.Equal(t, ids.XID(result.OrganizationID), account.Membership.OrganizationID)
	require.Equal(t, options.OrganizationName, account.Membership.OrganizationName)
	require.Equal(t, "owner", account.Membership.Role)

	bearer := claimURLToken(t, result.ClaimURL)
	token, err := store.GetAccessTokenByBearer(ctx, bearer)
	require.NoError(t, err)
	require.NotNil(t, token.ConsumedAt, "the automatic claim did not consume the invitation")
	_, err = store.ClaimAdminInvitation(ctx, auth.InvitationClaimInput{
		Bearer: bearer, ProviderSubject: subject + "-replay",
		Email: options.OwnerEmail, EmailVerified: true,
	})
	require.ErrorIs(t, err, auth.ErrInvitationInvalid, "the printed claim URL is still replayable")

	organizations := countOrganizationsNamed(t, harness, options.OrganizationName)
	require.Equal(t, 1, organizations)

	second, err := seed.Load(ctx, harness.Database, options)
	require.Error(t, err)
	require.ErrorContains(t, err, "make reset CONFIRM=1")
	require.Empty(t, second.OrganizationID)
	require.Nil(t, second.BoundOwner)
	require.Equal(t, organizations, countOrganizationsNamed(t, harness, options.OrganizationName),
		"the refused seed created an organization before failing")

	// The refusal left the first login usable instead of turning it into the
	// multiple-organizations case.
	again, err := store.ResolveAccount(ctx, subject)
	require.NoError(t, err)
	require.Equal(t, account.Membership.ID, again.Membership.ID)
}

func resultSchoolYearID(result seed.Result) (id ids.XID) {
	return ids.XID(result.SchoolYearID)
}

func claimURLToken(t *testing.T, claimURL string) string {
	t.Helper()
	parsed, err := url.Parse(claimURL)
	require.NoError(t, err)
	token := parsed.Query().Get("token")
	require.NotEmpty(t, token)
	return token
}

// countOrganizationsNamed is scoped to one test's unique organization name;
// the suite shares a database, so a global count would see other tests' rows.
func countOrganizationsNamed(t *testing.T, harness *testharness.Harness, name string) int {
	t.Helper()
	var count int
	require.NoError(t, harness.Migrator.QueryRow(harness.Context,
		`select count(*) from organizations where name = $1`, name).Scan(&count))
	return count
}

func allStudentMemberships(t *testing.T, harness *testharness.Harness, organizationID string) []data.HouseholdStudent {
	var result []data.HouseholdStudent
	require.NoError(t, harness.Database.InTenantRead(harness.Context, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.ListAllHouseholdStudentsForRegistry(ctx)
		return err
	}))
	return result
}

func allAdultMemberships(t *testing.T, harness *testharness.Harness, organizationID string) []data.HouseholdAdult {
	var result []data.HouseholdAdult
	require.NoError(t, harness.Database.InTenantRead(harness.Context, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.ListAllHouseholdAdultsForRegistry(ctx)
		return err
	}))
	return result
}

func allGuardianRelationships(t *testing.T, harness *testharness.Harness, organizationID string) []data.GuardianRelationship {
	var result []data.GuardianRelationship
	require.NoError(t, harness.Database.InTenantRead(harness.Context, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.ListAllGuardianRelationshipsForRegistry(ctx)
		return err
	}))
	return result
}

func countStudentMemberships(rows []data.HouseholdStudent) map[string]int {
	result := map[string]int{}
	for _, row := range rows {
		result[string(row.StudentID)]++
	}
	return result
}
