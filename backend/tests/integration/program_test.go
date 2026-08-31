package integration

import (
	"context"
	"errors"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestProgramMembershipRequiresGradeAndFlagsLaterRemoval(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "program integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic program year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-program", "Synthetic Program Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Program Room")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Enrichment")
	require.NoError(t, err)
	ungraded, err := factory.CreateUngradedStudent(ctx, year.ID, homeroom.ID, "Ungraded", "Synthetic")
	require.NoError(t, err)

	service := program.New(harness.Database)
	_, err = service.AddMembership(ctx, string(organizationID), actor, year.ID, programRow.ID, ungraded.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, program.ErrStudentGradeRequired))
	require.Contains(t, err.Error(), "Ungraded Synthetic")

	graded, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{
		LegalGivenName: "Graded", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	membership, err := service.AddMembership(ctx, string(organizationID), actor, year.ID, programRow.ID, graded.ID)
	require.NoError(t, err)
	require.False(t, membership.GradeMissing)
	require.Equal(t, "Graded", membership.LegalGivenName)

	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByID(ctx, year.ID, graded.ID)
		if err != nil {
			return err
		}
		updated, err := tx.UpdateStudent(ctx, year.ID, graded.ID, current.LegalGivenName, current.LegalFamilyName, current.PreferredGivenName, nil, current.HomeroomID, current.ExternalIdentifier, current.PriorYearStudentID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "student", ObjectID: &updated.ID, SchoolYearID: &updated.SchoolYearID, Reason: "synthetic grade removal", ChangeSummary: []byte(`{"grade_level_id":null}`)})
	})
	require.NoError(t, err)

	memberships, err := service.ListMemberships(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	require.True(t, memberships[0].GradeMissing)
	require.Equal(t, graded.ID, memberships[0].StudentID)

	count, err := service.CountStudentsWithoutGrade(ctx, string(organizationID), year.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}

func TestInterestAreaVocabularyPreservesIdentityAndAuditsChanges(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "interest-area integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic interest area year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic enrichment")
	require.NoError(t, err)
	service := program.New(harness.Database)

	first, err := service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Arts and crafts")
	require.NoError(t, err)
	second, err := service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Gardening")
	require.NoError(t, err)
	require.Equal(t, 1, first.Ordinal)
	require.Equal(t, 2, second.Ordinal)

	listed, err := service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	newLabel := "Arts & crafts"
	updated, err := service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Label: &newLabel})
	require.NoError(t, err)
	require.Equal(t, first.ID, updated.ID)
	require.Equal(t, newLabel, updated.Label)
	require.Equal(t, first.Ordinal, updated.Ordinal)

	retired := true
	updated, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Retired: &retired})
	require.NoError(t, err)
	require.NotNil(t, updated.RetiredAt)

	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID}, []ids.XID{listed[0].ID}, "retired areas are excluded from selection lists")
	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, true)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	retired = false
	updated, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Retired: &retired})
	require.NoError(t, err)
	require.Nil(t, updated.RetiredAt)
	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	objectType := "interest_area"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 5, "each vocabulary create, edit, retirement, and reactivation is audited")
	firstChanges, secondChanges := 0, 0
	for _, entry := range entries {
		require.Equal(t, string(audit.ActionVocabularyChange), entry.Action)
		require.NotNil(t, entry.ObjectID)
		switch string(*entry.ObjectID) {
		case string(first.ID):
			firstChanges++
		case string(second.ID):
			secondChanges++
		default:
			t.Errorf("unexpected interest-area audit object %q", string(*entry.ObjectID))
		}
	}
	require.Equal(t, 4, firstChanges)
	require.Equal(t, 1, secondChanges)
}

func TestClosedYearInterestAreaMutationIsRejected(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed interest-area integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic closed interest area year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic closed enrichment")
	require.NoError(t, err)
	service := program.New(harness.Database)
	area, err := service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Synthetic area")
	require.NoError(t, err)

	yearService := schoolyear.New(harness.Database)
	year, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	newLabel := "Closed edit"
	_, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, area.ID, program.InterestAreaUpdate{Label: &newLabel})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year label edit = %v", err)

	_, err = service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Closed create")
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year area create = %v", err)
}

func TestSessionsUseExplicitOrdinalsAndOwnMeetingDates(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "session integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic session year")
	require.NoError(t, err)
	nextYear, err := factory.CreateSchoolYear(ctx, "Synthetic next session year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic session programme")
	require.NoError(t, err)
	nextProgram, err := factory.CreateProgram(ctx, nextYear.ID, "Synthetic next programme")
	require.NoError(t, err)
	service := program.New(harness.Database)

	firstDate := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	secondDate := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	first, err := service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "First synthetic session", 7, []time.Time{firstDate, secondDate})
	require.NoError(t, err)
	require.Equal(t, data.SessionPlanning, first.State)
	require.Equal(t, 7, first.Ordinal)
	require.Equal(t, []time.Time{firstDate, secondDate}, first.MeetingDates)

	second, err := service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "Second synthetic session", 2, []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	_, err = service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "Duplicate ordinal", 2, []time.Time{time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)})
	var duplicate *pgconn.PgError
	require.ErrorAs(t, err, &duplicate)
	require.Equal(t, "23505", duplicate.Code)

	listed, err := service.ListSessions(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID, first.ID}, []ids.XID{listed[0].ID, listed[1].ID})
	require.Equal(t, []string{"2026-11-06"}, []string{listed[0].MeetingDates[0].Format("2006-01-02")})
	require.Equal(t, []string{"2026-10-02", "2026-10-16"}, []string{listed[1].MeetingDates[0].Format("2006-01-02"), listed[1].MeetingDates[1].Format("2006-01-02")})

	newName, newOrdinal := "Renamed synthetic session", 1
	updated, err := service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.SessionUpdate{Name: &newName, Ordinal: &newOrdinal})
	require.NoError(t, err)
	require.Equal(t, newName, updated.Name)
	require.Equal(t, newOrdinal, updated.Ordinal)

	dates, err := service.ListMeetingDates(ctx, string(organizationID), year.ID, programRow.ID, first.ID)
	require.NoError(t, err)
	require.Len(t, dates, 2)
	changedDate := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	_, err = service.UpdateMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, dates[0].ID, changedDate)
	require.NoError(t, err)
	require.NoError(t, service.DeleteMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, dates[1].ID))
	err = service.DeleteMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, dates[0].ID)
	require.ErrorIs(t, err, program.ErrSessionRequiresMeetingDate)

	_, err = service.GetSession(ctx, string(organizationID), nextYear.ID, nextProgram.ID, first.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	objectType := "meeting_date"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 4)
}

func TestClosedYearSessionAndMeetingDateMutationsAreRejected(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed session integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic closed session year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic closed session programme")
	require.NoError(t, err)
	date := time.Date(2026, 12, 4, 0, 0, 0, 0, time.UTC)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic closed session", 1, []time.Time{date})
	require.NoError(t, err)
	meetingDates, err := serviceMeetingDates(ctx, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
	require.NoError(t, err)

	yearService := schoolyear.New(harness.Database)
	year, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	service := program.New(harness.Database)
	newName := "Closed edit"
	_, err = service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionUpdate{Name: &newName})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year session edit = %v", err)
	_, err = service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "Closed create", 2, []time.Time{date.AddDate(0, 0, 7)})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year session create = %v", err)
	_, err = service.UpdateMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, meetingDates[0].ID, date.AddDate(0, 0, 7))
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year meeting date edit = %v", err)
	err = service.DeleteMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, meetingDates[0].ID)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year meeting date delete = %v", err)
	err = service.DeleteSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year session delete = %v", err)
}

func serviceMeetingDates(ctx context.Context, database *data.DB, organizationID ids.XID, yearID, programID, sessionID ids.XID) ([]data.MeetingDate, error) {
	return program.New(database).ListMeetingDates(ctx, string(organizationID), yearID, programID, sessionID)
}
