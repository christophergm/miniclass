package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

// SPEC §§14.4–14.5: metadata stays mutable except in Complete; deadline
// edits must not bypass the audited, warning-bearing reopening lifecycle.
func TestSessionUpdatesRespectVotingWindowAndAudit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "session update integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic session update year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic session update program")
	require.NoError(t, err)
	service := program.New(harness.Database)
	future := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	past := future.Add(-2 * time.Hour)
	for _, state := range []data.SessionState{data.SessionPlanning, data.SessionCatalogPublished, data.SessionVotingOpen, data.SessionVotingClosed, data.SessionAssigning, data.SessionPublished, data.SessionComplete} {
		t.Run(string(state), func(t *testing.T) {
			date := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
			session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic editable session", []time.Time{date})
			require.NoError(t, err)
			require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
				if _, err := tx.UpdateSession(ctx, year.ID, programRow.ID, session.ID, session.Name, &data.RankedChoiceConfiguration{RankDepth: 3, Deadline: &past}); err != nil {
					return err
				}
				if _, err := tx.UpdateSessionLifecycle(ctx, year.ID, programRow.ID, session.ID, state, false); err != nil {
					return err
				}
				tx.NoAuditRequired("synthetic session update state fixture")
				return nil
			}))
			name := "Synthetic renamed session"
			dates := []time.Time{date.AddDate(0, 0, 1)}
			input := program.SessionUpdate{Name: &name, Dates: &dates, RankedChoice: &data.RankedChoiceConfiguration{RankDepth: 3, Deadline: &past}}
			updated, err := service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, input)
			if state == data.SessionComplete {
				require.ErrorIs(t, err, program.ErrSessionReadOnly)
				stored, err := service.GetSession(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
				require.NoError(t, err)
				require.Equal(t, session.Name, stored.Name)
				require.Equal(t, []time.Time{date}, stored.MeetingDates)
				return
			}
			require.NoError(t, err)
			require.Equal(t, name, updated.Name)
			require.Equal(t, dates, updated.MeetingDates)
			require.Equal(t, state, updated.State)
			require.True(t, updated.RankedChoice.Deadline.Equal(past))
			_, err = service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, input)
			require.ErrorIs(t, err, program.ErrSessionNoChanges)
			input.RankedChoice.Deadline = &future
			_, err = service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, input)
			if state == data.SessionPlanning || state == data.SessionCatalogPublished {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, program.ErrRankedChoiceConfigurationLocked)
			}
		})
	}

	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic open session", []time.Time{future})
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, year.ID, programRow.ID, session.ID, 3, future)
	require.NoError(t, err)
	require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.UpdateSessionLifecycle(ctx, year.ID, programRow.ID, session.ID, data.SessionVotingOpen, false); err != nil {
			return err
		}
		tx.NoAuditRequired("synthetic open voting fixture")
		return nil
	}))
	later := future.Add(time.Hour)
	updated, err := service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionUpdate{RankedChoice: &data.RankedChoiceConfiguration{RankDepth: 3, Deadline: &later}})
	require.NoError(t, err)
	require.Equal(t, data.SessionVotingOpen, updated.State)
	require.True(t, updated.RankedChoice.Deadline.Equal(later))
	_, err = service.UpdateSession(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, program.SessionUpdate{Name: stringPtr("must roll back"), RankedChoice: &data.RankedChoiceConfiguration{RankDepth: 4, Deadline: &later}})
	require.ErrorIs(t, err, program.ErrRankedChoiceConfigurationLocked)
	stored, err := service.GetSession(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, session.Name, stored.Name)
	require.Equal(t, 3, stored.RankedChoice.RankDepth)

	objectType := "session"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	found := false
	for _, entry := range entries {
		if entry.Action != string(audit.ActionSessionChange) || entry.ObjectID == nil || *entry.ObjectID != session.ID {
			continue
		}
		var summary struct {
			RankedChoice data.RankedChoiceConfiguration `json:"ranked_choice"`
			Before       struct {
				RankedChoice data.RankedChoiceConfiguration `json:"ranked_choice"`
			} `json:"before"`
		}
		require.NoError(t, json.Unmarshal(entry.ChangeSummary, &summary))
		if summary.RankedChoice.Deadline == nil || !summary.RankedChoice.Deadline.Equal(later) {
			continue
		}
		found = true
		require.True(t, entry.OccurredAt.Valid)
		require.Equal(t, string(actor.Type), entry.ActorType)
		require.Equal(t, actor.Label, entry.ActorLabel)
		require.False(t, entry.Reason.Valid)
		require.NotNil(t, summary.Before.RankedChoice.Deadline)
		require.Equal(t, 3, summary.RankedChoice.RankDepth)
		require.True(t, summary.Before.RankedChoice.Deadline.Equal(future))
		require.True(t, summary.RankedChoice.Deadline.Equal(later))
	}
	require.True(t, found, "deadline edit was not audited")
}
