package handlers

import (
	"context"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/guardian"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

type GuardianOnboardingService = guardian.Service

type GuardianOnboardingHandler struct {
	service GuardianOnboardingService
}

func NewGuardianOnboardingHandler(service GuardianOnboardingService) *GuardianOnboardingHandler {
	return &GuardianOnboardingHandler{service: service}
}

type GuardianSchoolYearInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
}

type GuardianContactInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ContactID    string `path:"contactID" minLength:"1"`
}

type GuardianOnboardingSessionPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	SessionID    string `path:"sessionID" minLength:"1"`
}

type GuardianRegistrationEntryResponse struct {
	ID         string    `json:"id"`
	Token      string    `json:"token,omitempty" doc:"Returned only when a new entry is issued."`
	ExpiresAt  time.Time `json:"expires_at"`
	Generation int       `json:"generation"`
}

type GuardianRegistrationEntryOutput struct {
	Body GuardianRegistrationEntryResponse
}

type GuardianEmptyOutput struct{ Body struct{} }

func (h *GuardianOnboardingHandler) CreateRegistrationEntry(ctx context.Context, input *GuardianSchoolYearInput) (*GuardianRegistrationEntryOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "school year is required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	entry, err := h.service.CreateRegistrationEntry(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), administratorActor(account), time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianRegistrationEntryOutput{Body: GuardianRegistrationEntryResponse{ID: string(entry.ID), Token: entry.Token, ExpiresAt: entry.ExpiresAt, Generation: entry.Generation}}, nil
}

func (h *GuardianOnboardingHandler) GetRegistrationEntry(ctx context.Context, input *GuardianSchoolYearInput) (*GuardianRegistrationEntryOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "school year is required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	entry, err := h.service.GetRegistrationEntry(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)))
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianRegistrationEntryOutput{Body: GuardianRegistrationEntryResponse{ID: string(entry.ID), ExpiresAt: entry.ExpiresAt, Generation: entry.Generation}}, nil
}

func (h *GuardianOnboardingHandler) RevokeRegistrationEntry(ctx context.Context, input *GuardianSchoolYearInput) (*GuardianEmptyOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "school year is required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if err := h.service.RevokeRegistrationEntry(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), administratorActor(account), time.Now().UTC()); err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianEmptyOutput{}, nil
}

type GuardianInvitationImportInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	RawBody      []byte `contentType:"text/csv"`
}

type GuardianInvitationImportOutput struct {
	Body guardian.InvitationImportResult
}

func (h *GuardianOnboardingHandler) ImportInvitationContacts(ctx context.Context, input *GuardianInvitationImportInput) (*GuardianInvitationImportOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" || len(input.RawBody) == 0 {
		return nil, problems.New(http.StatusBadRequest, problems.ImportInvalid, "school year and a CSV document are required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	result, err := h.service.ImportInvitationContacts(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), input.RawBody, administratorActor(account), time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianInvitationImportOutput{Body: result}, nil
}

type GuardianInvitationExportOutput struct {
	Body func(huma.Context)
}

func (h *GuardianOnboardingHandler) ExportInvitationContacts(ctx context.Context, input *GuardianSchoolYearInput) (*GuardianInvitationExportOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "school year is required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	rows, err := h.service.ListInvitationContacts(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), administratorActor(account))
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write([]string{"email", "status", "expires_at", "created_at", "consumed_at"})
	for _, row := range rows {
		consumed := ""
		if row.ConsumedAt != nil {
			consumed = row.ConsumedAt.UTC().Format(time.RFC3339)
		}
		_ = writer.Write([]string{row.Email, row.Status, row.ExpiresAt.UTC().Format(time.RFC3339), row.CreatedAt.UTC().Format(time.RFC3339), consumed})
	}
	writer.Flush()
	content := []byte(builder.String())
	return &GuardianInvitationExportOutput{Body: func(output huma.Context) {
		output.SetHeader("Content-Type", "text/csv; charset=utf-8")
		output.SetHeader("Content-Disposition", "attachment; filename=guardian-invitations.csv")
		output.SetStatus(http.StatusOK)
		_, _ = output.BodyWriter().Write(content)
	}}, nil
}

func (h *GuardianOnboardingHandler) RevokeInvitationContact(ctx context.Context, input *GuardianContactInput) (*GuardianEmptyOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.ContactID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "school year and contact are required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if err := h.service.RevokeInvitationContact(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), ids.XID(strings.TrimSpace(input.ContactID)), administratorActor(account), time.Now().UTC()); err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianEmptyOutput{}, nil
}

func (h *GuardianOnboardingHandler) RevokeOnboardingSession(ctx context.Context, input *GuardianOnboardingSessionPathInput) (*GuardianEmptyOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.SessionID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.SessionInvalid, "school year and onboarding session are required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if err := h.service.RevokeOnboardingSession(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), ids.XID(strings.TrimSpace(input.SessionID)), administratorActor(account), time.Now().UTC()); err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianEmptyOutput{}, nil
}

type GuardianSignupNoticeInput struct {
	Body struct {
		Content *string `json:"content" doc:"Optional notice text. Null or empty removes the notice."`
	}
}

type GuardianPolicyResponse struct {
	TermsVersion   string                        `json:"terms_version"`
	TermsNotice    string                        `json:"terms_notice"`
	PrivacyVersion string                        `json:"privacy_version"`
	PrivacyNotice  string                        `json:"privacy_notice"`
	SignupNotice   *GuardianSignupNoticeResponse `json:"signup_notice,omitempty"`
}

type GuardianSignupNoticeResponse struct {
	Content string `json:"content"`
	Version int    `json:"version"`
	Hash    string `json:"hash"`
}

type GuardianPolicyOutput struct{ Body GuardianPolicyResponse }

func (h *GuardianOnboardingHandler) UpdateSignupNotice(ctx context.Context, input *GuardianSignupNoticeInput) (*GuardianPolicyOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ConsentInvalid, "signup notice content is required")
	}
	policy, err := h.service.UpdateSignupNotice(ctx, account.OrganizationID, input.Body.Content, administratorActor(account), time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianPolicyOutput{Body: guardianPolicyResponse(policy)}, nil
}

type GuardianBeginInput struct {
	Body struct {
		EntryToken string `json:"entry_token" minLength:"1"`
	}
}

type GuardianInvitationRedeemInput struct {
	Body struct {
		InvitationToken string `json:"invitation_token" minLength:"1"`
	}
}

type GuardianOTPRequestInput struct {
	Body struct {
		SessionToken string `json:"session_token" minLength:"1"`
		Email        string `json:"email" minLength:"1" format:"email"`
	}
}

type GuardianOTPVerifyInput struct {
	Body struct {
		SessionToken string `json:"session_token" minLength:"1"`
		ChallengeID  string `json:"challenge_id" minLength:"1"`
		Code         string `json:"code" minLength:"1" maxLength:"6"`
	}
}

type GuardianConsentInput struct {
	Body struct {
		SessionToken        string `json:"session_token" minLength:"1"`
		Email               string `json:"email" minLength:"1" format:"email"`
		TermsVersion        string `json:"terms_version" minLength:"1"`
		PrivacyVersion      string `json:"privacy_version" minLength:"1"`
		SignupNoticeVersion *int   `json:"signup_notice_version,omitempty"`
		SignupNoticeHash    string `json:"signup_notice_hash,omitempty"`
		SourceSurface       string `json:"source_surface,omitempty"`
	}
}

type GuardianCompleteInput struct {
	Body struct {
		SessionToken      string `json:"session_token" minLength:"1"`
		AdultGivenName    string `json:"adult_given_name" minLength:"1"`
		AdultFamilyName   string `json:"adult_family_name" minLength:"1"`
		StudentGivenName  string `json:"student_given_name" minLength:"1"`
		StudentFamilyName string `json:"student_family_name" minLength:"1"`
		GradeLevelID      string `json:"grade_level_id" minLength:"1"`
		HomeroomID        string `json:"homeroom_id" minLength:"1"`
		RelationshipType  string `json:"relationship_type" minLength:"1"`
	}
}

type GuardianSessionOutput struct {
	Body GuardianOnboardingSessionResponse
}

type GuardianOnboardingSessionResponse struct {
	SessionToken   string                 `json:"session_token"`
	SessionID      string                 `json:"session_id"`
	OrganizationID string                 `json:"organization_id"`
	SchoolYearID   string                 `json:"school_year_id"`
	Email          string                 `json:"email,omitempty"`
	Verified       bool                   `json:"mailbox_verified"`
	Consented      bool                   `json:"consented"`
	ExpiresAt      time.Time              `json:"expires_at"`
	IdleExpiresAt  time.Time              `json:"idle_expires_at"`
	Policy         GuardianPolicyResponse `json:"policy"`
}

func (h *GuardianOnboardingHandler) Begin(ctx context.Context, input *GuardianBeginInput) (*GuardianSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.RegistrationInvalid, "registration entry is required")
	}
	session, err := h.service.Begin(ctx, guardian.BeginInput{EntryToken: input.Body.EntryToken, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session)}, nil
}

func (h *GuardianOnboardingHandler) Redeem(ctx context.Context, input *GuardianInvitationRedeemInput) (*GuardianSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.InvitationInvalid, "invitation token is required")
	}
	session, err := h.service.Redeem(ctx, guardian.RedeemInput{InvitationToken: input.Body.InvitationToken, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session)}, nil
}

type GuardianOTPRequestOutput struct {
	Body struct {
		Accepted    bool   `json:"accepted"`
		ChallengeID string `json:"challenge_id"`
	}
}

func (h *GuardianOnboardingHandler) RequestOTP(ctx context.Context, input *GuardianOTPRequestInput) (*GuardianOTPRequestOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.SessionInvalid, "onboarding session is required")
	}
	result, err := h.service.RequestOTP(ctx, guardian.OTPRequestInput{SessionToken: input.Body.SessionToken, Email: input.Body.Email, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianOTPRequestOutput{Body: struct {
		Accepted    bool   `json:"accepted"`
		ChallengeID string `json:"challenge_id"`
	}{Accepted: result.Accepted, ChallengeID: result.ChallengeID}}, nil
}

func (h *GuardianOnboardingHandler) VerifyOTP(ctx context.Context, input *GuardianOTPVerifyInput) (*GuardianSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.OTPInvalid, "OTP verification fields are required")
	}
	session, err := h.service.VerifyOTP(ctx, guardian.OTPVerifyInput{SessionToken: input.Body.SessionToken, ChallengeID: input.Body.ChallengeID, Code: input.Body.Code, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session)}, nil
}

func (h *GuardianOnboardingHandler) AcceptConsent(ctx context.Context, input *GuardianConsentInput) (*GuardianSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ConsentRequired, "consent fields are required")
	}
	hash, err := decodeOptionalHash(input.Body.SignupNoticeHash)
	if err != nil {
		return nil, problems.New(http.StatusBadRequest, problems.ConsentInvalid, "signup notice hash is invalid")
	}
	session, err := h.service.AcceptConsent(ctx, guardian.ConsentInput{SessionToken: input.Body.SessionToken, Email: input.Body.Email, TermsVersion: input.Body.TermsVersion, PrivacyVersion: input.Body.PrivacyVersion, SignupNoticeVersion: input.Body.SignupNoticeVersion, SignupNoticeHash: hash, SourceSurface: input.Body.SourceSurface, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session)}, nil
}

type GuardianCompletionResponse struct {
	AdultID        string `json:"adult_id"`
	StudentID      string `json:"student_id"`
	RelationshipID string `json:"relationship_id"`
	OrganizationID string `json:"organization_id"`
	SchoolYearID   string `json:"school_year_id"`
}

type GuardianCompletionOutput struct{ Body GuardianCompletionResponse }

func (h *GuardianOnboardingHandler) Complete(ctx context.Context, input *GuardianCompleteInput) (*GuardianCompletionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ConsentRequired, "completion fields are required")
	}
	completion, err := h.service.Complete(ctx, guardian.CompleteInput{SessionToken: input.Body.SessionToken, AdultGivenName: input.Body.AdultGivenName, AdultFamilyName: input.Body.AdultFamilyName, StudentGivenName: input.Body.StudentGivenName, StudentFamilyName: input.Body.StudentFamilyName, GradeLevelID: ids.XID(strings.TrimSpace(input.Body.GradeLevelID)), HomeroomID: ids.XID(strings.TrimSpace(input.Body.HomeroomID)), RelationshipType: guardianRelationshipType(input.Body.RelationshipType), Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianCompletionOutput{Body: GuardianCompletionResponse{AdultID: string(completion.AdultID), StudentID: string(completion.StudentID), RelationshipID: string(completion.RelationshipID), OrganizationID: string(completion.OrganizationID), SchoolYearID: string(completion.SchoolYearID)}}, nil
}

type GuardianSessionInput struct {
	Body struct {
		SessionToken string `json:"session_token" minLength:"1"`
	}
}

func (h *GuardianOnboardingHandler) GetSession(ctx context.Context, input *GuardianSessionInput) (*GuardianSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil || strings.TrimSpace(input.Body.SessionToken) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.SessionInvalid, "onboarding session is required")
	}
	session, err := h.service.GetSession(ctx, input.Body.SessionToken, time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session)}, nil
}

func guardianServiceUnavailable() error {
	return problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "guardian onboarding is not configured")
}

func guardianOnboardingProblem(err error) error {
	switch {
	case errors.Is(err, guardian.ErrRegistrationInvalid):
		return problems.New(http.StatusNotFound, problems.RegistrationInvalid, "registration entry is invalid or expired")
	case errors.Is(err, guardian.ErrInvitationInvalid):
		return problems.New(http.StatusNotFound, problems.InvitationInvalid, "invitation is invalid or expired")
	case errors.Is(err, guardian.ErrOnboardingInvalid):
		return problems.New(http.StatusUnauthorized, problems.SessionInvalid, "onboarding session is invalid or expired")
	case errors.Is(err, guardian.ErrMailboxUnverified):
		return problems.New(http.StatusConflict, problems.InvitationEmailUnverified, "verify the guardian mailbox before continuing")
	case errors.Is(err, guardian.ErrConsentRequired):
		return problems.New(http.StatusConflict, problems.ConsentRequired, "current terms and privacy acceptance is required")
	case errors.Is(err, guardian.ErrConsentInvalid), errors.Is(err, guardian.ErrSignupNoticeInvalid):
		return problems.New(http.StatusBadRequest, problems.ConsentInvalid, "the submitted consent does not match the current policy")
	case errors.Is(err, guardian.ErrOnboardingRateLimit):
		return problems.New(http.StatusTooManyRequests, problems.RateLimited, "too many onboarding requests")
	case errors.Is(err, guardian.ErrOnboardingEmailConflict):
		return problems.New(http.StatusConflict, problems.ConsentInvalid, "this email requires administrator review before continuing")
	case errors.Is(err, guardian.ErrStudentAttributesRequired):
		return problems.New(http.StatusBadRequest, problems.ConsentInvalid, "grade and homeroom are required")
	case errors.Is(err, guardian.ErrOTPInvalid):
		return problems.New(http.StatusUnauthorized, problems.OTPInvalid, "OTP is invalid or expired")
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case data.IsSchoolYearPurged(err):
		return schoolYearPurgedProblem()
	case errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "guardian onboarding resource not found")
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "guardian onboarding operation failed")
	}
}

func guardianPolicyResponse(policy guardian.Policy) GuardianPolicyResponse {
	result := GuardianPolicyResponse{TermsVersion: policy.TermsVersion, TermsNotice: policy.TermsNotice, PrivacyVersion: policy.PrivacyVersion, PrivacyNotice: policy.PrivacyNotice}
	if policy.SignupNotice != nil {
		result.SignupNotice = &GuardianSignupNoticeResponse{Content: policy.SignupNotice.Content, Version: policy.SignupNotice.Version, Hash: hex.EncodeToString(policy.SignupNotice.Hash)}
	}
	return result
}

func guardianOnboardingSessionResponse(session guardian.Session) GuardianOnboardingSessionResponse {
	return GuardianOnboardingSessionResponse{SessionToken: session.Token, SessionID: string(session.ID), OrganizationID: string(session.OrganizationID), SchoolYearID: string(session.SchoolYearID), Email: session.Email, Verified: session.Verified, Consented: session.Consented, ExpiresAt: session.ExpiresAt, IdleExpiresAt: session.IdleExpiresAt, Policy: guardianPolicyResponse(session.Policy)}
}

func decodeOptionalHash(value string) ([]byte, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(decoded) != 32 {
		return nil, errors.New("invalid hash")
	}
	return decoded, nil
}

func guardianRelationshipType(value string) data.GuardianRelationshipType {
	return data.GuardianRelationshipType(strings.TrimSpace(value))
}
