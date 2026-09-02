package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type AdultAuthHandler struct{ service auth.AdultAuthentication }

func NewAdultAuthHandler(service auth.AdultAuthentication) *AdultAuthHandler {
	return &AdultAuthHandler{service: service}
}

type RequestAdultOTPInput struct {
	Body struct {
		OrganizationID string `json:"organization_id" minLength:"1"`
		SchoolYearID   string `json:"school_year_id" minLength:"1"`
		Email          string `json:"email" minLength:"1" format:"email"`
	}
}

type RequestAdultOTPOutput struct {
	Body struct {
		Accepted    bool   `json:"accepted"`
		ChallengeID string `json:"challenge_id" doc:"Opaque single-use challenge bearer; never a persisted identifier."`
	}
}

func (h *AdultAuthHandler) RequestOTP(ctx context.Context, input *RequestAdultOTPInput) (*RequestAdultOTPOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	result, err := h.service.RequestAdultOTP(ctx, auth.OTPRequest{
		OrganizationID: ids.XID(strings.TrimSpace(input.Body.OrganizationID)),
		SchoolYearID:   ids.XID(strings.TrimSpace(input.Body.SchoolYearID)),
		Email:          input.Body.Email,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	output := &RequestAdultOTPOutput{}
	output.Body.Accepted = result.Accepted
	output.Body.ChallengeID = result.ChallengeID
	return output, nil
}

type VerifyAdultOTPInput struct {
	Body struct {
		ChallengeID string `json:"challenge_id" minLength:"1"`
		Code        string `json:"code" minLength:"1" maxLength:"6"`
	}
}

type GuardianSessionResponse struct {
	SessionToken string    `json:"session_token"`
	SessionID    string    `json:"session_id"`
	AdultID      string    `json:"adult_id"`
	Organization string    `json:"organization_id"`
	SchoolYear   string    `json:"school_year_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	IdleExpires  time.Time `json:"idle_expires_at"`
	StudentIDs   []string  `json:"student_ids"`
}

type VerifyAdultOTPOutput struct{ Body GuardianSessionResponse }

func (h *AdultAuthHandler) VerifyOTP(ctx context.Context, input *VerifyAdultOTPInput) (*VerifyAdultOTPOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	session, err := h.service.VerifyAdultOTP(ctx, auth.OTPVerification{ChallengeID: input.Body.ChallengeID, Code: input.Body.Code, Now: time.Now().UTC()})
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	return &VerifyAdultOTPOutput{Body: guardianSessionResponse(session)}, nil
}

type GuardianMeOutput struct {
	Body struct {
		AdultID      string   `json:"adult_id"`
		Email        string   `json:"email"`
		Organization string   `json:"organization_id"`
		SchoolYear   string   `json:"school_year_id"`
		StudentIDs   []string `json:"student_ids"`
		Mode         string   `json:"mode"`
	}
}

func (h *AdultAuthHandler) GuardianMe(ctx context.Context, _ *struct{}) (*GuardianMeOutput, error) {
	principal, err := guardianPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return &GuardianMeOutput{Body: struct {
		AdultID      string   `json:"adult_id"`
		Email        string   `json:"email"`
		Organization string   `json:"organization_id"`
		SchoolYear   string   `json:"school_year_id"`
		StudentIDs   []string `json:"student_ids"`
		Mode         string   `json:"mode"`
	}{AdultID: string(principal.AdultID), Email: principal.Email, Organization: string(principal.OrganizationID), SchoolYear: string(principal.SchoolYearID), StudentIDs: stringifyIDs(principal.StudentIDs), Mode: auth.ModeGuardian}}, nil
}

type RevokeSessionOutput struct{ Body struct{} }

func (h *AdultAuthHandler) Revoke(ctx context.Context, _ *struct{}) (*RevokeSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	var sessionID ids.XID
	switch value := principal.(type) {
	case auth.AccountPrincipal:
		sessionID = value.SessionID
	case auth.GuardianPrincipal:
		sessionID = value.SessionID
	default:
		return nil, problems.New(http.StatusBadRequest, problems.SessionInvalid, "the principal has no application session")
	}
	if sessionID == "" {
		return nil, problems.New(http.StatusBadRequest, problems.SessionInvalid, "the principal has no application session")
	}
	if err := h.service.RevokeSessionByID(ctx, sessionID); err != nil {
		return nil, adultAuthProblem(err)
	}
	return &RevokeSessionOutput{}, nil
}

type MFAEnrollmentOutput struct {
	Body struct {
		Secret        string   `json:"secret"`
		RecoveryCodes []string `json:"recovery_codes"`
		Generation    int      `json:"generation"`
	}
}

func (h *AdultAuthHandler) EnrollMFA(ctx context.Context, _ *struct{}) (*MFAEnrollmentOutput, error) {
	principal, err := accountPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	enrollment, err := h.service.EnrollMFA(ctx, principal.UserID, principal.OrganizationID, administratorActor(principal), time.Now().UTC())
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	return &MFAEnrollmentOutput{Body: struct {
		Secret        string   `json:"secret"`
		RecoveryCodes []string `json:"recovery_codes"`
		Generation    int      `json:"generation"`
	}{Secret: enrollment.Secret, RecoveryCodes: enrollment.RecoveryCodes, Generation: enrollment.Generation}}, nil
}

type VerifyMFAInput struct {
	Body struct {
		Code         string `json:"code,omitempty" maxLength:"6"`
		RecoveryCode string `json:"recovery_code,omitempty"`
	}
}

type AdministrativeSessionOutput struct {
	Body struct {
		SessionToken string    `json:"session_token"`
		SessionID    string    `json:"session_id"`
		UserID       string    `json:"user_id"`
		ExpiresAt    time.Time `json:"expires_at"`
		IdleExpires  time.Time `json:"idle_expires_at"`
		Mode         string    `json:"mode"`
	}
}

func (h *AdultAuthHandler) VerifyMFA(ctx context.Context, input *VerifyMFAInput) (*AdministrativeSessionOutput, error) {
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	var session auth.AdministrativeSession
	var err error
	switch value := principal.(type) {
	case auth.AccountPrincipal:
		session, err = h.service.VerifyMFA(ctx, auth.MFAVerification{UserID: value.UserID, OrganizationID: value.OrganizationID, Code: input.Body.Code, RecoveryCode: input.Body.RecoveryCode, Now: time.Now().UTC()})
	case auth.GuardianPrincipal:
		session, err = h.service.VerifyMFAForGuardian(ctx, value, input.Body.Code, input.Body.RecoveryCode, time.Now().UTC())
	default:
		err = auth.ErrMFAInvalid
	}
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	output := &AdministrativeSessionOutput{}
	output.Body.SessionToken = session.Bearer
	output.Body.SessionID = string(session.SessionID)
	output.Body.UserID = string(session.UserID)
	output.Body.ExpiresAt = session.ExpiresAt
	output.Body.IdleExpires = session.IdleExpiresAt
	output.Body.Mode = auth.ModeAdministration
	return output, nil
}

type ResetMFAInput struct {
	Body struct {
		UserID string `json:"user_id" minLength:"1"`
		Reason string `json:"reason" minLength:"1"`
	}
}

func (h *AdultAuthHandler) ResetMFA(ctx context.Context, input *ResetMFAInput) (*struct{}, error) {
	principal, err := accountPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	if principal.Role != auth.RoleOwner {
		return nil, problems.New(http.StatusForbidden, problems.CapabilityRequired, "only an Owner may reset MFA")
	}
	err = h.service.ResetMFA(ctx, auth.MFAReset{OrganizationID: principal.OrganizationID, Actor: administratorActor(principal), TargetUserID: ids.XID(strings.TrimSpace(input.Body.UserID)), Reason: input.Body.Reason, Now: time.Now().UTC()})
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	return &struct{}{}, nil
}

type CreateAdultAccountLinkInput struct {
	Body struct {
		SchoolYearID string `json:"school_year_id" minLength:"1"`
		AdultID      string `json:"adult_id" minLength:"1"`
		UserID       string `json:"user_id" minLength:"1"`
	}
}

type AdultAccountLinkResponse struct {
	ID           string `json:"id"`
	Organization string `json:"organization_id"`
	SchoolYear   string `json:"school_year_id"`
	AdultID      string `json:"adult_id"`
	UserID       string `json:"user_id"`
}

type AdultAccountLinkOutput struct{ Body AdultAccountLinkResponse }

type ListAdultAccountLinksInput struct {
	SchoolYearID string `query:"school_year_id" minLength:"1" required:"true"`
}

type AdultAccountLinkListOutput struct{ Body []AdultAccountLinkResponse }

func (h *AdultAuthHandler) ListLinks(ctx context.Context, input *ListAdultAccountLinksInput) (*AdultAccountLinkListOutput, error) {
	principal, err := accountPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	links, err := h.service.ListAdultAccountLinks(ctx, principal.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)))
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	output := &AdultAccountLinkListOutput{Body: make([]AdultAccountLinkResponse, 0, len(links))}
	for _, link := range links {
		output.Body = append(output.Body, adultAccountLinkResponse(link))
	}
	return output, nil
}

func (h *AdultAuthHandler) CreateLink(ctx context.Context, input *CreateAdultAccountLinkInput) (*AdultAccountLinkOutput, error) {
	principal, err := accountPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	link, err := h.service.CreateAdultAccountLink(ctx, auth.AdultAccountLinkInput{OrganizationID: principal.OrganizationID, SchoolYearID: ids.XID(strings.TrimSpace(input.Body.SchoolYearID)), AdultID: ids.XID(strings.TrimSpace(input.Body.AdultID)), UserID: ids.XID(strings.TrimSpace(input.Body.UserID)), Actor: administratorActor(principal)})
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	return &AdultAccountLinkOutput{Body: adultAccountLinkResponse(link)}, nil
}

type DeleteAdultAccountLinkInput struct {
	LinkID       string `path:"linkID" minLength:"1"`
	SchoolYearID string `query:"school_year_id" minLength:"1" required:"true"`
}

func (h *AdultAuthHandler) DeleteLink(ctx context.Context, input *DeleteAdultAccountLinkInput) (*struct{}, error) {
	principal, err := accountPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	err = h.service.DeleteAdultAccountLink(ctx, auth.AdultAccountLinkInput{OrganizationID: principal.OrganizationID, SchoolYearID: ids.XID(strings.TrimSpace(input.SchoolYearID)), Actor: administratorActor(principal)}, ids.XID(strings.TrimSpace(input.LinkID)))
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	return &struct{}{}, nil
}

type GuardianModeInput struct {
	Body struct {
		SchoolYearID string `json:"school_year_id" minLength:"1"`
	}
}

func (h *AdultAuthHandler) GuardianMode(ctx context.Context, input *GuardianModeInput) (*VerifyAdultOTPOutput, error) {
	principal, err := accountPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if !principal.MFAAuthenticated {
		return nil, problems.New(http.StatusForbidden, problems.MFARequired, "MFA reauthentication is required before changing modes")
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "adult authentication is not configured")
	}
	session, err := h.service.CreateGuardianSessionForAccount(ctx, principal.UserID, principal.OrganizationID, ids.XID(strings.TrimSpace(input.Body.SchoolYearID)), time.Now().UTC())
	if err != nil {
		return nil, adultAuthProblem(err)
	}
	return &VerifyAdultOTPOutput{Body: guardianSessionResponse(session)}, nil
}

func accountPrincipal(ctx context.Context) (auth.AccountPrincipal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	account, ok := principal.(auth.AccountPrincipal)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusForbidden, problems.CapabilityRequired, "an administrative account principal is required")
	}
	return account, nil
}

func guardianPrincipal(ctx context.Context) (auth.GuardianPrincipal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.GuardianPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	guardian, ok := principal.(auth.GuardianPrincipal)
	if !ok {
		return auth.GuardianPrincipal{}, problems.New(http.StatusForbidden, problems.CapabilityRequired, "a guardian principal is required")
	}
	return guardian, nil
}

func guardianSessionResponse(session auth.GuardianSession) GuardianSessionResponse {
	return GuardianSessionResponse{SessionToken: session.Bearer, SessionID: string(session.SessionID), AdultID: string(session.AdultID), Organization: string(session.OrganizationID), SchoolYear: string(session.SchoolYearID), ExpiresAt: session.ExpiresAt, IdleExpires: session.IdleExpiresAt, StudentIDs: stringifyIDs(session.StudentIDs)}
}

func adultAccountLinkResponse(link auth.AdultAccountLink) AdultAccountLinkResponse {
	return AdultAccountLinkResponse{ID: string(link.ID), Organization: string(link.OrganizationID), SchoolYear: string(link.SchoolYearID), AdultID: string(link.AdultID), UserID: string(link.UserID)}
}

func stringifyIDs(values []ids.XID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func adultAuthProblem(err error) error {
	switch {
	case errors.Is(err, auth.ErrOTPInvalid):
		return problems.New(http.StatusUnauthorized, problems.OTPInvalid, "the OTP is invalid, expired, or already used")
	case errors.Is(err, auth.ErrOTPUnavailable):
		return problems.New(http.StatusServiceUnavailable, problems.OTPDeliveryUnavailable, "transactional OTP delivery is unavailable")
	case errors.Is(err, auth.ErrMFAAlreadyEnrolled):
		return problems.New(http.StatusConflict, problems.MFAAlreadyEnrolled, "MFA is already enrolled")
	case errors.Is(err, auth.ErrMFANotEnrolled):
		return problems.New(http.StatusConflict, problems.MFANotEnrolled, "MFA is not enrolled")
	case errors.Is(err, auth.ErrMFAInvalid):
		return problems.New(http.StatusUnauthorized, problems.MFAInvalid, "the MFA proof is invalid")
	case errors.Is(err, auth.ErrMFAResetReasonRequired):
		return problems.New(http.StatusBadRequest, problems.MFAResetReasonRequired, "an MFA reset reason is required")
	case errors.Is(err, auth.ErrAdultAccountLinkMissing), errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.AdultAccountLinkMissing, "the adult account link was not found")
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return problems.New(http.StatusConflict, problems.AdministratorConflict, "the adult account link already exists")
		}
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to complete adult authentication")
	}
}
