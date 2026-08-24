package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/auth"
)

// MeResponse is the authenticated account envelope. Organization and role
// are resolved server-side and never accepted from a URL or request body.
type MeResponse struct {
	Principal    MePrincipal    `json:"principal" doc:"The authenticated application principal."`
	Organization MeOrganization `json:"organization" doc:"The one organization resolved for the principal."`
	Role         string         `json:"role" doc:"The resolved organization role."`
}

type MePrincipal struct {
	ID    string `json:"id" doc:"Opaque application user identifier."`
	Email string `json:"email" doc:"Verified provider email."`
}

type MeOrganization struct {
	ID   string `json:"id" doc:"Opaque organization identifier."`
	Name string `json:"name" doc:"Organization display name."`
}

type MeOutput struct {
	Body MeResponse
}

// MeHandler returns the local principal resolved by authentication middleware.
type MeHandler struct{}

func (MeHandler) Handle(ctx context.Context, _ *struct{}) (*MeOutput, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	account, ok := principal.(auth.AccountPrincipal)
	if !ok {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "account principal has an unsupported type")
	}
	return &MeOutput{Body: MeResponse{
		Principal:    MePrincipal{ID: string(account.UserID), Email: account.Email},
		Organization: MeOrganization{ID: string(account.OrganizationID), Name: account.Organization},
		Role:         string(account.Role),
	}}, nil
}

// InvitationClaimer is the identity service operation needed by the claim
// endpoint. Keeping it as an interface makes the HTTP mapping unit-testable.
type InvitationClaimer = auth.InvitationClaimer

type ClaimInvitationInput struct {
	Body struct {
		Token string `json:"token" minLength:"1" doc:"The one-time admin invitation bearer value."`
	}
}

type ClaimInvitationOutput struct {
	Body MeResponse
}

// ClaimInvitationHandler consumes an invitation only after authentication
// middleware has verified the subject and email claims.
type ClaimInvitationHandler struct {
	claimer InvitationClaimer
}

func NewClaimInvitationHandler(claimer InvitationClaimer) *ClaimInvitationHandler {
	return &ClaimInvitationHandler{claimer: claimer}
}

func (h *ClaimInvitationHandler) Handle(ctx context.Context, input *ClaimInvitationInput) (*ClaimInvitationOutput, error) {
	if h == nil || h.claimer == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "invitation claim is not configured")
	}
	verified, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "verified identity is missing")
	}
	if input == nil || strings.TrimSpace(input.Body.Token) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.InvitationInvalid, "invitation token is required")
	}
	account, err := h.claimer.ClaimAdminInvitation(ctx, auth.InvitationClaimInput{
		Bearer: input.Body.Token, ProviderSubject: verified.Subject, Email: verified.Email, EmailVerified: verified.EmailVerified,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvitationUnverified):
			return nil, problems.New(http.StatusForbidden, problems.InvitationEmailUnverified, "the provider email is not verified")
		case errors.Is(err, auth.ErrInvitationEmail):
			return nil, problems.New(http.StatusForbidden, problems.InvitationEmailMismatch, "the invitation email does not match the verified email")
		case errors.Is(err, auth.ErrInvitationInvalid):
			return nil, problems.New(http.StatusForbidden, problems.InvitationInvalid, "the invitation token is invalid or no longer active")
		default:
			return nil, problems.New(http.StatusInternalServerError, problems.InternalError, "unable to claim invitation")
		}
	}
	return &ClaimInvitationOutput{Body: accountResponse(account)}, nil
}

func accountResponse(account auth.Account) MeResponse {
	return MeResponse{
		Principal:    MePrincipal{ID: string(account.User.ID), Email: account.User.Email},
		Organization: MeOrganization{ID: string(account.Membership.OrganizationID), Name: account.Membership.OrganizationName},
		Role:         account.Membership.Role,
	}
}
