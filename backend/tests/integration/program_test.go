package integration

import (
	"context"
	"encoding/json"
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

	reordered, err := service.ReorderInterestAreas(ctx, string(organizationID), actor, year.ID, programRow.ID, []ids.XID{second.ID, first.ID})
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID, first.ID}, []ids.XID{reordered[0].ID, reordered[1].ID})
	require.Equal(t, 1, reordered[0].Ordinal)
	require.Equal(t, 2, reordered[1].Ordinal)

	newLabel := "Arts & crafts"
	updated, err := service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Label: &newLabel})
	require.NoError(t, err)
	require.Equal(t, first.ID, updated.ID)
	require.Equal(t, newLabel, updated.Label)
	require.Equal(t, 2, updated.Ordinal)

	retired := true
	updated, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Retired: &retired})
	require.NoError(t, err)
	require.NotNil(t, updated.RetiredAt)

	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID}, []ids.XID{listed[0].ID}, "retired areas are excluded from selection lists")
	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, true)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID, first.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	retired = false
	updated, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Retired: &retired})
	require.NoError(t, err)
	require.Nil(t, updated.RetiredAt)
	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID, first.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	objectType := "interest_area"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 6, "each vocabulary create, reorder, edit, retirement, and reactivation is audited")
	firstChanges, secondChanges, reorderChanges := 0, 0, 0
	for _, entry := range entries {
		require.Equal(t, string(audit.ActionVocabularyChange), entry.Action)
		if entry.ObjectID == nil {
			reorderChanges++
			continue
		}
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
	require.Equal(t, 1, reorderChanges)
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

func TestSessionsOrderByFirstMeetingDateAndOwnMeetingDates(t *testing.T) {
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
	first, err := service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "First synthetic session", []time.Time{firstDate, secondDate})
	require.NoError(t, err)
	require.Equal(t, data.SessionPlanning, first.State)
	require.Equal(t, []time.Time{secondDate, firstDate}, first.MeetingDates)

	second, err := service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "Second synthetic session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	alpha, err := service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "Alpha synthetic session", []time.Time{time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	beta, err := service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "beta synthetic session", []time.Time{time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)

	listed, err := service.ListSessions(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, alpha.ID, beta.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID, listed[2].ID, listed[3].ID})

	newName := "Renamed synthetic session"
	updatedDates := []time.Time{time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)}
	updated, err := service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.SessionUpdate{Name: &newName, Dates: &updatedDates})
	require.NoError(t, err)
	require.Equal(t, newName, updated.Name)
	require.Equal(t, updatedDates, updated.MeetingDates)

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

func TestSessionLifecycleTransitionsWarnPreserveDraftsAndAudit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "session lifecycle integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic lifecycle year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic lifecycle programme")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic lifecycle session", []time.Time{time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	service := program.New(harness.Database)

	transition := func(next data.SessionState, confirm bool, reason string) program.SessionTransitionResult {
		result, err := service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionTransitionInput{NextState: next, Confirm: confirm, Reason: reason})
		require.NoError(t, err)
		return result
	}

	result := transition(data.SessionCatalogPublished, false, "")
	require.True(t, result.Applied)
	_, err = service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionTransitionInput{NextState: data.SessionVotingOpen})
	require.ErrorIs(t, err, program.ErrSessionTransitionGate)

	minimumGrade, err := factory.CreateGradeLevel(ctx, year.ID, "1", "Synthetic Grade One")
	require.NoError(t, err)
	maximumGrade, err := factory.CreateGradeLevel(ctx, year.ID, "6", "Synthetic Grade Six")
	require.NoError(t, err)
	_, err = factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic offering", "Synthetic description", nil, 12, minimumGrade.ID, maximumGrade.ID, "Synthetic room", "Synthetic entrance", "Synthetic directions", nil)
	require.NoError(t, err)

	for _, next := range []data.SessionState{data.SessionVotingOpen, data.SessionVotingClosed, data.SessionAssigning, data.SessionPublished} {
		result = transition(next, false, "")
		require.True(t, result.Applied)
	}

	preview := transition(data.SessionAssigning, false, "")
	require.False(t, preview.Applied)
	require.True(t, preview.RequiresConfirmation)
	require.Equal(t, data.SessionPublished, preview.Session.State)
	require.Contains(t, transitionWarningCodes(preview.Warnings), "stale-draft")
	require.Contains(t, transitionWarningCodes(preview.Warnings), "published-links-invalidated")

	_, err = service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionTransitionInput{NextState: data.SessionAssigning, Confirm: true})
	require.ErrorIs(t, err, program.ErrSessionTransitionReasonRequired)
	unchanged, err := service.GetSession(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, data.SessionPublished, unchanged.State)

	result = transition(data.SessionAssigning, true, "late family correction")
	require.True(t, result.Applied)
	require.True(t, result.Session.DraftAssignmentsStale)
	_, err = service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionTransitionInput{NextState: data.SessionPublished, Confirm: true})
	require.ErrorIs(t, err, program.ErrSessionTransitionGate)

	stored, err := service.GetSession(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, data.SessionAssigning, stored.State)
	require.True(t, stored.DraftAssignmentsStale)

	objectType := "session"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	var foundTransition bool
	for _, entry := range entries {
		if entry.Action != string(audit.ActionSessionStateTransition) || entry.ObjectID == nil || *entry.ObjectID != session.ID || entry.Reason.String != "late family correction" {
			continue
		}
		foundTransition = true
		require.True(t, entry.OccurredAt.Valid)
		var summary struct {
			FromState string `json:"from_state"`
			ToState   string `json:"to_state"`
			Backward  bool   `json:"backward"`
		}
		require.NoError(t, json.Unmarshal(entry.ChangeSummary, &summary))
		require.Equal(t, string(data.SessionPublished), summary.FromState)
		require.Equal(t, string(data.SessionAssigning), summary.ToState)
		require.True(t, summary.Backward)
	}
	require.True(t, foundTransition, "confirmed backward transition was not audited")

	completeSession, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic complete session", []time.Time{time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	completeTransition := func(next data.SessionState) {
		result, err := service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, completeSession.ID, program.SessionTransitionInput{NextState: next})
		require.NoError(t, err)
		require.True(t, result.Applied)
	}
	for _, next := range []data.SessionState{data.SessionCatalogPublished, data.SessionAssigning, data.SessionPublished, data.SessionComplete} {
		completeTransition(next)
	}
	_, err = service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, completeSession.ID, program.SessionUpdate{Name: stringPtr("not allowed")})
	require.ErrorIs(t, err, program.ErrSessionReadOnly)
	_, err = service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, completeSession.ID, program.SessionTransitionInput{NextState: data.SessionPlanning})
	require.ErrorIs(t, err, program.ErrSessionTransitionInvalid)
}

func stringPtr(value string) *string { return &value }

func transitionWarningCodes(warnings []program.SessionTransitionWarning) []string {
	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, warning.Code)
	}
	return result
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
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic closed session", []time.Time{date})
	require.NoError(t, err)
	_, err = factory.CreateMeetingDate(ctx, year.ID, programRow.ID, session.ID, date.AddDate(0, 0, 7))
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
	_, err = service.CreateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, "Closed create", []time.Time{date.AddDate(0, 0, 7)})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year session create = %v", err)
	_, err = service.UpdateMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, meetingDates[0].ID, date.AddDate(0, 0, 7))
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year meeting date edit = %v", err)
	err = service.DeleteMeetingDate(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, meetingDates[0].ID)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year meeting date delete = %v", err)
	_, err = service.TransitionSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionTransitionInput{NextState: data.SessionCatalogPublished})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year session transition = %v", err)
	err = service.DeleteSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year session delete = %v", err)
}

func serviceMeetingDates(ctx context.Context, database *data.DB, organizationID ids.XID, yearID, programID, sessionID ids.XID) ([]data.MeetingDate, error) {
	return program.New(database).ListMeetingDates(ctx, string(organizationID), yearID, programID, sessionID)
}

func TestOfferingsSupportGradeWindowsAndOptionalInterestAreas(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "offering integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic offering year")
	require.NoError(t, err)
	minimumGrade, err := factory.CreateGradeLevel(ctx, year.ID, "g3", "Synthetic Grade Three")
	require.NoError(t, err)
	maximumGrade, err := factory.CreateGradeLevel(ctx, year.ID, "g6", "Synthetic Grade Six")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic offering programme")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic offering session", []time.Time{time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	area, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic making")
	require.NoError(t, err)
	service := program.New(harness.Database)
	minimum := 3
	offering, err := service.CreateOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, "Synthetic Making", "Build a synthetic project", &minimum, 12, minimumGrade.ID, maximumGrade.ID, "Synthetic studio", "Synthetic foyer", "Synthetic instructions", nil)
	require.NoError(t, err)
	require.Equal(t, "Synthetic Making", offering.Name)
	require.Equal(t, "Build a synthetic project", offering.Description)
	require.Equal(t, &minimum, offering.MinimumViableEnrollment)
	require.Nil(t, offering.InterestAreaID)

	listed, err := service.ListOfferings(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, offering.ID, listed[0].ID)
	fetched, err := service.GetOffering(ctx, string(organizationID), year.ID, programRow.ID, session.ID, offering.ID)
	require.NoError(t, err)
	require.Equal(t, offering.ID, fetched.ID)

	newName := "Synthetic Making Updated"
	newCapacity := 15
	newMinimum := 4
	updated, err := service.UpdateOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, offering.ID, program.OfferingUpdate{Name: &newName, Capacity: &newCapacity, MinimumViableEnrollment: &newMinimum, InterestAreaID: &area.ID})
	require.NoError(t, err)
	require.Equal(t, newName, updated.Name)
	require.Equal(t, newCapacity, updated.Capacity)
	require.Equal(t, &newMinimum, updated.MinimumViableEnrollment)
	require.Equal(t, &area.ID, updated.InterestAreaID)

	_, err = service.CreateOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, "Impossible synthetic window", "Description", nil, 10, maximumGrade.ID, minimumGrade.ID, "", "", "", nil)
	require.ErrorIs(t, err, program.ErrOfferingGradeOrder)

	objectType := "offering"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		require.Equal(t, string(audit.ActionOfferingEdit), entry.Action)
		require.NotNil(t, entry.ObjectID)
		require.Equal(t, offering.ID, *entry.ObjectID)
		require.Equal(t, year.ID, *entry.SchoolYearID)
	}

	require.NoError(t, service.DeleteOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, offering.ID))
	_, err = service.GetOffering(ctx, string(organizationID), year.ID, programRow.ID, session.ID, offering.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestClosedYearOfferingMutationsAreRejected(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed offering integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic closed offering year")
	require.NoError(t, err)
	minimumGrade, err := factory.CreateGradeLevel(ctx, year.ID, "g1", "Synthetic Grade One")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic closed offering programme")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic closed offering session", []time.Time{time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic closed offering", "Description", nil, 10, minimumGrade.ID, minimumGrade.ID, "", "", "", nil)
	require.NoError(t, err)

	yearService := schoolyear.New(harness.Database)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	service := program.New(harness.Database)
	newName := "Closed offering edit"
	_, err = service.UpdateOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, offering.ID, program.OfferingUpdate{Name: &newName})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year offering edit = %v", err)
	_, err = service.CreateOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, "Closed offering create", "Description", nil, 10, minimumGrade.ID, minimumGrade.ID, "", "", "", nil)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year offering create = %v", err)
	err = service.DeleteOffering(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, offering.ID)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year offering delete = %v", err)
}

func TestObjectiveWeightsDefaultOverrideAndClearFallback(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	otherOrganizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "objective weights integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic objective weights year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic objective weights program")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic objective weights session", []time.Time{time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	service := program.New(harness.Database)

	defaults, err := service.GetProgramObjectiveWeights(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Equal(t, 3, defaults.Effective.RankHighMax)
	require.Equal(t, 10.0, defaults.Effective.RepeatOfferingPenalty)
	require.Equal(t, defaults.Defaults, defaults.Effective)

	updatedDefaults := defaults.Defaults
	updatedDefaults.RepeatOfferingPenalty = 12.5
	updatedDefaults.DeficitInfluence = 0.75
	_, err = service.UpdateProgramObjectiveWeights(ctx, string(organizationID), actor, year.ID, programRow.ID, updatedDefaults)
	require.NoError(t, err)

	withoutOverride, err := service.GetSessionObjectiveWeights(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, updatedDefaults, withoutOverride.Effective)
	require.Equal(t, data.ObjectiveWeightOverrides{}, withoutOverride.Overrides)

	override := 7.25
	overridden, err := service.UpdateSessionObjectiveWeights(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, data.ObjectiveWeightOverrides{RepeatOfferingPenalty: &override}, "synthetic test override")
	require.NoError(t, err)
	require.Equal(t, override, overridden.Effective.RepeatOfferingPenalty)
	require.Equal(t, updatedDefaults.DeficitInfluence, overridden.Effective.DeficitInfluence)
	require.NotNil(t, overridden.Overrides.RepeatOfferingPenalty)

	cleared, err := service.ClearSessionObjectiveWeights(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, updatedDefaults, cleared.Effective)
	require.Equal(t, data.ObjectiveWeightOverrides{}, cleared.Overrides)

	_, err = service.GetProgramObjectiveWeights(ctx, string(otherOrganizationID), year.ID, programRow.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows, "another organization cannot read objective weights")

	objectType := "program_objective_weights"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, string(audit.ActionObjectiveWeightsChange), entries[0].Action)
	objectType = "session_objective_weight_overrides"
	entries, err = harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Contains(t, []string{entries[0].Reason.String, entries[1].Reason.String}, "synthetic test override")
	require.Contains(t, []string{entries[0].Reason.String, entries[1].Reason.String}, "organizer cleared session objective-weight overrides")

	yearService := schoolyear.New(harness.Database)
	year, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)
	_, err = service.UpdateProgramObjectiveWeights(ctx, string(organizationID), actor, year.ID, programRow.ID, updatedDefaults)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year defaults edit = %v", err)
	_, err = service.UpdateSessionObjectiveWeights(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, data.ObjectiveWeightOverrides{RepeatOfferingPenalty: &override}, "closed-year test override")
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year override edit = %v", err)
}
