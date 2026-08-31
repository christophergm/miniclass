package integration

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestSessionNonParticipationPreservesMembershipAndExcludesParticipant(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "non-participation integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic non-participation year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-non-participation", "Synthetic Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Room")
	require.NoError(t, err)
	first, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "First", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	second, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Second", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Programme")
	require.NoError(t, err)
	firstMembership, err := factory.AddProgramMembership(ctx, year.ID, programRow.ID, first.ID)
	require.NoError(t, err)
	secondMembership, err := factory.AddProgramMembership(ctx, year.ID, programRow.ID, second.ID)
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Session", 1, []time.Time{time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	service := program.New(harness.Database)

	record, err := service.CreateSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, first.ID, "medical appointment")
	require.NoError(t, err)
	require.Equal(t, first.ID, record.StudentID)
	require.Equal(t, "medical appointment", record.Reason)
	_, err = service.CreateSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, first.ID, "duplicate reason")
	var duplicate *pgconn.PgError
	require.ErrorAs(t, err, &duplicate)
	require.Equal(t, "23505", duplicate.Code)

	nonParticipants, err := service.ListSessionNonParticipations(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, nonParticipants, 1)
	require.Equal(t, record.ID, nonParticipants[0].ID)
	memberships, err := service.ListMemberships(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{firstMembership.ID, secondMembership.ID}, []ids.XID{memberships[0].ID, memberships[1].ID})
	participating, err := service.ListParticipatingMemberships(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, participating, 1)
	require.Equal(t, second.ID, participating[0].StudentID)

	newReason := "family travel"
	updated, err := service.UpdateSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, record.ID, program.SessionNonParticipationUpdate{Reason: &newReason})
	require.NoError(t, err)
	require.Equal(t, newReason, updated.Reason)
	require.NoError(t, service.DeleteSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, record.ID))
	participating, err = service.ListParticipatingMemberships(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, participating, 2)

	objectType := "session_non_participation"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 3)
	for _, entry := range entries {
		require.Equal(t, string(audit.ActionSessionNonParticipation), entry.Action)
		require.NotEmpty(t, entry.Reason)
	}
}

func TestClosedYearSessionNonParticipationMutationsAreRejected(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed non-participation integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic closed non-participation year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-closed-non-participation", "Synthetic Closed Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Closed Room")
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Closed", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Closed Programme")
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Closed Session", 1, []time.Time{time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	service := program.New(harness.Database)
	record, err := service.CreateSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, student.ID, "temporary absence")
	require.NoError(t, err)

	closed := data.SchoolYearClosed
	yearService := schoolyear.New(harness.Database)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(closed)})
	require.NoError(t, err)

	reason := "closed edit"
	_, err = service.UpdateSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, record.ID, program.SessionNonParticipationUpdate{Reason: &reason})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year non-participation edit = %v", err)
	_, err = service.CreateSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, student.ID, "closed create")
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year non-participation create = %v", err)
	err = service.DeleteSessionNonParticipation(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, record.ID)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year non-participation delete = %v", err)
}
