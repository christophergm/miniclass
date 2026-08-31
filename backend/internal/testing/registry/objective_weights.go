package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "program_objective_weights", YearScoped: true, Factory: createProgramObjectiveWeights,
		ReadIDs: readProgramObjectiveWeightIDs, FetchByID: fetchProgramObjectiveWeightByID,
		UpdateByID: updateProgramObjectiveWeightByID, DeleteByID: deleteProgramObjectiveWeightByID,
		InsertWithForeignParent: insertProgramObjectiveWeightsWithForeignParent})
	Register(Entity{TableName: "session_objective_weight_overrides", YearScoped: true, Factory: createSessionObjectiveWeightOverrides,
		ReadIDs: readSessionObjectiveWeightOverrideIDs, FetchByID: fetchSessionObjectiveWeightOverrideByID,
		UpdateByID: updateSessionObjectiveWeightOverrideByID, DeleteByID: deleteSessionObjectiveWeightOverrideByID,
		InsertWithForeignParent: insertSessionObjectiveWeightOverridesWithForeignParent})
}

func createProgramObjectiveWeights(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 program objective weights factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic objective weights year %s", organizationID))
	if err != nil {
		return "", err
	}
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Objective Weights Program")
	if err != nil {
		return "", err
	}
	var result data.ProgramObjectiveWeights
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		result, err = tx.GetProgramObjectiveWeights(ctx, year.ID, programRow.ID)
		return err
	})
	return result.ID, err
}

func createSessionObjectiveWeightOverrides(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 session objective weights factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic session objective weights year %s", organizationID))
	if err != nil {
		return "", err
	}
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Session Objective Weights Program")
	if err != nil {
		return "", err
	}
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Objective Weights Session", []time.Time{time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	service := program.New(harness.Database)
	if _, err := service.UpdateSessionObjectiveWeights(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "objective weights fixture"}, year.ID, programRow.ID, session.ID, data.ObjectiveWeightOverrides{}, "layer 2 registry fixture"); err != nil {
		return "", err
	}
	var result data.SessionObjectiveWeightOverrides
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		result, err = tx.GetSessionObjectiveWeightOverrides(ctx, year.ID, programRow.ID, session.ID)
		return err
	})
	return result.ID, err
}

func readProgramObjectiveWeightIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllProgramObjectiveWeightsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchProgramObjectiveWeightByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindProgramObjectiveWeightsForRegistry(ctx, id)
	return row.ID != "", err
}
func updateProgramObjectiveWeightByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.UpdateProgramObjectiveWeightsForRegistry(ctx, id)
}
func deleteProgramObjectiveWeightByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteProgramObjectiveWeightsForRegistry(ctx, id)
}

func readSessionObjectiveWeightOverrideIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllSessionObjectiveWeightOverridesForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchSessionObjectiveWeightOverrideByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindSessionObjectiveWeightOverridesForRegistry(ctx, id)
	return row.ID != "", err
}
func updateSessionObjectiveWeightOverrideByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.UpdateSessionObjectiveWeightOverridesForRegistry(ctx, id)
}
func deleteSessionObjectiveWeightOverrideByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteSessionObjectiveWeightOverridesForRegistry(ctx, id)
}

func insertProgramObjectiveWeightsWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign objective weights fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Objective Weights Year")
	if err != nil {
		return err
	}
	programRow, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Objective Weights Program")
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
	_, err = tx.Exec(ctx, "insert into program_objective_weights (organization_id, school_year_id, program_id) values ($1, $2, $3)", string(foreignOrganizationID), string(year.ID), string(programRow.ID))
	return err
}

func insertSessionObjectiveWeightOverridesWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign session objective weights fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Session Objective Weights Year")
	if err != nil {
		return err
	}
	programRow, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Session Objective Weights Program")
	if err != nil {
		return err
	}
	session, err := foreignFactory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Foreign Objective Weights Session", []time.Time{time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)})
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
	_, err = tx.Exec(ctx, "insert into session_objective_weight_overrides (organization_id, school_year_id, program_id, session_id) values ($1, $2, $3, $4)", string(foreignOrganizationID), string(year.ID), string(programRow.ID), string(session.ID))
	return err
}
