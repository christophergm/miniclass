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
	"github.com/chrismott/miniclass/internal/vocabulary"
)

func init() {
	Register(Entity{TableName: "household_students", YearScoped: true, Factory: createHouseholdStudent, ReadIDs: readHouseholdStudentIDs,
		FetchByID: fetchHouseholdStudentByID, UpdateByID: updateHouseholdStudentByID, DeleteByID: deleteHouseholdStudentByID,
		InsertWithForeignParent: insertHouseholdStudentWithForeignParent})
}

func createHouseholdStudent(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create household student fixture: harness is nil")
	}
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 household student factory"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, fmt.Sprintf("Synthetic membership year %s", organizationID))
	if err != nil {
		return "", err
	}
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "synthetic-membership", "Synthetic Grade")
	if err != nil {
		return "", err
	}
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Synthetic Membership Room")
	if err != nil {
		return "", err
	}
	student, err := people.New(harness.Database).CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Membership Student", GradeLevelID: grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return "", err
	}
	household, err := people.New(harness.Database).CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Synthetic Membership Household"})
	if err != nil {
		return "", err
	}
	membership, err := people.New(harness.Database).AddStudentToHousehold(ctx, string(organizationID), year.ID, household.ID, student.ID, actor)
	if err != nil {
		return "", err
	}
	return membership.ID, nil
}

func readHouseholdStudentIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllHouseholdStudentsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchHouseholdStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, _, err := tx.FindHouseholdStudentForRegistry(ctx, id)
	return row.ID != "", err
}

func updateHouseholdStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindHouseholdStudentForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.TouchHouseholdStudent(ctx, year, id)
}

func deleteHouseholdStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindHouseholdStudentForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.DeleteHouseholdStudent(ctx, year, id)
}

func insertHouseholdStudentWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert household student fixture: app pool is nil")
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into household_students (organization_id, school_year_id, household_id, student_id) values ($1, public.xid(), public.xid(), public.xid())`, string(foreignOrganizationID))
	return err
}
