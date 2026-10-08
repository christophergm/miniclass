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
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestAssignmentBoardExcludesDeletedStudentsAndRetainedPlacements(t *testing.T) {
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "deleted assignment participant integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic deleted participant year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-deleted", "Synthetic Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Room")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Program")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Session", []time.Time{time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Offering", "", nil, 2, grade.ID, grade.ID, "", "", "", nil)
	require.NoError(t, err)
	students := make([]data.Student, 0, 4)
	for _, name := range []string{"ActivePlaced", "ActiveUnplaced", "DeletedPlaced", "DeletedUnplaced"} {
		student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: name, LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
		require.NoError(t, err)
		_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
		require.NoError(t, err)
		students = append(students, student)
	}
	service := program.New(harness.Database)
	firstMove, err := service.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID}, students[0].ID, offering.ID)
	require.NoError(t, err)
	_, err = service.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: firstMove.DraftRevision}, students[2].ID, offering.ID)
	require.NoError(t, err)
	peopleService := people.New(harness.Database)
	for _, student := range students[2:] {
		require.NoError(t, peopleService.DeleteStudent(ctx, string(organizationID), year.ID, student.ID, actor))
	}

	workspace, err := service.GetAssignmentWorkspace(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, workspace.Participants, 2)
	participantIDs := []ids.XID{workspace.Participants[0].StudentID, workspace.Participants[1].StudentID}
	require.ElementsMatch(t, []ids.XID{students[0].ID, students[1].ID}, participantIDs)
	require.Len(t, workspace.Assignments, 1)
	require.Equal(t, students[0].ID, workspace.Assignments[0].StudentID)

	quality, err := service.GetAssignmentQuality(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, quality.Placements, 1)
	require.Equal(t, students[0].ID, quality.Placements[0].Assignment.StudentID)
	require.Len(t, quality.Unplaced, 1)
	require.Equal(t, students[1].ID, quality.Unplaced[0].Assignment.StudentID)
	require.Len(t, quality.Offerings, 1)
	require.Equal(t, 1, quality.Offerings[0].Enrolled)
	require.Equal(t, map[string]int{workspace.Assignments[0].RealizedQuality: 1}, quality.QualityDistribution)
	for _, warning := range quality.Warnings {
		require.NotEqual(t, "catalog-capacity-short", warning.ID)
	}
	// Filtering the board must not erase retained membership or placement history.
	require.NoError(t, harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		memberships, err := tx.ListProgramMemberships(ctx, year.ID, programRow.ID)
		if err != nil {
			return err
		}
		require.Len(t, memberships, 4)
		assignments, err := tx.ListAssignments(ctx, year.ID, programRow.ID, session.ID)
		if err != nil {
			return err
		}
		require.Len(t, assignments, 2)
		return nil
	}))
}

func TestAssignmentOperationsMoveSwapPinAndRevision(t *testing.T) {
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "assignment operations integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic assignment operation year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-assignment", "Synthetic Assignment Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Assignment Room")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic assignment program")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic assignment session", []time.Time{time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	first, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "First", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	second, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Second", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, first.ID)
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, second.ID)
	require.NoError(t, err)
	foreignProgram, err := factory.CreateProgram(ctx, year.ID, "Synthetic foreign assignment program")
	require.NoError(t, err)
	foreign, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Foreign", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, foreignProgram.ID, foreign.ID)
	require.NoError(t, err)
	left, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic left", "", nil, 2, grade.ID, grade.ID, "", "", "", nil)
	require.NoError(t, err)
	right, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic right", "", nil, 2, grade.ID, grade.ID, "", "", "", nil)
	require.NoError(t, err)

	service := program.New(harness.Database)
	base := program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID}
	firstMove, err := service.MoveAssignment(ctx, string(organizationID), actor, base, first.ID, left.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), firstMove.DraftRevision)
	require.True(t, firstMove.Assignments[0].Pinned)
	secondMove, err := service.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: 1}, second.ID, right.ID)
	require.NoError(t, err)
	swapped, err := service.SwapAssignments(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: secondMove.DraftRevision}, first.ID, second.ID)
	require.NoError(t, err)
	placements := map[string]string{}
	for _, assignment := range swapped.Assignments {
		placements[string(assignment.StudentID)] = string(assignment.OfferingID)
	}
	require.Equal(t, string(right.ID), placements[string(first.ID)])
	require.Equal(t, string(left.ID), placements[string(second.ID)])
	unpinned, err := service.SetAssignmentPin(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: swapped.DraftRevision}, first.ID, false)
	require.NoError(t, err)
	require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		_, err := tx.CreateAssignmentExclusion(ctx, year.ID, programRow.ID, session.ID, first.ID, left.ID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionExclusionChange, ObjectType: "assignment_exclusion", SchoolYearID: &year.ID, ChangeSummary: []byte(`{"synthetic":true}`)})
	}))
	_, err = service.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: unpinned.DraftRevision}, first.ID, left.ID)
	var warning *program.HardRuleViolation
	require.ErrorAs(t, err, &warning)
	confirmed, err := service.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: unpinned.DraftRevision, ConfirmViolations: true}, first.ID, left.ID)
	require.NoError(t, err)
	require.Equal(t, int64(5), confirmed.DraftRevision)
	// An invalid cross-scope reference rolls back the revision CAS, so the next
	// valid exclusion can still use revision 5.
	_, err = service.AddAssignmentExclusion(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: confirmed.DraftRevision}, foreign.ID, left.ID)
	require.Error(t, err)
	added, err := service.AddAssignmentExclusion(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: confirmed.DraftRevision}, second.ID, left.ID)
	require.NoError(t, err)
	require.Equal(t, int64(6), added.DraftRevision)
	require.Len(t, added.Exclusions, 2)
	// The duplicate request is idempotent and does not advance the draft.
	repeated, err := service.AddAssignmentExclusion(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: added.DraftRevision}, second.ID, left.ID)
	require.NoError(t, err)
	require.Equal(t, added.DraftRevision, repeated.DraftRevision)
	var addedID string
	for _, exclusion := range added.Exclusions {
		if exclusion.StudentID == second.ID && exclusion.OfferingID == left.ID {
			addedID = string(exclusion.ID)
		}
	}
	require.NotEmpty(t, addedID)
	removed, err := service.RemoveAssignmentExclusion(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: added.DraftRevision}, ids.XID(addedID))
	require.NoError(t, err)
	require.Equal(t, int64(7), removed.DraftRevision)
	conflicting, err := service.AddAssignmentExclusion(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: removed.DraftRevision}, first.ID, left.ID)
	require.NoError(t, err)
	require.Len(t, conflicting.ConflictingAssignments, 1)
	require.Equal(t, first.ID, conflicting.ConflictingAssignments[0].StudentID)
	_, err = service.MoveAssignment(context.Background(), string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: unpinned.DraftRevision}, second.ID, left.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, program.ErrDraftRevisionConflict))
}
