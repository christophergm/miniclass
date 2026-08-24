package data

import (
	"context"
	"errors"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgtype"
)

// TenantAccessToken is the persisted portion of an invitation token. The raw
// bearer value never crosses the data boundary.
type TenantAccessToken struct {
	ID         ids.XID
	Purpose    string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ConsumedAt *time.Time
	Generation int
}

// TenantOrganizationMember is an organization member with the display data
// needed by administrator management. It is intentionally owned by data so
// callers cannot construct a tenant-free generated query facade.
type TenantOrganizationMember struct {
	ID                  ids.XID
	OrganizationID      ids.XID
	UserID              *ids.XID
	Role                string
	Email               string
	InvitedEmail        *string
	InvitationTokenID   *ids.XID
	InvitationExpiresAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (tx *Tx) CreateTenantAccessToken(ctx context.Context, tokenHash []byte, purpose string, expiresAt time.Time, generation int) (TenantAccessToken, error) {
	if tx == nil || tx.queries == nil {
		return TenantAccessToken{}, errors.New("create tenant access token: transaction is nil")
	}
	if len(tokenHash) != 32 || strings.TrimSpace(purpose) == "" || generation < 1 {
		return TenantAccessToken{}, errors.New("create tenant access token: invalid token fields")
	}
	row, err := tx.queries.CreateAccessToken(ctx, db.CreateAccessTokenParams{
		TokenHash:  tokenHash,
		Purpose:    db.AccessTokenPurpose(purpose),
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Generation: int32(generation),
	})
	if err != nil {
		return TenantAccessToken{}, err
	}
	return tenantAccessToken(row), nil
}

func (tx *Tx) GetTenantAccessToken(ctx context.Context, id ids.XID) (TenantAccessToken, error) {
	if tx == nil || tx.queries == nil {
		return TenantAccessToken{}, errors.New("get tenant access token: transaction is nil")
	}
	row, err := tx.queries.GetAccessTokenByID(ctx, id)
	if err != nil {
		return TenantAccessToken{}, err
	}
	return tenantAccessToken(row), nil
}

func (tx *Tx) RegenerateTenantAccessToken(ctx context.Context, id ids.XID, tokenHash []byte, expiresAt time.Time) (TenantAccessToken, error) {
	current, err := tx.GetTenantAccessToken(ctx, id)
	if err != nil {
		return TenantAccessToken{}, err
	}
	replacement, err := tx.CreateTenantAccessToken(ctx, tokenHash, current.Purpose, expiresAt, current.Generation+1)
	if err != nil {
		return TenantAccessToken{}, err
	}
	if err := tx.queries.RevokeAccessToken(ctx, id); err != nil {
		return TenantAccessToken{}, err
	}
	return replacement, nil
}

func (tx *Tx) RevokeTenantAccessToken(ctx context.Context, id ids.XID) error {
	if tx == nil || tx.queries == nil {
		return errors.New("revoke tenant access token: transaction is nil")
	}
	return tx.queries.RevokeAccessToken(ctx, id)
}

func (tx *Tx) CreateTenantOrganizationMember(ctx context.Context, role, email string, tokenID *ids.XID) (TenantOrganizationMember, error) {
	if tx == nil || tx.queries == nil {
		return TenantOrganizationMember{}, errors.New("create tenant organization member: transaction is nil")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || strings.TrimSpace(role) == "" || tokenID == nil {
		return TenantOrganizationMember{}, errors.New("create tenant organization member: invalid member fields")
	}
	emailValue := pgtype.Text{String: email, Valid: true}
	row, err := tx.queries.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID:    tx.organizationID,
		Role:              db.OrganizationRole(role),
		InvitedEmail:      emailValue,
		InvitationTokenID: tokenID,
	})
	if err != nil {
		return TenantOrganizationMember{}, err
	}
	return tenantOrganizationMember(row, email, nil), nil
}

func (tx *Tx) ListTenantOrganizationMembers(ctx context.Context) ([]TenantOrganizationMember, error) {
	if tx == nil || tx.queries == nil {
		return nil, errors.New("list tenant organization members: transaction is nil")
	}
	rows, err := tx.queries.ListOrganizationMembers(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]TenantOrganizationMember, 0, len(rows))
	for _, row := range rows {
		result = append(result, tenantOrganizationMemberSummary(row))
	}
	return result, nil
}

func (tx *Tx) GetTenantOrganizationMember(ctx context.Context, id ids.XID) (TenantOrganizationMember, error) {
	if tx == nil || tx.queries == nil {
		return TenantOrganizationMember{}, errors.New("get tenant organization member: transaction is nil")
	}
	row, err := tx.queries.GetOrganizationMember(ctx, db.GetOrganizationMemberParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return TenantOrganizationMember{}, err
	}
	return tenantOrganizationMemberGet(row), nil
}

func (tx *Tx) UpdateTenantOrganizationMemberRole(ctx context.Context, id ids.XID, role string) (TenantOrganizationMember, error) {
	if tx == nil || tx.queries == nil {
		return TenantOrganizationMember{}, errors.New("update tenant organization member: transaction is nil")
	}
	row, err := tx.queries.UpdateOrganizationMemberRole(ctx, db.UpdateOrganizationMemberRoleParams{
		ID: id, Role: db.OrganizationRole(role), OrganizationID: tx.organizationID,
	})
	if err != nil {
		return TenantOrganizationMember{}, err
	}
	return tenantOrganizationMember(row, "", nil), nil
}

func (tx *Tx) SetTenantOrganizationMemberInvitation(ctx context.Context, id ids.XID, tokenID *ids.XID) (bool, error) {
	if tx == nil || tx.queries == nil {
		return false, errors.New("set tenant organization invitation: transaction is nil")
	}
	rows, err := tx.queries.SetOrganizationMemberInvitation(ctx, db.SetOrganizationMemberInvitationParams{
		ID: id, InvitationTokenID: tokenID, OrganizationID: tx.organizationID,
	})
	return rows == 1, err
}

func (tx *Tx) DeleteTenantOrganizationMember(ctx context.Context, id ids.XID) (bool, error) {
	if tx == nil || tx.queries == nil {
		return false, errors.New("delete tenant organization member: transaction is nil")
	}
	rows, err := tx.queries.DeleteOrganizationMember(ctx, db.DeleteOrganizationMemberParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func tenantAccessToken(row db.AccessToken) TenantAccessToken {
	return TenantAccessToken{
		ID: row.ID, Purpose: string(row.Purpose), ExpiresAt: row.ExpiresAt.Time,
		RevokedAt: nullableTimestamp(row.RevokedAt), ConsumedAt: nullableTimestamp(row.ConsumedAt),
		Generation: int(row.Generation),
	}
}

func tenantOrganizationMember(row db.OrganizationMember, email string, invitationExpiresAt *time.Time) TenantOrganizationMember {
	return TenantOrganizationMember{
		ID: row.ID, OrganizationID: row.OrganizationID, UserID: row.UserID, Role: string(row.Role),
		Email: email, InvitedEmail: nullableText(row.InvitedEmail), InvitationTokenID: row.InvitationTokenID,
		InvitationExpiresAt: invitationExpiresAt, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func tenantOrganizationMemberSummary(row db.ListOrganizationMembersRow) TenantOrganizationMember {
	return TenantOrganizationMember{
		ID: row.ID, OrganizationID: row.OrganizationID, UserID: row.UserID, Role: string(row.Role),
		Email: row.UserEmail.String, InvitedEmail: nullableText(row.InvitedEmail), InvitationTokenID: row.InvitationTokenID,
		InvitationExpiresAt: nullableTimestamp(row.InvitationExpiresAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func tenantOrganizationMemberGet(row db.GetOrganizationMemberRow) TenantOrganizationMember {
	return TenantOrganizationMember{
		ID: row.ID, OrganizationID: row.OrganizationID, UserID: row.UserID, Role: string(row.Role),
		Email: row.UserEmail.String, InvitedEmail: nullableText(row.InvitedEmail), InvitationTokenID: row.InvitationTokenID,
		InvitationExpiresAt: nullableTimestamp(row.InvitationExpiresAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableTimestamp(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
