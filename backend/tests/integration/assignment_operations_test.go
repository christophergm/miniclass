package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

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
	_, err = service.MoveAssignment(context.Background(), string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ExpectedRevision: unpinned.DraftRevision}, second.ID, left.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, program.ErrDraftRevisionConflict))
}
