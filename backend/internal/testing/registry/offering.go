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
)

func init() {
	Register(Entity{TableName: "offerings", YearScoped: true, Factory: createOffering,
		ReadIDs: readOfferingIDs, FetchByID: fetchOfferingByID, UpdateByID: updateOfferingByID,
		DeleteByID: deleteOfferingByID, InsertWithForeignParent: insertOfferingWithForeignParent})
}

func createOffering(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create offering fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 offering factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic offering year %s", organizationID))
	if err != nil {
		return "", err
	}
	gradeMin, err := factory.CreateGradeLevel(ctx, year.ID, "min", "Synthetic Minimum Grade")
	if err != nil {
		return "", err
	}
	gradeMax, err := factory.CreateGradeLevel(ctx, year.ID, "max", "Synthetic Maximum Grade")
	if err != nil {
		return "", err
	}
	program, err := factory.CreateProgram(ctx, year.ID, "Synthetic Offering Program")
	if err != nil {
		return "", err
	}
	session, err := factory.CreateSession(ctx, year.ID, program.ID, "Synthetic Offering Session", 1, []time.Time{time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	row, err := factory.CreateOffering(ctx, year.ID, program.ID, session.ID, "Synthetic Offering", "Synthetic description", nil, 12, gradeMin.ID, gradeMax.ID, "Synthetic room", "Synthetic entrance", "Synthetic directions", nil)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func readOfferingIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllOfferingsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchOfferingByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindOfferingForRegistry(ctx, id)
	return row.ID != "", err
}

func updateOfferingByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindOfferingForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.TouchOfferingForRegistry(ctx, id)
}

func deleteOfferingByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteOfferingForRegistry(ctx, id)
}

func insertOfferingWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert offering fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign offering fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Offering Year")
	if err != nil {
		return err
	}
	minGrade, err := foreignFactory.CreateGradeLevel(ctx, year.ID, "foreign-min", "Synthetic Foreign Minimum Grade")
	if err != nil {
		return err
	}
	maxGrade, err := foreignFactory.CreateGradeLevel(ctx, year.ID, "foreign-max", "Synthetic Foreign Maximum Grade")
	if err != nil {
		return err
	}
	program, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Offering Program")
	if err != nil {
		return err
	}
	session, err := foreignFactory.CreateSession(ctx, year.ID, program.ID, "Synthetic Foreign Offering Session", 1, []time.Time{time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)})
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
	_, err = tx.Exec(ctx, `insert into offerings (organization_id, school_year_id, program_id, session_id, name, description, capacity, min_grade_level_id, max_grade_level_id) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, string(foreignOrganizationID), string(year.ID), string(program.ID), string(session.ID), "Foreign Offering", "Foreign description", 10, string(minGrade.ID), string(maxGrade.ID))
	return err
}
