package integration

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
	postgresTesting "github.com/chrismott/miniclass/internal/testing"
	"github.com/stretchr/testify/require"
)

func TestAdministratorManagementInvitesAuditsAndProtectsLastOwner(t *testing.T) {
	harness := postgresTesting.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	ownerUserID := mintUser(t, harness, "owner-subject", "owner@example.test")
	ownerMemberID := mintMember(t, harness, organizationID, ownerUserID, "owner", nil)
	actor := audit.Actor{Type: audit.ActorTypeUser, UserID: &ownerUserID, Label: "owner@example.test"}
	store := identity.NewStore(harness.Database)
	now := time.Now().UTC().Truncate(time.Second)

	invitation, err := store.InviteAdministrator(ctx, identity.InviteAdministratorInput{
		OrganizationID: string(organizationID), Actor: actor, Email: "Admin@Example.test", Role: "administrator",
		ClaimBaseURL: "https://planner.example/claim", Now: now,
	})
	require.NoError(t, err)
	require.Equal(t, "admin@example.test", invitation.Member.Email)
	require.True(t, invitation.Member.PendingInvitation)
	require.WithinDuration(t, now.Add(48*time.Hour), invitation.ExpiresAt, time.Second)

	oldURL, err := url.Parse(invitation.ClaimURL)
	require.NoError(t, err)
	oldBearer := oldURL.Query().Get("token")
	oldToken, err := store.GetAccessTokenByBearer(ctx, oldBearer)
	require.NoError(t, err)

	replacement, err := store.ResendAdministratorInvitation(ctx, identity.AdministratorActionInput{
		OrganizationID: string(organizationID), Actor: actor, MemberID: invitation.Member.ID, Now: now.Add(time.Minute),
	}, "https://planner.example/claim", time.Hour)
	require.NoError(t, err)
	require.Equal(t, 2, replacement.Generation)
	newURL, err := url.Parse(replacement.ClaimURL)
	require.NoError(t, err)
	newToken, err := store.GetAccessTokenByBearer(ctx, newURL.Query().Get("token"))
	require.NoError(t, err)
	require.NotEqual(t, oldToken.ID, newToken.ID)
	oldAfterResend, err := store.GetAccessTokenByBearer(ctx, oldBearer)
	require.NoError(t, err)
	require.NotNil(t, oldAfterResend.RevokedAt, "resend must revoke the previous URL")

	changed, err := store.ChangeAdministratorRole(ctx, identity.ChangeAdministratorRoleInput{
		AdministratorActionInput: identity.AdministratorActionInput{OrganizationID: string(organizationID), Actor: actor, MemberID: invitation.Member.ID},
		Role:                     "coordinator",
	})
	require.NoError(t, err)
	require.Equal(t, "coordinator", changed.Role)

	members, err := store.ListAdministrators(ctx, string(organizationID))
	require.NoError(t, err)
	require.Len(t, members, 2)
	var pending identity.Administrator
	for _, member := range members {
		if member.ID == invitation.Member.ID {
			pending = member
		}
	}
	require.Equal(t, "admin@example.test", pending.Email)
	require.Equal(t, "coordinator", pending.Role)
	require.True(t, pending.PendingInvitation)

	require.NoError(t, store.RevokeAdministratorInvitation(ctx, identity.AdministratorActionInput{
		OrganizationID: string(organizationID), Actor: actor, MemberID: invitation.Member.ID,
	}))
	revoked, err := store.GetAccessTokenByBearer(ctx, newURL.Query().Get("token"))
	require.NoError(t, err)
	require.NotNil(t, revoked.RevokedAt)
	require.NoError(t, store.RemoveAdministrator(ctx, identity.AdministratorActionInput{
		OrganizationID: string(organizationID), Actor: actor, MemberID: invitation.Member.ID,
	}))
	members, err = store.ListAdministrators(ctx, string(organizationID))
	require.NoError(t, err)
	require.Len(t, members, 1)

	_, err = store.ChangeAdministratorRole(ctx, identity.ChangeAdministratorRoleInput{
		AdministratorActionInput: identity.AdministratorActionInput{OrganizationID: string(organizationID), Actor: actor, MemberID: ownerMemberID},
		Role:                     "administrator",
	})
	require.ErrorIs(t, err, identity.ErrLastOwner)
	require.ErrorIs(t, store.RemoveAdministrator(ctx, identity.AdministratorActionInput{
		OrganizationID: string(organizationID), Actor: actor, MemberID: ownerMemberID,
	}), identity.ErrLastOwner)

	otherOrganizationID := harness.MintOrganization(t)
	otherMembers, err := store.ListAdministrators(ctx, string(otherOrganizationID))
	require.NoError(t, err)
	require.Empty(t, otherMembers)

	require.NoError(t, harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		count, err := tx.Queries().CountAuditLog(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 5, count)
		return nil
	}))
}

func mintUser(t *testing.T, harness *postgresTesting.Harness, subject, email string) ids.XID {
	t.Helper()
	var id ids.XID
	require.NoError(t, harness.Migrator.QueryRow(harness.Context, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, subject, email).Scan(&id))
	return id
}

func mintMember(t *testing.T, harness *postgresTesting.Harness, organizationID, userID ids.XID, role string, invitationTokenID *ids.XID) ids.XID {
	t.Helper()
	var id ids.XID
	require.NoError(t, harness.Migrator.QueryRow(harness.Context, `
		insert into organization_members (organization_id, user_id, role, invitation_token_id)
		values ($1, $2, $3, $4)
		returning id`, organizationID, userID, role, invitationTokenID).Scan(&id))
	return id
}
