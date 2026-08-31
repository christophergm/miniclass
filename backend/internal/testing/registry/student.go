package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{
		TableName: "students", YearScoped: true,
		Factory: createStudent, ReadIDs: readStudentIDs, FetchByID: fetchStudentByID,
		UpdateByID: updateStudentByID, DeleteByID: deleteStudentByID,
		InsertWithForeignParent: insertStudentWithForeignParent,
	})
}

func createStudent(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create student fixture: harness is nil")
	}
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 student factory"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic year %s", organizationID))
	if err != nil {
		return "", err
	}
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic", "Synthetic Grade")
	if err != nil {
		return "", err
	}
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Homeroom")
	if err != nil {
		return "", err
	}
	row, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: fmt.Sprintf("Student %s", organizationID),
		GradeLevelID: &grade.ID, HomeroomID: homeroom.ID,
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func fetchStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	rows, err := tx.ListAllActiveStudentsForRegistry(ctx)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func readStudentIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllActiveStudentsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func updateStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindStudentForRegistry(ctx, id)
	if err != nil {
		return false, err
	}
	if row.ID == "" {
		return false, nil
	}
	updatedName := row.LegalGivenName + " updated"
	_, err = tx.UpdateStudent(ctx, yearID, id, updatedName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.PriorYearStudentID)
	return err == nil, err
}

func deleteStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindStudentForRegistry(ctx, id)
	if err != nil {
		return false, err
	}
	if row.ID == "" {
		return false, nil
	}
	return tx.SoftDeleteStudent(ctx, yearID, id)
}

func insertStudentWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert student fixture: app pool is nil")
	}
	var schoolYearID, gradeLevelID, homeroomID ids.XID
	if err := harness.Migrator.QueryRow(ctx, "select public.xid(), public.xid(), public.xid()").Scan(&schoolYearID, &gradeLevelID, &homeroomID); err != nil {
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
	_, err = tx.Exec(ctx, `
		insert into students (organization_id, school_year_id, legal_given_name, legal_family_name, grade_level_id, homeroom_id)
		values ($1, $2, 'Cross', 'Parent', $3, $4)`, foreignOrganizationID, schoolYearID, gradeLevelID, homeroomID)
	return err
}
