package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "interest_areas", YearScoped: true, Factory: createInterestArea,
		ReadIDs: readInterestAreaIDs, FetchByID: fetchInterestAreaByID,
		UpdateByID: updateInterestAreaByID, DeleteByID: retireInterestAreaByID,
		InsertWithForeignParent: insertInterestAreaWithForeignParent})
}

func createInterestArea(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create interest area fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 interest-area factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic interest area year %s", organizationID))
	if err != nil {
		return "", err
	}
	program, err := factory.CreateProgram(ctx, year.ID, "Synthetic Interest Area Program")
	if err != nil {
		return "", err
	}
	row, err := factory.CreateInterestArea(ctx, year.ID, program.ID, fmt.Sprintf("Synthetic Interest Area %s", organizationID))
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func readInterestAreaIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestAreasForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchInterestAreaByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestAreaForRegistry(ctx, id)
	return row.ID != "", err
}

func updateInterestAreaByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestAreaForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.UpdateInterestAreaForRegistry(ctx, id, row.Label+" updated")
}

func retireInterestAreaByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestAreaForRegistry(ctx, id)
	if err != nil || row.ID == "" || row.RetiredAt != nil {
		return false, err
	}
	return tx.RetireInterestAreaForRegistry(ctx, id)
}

func insertInterestAreaWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert interest area fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign interest-area fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Interest Area Year")
	if err != nil {
		return err
	}
	program, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Interest Area Program")
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
	_, err = tx.Exec(ctx, "insert into interest_areas (organization_id, school_year_id, program_id, label, ordinal) values ($1, $2, $3, $4, $5)", string(foreignOrganizationID), string(year.ID), string(program.ID), "Foreign Interest Area", 1)
	return err
}
