package registry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/jackc/pgx/v5"
)

func init() {
	Register(Entity{TableName: "solve_runs", YearScoped: true, Immutable: true, Factory: createSolveRun,
		ReadIDs: readSolveRunIDs, FetchByID: fetchSolveRunByID,
		UpdateByID: immutableSolveRun, DeleteByID: immutableSolveRun,
		InsertWithForeignParent: insertSolveRunWithForeignParent})
}

func createSolveRun(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 solve run factory"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic solve run year %s", organizationID))
	if err != nil {
		return "", err
	}
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Solve Run Program")
	if err != nil {
		return "", err
	}
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Solve Run Session", []time.Time{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	var result data.SolveRun
	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		result, err = tx.CreateSolveRun(ctx, data.CreateSolveRunInput{
			SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, ContractVersion: "v1", Seed: 1,
			InputFingerprint: "0000000000000000000000000000000000000000000000000000000000000000",
			RequestDocument:  []byte(`{"version":"v1"}`), ResponseDocument: []byte(`{"version":"v1","status":"optimal"}`),
			SolverStatus: "optimal", DeterministicDuration: 0,
		})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSolveRun, ObjectType: "solve_run", ObjectID: &result.ID, SchoolYearID: &year.ID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return result.ID, err
}

func readSolveRunIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllSolveRunsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchSolveRunByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindSolveRunForRegistry(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return row.ID != "", err
}

func immutableSolveRun(context.Context, *data.Tx, ids.XID) (bool, error) { return false, nil }

func insertSolveRunWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign solve run fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Solve Run Year")
	if err != nil {
		return err
	}
	programRow, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Solve Run Program")
	if err != nil {
		return err
	}
	session, err := foreignFactory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Foreign Solve Run Session", []time.Time{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
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
	_, err = tx.Exec(ctx, `insert into solve_runs (organization_id, school_year_id, program_id, session_id, contract_version, seed, input_fingerprint, request_document, response_document, solver_status, deterministic_duration)
        values ($1, $2, $3, $4, 'v1', 1, '0000000000000000000000000000000000000000000000000000000000000000', '{"version":"v1"}', '{"version":"v1","status":"optimal"}', 'optimal', 0)`, string(foreignOrganizationID), string(year.ID), string(programRow.ID), string(session.ID))
	return err
}
