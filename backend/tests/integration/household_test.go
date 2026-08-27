package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
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
	relationships, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{})
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

// The Households column on a roster used to cost one request per household in
// the year. The year-scoped listing that replaced the fan-out is a new query
// path, so it gets its own tenancy proof (SPEC §9.2) rather than inheriting the
// per-household sub-resource's.
func TestHouseholdMembershipListingIsScopedToOneOrganizationAndYear(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	service := people.New(harness.Database)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "household membership listing test"}

	// Two tenants with the same shape, so a leak shows up as an extra row rather
	// than as an absent one.
	tenantA := newMembershipFixture(t, harness, service, actor, "A")
	tenantB := newMembershipFixture(t, harness, service, actor, "B")

	membershipA, err := service.ListHouseholdMembership(ctx, string(tenantA.organizationID), tenantA.year.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(tenantA.household.ID)}, householdIDsOfStudentMemberships(membershipA.Students))
	require.ElementsMatch(t, []string{string(tenantA.student.ID)}, studentIDsOf(membershipA.Students))
	require.ElementsMatch(t, []string{string(tenantA.adult.ID)}, adultIDsOf(membershipA.Adults))

	// The other tenant's rows exist and are reachable only under its own scope.
	membershipB, err := service.ListHouseholdMembership(ctx, string(tenantB.organizationID), tenantB.year.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(tenantB.student.ID)}, studentIDsOf(membershipB.Students))
	require.NotContains(t, studentIDsOf(membershipB.Students), string(tenantA.student.ID))
	require.NotContains(t, adultIDsOf(membershipB.Adults), string(tenantA.adult.ID))

	// One tenant asking for the other's year gets nothing, not the other's rows.
	crossYear, err := service.ListHouseholdMembership(ctx, string(tenantA.organizationID), tenantB.year.ID)
	require.Error(t, err)
	require.Empty(t, crossYear.Students)
	require.Empty(t, crossYear.Adults)

	// A second year in the same organisation is a separate answer.
	secondYear, err := schoolyear.New(harness.Database).Create(ctx, string(tenantA.organizationID), actor, "2027–2028")
	require.NoError(t, err)
	secondYearMembership, err := service.ListHouseholdMembership(ctx, string(tenantA.organizationID), secondYear.ID)
	require.NoError(t, err)
	require.Empty(t, secondYearMembership.Students)
	require.Empty(t, secondYearMembership.Adults)
}

// SPEC §21.3: a soft-deleted person is excluded from views while referential
// integrity with historical records is preserved. The membership row survives;
// the listings stop reporting it, so the household counts and the Households
// column no longer count a deleted person and no longer render a bare xid where
// a name would be.
func TestHouseholdMembershipExcludesSoftDeletedPeopleAndHouseholds(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	service := people.New(harness.Database)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "household membership soft delete test"}
	tenant := newMembershipFixture(t, harness, service, actor, "SoftDelete")
	organizationID := string(tenant.organizationID)

	require.NoError(t, service.DeleteStudent(ctx, organizationID, tenant.year.ID, tenant.student.ID, actor))

	membership, err := service.ListHouseholdMembership(ctx, organizationID, tenant.year.ID)
	require.NoError(t, err)
	require.Empty(t, studentIDsOf(membership.Students))
	require.ElementsMatch(t, []string{string(tenant.adult.ID)}, adultIDsOf(membership.Adults))

	// The per-household sub-resource, which still serves the household detail
	// page, agrees with the year-scoped listing.
	householdStudents, err := service.ListHouseholdStudents(ctx, organizationID, tenant.year.ID, tenant.household.ID)
	require.NoError(t, err)
	require.Empty(t, householdStudents)

	// Removing the retained membership row is still possible: the deletion path
	// looks the row up directly rather than through the filtered listing.
	require.NoError(t, service.RemoveStudentFromHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, tenant.student.ID, actor))

	require.NoError(t, service.Delete(ctx, organizationID, tenant.year.ID, tenant.adult.ID, actor))
	membership, err = service.ListHouseholdMembership(ctx, organizationID, tenant.year.ID)
	require.NoError(t, err)
	require.Empty(t, adultIDsOf(membership.Adults))
	householdAdults, err := service.ListHouseholdAdults(ctx, organizationID, tenant.year.ID, tenant.household.ID)
	require.NoError(t, err)
	require.Empty(t, householdAdults)

	// A soft-deleted household is absent from the household listing, so its
	// membership must be absent from the year's membership too; otherwise the
	// frontend indexes rows it can never name.
	survivor, err := service.CreateStudent(ctx, organizationID, tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Survivor", GradeLevelID: tenant.gradeID, HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	_, err = service.AddStudentToHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, survivor.ID, actor)
	require.NoError(t, err)
	require.NoError(t, service.DeleteHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, actor))

	membership, err = service.ListHouseholdMembership(ctx, organizationID, tenant.year.ID)
	require.NoError(t, err)
	require.Empty(t, studentIDsOf(membership.Students))
	households, err := service.ListHouseholds(ctx, organizationID, tenant.year.ID)
	require.NoError(t, err)
	require.Empty(t, households)
}

type membershipFixture struct {
	organizationID ids.XID
	year           data.SchoolYear
	gradeID        ids.XID
	homeroomID     ids.XID
	household      data.Household
	student        data.Student
	adult          data.Adult
}

// One organisation with one year, one household and one member of each kind.
// Every assertion in these tests is scoped to the fixture it created: the suite
// shares a database and isolates by organisation, so another test's rows are
// present.
func newMembershipFixture(t *testing.T, harness *testharness.Harness, service *people.Service, actor audit.Actor, label string) membershipFixture {
	t.Helper()
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "membership-grade", "Membership Grade "+label)
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Membership Room "+label)
	require.NoError(t, err)
	household, err := service.CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Membership Household " + label})
	require.NoError(t, err)
	student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Membership Student " + label, GradeLevelID: grade.ID, HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	adult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Membership Adult " + label, ParticipationIntent: data.AdultParticipationHelp,
	})
	require.NoError(t, err)
	_, err = service.AddStudentToHousehold(ctx, string(organizationID), year.ID, household.ID, student.ID, actor)
	require.NoError(t, err)
	_, err = service.AddAdultToHousehold(ctx, string(organizationID), year.ID, household.ID, adult.ID, actor)
	require.NoError(t, err)
	return membershipFixture{
		organizationID: organizationID, year: year, gradeID: grade.ID, homeroomID: homeroom.ID,
		household: household, student: student, adult: adult,
	}
}

func studentIDsOf(rows []data.HouseholdStudent) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.StudentID))
	}
	return result
}

func adultIDsOf(rows []data.HouseholdAdult) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.AdultID))
	}
	return result
}

func householdIDsOfStudentMemberships(rows []data.HouseholdStudent) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.HouseholdID))
	}
	return result
}

// A person detail page asks for one person's relationships. SPEC §8.2 makes the
// relationship a link between a named adult and a named student, so an unfiltered
// listing shows the whole school year's links on every person's page.
func TestGuardianRelationshipListFiltersToOnePerson(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian relationship filter test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "filter-grade", "Filter Grade")
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Filter Room")
	require.NoError(t, err)

	service := people.New(harness.Database)
	newStudent := func(name string) data.Student {
		student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
			LegalGivenName: "Synthetic", LegalFamilyName: name, GradeLevelID: grade.ID, HomeroomID: homeroom.ID,
		})
		require.NoError(t, err)
		return student
	}
	newAdult := func(name string) data.Adult {
		adult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
			LegalGivenName: "Synthetic", LegalFamilyName: name, ParticipationIntent: data.AdultParticipationHelp,
		})
		require.NoError(t, err)
		return adult
	}

	adultA, adultB := newAdult("Filter Adult A"), newAdult("Filter Adult B")
	studentA, studentB := newStudent("Filter Student A"), newStudent("Filter Student B")
	link := func(adult data.Adult, student data.Student) data.GuardianRelationship {
		relationship, err := service.CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{
			AdultID: adult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent,
		})
		require.NoError(t, err)
		return relationship
	}
	aToA, aToB, bToB := link(adultA, studentA), link(adultA, studentB), link(adultB, studentB)

	relationshipIDs := func(relationships []data.GuardianRelationship) []string {
		result := make([]string, 0, len(relationships))
		for _, relationship := range relationships {
			result = append(result, string(relationship.ID))
		}
		return result
	}

	byAdult, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{AdultID: adultA.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(aToA.ID), string(aToB.ID)}, relationshipIDs(byAdult))

	byStudent, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{StudentID: studentB.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(aToB.ID), string(bToB.ID)}, relationshipIDs(byStudent))

	byPair, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{AdultID: adultB.ID, StudentID: studentB.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(bToB.ID)}, relationshipIDs(byPair))

	// The zero filter still lists the year, scoped to this test's organisation.
	unfiltered, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(aToA.ID), string(aToB.ID), string(bToB.ID)}, relationshipIDs(unfiltered))
}
