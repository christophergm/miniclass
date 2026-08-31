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
		TableName:               "grade_levels",
		YearScoped:              true,
		Factory:                 createGradeLevel,
		ReadIDs:                 readGradeLevelIDs,
		FetchByID:               fetchGradeLevelByID,
		UpdateByID:              updateGradeLevelByID,
		DeleteByID:              retireGradeLevelByID,
		InsertWithForeignParent: insertGradeLevelWithForeignOrganization,
	})
}

func createGradeLevel(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create grade level fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 grade-level factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic year %s", organizationID))
	if err != nil {
		return "", err
	}
	row, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), year.ID, audit.Actor{
		Type: audit.ActorTypeSystem, Label: "layer 2 grade-level factory",
	}, "synthetic", fmt.Sprintf("Grade %s", organizationID))
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func readGradeLevelIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllGradeLevelsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchGradeLevelByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	_, yearID, err := tx.FindGradeLevelForRegistry(ctx, id)
	if yearID == "" && err == nil {
		return false, nil
	}
	return err == nil, err
}

func updateGradeLevelByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindGradeLevelForRegistry(ctx, id)
	if row.ID == "" && err == nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = tx.UpdateGradeLevel(ctx, yearID, id, row.Code, row.Label+" updated")
	return err == nil, err
}

func retireGradeLevelByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindGradeLevelForRegistry(ctx, id)
	if row.ID == "" && err == nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if row.RetiredAt != nil {
		return false, nil
	}
	_, err = tx.SetGradeLevelRetired(ctx, yearID, id, true)
	return err == nil, err
}

func insertGradeLevelWithForeignOrganization(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert grade level fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 foreign grade-level fixture"})
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
	_, err = tx.Exec(ctx, "insert into grade_levels (organization_id, school_year_id, code, label, ordinal) values ($1, $2, $3, $4, $5)", foreignOrganizationID, year.ID, "foreign", "Foreign", 1)
	return err
}
