package solver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/chrismott/miniclass/internal/solverclient"
	"github.com/chrismott/miniclass/internal/solvercontract"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestGradeWindowOverrideAuthoritativeSolve(t *testing.T) {
	for _, aboveMaximum := range []bool{false, true} {
		name := "below minimum"
		if aboveMaximum {
			name = "above maximum"
		}
		t.Run(name, func(t *testing.T) {
			testGradeWindowOverrideAuthoritativeSolve(t, aboveMaximum, "")
		})
	}
}

func TestGradeWindowOverrideAuthoritativeSolveSidecar(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("SOLVER_BASE_URL"))
	if baseURL == "" {
		t.Skip("SOLVER_BASE_URL is required for the real sidecar grade-override check")
	}
	for _, aboveMaximum := range []bool{false, true} {
		name := "below minimum"
		if aboveMaximum {
			name = "above maximum"
		}
		t.Run(name, func(t *testing.T) {
			testGradeWindowOverrideAuthoritativeSolve(t, aboveMaximum, baseURL)
		})
	}
}

// SPEC §§16.7 and 17.9 require an authorised pinned placement and its human
// judgement record to survive re-solve, while its grade warning remains visible.
func testGradeWindowOverrideAuthoritativeSolve(t *testing.T, aboveMaximum bool, baseURL string) {
	t.Helper()
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := snapshotPreferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic grade override year %s", organizationID))
	require.NoError(t, err)
	grades := make([]data.GradeLevel, 0, 3)
	for _, level := range []string{"junior", "middle", "senior"} {
		grade, err := factory.CreateGradeLevel(ctx, year.ID, "override-"+level, "Synthetic Override "+level)
		require.NoError(t, err)
		grades = append(grades, grade)
	}
	studentGrade := grades[0]
	if aboveMaximum {
		studentGrade = grades[2]
	}
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Grade Override Room")
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Grade Override", GradeLevelID: &studentGrade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Grade Override Program")
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Grade Override Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	ordinary, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Grade Eligible Offering", "Synthetic description", nil, 1, studentGrade.ID, studentGrade.ID, "", "", "", nil)
	require.NoError(t, err)
	target, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Middle Grade Offering", "Synthetic description", nil, 1, grades[1].ID, grades[1].ID, "", "", "", nil)
	require.NoError(t, err)

	programService := program.New(harness.Database)
	operation := program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID}
	initial, err := programService.MoveAssignment(ctx, string(organizationID), actor, operation, student.ID, ordinary.ID)
	require.NoError(t, err)
	require.Len(t, initial.Assignments, 1)
	operation.ExpectedRevision = initial.DraftRevision
	_, err = programService.MoveAssignment(ctx, string(organizationID), actor, operation, student.ID, target.ID)
	var violation *program.HardRuleViolation
	require.ErrorAs(t, err, &violation)
	require.Equal(t, []string{string(student.ID) + ":grade-window"}, violation.Rules)
	require.Equal(t, initial.Assignments, listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID), "unconfirmed move must leave the existing placement untouched")

	readOverrides := func() []data.AssignmentOverride {
		t.Helper()
		var overrides []data.AssignmentOverride
		require.NoError(t, harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
			current, err := tx.GetSession(ctx, year.ID, programRow.ID, session.ID)
			if err != nil {
				return err
			}
			require.Equal(t, operation.ExpectedRevision, current.DraftRevision)
			overrides, err = tx.ListAssignmentOverrides(ctx, year.ID, programRow.ID, session.ID)
			return err
		}))
		return overrides
	}
	require.Empty(t, readOverrides(), "unconfirmed move must not persist an override or advance the revision")

	operation.ConfirmViolations = true
	operation.Reason = "Synthetic organiser-approved grade exception"
	confirmed, err := programService.MoveAssignment(ctx, string(organizationID), actor, operation, student.ID, target.ID)
	require.NoError(t, err)
	require.Equal(t, initial.DraftRevision+1, confirmed.DraftRevision)
	require.Len(t, confirmed.Assignments, 1)
	placement := confirmed.Assignments[0]
	require.Equal(t, target.ID, placement.OfferingID)
	require.True(t, placement.Pinned)
	operation.ExpectedRevision = confirmed.DraftRevision
	beforeOverrides := readOverrides()
	require.Len(t, beforeOverrides, 1)
	override := beforeOverrides[0]
	require.Equal(t, placement.ID, override.AssignmentID)
	require.Equal(t, "grade-window", override.Rule, "the persisted rule vocabulary must not be migrated")
	require.Equal(t, operation.Reason, override.Reason)
	require.Equal(t, actor.Label, override.RecordedBy)
	require.NotEmpty(t, override.ID)
	require.False(t, override.CreatedAt.IsZero())

	assertGradeWarning := func() {
		t.Helper()
		quality, err := programService.GetAssignmentQuality(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
		require.NoError(t, err)
		require.Len(t, quality.Placements, 1)
		require.Len(t, quality.Overridden, 1)
		require.Equal(t, placement.ID, quality.Overridden[0].Assignment.ID)
		for _, warning := range quality.Placements[0].Warnings {
			if warning.ID == "grade-out-of-range" {
				require.Equal(t, &placement.ID, warning.AssignmentID)
				require.Equal(t, &student.ID, warning.StudentID)
				require.Equal(t, &target.ID, warning.OfferingID)
				return
			}
		}
		t.Fatal("authorised placement must retain its grade-out-of-range warning")
	}
	assertGradeWarning()

	seed := int64(327)
	stub := &snapshotCheckingResponseClient{t: t, responseClient: responseClient{response: solvercontract.Response{
		Version: solvercontract.Version, Seed: seed, Status: "optimal",
		Assignments: []solvercontract.Assignment{{ParticipantID: string(student.ID), OfferingID: string(target.ID), RealizedQuality: solvercontract.QualityNeutral}},
	}}}
	var client solverclient.Client = stub
	if baseURL != "" {
		client, err = solverclient.New(baseURL, 15*time.Second)
		require.NoError(t, err)
	}
	service := New(harness.Database, client)
	snapshot, err := service.CompileSnapshot(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, confirmed.DraftRevision, snapshot.DraftRevision)
	require.Equal(t, []solvercontract.PinnedPlacement{{ParticipantID: string(student.ID), OfferingID: string(target.ID)}}, snapshot.Request.Pins)
	require.Equal(t, []solvercontract.AuthorizedPinnedException{{ParticipantID: string(student.ID), OfferingID: string(target.ID), Rule: solvercontract.ExceptionRuleGrade}}, snapshot.Request.AuthorizedExceptions)
	require.Len(t, snapshot.Request.Participants, 1)
	if aboveMaximum {
		require.Greater(t, snapshot.Request.Participants[0].GradeOrdinal, grades[1].Ordinal)
	} else {
		require.Less(t, snapshot.Request.Participants[0].GradeOrdinal, grades[1].Ordinal)
	}
	snapshot.Request.Seed = seed
	stub.expected, err = solvercontract.CanonicalJSON(snapshot.Request)
	require.NoError(t, err)

	run, err := service.StartAuthoritative(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, &seed)
	require.NoError(t, err)
	require.Equal(t, "optimal", run.SolverStatus)
	require.Equal(t, "applied", run.ApplicationStatus)
	if baseURL == "" {
		require.Equal(t, 1, stub.calls)
	}
	loaded, err := service.Get(ctx, string(organizationID), year.ID, programRow.ID, session.ID, run.ID)
	require.NoError(t, err)
	var recorded solvercontract.Request
	require.NoError(t, json.Unmarshal(loaded.RequestDocument, &recorded))
	canonical, err := solvercontract.CanonicalJSON(recorded)
	require.NoError(t, err)
	require.Equal(t, stub.expected, canonical)
	var response solvercontract.Response
	require.NoError(t, json.Unmarshal(loaded.ResponseDocument, &response))
	require.Equal(t, []solvercontract.Assignment{{ParticipantID: string(student.ID), OfferingID: string(target.ID), RealizedQuality: solvercontract.QualityNeutral}}, response.Assignments)
	draft := listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
	require.Len(t, draft, 1)
	require.Equal(t, placement.ID, draft[0].ID)
	require.Equal(t, student.ID, draft[0].StudentID)
	require.Equal(t, target.ID, draft[0].OfferingID)
	require.True(t, draft[0].Pinned)
	require.Equal(t, &run.ID, draft[0].SolveRunID)
	operation.ExpectedRevision = confirmed.DraftRevision + 1
	require.Equal(t, beforeOverrides, readOverrides(), "re-solve must preserve override identity, reason, recorder and timestamps exactly")
	assertGradeWarning()
}
