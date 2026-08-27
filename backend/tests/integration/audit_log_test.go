package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/stretchr/testify/require"
)

// SPEC §20.1. The first page of the audit log carries no cursor, so the keyset
// comparison must be reachable with a null cursor. Sending the zero XID for
// cursor_id instead failed the public.xid20 domain check during parameter
// coercion and turned every unpaginated read into a 500.
func TestAuditLogReadsFirstPageWithoutCursorAndPagesWithOne(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "audit log integration test"}

	// One entry per transaction so occurred_at, which defaults to the
	// transaction's now(), differs between them.
	objectTypes := []string{"student", "school_year", "student"}
	for _, objectType := range objectTypes {
		objectType := objectType
		require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
			return tx.Record(ctx, audit.Entry{
				Action:        audit.ActionEdit,
				ObjectType:    objectType,
				ChangeSummary: []byte(`{"field":{"from":"a","to":"b"}}`),
			})
		}))
	}

	firstPage, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{PageSize: 2})
	require.NoError(t, err)
	require.Len(t, firstPage, 2)

	last := firstPage[len(firstPage)-1]
	cursorOccurredAt, cursorID := last.OccurredAt, last.ID
	secondPage, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{
		PageSize:         2,
		CursorOccurredAt: &cursorOccurredAt,
		CursorID:         &cursorID,
	})
	require.NoError(t, err)
	require.Len(t, secondPage, 1, "the cursor page holds the remaining entry for this organization")

	seen := map[ids.XID]bool{}
	for _, entry := range append(append([]data.AuditLogEntry{}, firstPage...), secondPage...) {
		require.False(t, seen[entry.ID], "keyset pagination returned %s twice", entry.ID)
		seen[entry.ID] = true
	}

	objectType := "school_year"
	filtered, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{PageSize: 50, ObjectType: &objectType})
	require.NoError(t, err)
	require.Len(t, filtered, 1, "object_type filters within this organization")
	require.Equal(t, "school_year", filtered[0].ObjectType)
}
