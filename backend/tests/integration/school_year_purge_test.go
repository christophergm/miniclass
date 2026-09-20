package integration

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/stretchr/testify/require"
)

func TestSchoolYearPurgeRetainsShellAndDoesNotCrossTenantOrYear(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	foreignOrganizationID := harness.MintOrganization(t)
	adminActor := audit.Actor{Type: audit.ActorTypeSystem, Label: "school-year purge fixture"}

	var userID ids.XID
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, "school-year-purge-owner-"+string(organizationID), "purge-owner@example.test").Scan(&userID))

	service := schoolyear.New(harness.Database)
	factory := factories.New(harness.Database, string(organizationID), adminActor)
	target, err := factory.CreateSchoolYear(ctx, "Synthetic purge target")
	require.NoError(t, err)
	sibling, err := factory.CreateSchoolYear(ctx, "Synthetic retained sibling")
	require.NoError(t, err)
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), adminActor)
	foreignYear, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic foreign year")
	require.NoError(t, err)

	targetGrade, err := factory.CreateGradeLevel(ctx, target.ID, "synthetic-target", "Synthetic Target Grade")
	require.NoError(t, err)
	targetHomeroom, err := factory.CreateHomeroom(ctx, target.ID, "Synthetic Target Homeroom")
	require.NoError(t, err)
	targetStudent, err := factory.CreateStudent(ctx, target.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Purge Target",
		GradeLevelID: &targetGrade.ID, HomeroomID: targetHomeroom.ID,
	})
	require.NoError(t, err)
	targetAdult, err := factory.CreateAdult(ctx, target.ID, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Purge Adult",
	})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, target.ID, people.GuardianRelationshipCreateInput{
		AdultID: targetAdult.ID, StudentID: targetStudent.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	targetProgram, err := factory.CreateProgram(ctx, target.ID, "Synthetic Purge Programme")
	require.NoError(t, err)
	_, err = factory.CreateInterestArea(ctx, target.ID, targetProgram.ID, "Synthetic Purge Area")
	require.NoError(t, err)
	targetSession, err := factory.CreateSession(ctx, target.ID, targetProgram.ID, "Synthetic Purge Session", []time.Time{
		time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, target.ID, targetProgram.ID, targetStudent.ID)
	require.NoError(t, err)
	_, err = factory.CreateSessionNonParticipation(ctx, target.ID, targetProgram.ID, targetSession.ID, targetStudent.ID, "Synthetic purge fixture")
	require.NoError(t, err)
	_, err = factory.CreateOffering(ctx, target.ID, targetProgram.ID, targetSession.ID, "Synthetic Purge Offering", "", nil, 20, targetGrade.ID, targetGrade.ID, "", "", "", nil)
	require.NoError(t, err)

	siblingGrade, err := factory.CreateGradeLevel(ctx, sibling.ID, "synthetic-sibling", "Synthetic Sibling Grade")
	require.NoError(t, err)
	siblingHomeroom, err := factory.CreateHomeroom(ctx, sibling.ID, "Synthetic Sibling Homeroom")
	require.NoError(t, err)
	_, err = factory.CreateStudent(ctx, sibling.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Retained Sibling",
		GradeLevelID: &siblingGrade.ID, HomeroomID: siblingHomeroom.ID,
	})
	require.NoError(t, err)
	foreignGrade, err := foreignFactory.CreateGradeLevel(ctx, foreignYear.ID, "synthetic-foreign", "Synthetic Foreign Grade")
	require.NoError(t, err)
	foreignHomeroom, err := foreignFactory.CreateHomeroom(ctx, foreignYear.ID, "Synthetic Foreign Homeroom")
	require.NoError(t, err)
	_, err = foreignFactory.CreateStudent(ctx, foreignYear.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Foreign Retained",
		GradeLevelID: &foreignGrade.ID, HomeroomID: foreignHomeroom.ID,
	})
	require.NoError(t, err)

	// A newly created year starts with its own empty vocabulary; no prior-year
	// source is consulted or copied.
	newYear, err := factory.CreateSchoolYear(ctx, "Synthetic independent year")
	require.NoError(t, err)
	vocabularySnapshot, err := vocabulary.New(harness.Database).List(ctx, string(organizationID), newYear.ID, true)
	require.NoError(t, err)
	require.Empty(t, vocabularySnapshot.Grades)

	_, err = service.Update(ctx, string(organizationID), target.ID, authRoleOwner, adminActor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = service.Update(ctx, string(organizationID), target.ID, authRoleAdministrator, adminActor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	userActor := audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: "purge-owner@example.test"}
	_, err = service.Purge(ctx, string(organizationID), target.ID, authRoleAdministrator, userActor, "PURGE Synthetic purge target")
	require.ErrorIs(t, err, schoolyear.ErrPurgeOwnerRequired)
	_, err = service.Purge(ctx, string(organizationID), target.ID, authRoleOwner, userActor, "wrong confirmation")
	require.ErrorIs(t, err, schoolyear.ErrPurgeConfirmationRequired)

	purged, err := service.Purge(ctx, string(organizationID), target.ID, authRoleOwner, userActor, "PURGE Synthetic purge target")
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearPurged, purged.State)
	require.Equal(t, userID, *purged.PurgedByUserID)
	require.NotNil(t, purged.PurgedAt)

	// Every current school-year table is checked by metadata rather than a
	// hand-maintained list, so adding a new year-scoped table without adding it
	// to the purge graph makes this regression fail.
	rows, err := harness.Migrator.Query(ctx, `
		select table_name
		from information_schema.columns
		where table_schema = current_schema()
		  and column_name = 'school_year_id'
		  and table_name not in ('school_years', 'audit_log')
		group by table_name
		order by table_name`)
	require.NoError(t, err)
	var tableNames []string
	for rows.Next() {
		var tableName string
		require.NoError(t, rows.Scan(&tableName))
		tableNames = append(tableNames, tableName)
	}
	require.NoError(t, rows.Err())
	rows.Close()

	migratorTx, err := harness.Migrator.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = migratorTx.Rollback(ctx) }()
	_, err = migratorTx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID)
	require.NoError(t, err)
	for _, tableName := range tableNames {
		query := "select count(*) from " + quoteIdentifier(harness.Schema) + "." + quoteIdentifier(tableName) + " where organization_id = $1 and school_year_id = $2"
		var count int64
		require.NoError(t, migratorTx.QueryRow(ctx, query, organizationID, target.ID).Scan(&count), tableName)
		require.Zero(t, count, tableName+" retained purged-year rows")
	}

	var auditCount int64
	var action, actorLabel string
	require.NoError(t, migratorTx.QueryRow(ctx, `
		select count(*), min(action), min(actor_label)
		from audit_log
		where organization_id = $1 and school_year_id = $2`, organizationID, target.ID).Scan(&auditCount, &action, &actorLabel))
	require.Equal(t, int64(1), auditCount)
	require.Equal(t, string(audit.ActionSchoolYearPurge), action)
	require.Equal(t, "Owner", actorLabel)

	retainedStudents, err := factory.CreateStudent(ctx, sibling.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Second Sibling",
		GradeLevelID: &siblingGrade.ID, HomeroomID: siblingHomeroom.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, retainedStudents.ID)
	listed, err := service.Get(ctx, string(organizationID), sibling.ID)
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearSetup, listed.State)

	foreignStudents, err := factory.CreateStudent(ctx, sibling.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Same Tenant Different Year",
		GradeLevelID: &siblingGrade.ID, HomeroomID: siblingHomeroom.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, foreignStudents.ID)
	foreignVisible, err := people.New(harness.Database).ListStudents(ctx, string(foreignOrganizationID), foreignYear.ID, false)
	require.NoError(t, err)
	require.Len(t, foreignVisible, 1)

	var priorYearColumnCount int64
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		select count(*)
		from information_schema.columns
		where table_schema = current_schema()
		  and table_name = 'students'
		  and column_name = 'prior_year_student_id'`).Scan(&priorYearColumnCount))
	require.Zero(t, priorYearColumnCount)

	_, err = service.Update(ctx, string(organizationID), target.ID, authRoleOwner, userActor, schoolyear.UpdateInput{Label: stringPointer("purged edit")})
	require.True(t, data.IsSchoolYearPurged(err), "purged edit = %v", err)

	var shellCount int64
	require.NoError(t, migratorTx.QueryRow(ctx, `
		select count(*) from school_years where organization_id = $1 and id = $2 and state = 'purged'`, organizationID, target.ID).Scan(&shellCount))
	require.Equal(t, int64(1), shellCount)

}
