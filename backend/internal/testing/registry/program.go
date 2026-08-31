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
	Register(Entity{TableName: "programs", YearScoped: true, Factory: createProgram, ReadIDs: readProgramIDs,
		FetchByID: fetchProgramByID, UpdateByID: updateProgramByID, DeleteByID: deleteProgramByID,
		InsertWithForeignParent: insertProgramWithForeignParent})
	Register(Entity{TableName: "program_memberships", YearScoped: true, Factory: createProgramMembership, ReadIDs: readProgramMembershipIDs,
		FetchByID: fetchProgramMembershipByID, UpdateByID: updateProgramMembershipByID, DeleteByID: deleteProgramMembershipByID,
		InsertWithForeignParent: insertProgramMembershipWithForeignParent})
}

func createProgram(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create program fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 program factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic program year %s", organizationID))
	if err != nil {
		return "", err
	}
	row, err := factory.CreateProgram(ctx, year.ID, "Synthetic Program")
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func createProgramMembership(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create program membership fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 membership factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic membership year %s", organizationID))
	if err != nil {
		return "", err
	}
	grade, err := factory.CreateGradeLevel(ctx, "synthetic-membership", "Synthetic Grade")
	if err != nil {
		return "", err
	}
	homeroom, err := factory.CreateHomeroom(ctx, "Synthetic Membership Room")
	if err != nil {
		return "", err
	}
	student, err := factory.CreateStudent(ctx, year.ID, structStudent(grade.ID, homeroom.ID))
	if err != nil {
		return "", err
	}
	program, err := factory.CreateProgram(ctx, year.ID, "Synthetic Membership Program")
	if err != nil {
		return "", err
	}
	row, err := factory.AddProgramMembership(ctx, year.ID, program.ID, student.ID)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func structStudent(gradeID, homeroomID ids.XID) people.StudentCreateInput {
	return people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Program Member", GradeLevelID: &gradeID, HomeroomID: homeroomID}
}

func readProgramIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllProgramsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchProgramByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindProgramForRegistry(ctx, id)
	return row.ID != "", err
}
func updateProgramByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindProgramForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.UpdateProgramForRegistry(ctx, id, row.Name+" updated")
}
func deleteProgramByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteProgramForRegistry(ctx, id)
}

func readProgramMembershipIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllProgramMembershipsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchProgramMembershipByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindProgramMembershipForRegistry(ctx, id)
	return row.ID != "", err
}
func updateProgramMembershipByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindProgramMembershipForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.TouchProgramMembershipForRegistry(ctx, id)
}
func deleteProgramMembershipByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteProgramMembershipForRegistry(ctx, id)
}

func insertProgramWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert program fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign program fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Program Year")
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "set local app.organization_id = $1", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "insert into programs (organization_id, school_year_id, name) values ($1, $2, $3)", string(foreignOrganizationID), string(year.ID), "Cross Parent Program")
	return err
}

func insertProgramMembershipWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert program membership fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign membership fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Membership Year")
	if err != nil {
		return err
	}
	grade, err := foreignFactory.CreateGradeLevel(ctx, "synthetic-foreign-membership", "Synthetic Foreign Grade")
	if err != nil {
		return err
	}
	homeroom, err := foreignFactory.CreateHomeroom(ctx, "Synthetic Foreign Membership Room")
	if err != nil {
		return err
	}
	student, err := foreignFactory.CreateStudent(ctx, year.ID, structStudent(grade.ID, homeroom.ID))
	if err != nil {
		return err
	}
	program, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Membership Program")
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "set local app.organization_id = $1", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into program_memberships (organization_id, school_year_id, program_id, student_id) values ($1, $2, $3, $4)`, string(foreignOrganizationID), string(year.ID), string(program.ID), string(student.ID))
	return err
}
