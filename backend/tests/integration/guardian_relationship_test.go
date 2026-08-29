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

// SPEC §8.2: a student MAY have more than one guardian, and those guardians
// need not have any recorded relationship to one another. There is no entity
// that could hold one, so each edge is its own record and the child is the only
// thing the two adults share.
func TestAStudentMayHaveTwoGuardiansAndEachEdgeIsItsOwnRecord(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian relationship integration test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "guardian-grade", "Guardian Grade")
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Guardian Room", nil)
	require.NoError(t, err)

	service := people.New(harness.Database)
	student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Guardian Student", GradeLevelID: xidPtr(grade.ID), HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	firstAdult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Guardian Adult One", ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
	})
	require.NoError(t, err)
	secondAdult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Guardian Adult Two", ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
	})
	require.NoError(t, err)

	first, err := service.CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: firstAdult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	second, err := service.CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: secondAdult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipGuardian,
	})
	require.NoError(t, err)
	byStudent, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{StudentID: student.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(first.ID), string(second.ID)}, guardianRelationshipIDs(byStudent))

	nextType := data.GuardianRelationshipGrandparent
	updated, err := service.UpdateGuardianRelationship(ctx, string(organizationID), year.ID, first.ID, actor, people.GuardianRelationshipUpdateInput{RelationshipType: &nextType})
	require.NoError(t, err)
	require.Equal(t, nextType, updated.RelationshipType)
	require.Equal(t, data.GuardianRelationshipGuardian, second.RelationshipType)

	// Each edge is removed on its own. Nothing groups the two adults, so losing
	// one guardian leaves the other adult's relationship to the child untouched.
	require.NoError(t, service.DeleteGuardianRelationship(ctx, string(organizationID), year.ID, first.ID, actor))
	byStudent, err = service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{StudentID: student.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(second.ID)}, guardianRelationshipIDs(byStudent))

	var auditCount int64
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		auditCount, err = tx.Queries().CountAuditLog(ctx)
		return err
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, auditCount, int64(10))
}

// The guardian edge is the only family construct (SPEC §8.2), so the year-scoped
// listing is the query path every family-shaped answer is derived through. It
// gets its own tenancy proof (SPEC §9.2) rather than inheriting the row-level
// policy's, because a missing year predicate leaks within a tenant.
func TestGuardianRelationshipListingIsScopedToOneOrganizationAndYear(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	service := people.New(harness.Database)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian relationship listing test"}

	// Two tenants with the same shape, so a leak shows up as an extra row rather
	// than as an absent one.
	tenantA := newGuardianFixture(t, harness, service, actor, "A")
	tenantB := newGuardianFixture(t, harness, service, actor, "B")
	relationshipA, err := service.CreateGuardianRelationship(ctx, string(tenantA.organizationID), tenantA.year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: tenantA.adult.ID, StudentID: tenantA.student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	relationshipB, err := service.CreateGuardianRelationship(ctx, string(tenantB.organizationID), tenantB.year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: tenantB.adult.ID, StudentID: tenantB.student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)

	listA, err := service.ListGuardianRelationships(ctx, string(tenantA.organizationID), tenantA.year.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(relationshipA.ID)}, guardianRelationshipIDs(listA))

	// The other tenant's rows exist and are reachable only under its own scope.
	listB, err := service.ListGuardianRelationships(ctx, string(tenantB.organizationID), tenantB.year.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(relationshipB.ID)}, guardianRelationshipIDs(listB))

	// One tenant asking for the other's year gets nothing, not the other's rows.
	crossYear, err := service.ListGuardianRelationships(ctx, string(tenantA.organizationID), tenantB.year.ID, data.GuardianRelationshipFilter{})
	require.Error(t, err)
	require.Empty(t, crossYear)

	// A filter naming the other tenant's adult does not reach across either: the
	// filter narrows a scoped listing and never widens it.
	foreign, err := service.ListGuardianRelationships(ctx, string(tenantA.organizationID), tenantA.year.ID, data.GuardianRelationshipFilter{AdultID: tenantB.adult.ID})
	require.NoError(t, err)
	require.Empty(t, foreign)

	// A second year in the same organisation is a separate answer.
	secondYear, err := schoolyear.New(harness.Database).Create(ctx, string(tenantA.organizationID), actor, "2027–2028")
	require.NoError(t, err)
	secondYearList, err := service.ListGuardianRelationships(ctx, string(tenantA.organizationID), secondYear.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.Empty(t, secondYearList)
}

// SPEC §21.3: relationship rows remain historical records, but a live listing
// excludes a row if either endpoint has been soft-deleted. Each side gets a
// separate fixture relationship so both filtered directions are exercised.
func TestGuardianRelationshipListingExcludesSoftDeletedPeople(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	service := people.New(harness.Database)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian relationship soft delete test"}
	tenant := newGuardianFixture(t, harness, service, actor, "GuardianSoftDelete")
	organizationID := string(tenant.organizationID)

	studentB, err := service.CreateStudent(ctx, organizationID, tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Guardian Student B", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	adultB, err := service.Create(ctx, organizationID, tenant.year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Guardian Adult B", ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
	})
	require.NoError(t, err)

	adultDeletedRelationship, err := service.CreateGuardianRelationship(ctx, organizationID, tenant.year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: adultB.ID, StudentID: tenant.student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	studentDeletedRelationship, err := service.CreateGuardianRelationship(ctx, organizationID, tenant.year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: tenant.adult.ID, StudentID: studentB.ID, RelationshipType: data.GuardianRelationshipGuardian,
	})
	require.NoError(t, err)

	byStudent, err := service.ListGuardianRelationships(ctx, organizationID, tenant.year.ID, data.GuardianRelationshipFilter{StudentID: tenant.student.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(adultDeletedRelationship.ID)}, guardianRelationshipIDs(byStudent))
	byAdult, err := service.ListGuardianRelationships(ctx, organizationID, tenant.year.ID, data.GuardianRelationshipFilter{AdultID: tenant.adult.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(studentDeletedRelationship.ID)}, guardianRelationshipIDs(byAdult))

	require.NoError(t, service.Delete(ctx, organizationID, tenant.year.ID, adultB.ID, actor))
	byStudent, err = service.ListGuardianRelationships(ctx, organizationID, tenant.year.ID, data.GuardianRelationshipFilter{StudentID: tenant.student.ID})
	require.NoError(t, err)
	require.Empty(t, byStudent)
	retainedAdultDeleted, err := service.GetGuardianRelationship(ctx, organizationID, tenant.year.ID, adultDeletedRelationship.ID)
	require.NoError(t, err)
	require.Equal(t, adultDeletedRelationship.ID, retainedAdultDeleted.ID)

	require.NoError(t, service.DeleteStudent(ctx, organizationID, tenant.year.ID, studentB.ID, actor))
	byAdult, err = service.ListGuardianRelationships(ctx, organizationID, tenant.year.ID, data.GuardianRelationshipFilter{AdultID: tenant.adult.ID})
	require.NoError(t, err)
	require.Empty(t, byAdult)
	retainedStudentDeleted, err := service.GetGuardianRelationship(ctx, organizationID, tenant.year.ID, studentDeletedRelationship.ID)
	require.NoError(t, err)
	require.Equal(t, studentDeletedRelationship.ID, retainedStudentDeleted.ID)

	unfiltered, err := service.ListGuardianRelationships(ctx, organizationID, tenant.year.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.Empty(t, unfiltered)

	// Restoring both people restores both rows, with no mutation of the link
	// rows themselves: the exclusion is a read-time predicate (SPEC §21.3).
	restoreAdult(t, harness, organizationID, adultB.ID)
	restoreStudent(t, harness, organizationID, studentB.ID)
	restored, err := service.ListGuardianRelationships(ctx, organizationID, tenant.year.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.ElementsMatch(t,
		[]string{string(adultDeletedRelationship.ID), string(studentDeletedRelationship.ID)},
		guardianRelationshipIDs(restored))
}

// SPEC §21.3 requires soft deletion to be reversible, but the restore endpoint
// is tracked separately in #103. Until it lands, clearing deleted_at directly
// is the closest available stand-in, and it is the stricter check for this
// issue: nothing but the person's own flag changes, so a row that reappears in
// a listing proves the exclusion is a read-time predicate and not a mutation.
func restoreStudent(t *testing.T, harness *testharness.Harness, organizationID string, studentID ids.XID) {
	t.Helper()
	restorePerson(t, harness, organizationID, studentID,
		`update students set deleted_at = null where id = $1 and organization_id = $2`)
}

func restoreAdult(t *testing.T, harness *testharness.Harness, organizationID string, adultID ids.XID) {
	t.Helper()
	restorePerson(t, harness, organizationID, adultID,
		`update adults set deleted_at = null where id = $1 and organization_id = $2`)
}

func restorePerson(t *testing.T, harness *testharness.Harness, organizationID string, personID ids.XID, statement string) {
	t.Helper()
	ctx := harness.Context
	tx, err := harness.Migrator.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	// Row-level security is forced on the person tables (SPEC §9.2), so this
	// transaction sets the same tenant setting an application transaction sets.
	_, err = tx.Exec(ctx, `select set_config('app.organization_id', $1, true)`, organizationID)
	require.NoError(t, err)
	tag, err := tx.Exec(ctx, statement, string(personID), organizationID)
	require.NoError(t, err)
	require.Equal(t, int64(1), tag.RowsAffected())
	require.NoError(t, tx.Commit(ctx))
}

type guardianFixture struct {
	organizationID ids.XID
	year           data.SchoolYear
	gradeID        ids.XID
	homeroomID     ids.XID
	student        data.Student
	adult          data.Adult
}

// One organisation with one year and one person of each kind, and deliberately
// no guardian edge between them: each test states the edges it needs. Every
// assertion in these tests is scoped to the fixture it created, because the
// suite shares a database and isolates by organisation, so another test's rows
// are present.
func newGuardianFixture(t *testing.T, harness *testharness.Harness, service *people.Service, actor audit.Actor, label string) guardianFixture {
	t.Helper()
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "relationship-grade", "Relationship Grade "+label)
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Relationship Room "+label, nil)
	require.NoError(t, err)
	student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Relationship Student " + label, GradeLevelID: xidPtr(grade.ID), HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	adult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Relationship Adult " + label, ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
	})
	require.NoError(t, err)
	return guardianFixture{
		organizationID: organizationID, year: year, gradeID: grade.ID, homeroomID: homeroom.ID,
		student: student, adult: adult,
	}
}

func guardianRelationshipIDs(rows []data.GuardianRelationship) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.ID))
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
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Filter Room", nil)
	require.NoError(t, err)

	service := people.New(harness.Database)
	newStudent := func(name string) data.Student {
		student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
			LegalGivenName: "Synthetic", LegalFamilyName: name, GradeLevelID: xidPtr(grade.ID), HomeroomID: homeroom.ID,
		})
		require.NoError(t, err)
		return student
	}
	newAdult := func(name string) data.Adult {
		adult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
			LegalGivenName: "Synthetic", LegalFamilyName: name, ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
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

	byAdult, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{AdultID: adultA.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(aToA.ID), string(aToB.ID)}, guardianRelationshipIDs(byAdult))

	byStudent, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{StudentID: studentB.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(aToB.ID), string(bToB.ID)}, guardianRelationshipIDs(byStudent))

	byPair, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{AdultID: adultB.ID, StudentID: studentB.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(bToB.ID)}, guardianRelationshipIDs(byPair))

	// The zero filter still lists the year, scoped to this test's organisation.
	unfiltered, err := service.ListGuardianRelationships(ctx, string(organizationID), year.ID, data.GuardianRelationshipFilter{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{string(aToA.ID), string(aToB.ID), string(bToB.ID)}, guardianRelationshipIDs(unfiltered))
}
