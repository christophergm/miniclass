package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
)

type AdministratorManager = identity.AdministratorManager

type AdministratorResponse struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Role                string     `json:"role"`
	PendingInvitation   bool       `json:"pending_invitation"`
	InvitationExpiresAt *time.Time `json:"invitation_expires_at,omitempty"`
}

type AdministratorListOutput struct {
	Body struct {
		Members []AdministratorResponse `json:"members"`
	}
}

type AdministratorIDInput struct {
	MemberID string `path:"memberID" doc:"Opaque administrator membership identifier."`
}

type InviteAdministratorInput struct {
	Body struct {
		Email string `json:"email" minLength:"1" doc:"Email address to invite."`
		Role  string `json:"role,omitempty" doc:"administrator or coordinator; defaults to administrator."`
	}
}

type InviteAdministratorOutput struct {
	Body InvitationResponse
}

type InvitationResponse struct {
	Member     AdministratorResponse `json:"member"`
	ClaimURL   string                `json:"claim_url"`
	ExpiresAt  time.Time             `json:"expires_at"`
	Generation int                   `json:"generation"`
}

type ChangeAdministratorRoleInput struct {
	MemberID string `path:"memberID"`
	Body     struct {
		Role string `json:"role" minLength:"1" doc:"owner, administrator, or coordinator."`
	}
}

type AdministratorOutput struct {
	Body AdministratorResponse
}

type EmptyAdministratorOutput struct {
	Body struct{}
}

type AdministratorHandler struct {
	manager       AdministratorManager
	claimBaseURL  string
	invitationTTL time.Duration
}

func NewAdministratorHandler(manager AdministratorManager, claimBaseURL string) *AdministratorHandler {
	return &AdministratorHandler{manager: manager, claimBaseURL: claimBaseURL, invitationTTL: 48 * time.Hour}
}

func (h *AdministratorHandler) List(ctx context.Context, _ *struct{}) (*AdministratorListOutput, error) {
	principal, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.manager == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "administrator management is not configured")
	}
	members, err := h.manager.ListAdministrators(ctx, string(principal.OrganizationID))
	if err != nil {
		return nil, administratorProblem(err)
	}
	output := &AdministratorListOutput{}
	output.Body.Members = make([]AdministratorResponse, 0, len(members))
	for _, member := range members {
		output.Body.Members = append(output.Body.Members, administratorResponse(member))
	}
	return output, nil
}

func (h *AdministratorHandler) Invite(ctx context.Context, input *InviteAdministratorInput) (*InviteAdministratorOutput, error) {
	principal, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.manager == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "administrator management is not configured")
	}
	if input == nil || strings.TrimSpace(input.Body.Email) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.AdministratorRoleInvalid, "administrator email is required")
	}
	invitation, err := h.manager.InviteAdministrator(ctx, identity.InviteAdministratorInput{
		OrganizationID: string(principal.OrganizationID), Actor: administratorActor(principal), Email: input.Body.Email,
		Role: input.Body.Role, ClaimBaseURL: h.claimBaseURL, InvitationTTL: h.invitationTTL,
	})
	if err != nil {
		return nil, administratorProblem(err)
	}
	return &InviteAdministratorOutput{Body: invitationResponse(invitation)}, nil
}

func (h *AdministratorHandler) Resend(ctx context.Context, input *AdministratorIDInput) (*InviteAdministratorOutput, error) {
	principal, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	memberID, err := parseAdministratorID(input)
	if err != nil {
		return nil, err
	}
	if h == nil || h.manager == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "administrator management is not configured")
	}
	invitation, err := h.manager.ResendAdministratorInvitation(ctx, identity.AdministratorActionInput{
		OrganizationID: string(principal.OrganizationID), Actor: administratorActor(principal), MemberID: memberID,
	}, h.claimBaseURL, h.invitationTTL)
	if err != nil {
		return nil, administratorProblem(err)
	}
	return &InviteAdministratorOutput{Body: invitationResponse(invitation)}, nil
}

func (h *AdministratorHandler) Revoke(ctx context.Context, input *AdministratorIDInput) (*EmptyAdministratorOutput, error) {
	principal, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	memberID, err := parseAdministratorID(input)
	if err != nil {
		return nil, err
	}
	if h == nil || h.manager == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "administrator management is not configured")
	}
	err = h.manager.RevokeAdministratorInvitation(ctx, identity.AdministratorActionInput{
		OrganizationID: string(principal.OrganizationID), Actor: administratorActor(principal), MemberID: memberID,
	})
	if err != nil {
		return nil, administratorProblem(err)
	}
	return &EmptyAdministratorOutput{}, nil
}

func (h *AdministratorHandler) ChangeRole(ctx context.Context, input *ChangeAdministratorRoleInput) (*AdministratorOutput, error) {
	principal, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.MemberID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "administrator identifier is required")
	}
	if h == nil || h.manager == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "administrator management is not configured")
	}
	memberID := ids.XID(strings.TrimSpace(input.MemberID))
	member, err := h.manager.ChangeAdministratorRole(ctx, identity.ChangeAdministratorRoleInput{
		AdministratorActionInput: identity.AdministratorActionInput{OrganizationID: string(principal.OrganizationID), Actor: administratorActor(principal), MemberID: memberID},
		Role:                     input.Body.Role,
	})
	if err != nil {
		return nil, administratorProblem(err)
	}
	return &AdministratorOutput{Body: administratorResponse(member)}, nil
}

func (h *AdministratorHandler) Remove(ctx context.Context, input *AdministratorIDInput) (*EmptyAdministratorOutput, error) {
	principal, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	memberID, err := parseAdministratorID(input)
	if err != nil {
		return nil, err
	}
	if h == nil || h.manager == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "administrator management is not configured")
	}
	err = h.manager.RemoveAdministrator(ctx, identity.AdministratorActionInput{
		OrganizationID: string(principal.OrganizationID), Actor: administratorActor(principal), MemberID: memberID,
	})
	if err != nil {
		return nil, administratorProblem(err)
	}
	return &EmptyAdministratorOutput{}, nil
}

func administratorPrincipal(ctx context.Context) (auth.AccountPrincipal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	account, ok := principal.(auth.AccountPrincipal)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal has an unsupported type")
	}
	return account, nil
}

func administratorActor(principal auth.AccountPrincipal) audit.Actor {
	userID := principal.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: principal.Email}
}

func parseAdministratorID(input *AdministratorIDInput) (ids.XID, error) {
	if input == nil || strings.TrimSpace(input.MemberID) == "" {
		return "", problems.New(http.StatusBadRequest, problems.ResourceNotFound, "administrator identifier is required")
	}
	return ids.XID(strings.TrimSpace(input.MemberID)), nil
}

func administratorResponse(member identity.Administrator) AdministratorResponse {
	return AdministratorResponse{ID: string(member.ID), Email: member.Email, Role: member.Role, PendingInvitation: member.PendingInvitation, InvitationExpiresAt: member.InvitationExpiresAt}
}

func invitationResponse(invitation identity.Invitation) InvitationResponse {
	return InvitationResponse{Member: administratorResponse(invitation.Member), ClaimURL: invitation.ClaimURL, ExpiresAt: invitation.ExpiresAt, Generation: invitation.Generation}
}

func administratorProblem(err error) error {
	switch {
	case errors.Is(err, identity.ErrAdministratorNotFound):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "administrator not found")
	case errors.Is(err, identity.ErrAdministratorAlreadyExists):
		return problems.New(http.StatusConflict, problems.AdministratorConflict, "an administrator with that email already exists")
	case errors.Is(err, identity.ErrAdministratorRole):
		return problems.New(http.StatusBadRequest, problems.AdministratorRoleInvalid, "administrator role must be owner, administrator, or coordinator")
	case errors.Is(err, identity.ErrLastOwner):
		return problems.New(http.StatusConflict, problems.LastOwner, "the last owner cannot be removed or demoted")
	case errors.Is(err, identity.ErrInvitationNotPending):
		return problems.New(http.StatusConflict, problems.InvitationNotPending, "the administrator does not have a pending invitation")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "administrator operation failed")
	}
}
