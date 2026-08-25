package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/registry"
	"github.com/stretchr/testify/require"
)

func TestLayerTwoEntityIsolation(t *testing.T) {
	harness := testharness.Open(t)
	for _, entity := range registry.Entries() {
		entity := entity
		t.Run(entity.TableName, func(t *testing.T) {
			ctx := harness.Context
			organizationA := harness.MintOrganization(t)
			organizationB := harness.MintOrganization(t)
			id, err := entity.Factory(ctx, harness, organizationA)
			require.NoError(t, err)

			var foreignRead []string
			err = harness.Database.InTenantRead(ctx, string(organizationB), func(ctx context.Context, tx *data.Tx) error {
				ids, err := entity.ReadIDs(ctx, tx)
				for _, value := range ids {
					foreignRead = append(foreignRead, string(value))
				}
				return err
			})
			require.NoError(t, err)
			require.NotContains(t, foreignRead, string(id), "foreign organization can read the entity")

			var fetched bool
			err = harness.Database.InTenantRead(ctx, string(organizationB), func(ctx context.Context, tx *data.Tx) error {
				var err error
				fetched, err = entity.FetchByID(ctx, tx, id)
				return err
			})
			require.NoError(t, err)
			require.False(t, fetched, "foreign organization can fetch the entity by id")

			var updated bool
			actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 update probe"}
			err = harness.Database.InTenant(ctx, string(organizationB), actor, func(ctx context.Context, tx *data.Tx) error {
				var err error
				updated, err = entity.UpdateByID(ctx, tx, id)
				tx.NoAuditRequired("layer 2 cross-organization update probe")
				return err
			})
			require.NoError(t, err)
			require.False(t, updated, "foreign organization can update the entity")

			var deleted bool
			err = harness.Database.InTenant(ctx, string(organizationB), actor, func(ctx context.Context, tx *data.Tx) error {
				var err error
				deleted, err = entity.DeleteByID(ctx, tx, id)
				tx.NoAuditRequired("layer 2 cross-organization delete probe")
				return err
			})
			require.NoError(t, err)
			require.False(t, deleted, "foreign organization can delete the entity")

			err = entity.InsertWithForeignParent(ctx, harness, organizationA, organizationB)
			require.Error(t, err, "foreign parent insert unexpectedly succeeded")
			assertSQLState(t, err, "42501")
		})
	}
}

func TestLayerTwoRegistryIsDeterministic(t *testing.T) {
	entries := registry.Entries()
	require.NotEmpty(t, entries)
	for _, table := range []string{"school_years", "grade_levels", "homerooms"} {
		entry, ok := registry.ForTable(table)
		require.True(t, ok, table+" is missing from the registry")
		require.Equal(t, table, entry.TableName)
	}
	schoolYears, ok := registry.ForTable("school_years")
	require.True(t, ok)
	require.True(t, schoolYears.YearScoped)
}
