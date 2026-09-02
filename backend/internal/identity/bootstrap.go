package identity

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/auth"
	data "github.com/chrismott/miniclass/internal/data"
	identitydata "github.com/chrismott/miniclass/internal/data/identity"
	"github.com/chrismott/miniclass/internal/ids"
)

const defaultInvitationLifetime = 48 * time.Hour

// Store is the identity service's unscoped data boundary. Keeping the
// accessor private here ensures callers cannot use it to reach generated SQL
// or accidentally add tenant state to an identity transaction.
type Store struct {
	database       *identitydata.DB
	tenantDatabase *data.DB
	authKey        []byte
	otpDelivery    auth.OTPDelivery
}

// NewStore creates the identity data boundary from the application's database
// owner. Identity callers never need to import internal/data/identity.
func NewStore(database *data.DB) *Store {
	return NewStoreWithAuth(database, nil, nil)
}

// NewStoreWithAuth creates the identity store with the key material and
// transactional OTP delivery boundary used by Phase 4.
func NewStoreWithAuth(database *data.DB, authKey []byte, delivery auth.OTPDelivery) *Store {
	if database == nil {
		return nil
	}
	if len(authKey) == 0 {
		fallback := sha256.Sum256([]byte("miniclass-development-adult-auth-key"))
		authKey = fallback[:]
	}
	key := append([]byte(nil), authKey...)
	if len(key) != 32 {
		digest := sha256.Sum256(key)
		key = digest[:]
	}
	return &Store{database: identitydata.New(database.Pool()), tenantDatabase: database, authKey: key, otpDelivery: delivery}
}

// ResolveAccount maps a verified provider subject to one local membership.
func (s *Store) ResolveAccount(ctx context.Context, providerSubject string) (auth.Account, error) {
	if s == nil || s.database == nil {
		return auth.Account{}, errors.New("resolve account: data accessor is nil")
	}
	account, err := s.database.ResolveAccount(ctx, providerSubject)
	if errors.Is(err, identitydata.ErrNoOrganization) {
		return auth.Account{}, auth.ErrNoOrganization
	}
	if errors.Is(err, identitydata.ErrMultipleOrganizations) {
		return auth.Account{}, auth.ErrMultipleOrganizations
	}
	if err != nil {
		return auth.Account{}, err
	}
	return auth.Account{
		User: auth.AccountUser{ID: account.User.ID, ProviderSubject: account.User.ProviderSubject, Email: account.User.Email},
		Membership: auth.AccountMembership{
			ID: account.Membership.ID, OrganizationID: account.Membership.OrganizationID,
			OrganizationName: account.Membership.OrganizationName, Role: account.Membership.Role,
		},
	}, nil
}

// ClaimAdminInvitation binds a verified provider identity to an invitation.
func (s *Store) ClaimAdminInvitation(ctx context.Context, input auth.InvitationClaimInput) (auth.Account, error) {
	if s == nil || s.database == nil {
		return auth.Account{}, errors.New("claim admin invitation: data accessor is nil")
	}
	account, err := s.database.ClaimAdminInvitation(ctx, identitydata.ClaimInput{
		Bearer: input.Bearer, ProviderSubject: input.ProviderSubject, Email: input.Email, EmailVerified: input.EmailVerified,
	})
	if errors.Is(err, identitydata.ErrInvitationUnverified) {
		return auth.Account{}, auth.ErrInvitationUnverified
	}
	if errors.Is(err, identitydata.ErrInvitationEmail) {
		return auth.Account{}, auth.ErrInvitationEmail
	}
	if errors.Is(err, identitydata.ErrInvitationInvalid) {
		return auth.Account{}, auth.ErrInvitationInvalid
	}
	if err != nil {
		return auth.Account{}, err
	}
	return auth.Account{
		User: auth.AccountUser{ID: account.User.ID, ProviderSubject: account.User.ProviderSubject, Email: account.User.Email},
		Membership: auth.AccountMembership{
			ID: account.Membership.ID, OrganizationID: account.Membership.OrganizationID,
			OrganizationName: account.Membership.OrganizationName, Role: account.Membership.Role,
		},
	}, nil
}

// GetAccessTokenByBearer resolves a bearer value to its persisted token
// record without exposing raw token hashes to callers.
func (s *Store) GetAccessTokenByBearer(ctx context.Context, bearer string) (identitydata.AccessToken, error) {
	if s == nil || s.database == nil {
		return identitydata.AccessToken{}, errors.New("get access token: data accessor is nil")
	}
	hash, err := HashAccessToken(bearer)
	if err != nil {
		return identitydata.AccessToken{}, err
	}
	var token identitydata.AccessToken
	err = s.database.InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		token, err = tx.GetAccessTokenByHash(ctx, hash)
		return err
	})
	return token, err
}

// GetInvitationMember retrieves the membership attached to an invitation.
func (s *Store) GetInvitationMember(ctx context.Context, tokenID ids.XID) (identitydata.OrganizationMember, error) {
	if s == nil || s.database == nil {
		return identitydata.OrganizationMember{}, errors.New("get organization member: data accessor is nil")
	}
	var member identitydata.OrganizationMember
	err := s.database.InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		member, err = tx.GetOrganizationMemberByInvitationToken(ctx, tokenID)
		return err
	})
	return member, err
}

// ConsumeAccessToken atomically performs the one-time transition for a token.
func (s *Store) ConsumeAccessToken(ctx context.Context, tokenID ids.XID) (bool, error) {
	if s == nil || s.database == nil {
		return false, errors.New("consume access token: data accessor is nil")
	}
	var consumed bool
	err := s.database.InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		consumed, err = tx.ConsumeAccessToken(ctx, tokenID)
		return err
	})
	return consumed, err
}

// BootstrapInput describes the first organization and Owner invitation.
type BootstrapInput struct {
	OrganizationName string
	HomeroomLabel    string
	OwnerEmail       string
	ClaimBaseURL     string
	InvitationTTL    time.Duration
	Now              time.Time
}

// BootstrapResult contains the created records and the one value that may be
// printed to the operator. The token hash is available in Token; TokenValue is
// intentionally not returned by the data layer.
type BootstrapResult struct {
	Organization identitydata.Organization
	Member       identitydata.OrganizationMember
	Token        identitydata.AccessToken
	TokenValue   string
	ClaimURL     string
}

// Bootstrap creates the organization, pending Owner membership, and invitation
// token in one unscoped transaction. It uses the same normal application data
// path as later identity operations and does not grant the process a runtime
// database privilege.
func Bootstrap(ctx context.Context, store *Store, input BootstrapInput) (BootstrapResult, error) {
	if store == nil || store.database == nil {
		return BootstrapResult{}, errors.New("bootstrap identity: data accessor is nil")
	}
	name := strings.TrimSpace(input.OrganizationName)
	label := strings.TrimSpace(input.HomeroomLabel)
	email := strings.ToLower(strings.TrimSpace(input.OwnerEmail))
	if name == "" {
		return BootstrapResult{}, errors.New("bootstrap identity: organization name is required")
	}
	if label == "" {
		return BootstrapResult{}, errors.New("bootstrap identity: homeroom label is required")
	}
	if email == "" {
		return BootstrapResult{}, errors.New("bootstrap identity: owner email is required")
	}
	if strings.TrimSpace(input.ClaimBaseURL) == "" {
		return BootstrapResult{}, errors.New("bootstrap identity: claim base URL is required")
	}
	if err := validateClaimBaseURL(input.ClaimBaseURL); err != nil {
		return BootstrapResult{}, fmt.Errorf("bootstrap identity: %w", err)
	}
	ttl := input.InvitationTTL
	if ttl == 0 {
		ttl = defaultInvitationLifetime
	}
	if ttl < 0 {
		return BootstrapResult{}, errors.New("bootstrap identity: invitation TTL must be positive")
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	bearer, err := GenerateAccessToken()
	if err != nil {
		return BootstrapResult{}, err
	}

	var result BootstrapResult
	err = store.database.InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		organization, err := tx.CreateOrganization(ctx, name, label)
		if err != nil {
			return err
		}
		token, err := tx.CreateAccessToken(ctx, bearer.Hash, PurposeAdminInvitation, now.Add(ttl), 1)
		if err != nil {
			return err
		}
		member, err := tx.CreateOrganizationMember(ctx, organization.ID, nil, "owner", &email, &token.ID)
		if err != nil {
			return err
		}
		result = BootstrapResult{
			Organization: organization,
			Member:       member,
			Token:        token,
			TokenValue:   bearer.Value,
		}
		return nil
	})
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("bootstrap identity: %w", err)
	}

	claimURL, err := addTokenToURL(input.ClaimBaseURL, bearer.Value)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("bootstrap identity: %w", err)
	}
	result.ClaimURL = claimURL
	return result, nil
}

// RegeneratedInvitation is the replacement bearer value and persisted token
// for an existing administrator invitation.
type RegeneratedInvitation struct {
	Token      identitydata.AccessToken
	TokenValue string
	ClaimURL   string
}

// RegenerateAdminInvitation issues a fresh generation for a pending Owner
// invitation, revokes the old generation, and repoints the membership in one
// transaction. The previous bearer URL therefore stops resolving as soon as
// this function returns successfully.
func RegenerateAdminInvitation(ctx context.Context, store *Store, tokenID string, claimBaseURL string, invitationTTL time.Duration, now time.Time) (RegeneratedInvitation, error) {
	if store == nil || store.database == nil {
		return RegeneratedInvitation{}, errors.New("regenerate invitation: data accessor is nil")
	}
	if strings.TrimSpace(tokenID) == "" {
		return RegeneratedInvitation{}, errors.New("regenerate invitation: token id is required")
	}
	if err := validateClaimBaseURL(claimBaseURL); err != nil {
		return RegeneratedInvitation{}, fmt.Errorf("regenerate invitation: %w", err)
	}
	ttl := invitationTTL
	if ttl == 0 {
		ttl = defaultInvitationLifetime
	}
	if ttl < 0 {
		return RegeneratedInvitation{}, errors.New("regenerate invitation: invitation TTL must be positive")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	bearer, err := GenerateAccessToken()
	if err != nil {
		return RegeneratedInvitation{}, err
	}

	var result RegeneratedInvitation
	err = store.database.InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		current, err := tx.GetAccessTokenByID(ctx, ids.XID(tokenID))
		if err != nil {
			return err
		}
		if current.Purpose != PurposeAdminInvitation {
			return fmt.Errorf("token purpose is %q, want %q", current.Purpose, PurposeAdminInvitation)
		}
		if current.RevokedAt != nil || current.ConsumedAt != nil {
			return errors.New("invitation token is no longer active")
		}
		replacement, err := tx.RegenerateAccessToken(ctx, current.ID, bearer.Hash, now.Add(ttl))
		if err != nil {
			return err
		}
		replaced, err := tx.ReplaceOrganizationMemberInvitation(ctx, current.ID, replacement.ID)
		if err != nil {
			return err
		}
		if !replaced {
			return errors.New("token is not attached to an organization invitation")
		}
		result = RegeneratedInvitation{Token: replacement, TokenValue: bearer.Value}
		return nil
	})
	if err != nil {
		return RegeneratedInvitation{}, fmt.Errorf("regenerate invitation: %w", err)
	}
	result.ClaimURL, err = addTokenToURL(claimBaseURL, bearer.Value)
	if err != nil {
		return RegeneratedInvitation{}, fmt.Errorf("regenerate invitation: %w", err)
	}
	return result, nil
}

func addTokenToURL(baseURL, token string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("claim base URL must be an absolute URL")
	}
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func validateClaimBaseURL(baseURL string) error {
	_, err := addTokenToURL(baseURL, "placeholder")
	return err
}
