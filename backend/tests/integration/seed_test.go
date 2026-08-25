package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/data"
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

func resultSchoolYearID(result seed.Result) (id ids.XID) {
	return ids.XID(result.SchoolYearID)
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
