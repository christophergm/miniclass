package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/jackc/pgx/v5"
)

func init() {
	Register(Entity{
		TableName:               "school_years",
		YearScoped:              true,
		Factory:                 createSchoolYear,
		ReadIDs:                 readSchoolYearIDs,
		FetchByID:               fetchSchoolYearByID,
		UpdateByID:              updateSchoolYearByID,
		DeleteByID:              deleteSchoolYearByID,
		InsertWithForeignParent: insertSchoolYearWithForeignOrganization,
	})
}

// createSchoolYear is the school-years registry factory. It uses the normal
// tenant service so the fixture also exercises the audit-required write path.
func createSchoolYear(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create school year fixture: harness is nil")
	}
	row, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), audit.Actor{
		Type: audit.ActorTypeSystem, Label: "layer 2 school-year factory",
	}, fmt.Sprintf("Synthetic year %s", organizationID))
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

// fetchSchoolYearByID is intentionally a tenant-transaction operation: RLS
// makes a foreign organization's ID indistinguishable from an absent row.
func fetchSchoolYearByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	_, err := tx.GetSchoolYearByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func readSchoolYearIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListSchoolYears(ctx)
	if err != nil {
		return nil, err
	}
	idsResult := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		idsResult = append(idsResult, row.ID)
	}
	return idsResult, nil
}

func updateSchoolYearByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.GetSchoolYearByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, err
	}
	_, err = tx.UpdateSchoolYearLabel(ctx, id, row.Label+" updated")
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func deleteSchoolYearByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	rows, err := tx.DeleteSchoolYear(ctx, id)
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// insertSchoolYearWithForeignOrganization attempts the parent mismatch using
// the app role. RLS must reject it even though the row shape is otherwise
// valid, proving that the organization parent cannot be crossed by a caller.
func insertSchoolYearWithForeignOrganization(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert school year fixture: app pool is nil")
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "insert into school_years (organization_id, label) values ($1, $2)", string(foreignOrganizationID), "cross-parent probe")
	return err
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
