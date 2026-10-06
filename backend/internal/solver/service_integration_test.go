package solver

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/solvercontract"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

type responseClient struct{ response solvercontract.Response }

func (c *responseClient) Solve(context.Context, solvercontract.Request) (solvercontract.Response, error) {
	return c.response, nil
}

func TestStartReplacesSuccessfulDraftAndPreservesItOnInfeasibility(t *testing.T) {
	harness := testharness.Open(t)
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "solver persistence test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	ctx := context.Background()
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic solver year %s", organizationID))
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "solver", "Synthetic Solver Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Solver Room")
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Solver", LegalFamilyName: "Student", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Solver Program")
	require.NoError(t, err)
	require.NoError(t, addMembership(ctx, harness.Database, organizationID, actor, year.ID, programRow.ID, student.ID))
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Solver Session", []time.Time{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Solver Offering", "Synthetic description", nil, 2, grade.ID, grade.ID, "Synthetic room", "Synthetic entrance", "Synthetic directions", nil)
	require.NoError(t, err)

	request := solvercontract.Request{Version: solvercontract.Version, Seed: 33, MaxDeterministicTime: 1, QualityConfig: solvercontract.QualityConfig{HighRankMax: 3}, Participants: []solvercontract.Participant{{ID: string(student.ID), GradeOrdinal: 1}}, Offerings: []solvercontract.Offering{{ID: string(offering.ID), Capacity: 2, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}}, Pins: []solvercontract.PinnedPlacement{{ParticipantID: string(student.ID), OfferingID: string(offering.ID)}}}
	client := &responseClient{response: solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "optimal", Assignments: []solvercontract.Assignment{{ParticipantID: string(student.ID), OfferingID: string(offering.ID), RealizedQuality: solvercontract.QualityTop}}}}
	service := New(harness.Database, client)
	snapshot, err := service.CompileSnapshot(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Zero(t, snapshot.DraftRevision)
	require.Equal(t, []solvercontract.Participant{{ID: string(student.ID), GradeOrdinal: 1, InterestProfile: []solvercontract.InterestRating{}}}, snapshot.Request.Participants)
	require.Equal(t, []solvercontract.Offering{{ID: string(offering.ID), Capacity: 2, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}}, snapshot.Request.Offerings)
	require.Empty(t, snapshot.Request.Pins)
	require.Empty(t, snapshot.Request.PriorPlacements)
	run, err := service.Start(ctx, string(organizationID), actor, StartInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Request: request, Seed: &request.Seed})
	require.NoError(t, err)
	require.Equal(t, "applied", run.ApplicationStatus)

	assignments := listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
	require.Len(t, assignments, 1)
	assignmentID := assignments[0].ID
	require.Equal(t, &run.ID, assignments[0].SolveRunID)
	require.Equal(t, "top", assignments[0].RealizedQuality)
	snapshot, err = service.CompileSnapshot(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), snapshot.DraftRevision)
	require.Equal(t, []solvercontract.PinnedPlacement{{ParticipantID: string(student.ID), OfferingID: string(offering.ID)}}, snapshot.Request.Pins)
	require.Equal(t, []solvercontract.Placement{{ParticipantID: string(student.ID), OfferingID: string(offering.ID)}}, snapshot.Request.PriorPlacements)

	client.response = solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "optimal", Assignments: []solvercontract.Assignment{{ParticipantID: string(student.ID), OfferingID: string(offering.ID), RealizedQuality: solvercontract.QualityTop}}}
	secondRun, err := service.Start(ctx, string(organizationID), actor, StartInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Request: request, Seed: &request.Seed})
	require.NoError(t, err)
	require.Equal(t, "applied", secondRun.ApplicationStatus)
	assignments = listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
	require.Equal(t, assignmentID, assignments[0].ID, "unchanged placement retains its editing context")
	require.Equal(t, &secondRun.ID, assignments[0].SolveRunID)
	require.True(t, assignments[0].Pinned)

	client.response = solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "infeasible", Assignments: []solvercontract.Assignment{}}
	infeasibleRun, err := service.Start(ctx, string(organizationID), actor, StartInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Request: request, Seed: &request.Seed})
	require.NoError(t, err)
	require.Equal(t, "not_applicable", infeasibleRun.ApplicationStatus)
	assignments = listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
	require.Len(t, assignments, 1)
	require.Equal(t, &secondRun.ID, assignments[0].SolveRunID)
	require.Equal(t, "top", assignments[0].RealizedQuality)

	changed := request
	changed.MaxDeterministicTime = 2
	_, err = service.Rerun(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, run.ID, changed)
	require.True(t, errors.Is(err, ErrInputFingerprintMismatch))
}

func TestStartRetainsSupersededSuccessfulRunWithoutReplacingDraft(t *testing.T) {
	harness := testharness.Open(t)
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "stale solve test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	ctx := context.Background()
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Stale solver year %s", organizationID))
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "stale", "Stale Solver Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Stale Solver Room")
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Stale", LegalFamilyName: "Student", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Stale Solver Program")
	require.NoError(t, err)
	require.NoError(t, addMembership(ctx, harness.Database, organizationID, actor, year.ID, programRow.ID, student.ID))
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Stale Solver Session", []time.Time{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Stale Solver Offering", "Synthetic description", nil, 2, grade.ID, grade.ID, "Synthetic room", "Synthetic entrance", "Synthetic directions", nil)
	require.NoError(t, err)
	request := solvercontract.Request{Version: solvercontract.Version, Seed: 44, MaxDeterministicTime: 1, QualityConfig: solvercontract.QualityConfig{HighRankMax: 3}, Participants: []solvercontract.Participant{{ID: string(student.ID), GradeOrdinal: 1}}, Offerings: []solvercontract.Offering{{ID: string(offering.ID), Capacity: 2, MinGradeOrdinal: 1, MaxGradeOrdinal: 1}}}
	service := New(harness.Database, &responseClient{response: solvercontract.Response{Version: solvercontract.Version, Seed: request.Seed, Status: "optimal", Assignments: []solvercontract.Assignment{{ParticipantID: string(student.ID), OfferingID: string(offering.ID), RealizedQuality: solvercontract.QualityTop}}}})
	first, err := service.Start(ctx, string(organizationID), actor, StartInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Request: request, Seed: &request.Seed})
	require.NoError(t, err)
	staleRevision := int64(0)
	superseded, err := service.Start(ctx, string(organizationID), actor, StartInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Request: request, Seed: &request.Seed, DraftRevision: &staleRevision})
	require.NoError(t, err)
	require.Equal(t, "superseded", superseded.ApplicationStatus)
	assignments := listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
	require.Len(t, assignments, 1)
	require.Equal(t, &first.ID, assignments[0].SolveRunID)
	snapshot, err := service.CompileSnapshot(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), snapshot.DraftRevision)
	loaded, err := service.Get(ctx, string(organizationID), year.ID, programRow.ID, session.ID, superseded.ID)
	require.NoError(t, err)
	require.Equal(t, "superseded", loaded.ApplicationStatus)
}

func addMembership(ctx context.Context, database *data.DB, organizationID ids.XID, actor audit.Actor, schoolYearID, programID, studentID ids.XID) error {
	return database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.CreateProgramMembership(ctx, schoolYearID, programID, studentID); err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "program_membership", SchoolYearID: &schoolYearID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
}

func listDraft(t *testing.T, database *data.DB, organizationID, schoolYearID, programID, sessionID ids.XID) []data.Assignment {
	t.Helper()
	var assignments []data.Assignment
	err := database.InTenantRead(context.Background(), string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		assignments, err = tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		return err
	})
	require.NoError(t, err)
	return assignments
}
