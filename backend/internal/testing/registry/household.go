package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
)

func init() {
	Register(Entity{TableName: "households", YearScoped: true, Factory: createHousehold, ReadIDs: readHouseholdIDs,
		FetchByID: fetchHouseholdByID, UpdateByID: updateHouseholdByID, DeleteByID: deleteHouseholdByID,
		InsertWithForeignParent: insertHouseholdWithForeignParent})
}

func createHousehold(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create household fixture: harness is nil")
	}
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 household factory"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, fmt.Sprintf("Synthetic household year %s", organizationID))
	if err != nil {
		return "", err
	}
	row, err := people.New(harness.Database).CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Synthetic Household"})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func fetchHouseholdByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, _, err := tx.FindHouseholdForRegistry(ctx, id)
	return row.ID != "", err
}

func readHouseholdIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllActiveHouseholdsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func updateHouseholdByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindHouseholdForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	_, err = tx.UpdateHousehold(ctx, year, id, row.DisplayName+" updated")
	return err == nil, err
}

func deleteHouseholdByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindHouseholdForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.SoftDeleteHousehold(ctx, year, id)
}

func insertHouseholdWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert household fixture: app pool is nil")
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into households (organization_id, school_year_id, display_name) values ($1, public.xid(), 'cross-parent probe')`, string(foreignOrganizationID))
	return err
}
