package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/stretchr/testify/require"
)

// SPEC §21.3 makes soft deletion reversible, and §5.4 makes the judgement
// behind the reversal data. The student and adult cases live with their own
// CRUD tests; households had no coverage of either half, and no test anywhere
// read the restore entry back to prove the actor, time and reason survived.
func TestHouseholdSoftDeleteVisibilityRestoreAndAudit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "household restore integration test"}
	service := people.New(harness.Database)
	tenant := newMembershipFixture(t, harness, service, actor, "HouseholdRestore")
	organizationID := string(tenant.organizationID)

	active, err := service.ListHouseholds(ctx, organizationID, tenant.year.ID, false)
	require.NoError(t, err)
	require.Contains(t, rosterHouseholdIDs(active), string(tenant.household.ID))

	require.NoError(t, service.DeleteHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, actor))

	active, err = service.ListHouseholds(ctx, organizationID, tenant.year.ID, false)
	require.NoError(t, err)
	require.NotContains(t, rosterHouseholdIDs(active), string(tenant.household.ID), "the default listing still names the deleted household")

	including, err := service.ListHouseholds(ctx, organizationID, tenant.year.ID, true)
	require.NoError(t, err)
	deleted := findHousehold(including, tenant.household.ID)
	require.NotNil(t, deleted, "include_deleted did not return the deleted household")
	require.NotNil(t, deleted.DeletedAt, "the deleted household came back without deleted_at")

	const reason = "the family withdrew the request"
	restored, err := service.RestoreHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, actor, reason)
	require.NoError(t, err)
	require.Nil(t, restored.DeletedAt)

	active, err = service.ListHouseholds(ctx, organizationID, tenant.year.ID, false)
	require.NoError(t, err)
	require.Contains(t, rosterHouseholdIDs(active), string(tenant.household.ID), "the restored household is missing from the default listing")

	// Restoring a household that is not deleted is a no-op the caller should
	// hear about rather than a silent second restore.
	_, err = service.RestoreHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, actor, reason)
	require.ErrorIs(t, err, people.ErrRestoreNotDeleted)

	// SPEC §5.4: the reason is a record, not a formality.
	_, err = service.RestoreHousehold(ctx, organizationID, tenant.year.ID, tenant.household.ID, actor, "   ")
	require.ErrorIs(t, err, people.ErrRestoreReasonRequired)

	objectType := "household"
	entries, err := harness.Database.ListAuditLog(ctx, organizationID, data.AuditLogFilter{PageSize: 50, ObjectType: &objectType})
	require.NoError(t, err)
	var restoreEntries []data.AuditLogEntry
	for _, entry := range entries {
		if entry.Action == string(audit.ActionRestore) {
			restoreEntries = append(restoreEntries, entry)
		}
	}
	require.Len(t, restoreEntries, 1, "the successful restore should be audited exactly once")
	entry := restoreEntries[0]
	require.NotNil(t, entry.ObjectID)
	require.Equal(t, tenant.household.ID, *entry.ObjectID)
	require.NotNil(t, entry.SchoolYearID)
	require.Equal(t, tenant.year.ID, *entry.SchoolYearID)
	require.Equal(t, actor.Label, entry.ActorLabel)
	require.True(t, entry.OccurredAt.Valid, "the restore entry has no occurrence time")
	require.True(t, entry.Reason.Valid)
	require.Equal(t, reason, entry.Reason.String)
	require.NotEmpty(t, entry.ChangeSummary, "the restore entry records no change summary")
}

// SPEC §9.2: cross-tenant access must not be reachable through any
// authenticated path, and a new query path without an isolation test is a
// defect. Including deleted rows and restoring them are both new paths, and
// the include-deleted listing is the one that deliberately drops a filter.
func TestDeletedRosterVisibilityAndRestoreAreTenantScoped(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "roster restore isolation test"}
	service := people.New(harness.Database)
	owner := newMembershipFixture(t, harness, service, actor, "RestoreIsolationOwner")
	intruder := newMembershipFixture(t, harness, service, actor, "RestoreIsolationIntruder")
	ownerID, intruderID := string(owner.organizationID), string(intruder.organizationID)

	require.NoError(t, service.DeleteStudent(ctx, ownerID, owner.year.ID, owner.student.ID, actor))
	require.NoError(t, service.Delete(ctx, ownerID, owner.year.ID, owner.adult.ID, actor))
	require.NoError(t, service.DeleteHousehold(ctx, ownerID, owner.year.ID, owner.household.ID, actor))

	// The outer guard: the intruder naming the owner's year by identifier
	// cannot resolve it at all, so no listing or restore reaches the row.
	_, err := service.ListStudents(ctx, intruderID, owner.year.ID, true)
	require.Error(t, err, "a foreign organisation can list deleted students")
	_, err = service.List(ctx, intruderID, owner.year.ID, true)
	require.Error(t, err, "a foreign organisation can list deleted adults")
	_, err = service.ListHouseholds(ctx, intruderID, owner.year.ID, true)
	require.Error(t, err, "a foreign organisation can list deleted households")
	_, err = service.RestoreStudent(ctx, intruderID, owner.year.ID, owner.student.ID, actor, "cross-tenant probe")
	require.Error(t, err, "a foreign organisation can restore a student")
	_, err = service.Restore(ctx, intruderID, owner.year.ID, owner.adult.ID, actor, "cross-tenant probe")
	require.Error(t, err, "a foreign organisation can restore an adult")
	_, err = service.RestoreHousehold(ctx, intruderID, owner.year.ID, owner.household.ID, actor, "cross-tenant probe")
	require.Error(t, err, "a foreign organisation can restore a household")

	// SPEC §9.2 requires the guard to hold without each query remembering to
	// filter, so the new statements are also probed directly, below the
	// service's school-year check that stopped the calls above.
	require.NoError(t, harness.Database.InTenantRead(ctx, intruderID, func(ctx context.Context, tx *data.Tx) error {
		students, err := tx.ListStudents(ctx, owner.year.ID, true)
		if err != nil {
			return err
		}
		require.NotContains(t, rosterStudentIDs(students), string(owner.student.ID), "include_deleted leaks a foreign organisation's students")
		adults, err := tx.ListAdults(ctx, owner.year.ID, true)
		if err != nil {
			return err
		}
		require.NotContains(t, rosterAdultIDs(adults), string(owner.adult.ID), "include_deleted leaks a foreign organisation's adults")
		households, err := tx.ListHouseholds(ctx, owner.year.ID, true)
		if err != nil {
			return err
		}
		require.NotContains(t, rosterHouseholdIDs(households), string(owner.household.ID), "include_deleted leaks a foreign organisation's households")
		return nil
	}))

	// Restore is a write, so a failure here would be a cross-tenant mutation
	// rather than a leak. The update must match no row.
	require.NoError(t, harness.Database.InTenant(ctx, intruderID, actor, func(ctx context.Context, tx *data.Tx) error {
		tx.NoAuditRequired("cross-organization restore probe")
		_, err := tx.RestoreStudent(ctx, owner.year.ID, owner.student.ID)
		require.Error(t, err, "a foreign organisation can restore a student row")
		_, err = tx.RestoreAdult(ctx, owner.year.ID, owner.adult.ID)
		require.Error(t, err, "a foreign organisation can restore an adult row")
		_, err = tx.RestoreHousehold(ctx, owner.year.ID, owner.household.ID)
		require.Error(t, err, "a foreign organisation can restore a household row")
		return nil
	}))

	// The owner's rows are untouched: still deleted, and still restorable by
	// the organisation that owns them.
	ownerStudents, err := service.ListStudents(ctx, ownerID, owner.year.ID, true)
	require.NoError(t, err)
	require.Contains(t, rosterStudentIDs(ownerStudents), string(owner.student.ID))
	activeStudents, err := service.ListStudents(ctx, ownerID, owner.year.ID, false)
	require.NoError(t, err)
	require.NotContains(t, rosterStudentIDs(activeStudents), string(owner.student.ID), "the failed cross-tenant restore changed the owner's row")

	restored, err := service.RestoreStudent(ctx, ownerID, owner.year.ID, owner.student.ID, actor, "the owning organisation restores it")
	require.NoError(t, err)
	require.Nil(t, restored.DeletedAt)
}

func rosterStudentIDs(rows []data.Student) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.ID))
	}
	return result
}

func rosterHouseholdIDs(rows []data.Household) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.ID))
	}
	return result
}

func rosterAdultIDs(rows []data.Adult) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, string(row.ID))
	}
	return result
}

func findHousehold(rows []data.Household, id ids.XID) *data.Household {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}
