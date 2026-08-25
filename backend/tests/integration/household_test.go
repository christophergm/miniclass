package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/stretchr/testify/require"
)

func TestHouseholdsAllowMultipleMembershipsAndSeparateGuardianRelationships(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "household integration test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "household-grade", "Household Grade")
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Household Room")
	require.NoError(t, err)

	service := people.New(harness.Database)
	student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Household Student", GradeLevelID: grade.ID, HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	adult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Household Adult", ParticipationIntent: data.AdultParticipationHelp,
	})
	require.NoError(t, err)
	householdA, err := service.CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Primary Household"})
	require.NoError(t, err)
	householdB, err := service.CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Second Household"})
	require.NoError(t, err)

	// A student and adult can each belong to more than one household.
	_, err = service.AddStudentToHousehold(ctx, string(organizationID), year.ID, householdA.ID, student.ID, actor)
	require.NoError(t, err)
	_, err = service.AddStudentToHousehold(ctx, string(organizationID), year.ID, householdB.ID, student.ID, actor)
	require.NoError(t, err)
	_, err = service.AddAdultToHousehold(ctx, string(organizationID), year.ID, householdA.ID, adult.ID, actor)
	require.NoError(t, err)
	_, err = service.AddAdultToHousehold(ctx, string(organizationID), year.ID, householdB.ID, adult.ID, actor)
	require.NoError(t, err)
	studentsA, err := service.ListHouseholdStudents(ctx, string(organizationID), year.ID, householdA.ID)
	require.NoError(t, err)
	studentsB, err := service.ListHouseholdStudents(ctx, string(organizationID), year.ID, householdB.ID)
	require.NoError(t, err)
	require.Len(t, studentsA, 1)
	require.Len(t, studentsB, 1)
	require.NotEqual(t, studentsA[0].ID, studentsB[0].ID)
	require.Equal(t, year.ID, studentsA[0].SchoolYearID)
	require.Equal(t, householdA.ID, studentsA[0].HouseholdID)
	require.Equal(t, student.ID, studentsA[0].StudentID)

	// A relationship is its own record and need not imply a household membership.
	relationship, err := service.CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: adult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	nextType := data.GuardianRelationshipGuardian
	updated, err := service.UpdateGuardianRelationship(ctx, string(organizationID), year.ID, relationship.ID, actor, people.GuardianRelationshipUpdateInput{RelationshipType: &nextType})
	require.NoError(t, err)
	require.Equal(t, nextType, updated.RelationshipType)

	// Empty household membership is valid; it is not a blocking validation.
	emptyHousehold, err := service.CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Empty Household"})
	require.NoError(t, err)
	emptyStudents, err := service.ListHouseholdStudents(ctx, string(organizationID), year.ID, emptyHousehold.ID)
	require.NoError(t, err)
	require.Empty(t, emptyStudents)

	require.NoError(t, service.RemoveStudentFromHousehold(ctx, string(organizationID), year.ID, householdA.ID, student.ID, actor))
	studentsA, err = service.ListHouseholdStudents(ctx, string(organizationID), year.ID, householdA.ID)
	require.NoError(t, err)
	require.Empty(t, studentsA)
	studentsB, err = service.ListHouseholdStudents(ctx, string(organizationID), year.ID, householdB.ID)
	require.NoError(t, err)
	require.Len(t, studentsB, 1)
	require.NoError(t, service.DeleteGuardianRelationship(ctx, string(organizationID), year.ID, relationship.ID, actor))
	relationships, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID)
	require.NoError(t, err)
	require.Empty(t, relationships)

	var auditCount int64
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		auditCount, err = tx.Queries().CountAuditLog(ctx)
		return err
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, auditCount, int64(13))
}
