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

			if !entity.Immutable {
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
			}

			err = entity.InsertWithForeignParent(ctx, harness, organizationA, organizationB)
			require.Error(t, err, "foreign parent insert unexpectedly succeeded")
			assertSQLState(t, err, "42501")
		})
	}
}

func TestLayerTwoRegistryIsDeterministic(t *testing.T) {
	entries := registry.Entries()
	require.NotEmpty(t, entries)

	// Ensure essential tables are present and year-scoped where appropriate.
	for _, table := range []string{"school_years", "grade_levels", "homerooms", "adults", "students", "guardian_relationships", "programs", "program_memberships", "interest_areas", "sessions", "meeting_dates", "offerings", "session_non_participations", "program_objective_weights", "session_objective_weight_overrides", "interest_profile_submissions", "interest_profile_responses", "ranked_choice_submissions", "ranked_choice_responses", "ranked_choice_access_codes", "interest_profile_surveys", "interest_profile_survey_audience_students", "interest_profile_survey_questions", "interest_profile_survey_scale_options", "interest_profile_survey_audience_snapshots", "interest_profile_survey_access_codes"} {
		entry, ok := registry.ForTable(table)
		require.True(t, ok, table+" is missing from the registry")
		require.Equal(t, table, entry.TableName)
	}

	schoolYears, ok := registry.ForTable("school_years")
	require.True(t, ok)
	require.True(t, schoolYears.YearScoped)
	gradeLevels, ok := registry.ForTable("grade_levels")
	require.True(t, ok)
	require.True(t, gradeLevels.YearScoped)
	homerooms, ok := registry.ForTable("homerooms")
	require.True(t, ok)
	require.True(t, homerooms.YearScoped)
	adults, ok := registry.ForTable("adults")
	require.True(t, ok)
	require.True(t, adults.YearScoped)
	students, ok := registry.ForTable("students")
	require.True(t, ok)
	require.True(t, students.YearScoped)
	guardianRelationships, ok := registry.ForTable("guardian_relationships")
	require.True(t, ok)
	require.True(t, guardianRelationships.YearScoped)
	programs, ok := registry.ForTable("programs")
	require.True(t, ok)
	require.True(t, programs.YearScoped)
	memberships, ok := registry.ForTable("program_memberships")
	require.True(t, ok)
	require.True(t, memberships.YearScoped)
	interestAreas, ok := registry.ForTable("interest_areas")
	require.True(t, ok)
	require.True(t, interestAreas.YearScoped)
	sessions, ok := registry.ForTable("sessions")
	require.True(t, ok)
	require.True(t, sessions.YearScoped)
	meetingDates, ok := registry.ForTable("meeting_dates")
	require.True(t, ok)
	require.True(t, meetingDates.YearScoped)
	programObjectiveWeights, ok := registry.ForTable("program_objective_weights")
	require.True(t, ok)
	require.True(t, programObjectiveWeights.YearScoped)
	sessionObjectiveWeightOverrides, ok := registry.ForTable("session_objective_weight_overrides")
	require.True(t, ok)
	require.True(t, sessionObjectiveWeightOverrides.YearScoped)
	for _, table := range []string{"interest_profile_submissions", "interest_profile_responses", "ranked_choice_submissions", "ranked_choice_responses", "interest_profile_survey_audience_snapshots"} {
		entry, ok := registry.ForTable(table)
		require.True(t, ok)
		require.True(t, entry.YearScoped)
		require.True(t, entry.Immutable)
	}
	accessCodes, ok := registry.ForTable("ranked_choice_access_codes")
	require.True(t, ok)
	require.True(t, accessCodes.YearScoped)
	require.False(t, accessCodes.Immutable)
}
