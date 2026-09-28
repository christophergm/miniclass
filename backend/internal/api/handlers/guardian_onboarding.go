package handlers

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
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

type GuardianInvitationContactsInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	Cursor       string `query:"cursor"`
	Limit        int32  `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

type GuardianInvitationContactResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status" enum:"pending,redeemed,expired,revoked"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type GuardianInvitationContactsOutput struct {
	Body struct {
		Contacts   []GuardianInvitationContactResponse `json:"contacts"`
		OpenCount  int64                               `json:"open_count"`
		NextCursor string                              `json:"next_cursor,omitempty"`
	}
}

type GuardianOnboardingSessionPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	SessionID    string `path:"sessionID" minLength:"1"`
}

type GuardianRegistrationEntryResponse struct {
	ID              string     `json:"id" doc:"Public shared registration-link identifier."`
	Status          string     `json:"status" enum:"active,expired,revoked"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	RevocationKind  string     `json:"revocation_kind,omitempty" enum:"manual,replaced,legacy_cutover"`
	RevokedByUserID string     `json:"revoked_by_user_id,omitempty"`
	Generation      int        `json:"generation"`
}

type GuardianRegistrationEntryOutput struct {
	Body GuardianRegistrationEntryResponse
}

type GuardianRegistrationEntryIssueInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	Body         struct {
		ExpiresAt time.Time `json:"expires_at"`
	}
}

type GuardianRegistrationLinkInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	LinkID       string `path:"linkID" minLength:"1"`
}
type GuardianRegistrationLinksInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	Cursor       string `query:"cursor"`
	Limit        int32  `query:"limit" minimum:"1" maximum:"100" default:"10"`
}
type GuardianRegistrationLinkUpdateInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	LinkID       string `path:"linkID" minLength:"1"`
	Body         struct {
		ExpiresAt time.Time `json:"expires_at"`
	}
}
type GuardianRegistrationLinksOutput struct {
	Body struct {
		Entries    []GuardianRegistrationEntryResponse `json:"entries"`
		NextCursor string                              `json:"next_cursor,omitempty"`
	}
}
type GuardianEmptyOutput struct{ Body struct{} }

func (h *GuardianOnboardingHandler) CreateRegistrationEntry(ctx context.Context, input *GuardianRegistrationEntryIssueInput) (*GuardianRegistrationEntryOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" || input.Body.ExpiresAt.IsZero() {
		return nil, problems.New(http.StatusBadRequest, problems.RegistrationInvalid, "school year and future expiration are required")
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	now := time.Now().UTC()
	entry, err := h.service.IssueRegistrationEntry(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), input.Body.ExpiresAt.UTC(), administratorActor(account), now)
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianRegistrationEntryOutput{Body: guardianRegistrationEntryResponse(entry)}, nil
}

func guardianRegistrationEntryResponse(entry guardian.RegistrationEntry) GuardianRegistrationEntryResponse {
	response := GuardianRegistrationEntryResponse{ID: string(entry.ID), Status: entry.Status, ExpiresAt: entry.ExpiresAt, CreatedAt: entry.CreatedAt, RevokedAt: entry.RevokedAt, RevocationKind: entry.RevocationKind, Generation: entry.Generation}
	if entry.RevokedByUserID != nil {
		response.RevokedByUserID = string(*entry.RevokedByUserID)
	}
	return response
}

func (h *GuardianOnboardingHandler) ListRegistrationEntries(ctx context.Context, input *GuardianRegistrationLinksInput) (*GuardianRegistrationLinksOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "school year is required")
	}
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}
	var cursor *guardian.RegistrationEntryCursor
	if input.Cursor != "" {
		value, err := decodeGuardianRegistrationCursor(input.Cursor)
		if err != nil {
			return nil, problems.New(http.StatusBadRequest, problems.RegistrationInvalid, "cursor is invalid")
		}
		cursor = &value
	}
	page, err := h.service.ListRegistrationEntries(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), cursor, limit, time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	output := &GuardianRegistrationLinksOutput{}
	output.Body.Entries = make([]GuardianRegistrationEntryResponse, len(page.Entries))
	for i, entry := range page.Entries {
		output.Body.Entries[i] = guardianRegistrationEntryResponse(entry)
	}
	if page.NextCursor != nil {
		output.Body.NextCursor = encodeGuardianRegistrationCursor(*page.NextCursor)
	}
	return output, nil
}

func (h *GuardianOnboardingHandler) UpdateRegistrationEntry(ctx context.Context, input *GuardianRegistrationLinkUpdateInput) (*GuardianRegistrationEntryOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil || input.Body.ExpiresAt.IsZero() {
		return nil, problems.New(http.StatusBadRequest, problems.RegistrationInvalid, "future expiration is required")
	}
	entry, err := h.service.UpdateRegistrationEntry(ctx, account.OrganizationID, ids.XID(input.SchoolYearID), ids.XID(input.LinkID), guardian.RegistrationEntryUpdateInput{ExpiresAt: input.Body.ExpiresAt.UTC()}, administratorActor(account), time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianRegistrationEntryOutput{Body: guardianRegistrationEntryResponse(entry)}, nil
}

func (h *GuardianOnboardingHandler) RevokeRegistrationEntryByID(ctx context.Context, input *GuardianRegistrationLinkInput) (*GuardianEmptyOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.LinkID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.RegistrationInvalid, "school year and link are required")
	}
	if err := h.service.RevokeRegistrationEntryByID(ctx, account.OrganizationID, ids.XID(input.SchoolYearID), ids.XID(input.LinkID), administratorActor(account), time.Now().UTC()); err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianEmptyOutput{}, nil
}

func encodeGuardianRegistrationCursor(cursor guardian.RegistrationEntryCursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}
func decodeGuardianRegistrationCursor(value string) (guardian.RegistrationEntryCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return guardian.RegistrationEntryCursor{}, err
	}
	var cursor guardian.RegistrationEntryCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return guardian.RegistrationEntryCursor{}, errors.New("invalid cursor")
	}
	return cursor, nil
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
	return &GuardianRegistrationEntryOutput{Body: guardianRegistrationEntryResponse(entry)}, nil
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

func (h *GuardianOnboardingHandler) ListInvitationContacts(ctx context.Context, input *GuardianInvitationContactsInput) (*GuardianInvitationContactsOutput, error) {
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
	limit := input.Limit
	if limit == 0 {
		limit = 50
	}
	var cursor *guardian.InvitationContactCursor
	if input.Cursor != "" {
		value, err := decodeGuardianInvitationContactCursor(input.Cursor)
		if err != nil {
			return nil, problems.New(http.StatusBadRequest, problems.InvitationInvalid, "cursor is invalid")
		}
		cursor = &value
	}
	page, err := h.service.ListInvitationContactPage(ctx, account.OrganizationID, ids.XID(strings.TrimSpace(input.SchoolYearID)), cursor, limit, time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	output := &GuardianInvitationContactsOutput{}
	output.Body.OpenCount = page.OpenCount
	output.Body.Contacts = make([]GuardianInvitationContactResponse, len(page.Contacts))
	for i, contact := range page.Contacts {
		output.Body.Contacts[i] = GuardianInvitationContactResponse{ID: string(contact.ID), Email: contact.Email, Status: contact.Status, ExpiresAt: contact.ExpiresAt, CreatedAt: contact.CreatedAt}
	}
	if page.NextCursor != nil {
		output.Body.NextCursor = encodeGuardianInvitationContactCursor(*page.NextCursor)
	}
	return output, nil
}

func encodeGuardianInvitationContactCursor(cursor guardian.InvitationContactCursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeGuardianInvitationContactCursor(value string) (guardian.InvitationContactCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return guardian.InvitationContactCursor{}, err
	}
	var cursor guardian.InvitationContactCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return guardian.InvitationContactCursor{}, errors.New("invalid cursor")
	}
	return cursor, nil
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

func (h *GuardianOnboardingHandler) GetSignupNotice(ctx context.Context, _ *struct{}) (*GuardianPolicyOutput, error) {
	account, err := administratorPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	policy, err := h.service.GetSignupNotice(ctx, account.OrganizationID)
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianPolicyOutput{Body: guardianPolicyResponse(policy)}, nil
}

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
type GuardianBeginPathInput struct {
	RegistrationLinkID string `path:"registrationLinkID" minLength:"1"`
}

type GuardianRegistrationLandingResponse struct {
	OrganizationName string `json:"organization_name"`
	SchoolYearLabel  string `json:"school_year_label"`
}

type GuardianRegistrationLandingOutput struct {
	Body GuardianRegistrationLandingResponse
}

func (h *GuardianOnboardingHandler) GetRegistrationLanding(ctx context.Context, input *GuardianBeginPathInput) (*GuardianRegistrationLandingOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil || strings.TrimSpace(input.RegistrationLinkID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.RegistrationInvalid, "registration entry is invalid or expired")
	}
	landing, err := h.service.RegistrationLanding(ctx, input.RegistrationLinkID, time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianRegistrationLandingOutput{Body: GuardianRegistrationLandingResponse{OrganizationName: landing.OrganizationName, SchoolYearLabel: landing.SchoolYearLabel}}, nil
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
		SessionToken    string `json:"session_token" minLength:"1"`
		AdultGivenName  string `json:"adult_given_name" minLength:"1"`
		AdultFamilyName string `json:"adult_family_name" minLength:"1"`
	}
}

type GuardianSessionOutput struct {
	Body GuardianOnboardingSessionResponse
}

type GuardianOnboardingSessionResponse struct {
	SessionToken         string                     `json:"session_token"`
	SessionID            string                     `json:"session_id"`
	OrganizationID       string                     `json:"organization_id"`
	SchoolYearID         string                     `json:"school_year_id"`
	Email                string                     `json:"email,omitempty"`
	Verified             bool                       `json:"mailbox_verified"`
	Consented            bool                       `json:"consented"`
	ExpiresAt            time.Time                  `json:"expires_at"`
	IdleExpiresAt        time.Time                  `json:"idle_expires_at"`
	Policy               GuardianPolicyResponse     `json:"policy"`
	GradeLevels          []GuardianVocabularyOption `json:"grade_levels"`
	Homerooms            []GuardianVocabularyOption `json:"homerooms"`
	ExistingGuardian     bool                       `json:"existing_guardian,omitempty"`
	GuardianSessionToken string                     `json:"guardian_session_token,omitempty"`
}

// GuardianVocabularyOption is a display label paired with the opaque value
// submitted by an authorized guardian flow.
type GuardianVocabularyOption struct {
	ID    string `json:"id" doc:"Opaque vocabulary identifier."`
	Label string `json:"label"`
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
	return h.sessionOutput(ctx, session)
}

func (h *GuardianOnboardingHandler) BeginPath(ctx context.Context, input *GuardianBeginPathInput) (*GuardianSessionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil || strings.TrimSpace(input.RegistrationLinkID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.RegistrationInvalid, "registration link is required")
	}
	session, err := h.service.Begin(ctx, guardian.BeginInput{EntryToken: input.RegistrationLinkID, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return h.sessionOutput(ctx, session)
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
	return h.sessionOutput(ctx, session)
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
	return h.sessionOutput(ctx, session)
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
	return h.sessionOutput(ctx, session)
}

type GuardianCompletionResponse struct {
	AdultID        string    `json:"adult_id"`
	OrganizationID string    `json:"organization_id"`
	SchoolYearID   string    `json:"school_year_id"`
	SessionToken   string    `json:"session_token" doc:"Guardian access-session bearer; returned only at completion."`
	SessionID      string    `json:"session_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	IdleExpiresAt  time.Time `json:"idle_expires_at"`
	StudentIDs     []string  `json:"student_ids"`
}

type GuardianCompletionOutput struct{ Body GuardianCompletionResponse }

func (h *GuardianOnboardingHandler) Complete(ctx context.Context, input *GuardianCompleteInput) (*GuardianCompletionOutput, error) {
	if h == nil || h.service == nil {
		return nil, guardianServiceUnavailable()
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ConsentRequired, "completion fields are required")
	}
	completion, err := h.service.Complete(ctx, guardian.CompleteInput{SessionToken: input.Body.SessionToken, AdultGivenName: input.Body.AdultGivenName, AdultFamilyName: input.Body.AdultFamilyName, Now: time.Now().UTC()})
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianCompletionOutput{Body: GuardianCompletionResponse{AdultID: string(completion.AdultID), OrganizationID: string(completion.OrganizationID), SchoolYearID: string(completion.SchoolYearID), SessionToken: completion.SessionToken, SessionID: string(completion.SessionID), ExpiresAt: completion.ExpiresAt, IdleExpiresAt: completion.IdleExpiresAt, StudentIDs: stringifyIDs(completion.StudentIDs)}}, nil
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
	return h.sessionOutput(ctx, session)
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

func (h *GuardianOnboardingHandler) sessionOutput(ctx context.Context, session guardian.Session) (*GuardianSessionOutput, error) {
	if session.GuardianSessionToken != "" {
		return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session, guardian.OnboardingVocabulary{})}, nil
	}
	vocabulary, err := h.service.OnboardingVocabulary(ctx, session.Token, time.Now().UTC())
	if err != nil {
		return nil, guardianOnboardingProblem(err)
	}
	return &GuardianSessionOutput{Body: guardianOnboardingSessionResponse(session, vocabulary)}, nil
}

func guardianOnboardingSessionResponse(session guardian.Session, vocabulary guardian.OnboardingVocabulary) GuardianOnboardingSessionResponse {
	gradeLevels := make([]GuardianVocabularyOption, 0, len(vocabulary.GradeLevels))
	for _, grade := range vocabulary.GradeLevels {
		gradeLevels = append(gradeLevels, GuardianVocabularyOption{ID: string(grade.ID), Label: grade.Label})
	}
	homerooms := make([]GuardianVocabularyOption, 0, len(vocabulary.Homerooms))
	for _, homeroom := range vocabulary.Homerooms {
		homerooms = append(homerooms, GuardianVocabularyOption{ID: string(homeroom.ID), Label: homeroom.Label})
	}
	return GuardianOnboardingSessionResponse{SessionToken: session.Token, SessionID: string(session.ID), OrganizationID: string(session.OrganizationID), SchoolYearID: string(session.SchoolYearID), Email: session.Email, Verified: session.Verified, Consented: session.Consented, ExpiresAt: session.ExpiresAt, IdleExpiresAt: session.IdleExpiresAt, Policy: guardianPolicyResponse(session.Policy), GradeLevels: gradeLevels, Homerooms: homerooms, ExistingGuardian: session.ExistingGuardian, GuardianSessionToken: session.GuardianSessionToken}
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
