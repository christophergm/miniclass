package registry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/jackc/pgx/v5"
)

func init() {
	Register(Entity{TableName: "assignments", YearScoped: true, Factory: createAssignment,
		ReadIDs: readAssignmentIDs, FetchByID: fetchAssignmentByID,
		UpdateByID: touchAssignment, DeleteByID: deleteAssignment,
		InsertWithForeignParent: insertAssignmentWithForeignParent})
}

func createAssignment(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 assignment factory"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic assignment year %s", organizationID))
	if err != nil {
		return "", err
	}
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "assignment", "Synthetic Assignment Grade")
	if err != nil {
		return "", err
	}
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Assignment Room")
	if err != nil {
		return "", err
	}
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Assignment Program")
	if err != nil {
		return "", err
	}
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Assignment Session", []time.Time{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Assigned", LegalFamilyName: "Student", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return "", err
	}
	offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Assignment Offering", "Synthetic assignment description", nil, 12, grade.ID, grade.ID, "Synthetic room", "Synthetic entrance", "Synthetic directions", nil)
	if err != nil {
		return "", err
	}
	var result data.Assignment
	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		run, err := tx.CreateSolveRun(ctx, data.CreateSolveRunInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ContractVersion: "v1", Seed: 1,
			InputFingerprint: "0000000000000000000000000000000000000000000000000000000000000000", RequestDocument: []byte(`{"version":"v1"}`), ResponseDocument: []byte(`{"version":"v1","status":"optimal"}`),
			EffectiveWeightsDocument: []byte(`{}`), MetricsDocument: []byte(`{}`), SolverStatus: "optimal", ApplicationStatus: "applied", DeterministicDuration: 0})
		if err != nil {
			return err
		}
		result, err = tx.CreateAssignment(ctx, data.CreateAssignmentInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, StudentID: student.ID, OfferingID: offering.ID, SolveRunID: &run.ID, Origin: "solver", RealizedQuality: "neutral"})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSolveRun, ObjectType: "assignment", ObjectID: &result.ID, SchoolYearID: &year.ID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return result.ID, err
}

func readAssignmentIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllAssignmentsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchAssignmentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindAssignmentForRegistry(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return row.ID != "", err
}

func touchAssignment(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchAssignmentForRegistry(ctx, id)
}

func deleteAssignment(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteAssignmentForRegistry(ctx, id)
}

func insertAssignmentWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign assignment fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Assignment Year")
	if err != nil {
		return err
	}
	grade, err := foreignFactory.CreateGradeLevel(ctx, year.ID, "foreign-assignment", "Synthetic Foreign Assignment Grade")
	if err != nil {
		return err
	}
	homeroom, err := foreignFactory.CreateHomeroom(ctx, year.ID, "Synthetic Foreign Assignment Room")
	if err != nil {
		return err
	}
	programRow, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Assignment Program")
	if err != nil {
		return err
	}
	session, err := foreignFactory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Foreign Assignment Session", []time.Time{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return err
	}
	student, err := foreignFactory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Foreign", LegalFamilyName: "Assigned Student", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return err
	}
	offering, err := foreignFactory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Foreign Assignment Offering", "Synthetic foreign assignment description", nil, 12, grade.ID, grade.ID, "Synthetic room", "Synthetic entrance", "Synthetic directions", nil)
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into assignments (organization_id, school_year_id, program_id, session_id, student_id, offering_id, solve_run_id, origin, pinned, realized_quality)
        values ($1, $2, $3, $4, $5, $6, $7, 'solver', false, 'neutral')`, string(foreignOrganizationID), string(year.ID), string(programRow.ID), string(session.ID), string(student.ID), string(offering.ID), "00000000000000000000")
	return err
}
