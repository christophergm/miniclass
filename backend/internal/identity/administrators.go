package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var (
	ErrAdministratorAlreadyExists = errors.New("administrator already exists")
	ErrAdministratorNotFound      = errors.New("administrator not found")
	ErrAdministratorRole          = errors.New("administrator role is invalid")
	ErrInvitationNotPending       = errors.New("administrator invitation is not pending")
	ErrLastOwner                  = errors.New("the last owner cannot be removed or demoted")
)

const (
	RoleAdministrator = string(auth.RoleAdministrator)
	RoleCoordinator   = string(auth.RoleCoordinator)
	RoleOwner         = string(auth.RoleOwner)
)

// Administrator is the public identity-management representation of one
// organization member. Pending invitations have no provider user yet.
type Administrator struct {
	ID                  ids.XID    `json:"id"`
	Email               string     `json:"email"`
	Role                string     `json:"role"`
	PendingInvitation   bool       `json:"pending_invitation"`
	InvitationExpiresAt *time.Time `json:"invitation_expires_at,omitempty"`
}

type Invitation struct {
	Member     Administrator `json:"member"`
	ClaimURL   string        `json:"claim_url"`
	ExpiresAt  time.Time     `json:"expires_at"`
	Generation int           `json:"generation"`
}

type InviteAdministratorInput struct {
	OrganizationID string
	Actor          audit.Actor
	Email          string
	Role           string
	ClaimBaseURL   string
	InvitationTTL  time.Duration
	Now            time.Time
}

type AdministratorActionInput struct {
	OrganizationID string
	Actor          audit.Actor
	MemberID       ids.XID
	Now            time.Time
}

type ChangeAdministratorRoleInput struct {
	AdministratorActionInput
	Role string
}

// AdministratorManager is the identity use-case boundary used by API
// handlers. It keeps tenant transactions and audit writes out of HTTP code.
type AdministratorManager interface {
	InviteAdministrator(context.Context, InviteAdministratorInput) (Invitation, error)
	ListAdministrators(context.Context, string) ([]Administrator, error)
	ResendAdministratorInvitation(context.Context, AdministratorActionInput, string, time.Duration) (Invitation, error)
	RevokeAdministratorInvitation(context.Context, AdministratorActionInput) error
	ChangeAdministratorRole(context.Context, ChangeAdministratorRoleInput) (Administrator, error)
	RemoveAdministrator(context.Context, AdministratorActionInput) error
}

func (s *Store) InviteAdministrator(ctx context.Context, input InviteAdministratorInput) (Invitation, error) {
	if err := validateAdministratorInput(input.OrganizationID, input.Email, input.ClaimBaseURL); err != nil {
		return Invitation{}, err
	}
	role := normalizeRole(input.Role)
	if role != RoleAdministrator && role != RoleCoordinator {
		return Invitation{}, ErrAdministratorRole
	}
	ttl, now, err := invitationTiming(input.InvitationTTL, input.Now)
	if err != nil {
		return Invitation{}, err
	}
	bearer, err := GenerateAccessToken()
	if err != nil {
		return Invitation{}, err
	}

	var member data.TenantOrganizationMember
	var token data.TenantAccessToken
	err = s.inTenant(ctx, input.OrganizationID, input.Actor, func(ctx context.Context, tx *data.Tx) error {
		members, err := tx.ListTenantOrganizationMembers(ctx)
		if err != nil {
			return err
		}
		for _, existing := range members {
			if strings.EqualFold(existing.Email, input.Email) || (existing.InvitedEmail != nil && strings.EqualFold(*existing.InvitedEmail, input.Email)) {
				return ErrAdministratorAlreadyExists
			}
		}
		token, err = tx.CreateTenantAccessToken(ctx, bearer.Hash, PurposeAdminInvitation, now.Add(ttl), 1)
		if err != nil {
			return err
		}
		member, err = tx.CreateTenantOrganizationMember(ctx, role, input.Email, &token.ID)
		if err != nil {
			return err
		}
		return recordAdministratorAudit(ctx, tx, audit.Entry{
			Action: audit.ActionAdministratorAdd, ObjectType: "administrator", ObjectID: &member.ID,
			ChangeSummary: summary(map[string]any{"email": input.Email, "role": role}),
		})
	})
	if err != nil {
		return Invitation{}, fmt.Errorf("invite administrator: %w", err)
	}
	claimURL, err := addTokenToURL(input.ClaimBaseURL, bearer.Value)
	if err != nil {
		return Invitation{}, fmt.Errorf("invite administrator: %w", err)
	}
	return Invitation{Member: administratorFromMember(member), ClaimURL: claimURL, ExpiresAt: token.ExpiresAt, Generation: token.Generation}, nil
}

func (s *Store) ListAdministrators(ctx context.Context, organizationID string) ([]Administrator, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, errors.New("list administrators: organization id is required")
	}
	var members []data.TenantOrganizationMember
	err := s.inTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		members, err = tx.ListTenantOrganizationMembers(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list administrators: %w", err)
	}
	result := make([]Administrator, 0, len(members))
	for _, member := range members {
		result = append(result, administratorFromMember(member))
	}
	return result, nil
}

func (s *Store) ResendAdministratorInvitation(ctx context.Context, input AdministratorActionInput, claimBaseURL string, ttl time.Duration) (Invitation, error) {
	if err := validateAdministratorInput(input.OrganizationID, "valid@example.test", claimBaseURL); err != nil {
		return Invitation{}, err
	}
	ttl, now, err := invitationTiming(ttl, input.Now)
	if err != nil {
		return Invitation{}, err
	}
	bearer, err := GenerateAccessToken()
	if err != nil {
		return Invitation{}, err
	}
	var member data.TenantOrganizationMember
	var token data.TenantAccessToken
	err = s.inTenant(ctx, input.OrganizationID, input.Actor, func(ctx context.Context, tx *data.Tx) error {
		var err error
		member, err = tx.GetTenantOrganizationMember(ctx, input.MemberID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAdministratorNotFound
		}
		if err != nil {
			return err
		}
		if member.UserID != nil || member.InvitedEmail == nil {
			return ErrInvitationNotPending
		}
		if member.InvitationTokenID == nil {
			token, err = tx.CreateTenantAccessToken(ctx, bearer.Hash, PurposeAdminInvitation, now.Add(ttl), 1)
		} else {
			token, err = tx.RegenerateTenantAccessToken(ctx, *member.InvitationTokenID, bearer.Hash, now.Add(ttl))
		}
		if err != nil {
			return err
		}
		updated, err := tx.SetTenantOrganizationMemberInvitation(ctx, member.ID, &token.ID)
		if err != nil {
			return err
		}
		if !updated {
			return ErrAdministratorNotFound
		}
		member.InvitationTokenID = &token.ID
		member.InvitationExpiresAt = &token.ExpiresAt
		return recordAdministratorAudit(ctx, tx, audit.Entry{
			Action: audit.ActionLinkRegenerate, ObjectType: "administrator_invitation", ObjectID: &member.ID,
			ChangeSummary: summary(map[string]any{"generation": token.Generation}),
		})
	})
	if err != nil {
		return Invitation{}, fmt.Errorf("resend administrator invitation: %w", err)
	}
	claimURL, err := addTokenToURL(claimBaseURL, bearer.Value)
	if err != nil {
		return Invitation{}, fmt.Errorf("resend administrator invitation: %w", err)
	}
	return Invitation{Member: administratorFromMember(member), ClaimURL: claimURL, ExpiresAt: token.ExpiresAt, Generation: token.Generation}, nil
}

func (s *Store) RevokeAdministratorInvitation(ctx context.Context, input AdministratorActionInput) error {
	err := s.inTenant(ctx, input.OrganizationID, input.Actor, func(ctx context.Context, tx *data.Tx) error {
		member, err := tx.GetTenantOrganizationMember(ctx, input.MemberID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAdministratorNotFound
		}
		if err != nil {
			return err
		}
		if member.UserID != nil || member.InvitedEmail == nil {
			return ErrInvitationNotPending
		}
		if member.InvitationTokenID != nil {
			if err := tx.RevokeTenantAccessToken(ctx, *member.InvitationTokenID); err != nil {
				return err
			}
		}
		updated, err := tx.SetTenantOrganizationMemberInvitation(ctx, member.ID, nil)
		if err != nil {
			return err
		}
		if !updated {
			return ErrAdministratorNotFound
		}
		return recordAdministratorAudit(ctx, tx, audit.Entry{
			Action: audit.ActionLinkRevoke, ObjectType: "administrator_invitation", ObjectID: &member.ID,
			ChangeSummary: summary(map[string]any{"revoked": true}),
		})
	})
	if err != nil {
		return fmt.Errorf("revoke administrator invitation: %w", err)
	}
	return nil
}

func (s *Store) ChangeAdministratorRole(ctx context.Context, input ChangeAdministratorRoleInput) (Administrator, error) {
	role := normalizeRole(input.Role)
	if role != RoleOwner && role != RoleAdministrator && role != RoleCoordinator {
		return Administrator{}, ErrAdministratorRole
	}
	var result data.TenantOrganizationMember
	err := s.inTenant(ctx, input.OrganizationID, input.Actor, func(ctx context.Context, tx *data.Tx) error {
		member, err := tx.GetTenantOrganizationMember(ctx, input.MemberID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAdministratorNotFound
		}
		if err != nil {
			return err
		}
		if member.Role == RoleOwner && role != RoleOwner && sameUser(member.UserID, input.Actor.UserID) {
			anotherOwner, err := hasAnotherOwner(ctx, tx, member.ID)
			if err != nil {
				return err
			}
			if !anotherOwner {
				return ErrLastOwner
			}
		}
		result, err = tx.UpdateTenantOrganizationMemberRole(ctx, member.ID, role)
		if err != nil {
			return err
		}
		result.Email = member.Email
		result.InvitedEmail = member.InvitedEmail
		result.InvitationTokenID = member.InvitationTokenID
		result.InvitationExpiresAt = member.InvitationExpiresAt
		return recordAdministratorAudit(ctx, tx, audit.Entry{
			Action: audit.ActionPermissionChange, ObjectType: "administrator", ObjectID: &member.ID,
			ChangeSummary: summary(map[string]any{"before_role": member.Role, "after_role": role}),
		})
	})
	if err != nil {
		return Administrator{}, fmt.Errorf("change administrator role: %w", err)
	}
	return administratorFromMember(result), nil
}

func (s *Store) RemoveAdministrator(ctx context.Context, input AdministratorActionInput) error {
	err := s.inTenant(ctx, input.OrganizationID, input.Actor, func(ctx context.Context, tx *data.Tx) error {
		member, err := tx.GetTenantOrganizationMember(ctx, input.MemberID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAdministratorNotFound
		}
		if err != nil {
			return err
		}
		if member.Role == RoleOwner && sameUser(member.UserID, input.Actor.UserID) {
			anotherOwner, err := hasAnotherOwner(ctx, tx, member.ID)
			if err != nil {
				return err
			}
			if !anotherOwner {
				return ErrLastOwner
			}
		}
		if member.InvitationTokenID != nil {
			if err := tx.RevokeTenantAccessToken(ctx, *member.InvitationTokenID); err != nil {
				return err
			}
		}
		deleted, err := tx.DeleteTenantOrganizationMember(ctx, member.ID)
		if err != nil {
			return err
		}
		if !deleted {
			return ErrAdministratorNotFound
		}
		return recordAdministratorAudit(ctx, tx, audit.Entry{
			Action: audit.ActionAdministratorRemove, ObjectType: "administrator", ObjectID: &member.ID,
			ChangeSummary: summary(map[string]any{"email": administratorEmail(member), "role": member.Role}),
		})
	})
	if err != nil {
		return fmt.Errorf("remove administrator: %w", err)
	}
	return nil
}

func (s *Store) inTenant(ctx context.Context, organizationID string, actor audit.Actor, fn func(context.Context, *data.Tx) error) error {
	if s == nil || s.tenantDatabase == nil {
		return errors.New("administrator management: data accessor is nil")
	}
	return s.tenantDatabase.InTenant(ctx, organizationID, actor, fn)
}

func (s *Store) inTenantRead(ctx context.Context, organizationID string, fn func(context.Context, *data.Tx) error) error {
	if s == nil || s.tenantDatabase == nil {
		return errors.New("administrator management: data accessor is nil")
	}
	return s.tenantDatabase.InTenantRead(ctx, organizationID, fn)
}

func hasAnotherOwner(ctx context.Context, tx *data.Tx, excluded ids.XID) (bool, error) {
	members, err := tx.ListTenantOrganizationMembers(ctx)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.ID != excluded && member.Role == RoleOwner {
			return true, nil
		}
	}
	return false, nil
}

func sameUser(memberID, actorID *ids.XID) bool {
	return memberID != nil && actorID != nil && *memberID == *actorID
}

func administratorFromMember(member data.TenantOrganizationMember) Administrator {
	return Administrator{ID: member.ID, Email: administratorEmail(member), Role: member.Role, PendingInvitation: member.UserID == nil, InvitationExpiresAt: member.InvitationExpiresAt}
}

func administratorEmail(member data.TenantOrganizationMember) string {
	if member.UserID == nil && member.InvitedEmail != nil {
		return *member.InvitedEmail
	}
	return member.Email
}

func normalizeRole(role string) string { return strings.ToLower(strings.TrimSpace(role)) }

func validateAdministratorInput(organizationID, email, claimBaseURL string) error {
	if strings.TrimSpace(organizationID) == "" {
		return errors.New("administrator organization id is required")
	}
	if strings.TrimSpace(email) == "" {
		return errors.New("administrator email is required")
	}
	if _, err := addTokenToURL(claimBaseURL, "placeholder"); err != nil {
		return err
	}
	return nil
}

func invitationTiming(ttl time.Duration, now time.Time) (time.Duration, time.Time, error) {
	if ttl == 0 {
		ttl = defaultInvitationLifetime
	}
	if ttl < 0 {
		return 0, time.Time{}, errors.New("invitation TTL must be positive")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return ttl, now.UTC(), nil
}

func recordAdministratorAudit(ctx context.Context, tx *data.Tx, entry audit.Entry) error {
	return tx.Record(ctx, entry)
}

func summary(value map[string]any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
