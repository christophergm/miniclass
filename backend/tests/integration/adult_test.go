package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestAdultCRUDSoftDeleteAndAudit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "adult integration test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)

	preferred := "Alex"
	email := "alex@example.test"
	externalIdentifier := "adult-1"
	service := people.New(harness.Database)
	created, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Alexander", LegalFamilyName: "Rivera", PreferredGivenName: &preferred,
		Email: &email, ExternalIdentifier: &externalIdentifier, ParticipationIntent: data.AdultParticipationHelp,
	})
	require.NoError(t, err)
	require.Equal(t, preferred, *created.PreferredGivenName)
	require.Equal(t, data.AdultParticipationHelp, created.ParticipationIntent)

	newIntent := data.AdultParticipationLead
	updated, err := service.Update(ctx, string(organizationID), year.ID, created.ID, actor, people.AdultUpdateInput{ParticipationIntent: &newIntent})
	require.NoError(t, err)
	require.Equal(t, data.AdultParticipationLead, updated.ParticipationIntent)

	listed, err := service.List(ctx, string(organizationID), year.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	require.NoError(t, service.Delete(ctx, string(organizationID), year.ID, created.ID, actor))
	listed, err = service.List(ctx, string(organizationID), year.ID)
	require.NoError(t, err)
	require.Empty(t, listed)
	_, err = service.Get(ctx, string(organizationID), year.ID, created.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// The partial external-identifier index allows a replacement after the
	// old row has been soft-deleted.
	replacement, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Jordan", LegalFamilyName: "Rivera", ExternalIdentifier: &externalIdentifier,
		ParticipationIntent: data.AdultParticipationUnavailable,
	})
	require.NoError(t, err)
	require.NotEqual(t, created.ID, replacement.ID)

	var auditCount int64
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		auditCount, err = tx.Queries().CountAuditLog(ctx)
		return err
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, auditCount, int64(5))
}
