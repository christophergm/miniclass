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
	"github.com/chrismott/miniclass/internal/vocabulary"
)

func init() {
	Register(Entity{
		TableName:               "homerooms",
		YearScoped:              true,
		Factory:                 createHomeroom,
		ReadIDs:                 readHomeroomIDs,
		FetchByID:               fetchHomeroomByID,
		UpdateByID:              updateHomeroomByID,
		DeleteByID:              retireHomeroomByID,
		InsertWithForeignParent: insertHomeroomWithForeignOrganization,
	})
}

func createHomeroom(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create homeroom fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 homeroom factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic year %s", organizationID))
	if err != nil {
		return "", err
	}
	row, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), year.ID, audit.Actor{
		Type: audit.ActorTypeSystem, Label: "layer 2 homeroom factory",
	}, fmt.Sprintf("Synthetic Homeroom %s", organizationID), nil)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func readHomeroomIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllHomeroomsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchHomeroomByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	_, yearID, err := tx.FindHomeroomForRegistry(ctx, id)
	if yearID == "" && err == nil {
		return false, nil
	}
	return err == nil, err
}

func updateHomeroomByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindHomeroomForRegistry(ctx, id)
	if row.ID == "" && err == nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = tx.UpdateHomeroom(ctx, yearID, id, row.Name+" updated", row.ExternalIdentifier)
	return err == nil, err
}

func retireHomeroomByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindHomeroomForRegistry(ctx, id)
	if row.ID == "" && err == nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if row.RetiredAt != nil {
		return false, nil
	}
	_, err = tx.SetHomeroomRetired(ctx, yearID, id, true)
	return err == nil, err
}

func insertHomeroomWithForeignOrganization(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert homeroom fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 foreign homeroom fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic foreign year %s", foreignOrganizationID))
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
	_, err = tx.Exec(ctx, "insert into homerooms (organization_id, school_year_id, name) values ($1, $2, $3)", foreignOrganizationID, year.ID, "Foreign Homeroom")
	return err
}
