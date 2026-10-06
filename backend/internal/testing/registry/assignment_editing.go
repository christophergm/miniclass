package registry

import (
	"context"
	"errors"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/jackc/pgx/v5"
)

func init() {
	Register(Entity{TableName: "assignment_exclusions", YearScoped: true, Factory: createAssignmentExclusion,
		ReadIDs: readAssignmentExclusionIDs, FetchByID: fetchAssignmentExclusion, UpdateByID: touchAssignmentExclusion,
		DeleteByID: deleteAssignmentExclusion, InsertWithForeignParent: insertForeignAssignmentExclusion})
	Register(Entity{TableName: "assignment_overrides", YearScoped: true, Factory: createAssignmentOverride,
		ReadIDs: readAssignmentOverrideIDs, FetchByID: fetchAssignmentOverride, UpdateByID: touchAssignmentOverride,
		DeleteByID: deleteAssignmentOverride, InsertWithForeignParent: insertForeignAssignmentOverride})
}

func assignmentFixture(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (data.Assignment, error) {
	id, err := createAssignment(ctx, harness, organizationID)
	if err != nil {
		return data.Assignment{}, err
	}
	var assignment data.Assignment
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		assignment, err = tx.FindAssignmentForRegistry(ctx, id)
		return err
	})
	return assignment, err
}

func createAssignmentExclusion(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	assignment, err := assignmentFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	var result data.AssignmentExclusion
	err = harness.Database.InTenant(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "assignment exclusion fixture"}, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.CreateAssignmentExclusion(ctx, assignment.SchoolYearID, assignment.ProgramID, assignment.SessionID, assignment.StudentID, assignment.OfferingID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "assignment_exclusion", ObjectID: &result.ID, SchoolYearID: &assignment.SchoolYearID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return result.ID, err
}

func createAssignmentOverride(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	assignment, err := assignmentFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	var result data.AssignmentOverride
	err = harness.Database.InTenant(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "assignment override fixture"}, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.CreateAssignmentOverride(ctx, assignment.SchoolYearID, assignment.ProgramID, assignment.SessionID, assignment.ID, "capacity", "synthetic override", "fixture organizer")
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "assignment_override", ObjectID: &result.ID, SchoolYearID: &assignment.SchoolYearID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return result.ID, err
}

func readAssignmentExclusionIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllAssignmentExclusionsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchAssignmentExclusion(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindAssignmentExclusionForRegistry(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return row.ID != "", err
}
func touchAssignmentExclusion(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchAssignmentExclusionForRegistry(ctx, id)
}
func deleteAssignmentExclusion(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteAssignmentExclusionForRegistry(ctx, id)
}

func readAssignmentOverrideIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllAssignmentOverridesForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchAssignmentOverride(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindAssignmentOverrideForRegistry(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return row.ID != "", err
}
func touchAssignmentOverride(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchAssignmentOverrideForRegistry(ctx, id)
}
func deleteAssignmentOverride(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteAssignmentOverrideForRegistry(ctx, id)
}

func foreignAssignmentScope(ctx context.Context, harness *testharness.Harness, foreignOrganizationID ids.XID) (data.Assignment, error) {
	return assignmentFixture(ctx, harness, foreignOrganizationID)
}

func insertForeignAssignmentExclusion(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	assignment, err := foreignAssignmentScope(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into assignment_exclusions (organization_id, school_year_id, program_id, session_id, student_id, offering_id) values ($1, $2, $3, $4, $5, $6)`, foreignOrganizationID, assignment.SchoolYearID, assignment.ProgramID, assignment.SessionID, assignment.StudentID, assignment.OfferingID)
	return err
}

func insertForeignAssignmentOverride(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	assignment, err := foreignAssignmentScope(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into assignment_overrides (organization_id, school_year_id, program_id, session_id, assignment_id, rule, reason, recorded_by) values ($1, $2, $3, $4, $5, 'capacity', 'foreign probe', 'probe')`, foreignOrganizationID, assignment.SchoolYearID, assignment.ProgramID, assignment.SessionID, assignment.ID)
	return err
}
