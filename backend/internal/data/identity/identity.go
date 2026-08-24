// Package identity is the unscoped data accessor for the four identity
// tables. It never sets app.organization_id and cannot be used to reach the
// tenant domain through its transaction handle.
package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB provides transaction-scoped access to identity tables.
type DB struct {
	pool *pgxpool.Pool
}

// New creates an unscoped identity accessor over a pool owned by data.DB.
func New(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// Organization is the application-facing organization record.
type Organization struct {
	ID            ids.XID
	Name          string
	HomeroomLabel string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AccessToken is the persisted portion of a token. The bearer value is never
// present here; only its SHA-256 digest is accepted by the data layer.
type AccessToken struct {
	ID         ids.XID
	TokenHash  []byte
	Purpose    string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ConsumedAt *time.Time
	Generation int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// OrganizationMember is the application-facing membership record.
type OrganizationMember struct {
	ID                ids.XID
	OrganizationID    ids.XID
	UserID            *ids.XID
	Role              string
	InvitedEmail      *string
	InvitationTokenID *ids.XID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Tx is the only object handed to identity callbacks. Generated sqlc queries
// remain private to this package, keeping all database access behind data.
type Tx struct {
	queries *db.Queries
}

// InTx runs an identity transaction without setting any tenant GUC.
func (d *DB) InTx(ctx context.Context, fn func(context.Context, *Tx) error) error {
	return d.inTx(ctx, pgx.ReadWrite, fn)
}

// InReadTx runs a read-only identity transaction without setting any tenant
// GUC.
func (d *DB) InReadTx(ctx context.Context, fn func(context.Context, *Tx) error) error {
	return d.inTx(ctx, pgx.ReadOnly, fn)
}

func (d *DB) inTx(ctx context.Context, accessMode pgx.TxAccessMode, fn func(context.Context, *Tx) error) error {
	if d == nil || d.pool == nil {
		return errors.New("begin identity transaction: connection pool is nil")
	}
	if fn == nil {
		return errors.New("begin identity transaction: callback is nil")
	}

	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: accessMode})
	if err != nil {
		return fmt.Errorf("begin identity transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(ctx, &Tx{queries: db.New(tx)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity transaction: %w", err)
	}
	return nil
}

// CreateOrganization inserts an organization in the current identity
// transaction.
func (tx *Tx) CreateOrganization(ctx context.Context, name, homeroomLabel string) (Organization, error) {
	if strings.TrimSpace(name) == "" {
		return Organization{}, errors.New("create organization: name is empty")
	}
	if strings.TrimSpace(homeroomLabel) == "" {
		return Organization{}, errors.New("create organization: homeroom label is empty")
	}
	row, err := tx.queries.CreateOrganization(ctx, db.CreateOrganizationParams{
		Name:          name,
		HomeroomLabel: homeroomLabel,
	})
	if err != nil {
		return Organization{}, fmt.Errorf("create organization: %w", err)
	}
	return Organization{
		ID:            row.ID,
		Name:          row.Name,
		HomeroomLabel: row.HomeroomLabel,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}, nil
}

// CreateAccessToken stores a hashed access token with no object or tenant
// scope. Purpose-specific ownership is represented by later typed relations.
func (tx *Tx) CreateAccessToken(ctx context.Context, tokenHash []byte, purpose string, expiresAt time.Time, generation int) (AccessToken, error) {
	if len(tokenHash) != 32 {
		return AccessToken{}, errors.New("create access token: hash must be 32 bytes")
	}
	if strings.TrimSpace(purpose) == "" {
		return AccessToken{}, errors.New("create access token: purpose is empty")
	}
	if generation < 1 {
		return AccessToken{}, errors.New("create access token: generation must be positive")
	}
	row, err := tx.queries.CreateAccessToken(ctx, db.CreateAccessTokenParams{
		TokenHash:  tokenHash,
		Purpose:    db.AccessTokenPurpose(purpose),
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Generation: int32(generation),
	})
	if err != nil {
		return AccessToken{}, fmt.Errorf("create access token: %w", err)
	}
	return accessToken(row)
}

// CreateOrganizationMember inserts either an active member or an invitation.
// An invitation has a nil user ID and a non-nil invited email.
func (tx *Tx) CreateOrganizationMember(ctx context.Context, organizationID ids.XID, userID *ids.XID, role string, invitedEmail *string, invitationTokenID *ids.XID) (OrganizationMember, error) {
	if strings.TrimSpace(string(organizationID)) == "" {
		return OrganizationMember{}, errors.New("create organization member: organization id is empty")
	}
	if strings.TrimSpace(role) == "" {
		return OrganizationMember{}, errors.New("create organization member: role is empty")
	}
	if userID == nil && invitedEmail == nil {
		return OrganizationMember{}, errors.New("create organization member: invitation email is required without a user")
	}
	if userID != nil && invitedEmail != nil {
		return OrganizationMember{}, errors.New("create organization member: invited email cannot accompany a user")
	}
	var email pgtype.Text
	if invitedEmail != nil {
		normalizedEmail := strings.ToLower(strings.TrimSpace(*invitedEmail))
		if normalizedEmail == "" {
			return OrganizationMember{}, errors.New("create organization member: invitation email is empty")
		}
		email = pgtype.Text{String: normalizedEmail, Valid: true}
	}
	row, err := tx.queries.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID:    organizationID,
		UserID:            userID,
		Role:              db.OrganizationRole(role),
		InvitedEmail:      email,
		InvitationTokenID: invitationTokenID,
	})
	if err != nil {
		return OrganizationMember{}, fmt.Errorf("create organization member: %w", err)
	}
	return organizationMember(row)
}

// GetAccessTokenByHash retrieves a token by its digest. Callers must perform
// expiry, revocation, and consumption checks before treating it as usable.
func (tx *Tx) GetAccessTokenByHash(ctx context.Context, tokenHash []byte) (AccessToken, error) {
	if len(tokenHash) != 32 {
		return AccessToken{}, errors.New("get access token: hash must be 32 bytes")
	}
	row, err := tx.queries.GetAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		return AccessToken{}, fmt.Errorf("get access token: %w", err)
	}
	return accessToken(row)
}

// GetAccessTokenByID retrieves a token by its opaque identifier inside the
// current identity transaction.
func (tx *Tx) GetAccessTokenByID(ctx context.Context, id ids.XID) (AccessToken, error) {
	if strings.TrimSpace(string(id)) == "" {
		return AccessToken{}, errors.New("get access token: id is empty")
	}
	row, err := tx.queries.GetAccessTokenByID(ctx, id)
	if err != nil {
		return AccessToken{}, fmt.Errorf("get access token: %w", err)
	}
	return accessToken(row)
}

// RegenerateAccessToken issues a new token generation and revokes the prior
// row in the same transaction. Keeping the old row preserves its consumed or
// revoked state while making its bearer value unusable immediately.
func (tx *Tx) RegenerateAccessToken(ctx context.Context, id ids.XID, tokenHash []byte, expiresAt time.Time) (AccessToken, error) {
	if strings.TrimSpace(string(id)) == "" {
		return AccessToken{}, errors.New("regenerate access token: id is empty")
	}
	if len(tokenHash) != 32 {
		return AccessToken{}, errors.New("regenerate access token: hash must be 32 bytes")
	}
	current, err := tx.GetAccessTokenByID(ctx, id)
	if err != nil {
		return AccessToken{}, fmt.Errorf("regenerate access token: %w", err)
	}
	regenerated, err := tx.CreateAccessToken(ctx, tokenHash, current.Purpose, expiresAt, current.Generation+1)
	if err != nil {
		return AccessToken{}, fmt.Errorf("regenerate access token: %w", err)
	}
	if err := tx.RevokeAccessToken(ctx, current.ID); err != nil {
		return AccessToken{}, fmt.Errorf("regenerate access token: %w", err)
	}
	return regenerated, nil
}

// ReplaceOrganizationMemberInvitation points the pending invitation at a new
// token generation. It returns false when no membership referenced the old
// token, allowing the use case to roll back rather than orphaning a token.
func (tx *Tx) ReplaceOrganizationMemberInvitation(ctx context.Context, oldTokenID, newTokenID ids.XID) (bool, error) {
	if strings.TrimSpace(string(oldTokenID)) == "" || strings.TrimSpace(string(newTokenID)) == "" {
		return false, errors.New("replace organization invitation: token id is empty")
	}
	rows, err := tx.queries.ReplaceOrganizationMemberInvitation(ctx, db.ReplaceOrganizationMemberInvitationParams{
		InvitationTokenID:   &oldTokenID,
		InvitationTokenID_2: &newTokenID,
	})
	if err != nil {
		return false, fmt.Errorf("replace organization invitation: %w", err)
	}
	return rows == 1, nil
}

// GetOrganizationMemberByInvitationToken retrieves the membership attached to
// an invitation token.
func (tx *Tx) GetOrganizationMemberByInvitationToken(ctx context.Context, tokenID ids.XID) (OrganizationMember, error) {
	if strings.TrimSpace(string(tokenID)) == "" {
		return OrganizationMember{}, errors.New("get organization member: token id is empty")
	}
	row, err := tx.queries.GetOrganizationMemberByInvitationToken(ctx, &tokenID)
	if err != nil {
		return OrganizationMember{}, fmt.Errorf("get organization member: %w", err)
	}
	return organizationMember(row)
}

// RevokeAccessToken makes a token unusable. The update is idempotent.
func (tx *Tx) RevokeAccessToken(ctx context.Context, id ids.XID) error {
	if strings.TrimSpace(string(id)) == "" {
		return errors.New("revoke access token: id is empty")
	}
	if err := tx.queries.RevokeAccessToken(ctx, id); err != nil {
		return fmt.Errorf("revoke access token: %w", err)
	}
	return nil
}

// ConsumeAccessToken marks a currently usable token as consumed. The returned
// boolean distinguishes a successful one-time transition from an already
// revoked, consumed, or expired token.
func (tx *Tx) ConsumeAccessToken(ctx context.Context, id ids.XID) (bool, error) {
	if strings.TrimSpace(string(id)) == "" {
		return false, errors.New("consume access token: id is empty")
	}
	// The generated exec query intentionally has no row count API. A follow-up
	// read in the same transaction provides the observable state while keeping
	// the transition itself in SQL.
	rows, err := tx.queries.ConsumeAccessToken(ctx, id)
	if err != nil {
		return false, fmt.Errorf("consume access token: %w", err)
	}
	return rows == 1, nil
}

func accessToken(row db.AccessToken) (AccessToken, error) {
	expiresAt, err := validTime(row.ExpiresAt, "expires_at")
	if err != nil {
		return AccessToken{}, err
	}
	createdAt, err := validTime(row.CreatedAt, "created_at")
	if err != nil {
		return AccessToken{}, err
	}
	updatedAt, err := validTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return AccessToken{}, err
	}
	return AccessToken{
		ID:         row.ID,
		TokenHash:  append([]byte(nil), row.TokenHash...),
		Purpose:    string(row.Purpose),
		ExpiresAt:  expiresAt,
		RevokedAt:  nullableTime(row.RevokedAt),
		ConsumedAt: nullableTime(row.ConsumedAt),
		Generation: int(row.Generation),
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}

func organizationMember(row db.OrganizationMember) (OrganizationMember, error) {
	createdAt, err := validTime(row.CreatedAt, "created_at")
	if err != nil {
		return OrganizationMember{}, err
	}
	updatedAt, err := validTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return OrganizationMember{}, err
	}
	var invitedEmail *string
	if row.InvitedEmail.Valid {
		value := row.InvitedEmail.String
		invitedEmail = &value
	}
	return OrganizationMember{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		UserID:            row.UserID,
		Role:              string(row.Role),
		InvitedEmail:      invitedEmail,
		InvitationTokenID: row.InvitationTokenID,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func validTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("identity row: %s is null", name)
	}
	return value.Time, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
