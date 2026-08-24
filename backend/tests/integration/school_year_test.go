package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestSchoolYearLifecycleAndClosedYearGuard(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	service := schoolyear.New(harness.Database)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "school-year integration test"}

	created, err := service.Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearSetup, created.State)

	active, err := service.Update(ctx, string(organizationID), created.ID, authRoleOwner, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearActive),
	})
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearActive, active.State)

	_, err = service.Update(ctx, string(organizationID), created.ID, authRoleOwner, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearSetup),
	})
	require.ErrorIs(t, err, schoolyear.ErrInvalidTransition)

	closed, err := service.Update(ctx, string(organizationID), created.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearClosed),
	})
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearClosed, closed.State)

	newLabel := "closed edit"
	_, err = service.Update(ctx, string(organizationID), created.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{
		Label: &newLabel,
	})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year edit = %v", err)

	_, err = service.Update(ctx, string(organizationID), created.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearActive), Reason: "correction without Owner approval",
	})
	require.ErrorIs(t, err, schoolyear.ErrOwnerRequired)

	_, err = service.Update(ctx, string(organizationID), created.ID, authRoleOwner, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearActive),
	})
	require.ErrorIs(t, err, schoolyear.ErrReasonRequired)

	reopened, err := service.Update(ctx, string(organizationID), created.ID, authRoleOwner, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearActive), Reason: "correct a verified historical import error",
	})
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearActive, reopened.State)

	second, err := service.Create(ctx, string(organizationID), actor, "2027–2028")
	require.NoError(t, err)
	second, err = service.Update(ctx, string(organizationID), second.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearActive),
	})
	require.NoError(t, err)
	require.Equal(t, data.SchoolYearActive, second.State, "a concurrent active year must be allowed")

	// Audit remains writable after the year is closed; this is the deliberate
	// trigger exemption needed to record closing and later historical events.
	var auditCount int64
	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		id := created.ID
		if err := tx.Record(ctx, audit.Entry{
			Action: audit.ActionManualOperation, ObjectType: "school_year_note", ObjectID: &id,
			SchoolYearID: &id, Reason: "post-close historical note",
		}); err != nil {
			return err
		}
		var err error
		auditCount, err = tx.Queries().CountAuditLog(ctx)
		return err
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, auditCount, int64(6))

	// A closed year cannot be deleted, either.
	_, err = service.Update(ctx, string(organizationID), created.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{
		State: statePtr(data.SchoolYearClosed),
	})
	require.NoError(t, err)
	_, err = service.Get(ctx, string(organizationID), created.ID)
	require.NoError(t, err)
	err = service.Delete(ctx, string(organizationID), created.ID, actor)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year delete = %v", err)
}

func TestSchoolYearCrossOrganizationFetchIsNotFound(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationA := harness.MintOrganization(t)
	organizationB := harness.MintOrganization(t)
	service := schoolyear.New(harness.Database)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "school-year cross-tenant test"}
	created, err := service.Create(ctx, string(organizationA), actor, "2026–2027")
	require.NoError(t, err)

	_, err = service.Get(ctx, string(organizationB), created.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, pgx.ErrNoRows), "foreign fetch = %v", err)
}

func statePtr(value data.SchoolYearState) *data.SchoolYearState { return &value }

// Keep the role values in one place in this integration package so the test
// reads as a lifecycle table without importing the HTTP capability layer.
const (
	authRoleOwner         = "owner"
	authRoleAdministrator = "administrator"
)
