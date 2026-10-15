package integration

import (
	"context"
	"encoding/json"
	"sync"
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

type assignmentRemovalFixture struct {
	harness        *testharness.Harness
	organizationID ids.XID
	actor          audit.Actor
	service        *program.Service
	input          program.AssignmentOperationInput
	assignments    []data.Assignment
}

func newAssignmentRemovalFixture(t *testing.T) assignmentRemovalFixture {
	t.Helper()
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "Synthetic removal operator"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic removal year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-removal", "Synthetic Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Room")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic removal program")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic removal session", []time.Time{time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	fixture := assignmentRemovalFixture{harness: harness, organizationID: organizationID, actor: actor, service: program.New(harness.Database), input: program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Reason: "  Synthetic removal reason  "}}
	for i, name := range []string{"Pinned", "UnpinnedLeft", "UnpinnedRight"} {
		student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: name, LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
		require.NoError(t, err)
		_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
		require.NoError(t, err)
		offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic "+name, "", nil, 3, grade.ID, grade.ID, "", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
			assignment, err := tx.CreateAssignment(ctx, data.CreateAssignmentInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, StudentID: student.ID, OfferingID: offering.ID, Origin: "manual", Pinned: i == 0, RealizedQuality: "high"})
			if err != nil {
				return err
			}
			fixture.assignments = append(fixture.assignments, assignment)
			if _, err := tx.CreateAssignmentOverride(ctx, year.ID, programRow.ID, session.ID, assignment.ID, "capacity", "Synthetic fixture override", actor.Label); err != nil {
				return err
			}
			return tx.Record(ctx, audit.Entry{Action: audit.ActionManualOperation, ObjectType: "assignment", SchoolYearID: &year.ID, ChangeSummary: []byte(`{"synthetic":true}`)})
		}))
	}
	return fixture
}

func (f assignmentRemovalFixture) workspace(t *testing.T) program.AssignmentWorkspace {
	t.Helper()
	workspace, err := f.service.GetAssignmentWorkspace(f.harness.Context, string(f.organizationID), f.input.SchoolYearID, f.input.ProgramID, f.input.SessionID)
	require.NoError(t, err)
	return workspace
}

func (f assignmentRemovalFixture) assertAudit(t *testing.T, operation string, removed []data.Assignment) {
	t.Helper()
	objectType := "session"
	entries, err := f.harness.Database.ListAuditLog(f.harness.Context, string(f.organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	var removals []data.AuditLogEntry
	for _, entry := range entries {
		if entry.Action == string(audit.ActionManualOperation) {
			removals = append(removals, entry)
		}
	}
	require.Len(t, removals, 1)
	entry := removals[0]
	require.Equal(t, string(f.actor.Type), entry.ActorType)
	require.Equal(t, f.actor.Label, entry.ActorLabel)
	require.True(t, entry.OccurredAt.Valid)
	require.False(t, entry.OccurredAt.Time.IsZero())
	require.Equal(t, "Synthetic removal reason", entry.Reason.String)
	require.Equal(t, &f.input.SessionID, entry.ObjectID)
	require.Equal(t, &f.input.SchoolYearID, entry.SchoolYearID)
	var summary struct {
		Operation     string            `json:"operation"`
		ProgramID     ids.XID           `json:"program_id"`
		SessionID     ids.XID           `json:"session_id"`
		Removed       []data.Assignment `json:"removed_assignments"`
		DraftRevision int64             `json:"draft_revision"`
	}
	require.NoError(t, json.Unmarshal(entry.ChangeSummary, &summary))
	require.Equal(t, operation, summary.Operation)
	require.Equal(t, f.input.ProgramID, summary.ProgramID)
	require.Equal(t, f.input.SessionID, summary.SessionID)
	require.Equal(t, int64(1), summary.DraftRevision)
	// JSON preserves timestamp instants, not the database driver's location objects.
	normalize := func(assignments []data.Assignment) []data.Assignment {
		result := append([]data.Assignment(nil), assignments...)
		for i := range result {
			result[i].CreatedAt = result[i].CreatedAt.UTC()
			result[i].UpdatedAt = result[i].UpdatedAt.UTC()
		}
		return result
	}
	require.ElementsMatch(t, normalize(removed), normalize(summary.Removed))
}

func TestRemoveAssignmentReturnsPinnedOrUnpinnedStudentToUnplaced(t *testing.T) {
	for index, name := range []string{"pinned", "unpinned"} {
		t.Run(name, func(t *testing.T) {
			f := newAssignmentRemovalFixture(t)
			target := f.assignments[index]
			result, err := f.service.RemoveAssignment(f.harness.Context, string(f.organizationID), f.actor, f.input, target.StudentID)
			require.NoError(t, err)
			require.Equal(t, int64(1), result.DraftRevision)
			require.Len(t, result.Assignments, 2)
			workspace := f.workspace(t)
			require.Equal(t, result.DraftRevision, workspace.Session.DraftRevision)
			require.ElementsMatch(t, result.Assignments, workspace.Assignments)
			require.Len(t, workspace.Participants, 3, "removal does not remove membership")
			require.Len(t, workspace.Overrides, 2)
			for _, override := range workspace.Overrides {
				require.NotEqual(t, target.ID, override.AssignmentID)
			}
			quality, err := f.service.GetAssignmentQuality(f.harness.Context, string(f.organizationID), f.input.SchoolYearID, f.input.ProgramID, f.input.SessionID)
			require.NoError(t, err)
			require.Len(t, quality.Unplaced, 1)
			require.Equal(t, target.StudentID, quality.Unplaced[0].Assignment.StudentID)
			repeated := f.input
			repeated.ExpectedRevision = result.DraftRevision
			noop, err := f.service.RemoveAssignment(f.harness.Context, string(f.organizationID), f.actor, repeated, target.StudentID)
			require.NoError(t, err)
			require.Equal(t, result, noop)
			_, err = f.service.RemoveAssignment(f.harness.Context, string(f.organizationID), f.actor, f.input, target.StudentID)
			require.ErrorIs(t, err, program.ErrDraftRevisionConflict)
			f.assertAudit(t, "remove", []data.Assignment{target})
		})
	}
}

func TestRemoveUnpinnedAssignmentsPreservesPinsAcrossWholeSession(t *testing.T) {
	f := newAssignmentRemovalFixture(t)
	ctx := f.harness.Context
	// A second session in the same program must not be touched.
	factory := factories.New(f.harness.Database, string(f.organizationID), f.actor)
	otherSession, err := factory.CreateSession(ctx, f.input.SchoolYearID, f.input.ProgramID, "Synthetic other session", []time.Time{time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, f.input.SchoolYearID, f.input.ProgramID, otherSession.ID, "Synthetic other offering", "", nil, 3, f.workspace(t).Offerings[0].MinGradeLevelID, f.workspace(t).Offerings[0].MaxGradeLevelID, "", "", "", nil)
	require.NoError(t, err)
	var otherAssignment data.Assignment
	require.NoError(t, f.harness.Database.InTenant(ctx, string(f.organizationID), f.actor, func(ctx context.Context, tx *data.Tx) error {
		var err error
		otherAssignment, err = tx.CreateAssignment(ctx, data.CreateAssignmentInput{SchoolYearID: f.input.SchoolYearID, ProgramID: f.input.ProgramID, SessionID: otherSession.ID, StudentID: f.assignments[1].StudentID, OfferingID: offering.ID, Origin: "manual", RealizedQuality: "neutral"})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionManualOperation, ObjectType: "assignment", ChangeSummary: []byte(`{"synthetic":true}`)})
	}))
	// Even a retained placement hidden from the board remains part of Remove all.
	require.NoError(t, people.New(f.harness.Database).DeleteStudent(ctx, string(f.organizationID), f.input.SchoolYearID, f.assignments[2].StudentID, f.actor))
	result, err := f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, f.input)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.DraftRevision)
	require.Len(t, result.Assignments, 1)
	pinned := result.Assignments[0]
	require.Equal(t, f.assignments[0].ID, pinned.ID)
	require.Equal(t, f.assignments[0].StudentID, pinned.StudentID)
	require.Equal(t, f.assignments[0].OfferingID, pinned.OfferingID)
	require.Equal(t, f.assignments[0].Origin, pinned.Origin)
	require.Equal(t, f.assignments[0].RealizedQuality, pinned.RealizedQuality)
	require.True(t, pinned.Pinned)
	workspace := f.workspace(t)
	require.Len(t, workspace.Overrides, 1)
	require.Equal(t, pinned.ID, workspace.Overrides[0].AssignmentID)
	require.NoError(t, f.harness.Database.InTenantRead(ctx, string(f.organizationID), func(ctx context.Context, tx *data.Tx) error {
		assignments, err := tx.ListAssignments(ctx, f.input.SchoolYearID, f.input.ProgramID, f.input.SessionID)
		require.NoError(t, err)
		require.Equal(t, result.Assignments, assignments, "includes hidden/deleted student assignments")
		other, err := tx.ListAssignments(ctx, f.input.SchoolYearID, f.input.ProgramID, otherSession.ID)
		require.NoError(t, err)
		require.Equal(t, []data.Assignment{otherAssignment}, other)
		return nil
	}))
	f.input.ExpectedRevision = 1
	noop, err := f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, f.input)
	require.NoError(t, err)
	require.Equal(t, result, noop)
	f.assertAudit(t, "remove-all", f.assignments[1:])
}

func TestAssignmentRemovalsRejectStaleRevisionAndForeignScope(t *testing.T) {
	f, foreign := newAssignmentRemovalFixture(t), newAssignmentRemovalFixture(t)
	ctx := f.harness.Context
	before := f.workspace(t)
	stale := f.input
	stale.ExpectedRevision = 9
	_, err := f.service.RemoveAssignment(ctx, string(f.organizationID), f.actor, stale, f.assignments[0].StudentID)
	require.ErrorIs(t, err, program.ErrDraftRevisionConflict)
	_, err = f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, stale)
	require.ErrorIs(t, err, program.ErrDraftRevisionConflict)
	_, err = f.service.RemoveAssignment(ctx, string(f.organizationID), f.actor, foreign.input, foreign.assignments[0].StudentID)
	require.Error(t, err)
	_, err = f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, foreign.input)
	require.Error(t, err)
	wrongProgram := f.input
	wrongProgram.ProgramID = foreign.input.ProgramID
	_, err = f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, wrongProgram)
	require.Error(t, err)
	noop, err := f.service.RemoveAssignment(ctx, string(f.organizationID), f.actor, f.input, foreign.assignments[0].StudentID)
	require.NoError(t, err)
	require.Zero(t, noop.DraftRevision)
	require.ElementsMatch(t, f.assignments, noop.Assignments)
	require.Equal(t, before, f.workspace(t), "failed requests and absent-target no-op leave draft unchanged")
	require.ElementsMatch(t, foreign.assignments, foreign.workspace(t).Assignments)
	require.Zero(t, foreign.workspace(t).Session.DraftRevision)
}

func TestAssignmentRemovalsEmptyDraftAndStaleNoop(t *testing.T) {
	f := newAssignmentRemovalFixture(t)
	ctx := f.harness.Context
	_, err := f.service.RemoveAssignment(ctx, string(f.organizationID), f.actor, f.input, f.assignments[0].StudentID)
	require.NoError(t, err)
	f.input.ExpectedRevision = 1
	cleared, err := f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, f.input)
	require.NoError(t, err)
	require.Equal(t, int64(2), cleared.DraftRevision)
	require.NotNil(t, cleared.Assignments)
	require.Empty(t, cleared.Assignments)
	_, err = f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, f.input)
	require.ErrorIs(t, err, program.ErrDraftRevisionConflict)
	f.input.ExpectedRevision = 2
	empty, err := f.service.RemoveUnpinnedAssignments(ctx, string(f.organizationID), f.actor, f.input)
	require.NoError(t, err)
	require.Equal(t, cleared, empty)
	require.Empty(t, f.workspace(t).Overrides)
}

func TestAssignmentRemovalsConcurrentRequestsHaveOneWinner(t *testing.T) {
	f := newAssignmentRemovalFixture(t)
	var wg sync.WaitGroup
	results := make([]program.AssignmentOperationResult, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := range results {
		wg.Go(func() {
			<-start
			results[i], errs[i] = f.service.RemoveUnpinnedAssignments(f.harness.Context, string(f.organizationID), f.actor, f.input)
		})
	}
	close(start)
	wg.Wait()
	if errs[0] != nil {
		errs[0], errs[1] = errs[1], errs[0]
		results[0], results[1] = results[1], results[0]
	}
	require.NoError(t, errs[0])
	require.ErrorIs(t, errs[1], program.ErrDraftRevisionConflict)
	require.Equal(t, int64(1), results[0].DraftRevision)
	require.Len(t, results[0].Assignments, 1)
	require.Equal(t, results[0].Assignments, f.workspace(t).Assignments)
	f.assertAudit(t, "remove-all", f.assignments[1:])
}
