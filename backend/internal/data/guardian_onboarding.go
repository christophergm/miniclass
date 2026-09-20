package data

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
)

// GuardianToken is the tenant-scoped part of an onboarding bearer. The raw
// token is deliberately never represented here; only its verifier lives in
// the identity layer.
type GuardianToken struct {
	ID                 ids.XID
	Purpose            string
	ExpiresAt          time.Time
	Generation         int
	OrganizationID     ids.XID
	SchoolYearID       ids.XID
	ParentTokenID      *ids.XID
	RequestedEmailHash []byte
	MailboxVerifiedAt  *time.Time
	IdleExpiresAt      *time.Time
}

type GuardianInvitationContact struct {
	ID                ids.XID
	OrganizationID    ids.XID
	SchoolYearID      ids.XID
	InvitationTokenID ids.XID
	Email             string
	AcceptedConsentID *ids.XID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type GuardianInvitationContactState struct {
	GuardianInvitationContact
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ConsumedAt *time.Time
	Generation int
}

type GuardianOnboardingConsent struct {
	ID                  ids.XID
	OrganizationID      ids.XID
	SchoolYearID        ids.XID
	SessionTokenID      ids.XID
	VerifiedEmail       string
	TermsVersion        string
	PrivacyVersion      string
	SignupNoticeVersion *int
	SignupNoticeHash    []byte
	AcceptedAt          time.Time
	SourceSurface       string
}

type GuardianSignupNotice struct {
	OrganizationID ids.XID
	Content        *string
	Version        int
	ContentHash    []byte
}

func (tx *Tx) CreateGuardianRegistrationEntry(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID ids.XID) (GuardianToken, error) {
	row, err := tx.queries.CreateGuardianRegistrationEntry(ctx, db.CreateGuardianRegistrationEntryParams{TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: &organizationID, SchoolYearID: &schoolYearID})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) RevokeGuardianRegistrationEntries(ctx context.Context, organizationID, schoolYearID ids.XID, at time.Time) (int64, error) {
	return tx.queries.RevokeGuardianRegistrationEntries(ctx, db.RevokeGuardianRegistrationEntriesParams{OrganizationID: &organizationID, SchoolYearID: &schoolYearID, RevokedAt: timestamp(at)})
}

func (tx *Tx) CreateGuardianInvitationToken(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID ids.XID) (GuardianToken, error) {
	row, err := tx.queries.CreateGuardianInvitationToken(ctx, db.CreateGuardianInvitationTokenParams{TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: &organizationID, SchoolYearID: &schoolYearID})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) RevokeGuardianInvitationToken(ctx context.Context, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.RevokeGuardianInvitationToken(ctx, db.RevokeGuardianInvitationTokenParams{ID: id, RevokedAt: timestamp(at)})
	return rows == 1, err
}

func (tx *Tx) ConsumeGuardianInvitationToken(ctx context.Context, id ids.XID, at time.Time) (GuardianToken, error) {
	row, err := tx.queries.ConsumeGuardianInvitationToken(ctx, db.ConsumeGuardianInvitationTokenParams{ID: id, ConsumedAt: timestamp(at)})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) LockGuardianOnboardingEmail(ctx context.Context, schoolYearID ids.XID, email string) error {
	return tx.queries.LockGuardianOnboardingEmail(ctx, db.LockGuardianOnboardingEmailParams{OrganizationID: string(tx.organizationID), SchoolYearID: string(schoolYearID), Email: strings.ToLower(strings.TrimSpace(email))})
}

func (tx *Tx) LockGuardianRegistrationEntry(ctx context.Context, id ids.XID) error {
	_, err := tx.queries.LockGuardianRegistrationEntry(ctx, id)
	return err
}

func (tx *Tx) CountRecentGuardianOnboardingSessionsForParent(ctx context.Context, parentTokenID ids.XID, since time.Time) (int64, error) {
	return tx.queries.CountRecentGuardianOnboardingSessionsForParent(ctx, db.CountRecentGuardianOnboardingSessionsForParentParams{ParentTokenID: &parentTokenID, CreatedAt: timestamp(since)})
}

func (tx *Tx) CreateGuardianOnboardingSession(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID, parentTokenID *ids.XID, now, idleExpiresAt time.Time) (GuardianToken, error) {
	row, err := tx.queries.CreateGuardianOnboardingSession(ctx, db.CreateGuardianOnboardingSessionParams{TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: organizationID, SchoolYearID: schoolYearID, ParentTokenID: parentTokenID, LastSeenAt: timestamp(now), IdleExpiresAt: timestamp(idleExpiresAt)})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) TouchGuardianOnboardingSession(ctx context.Context, id ids.XID, now, idleExpiresAt time.Time) (bool, error) {
	rows, err := tx.queries.TouchGuardianOnboardingSession(ctx, db.TouchGuardianOnboardingSessionParams{ID: id, LastSeenAt: timestamp(now), IdleExpiresAt: timestamp(idleExpiresAt)})
	return rows == 1, err
}

func (tx *Tx) VerifyGuardianOnboardingSession(ctx context.Context, id ids.XID, emailHash []byte, verifiedAt time.Time) (GuardianToken, error) {
	row, err := tx.queries.VerifyGuardianOnboardingSession(ctx, db.VerifyGuardianOnboardingSessionParams{ID: id, RequestedEmailHash: emailHash, MailboxVerifiedAt: timestamp(verifiedAt)})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) CompleteGuardianOnboardingSession(ctx context.Context, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.CompleteGuardianOnboardingSession(ctx, db.CompleteGuardianOnboardingSessionParams{ID: id, ConsumedAt: timestamp(at)})
	return rows == 1, err
}

func (tx *Tx) RevokeGuardianOnboardingSession(ctx context.Context, schoolYearID, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.RevokeGuardianOnboardingSession(ctx, db.RevokeGuardianOnboardingSessionParams{ID: id, RevokedAt: timestamp(at), OrganizationID: &tx.organizationID, SchoolYearID: &schoolYearID})
	return rows == 1, err
}

func (tx *Tx) CreateGuardianOnboardingOTP(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID, sessionID ids.XID, verifierHash, emailHash []byte) (GuardianToken, error) {
	row, err := tx.queries.CreateGuardianOnboardingOTP(ctx, db.CreateGuardianOnboardingOTPParams{TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: &organizationID, SchoolYearID: &schoolYearID, ParentTokenID: &sessionID, VerifierHash: verifierHash, RequestedEmailHash: emailHash})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) CountRecentGuardianOnboardingOTPRequests(ctx context.Context, organizationID, schoolYearID *ids.XID, emailHash []byte, since time.Time) (int64, error) {
	return tx.queries.CountRecentGuardianOnboardingOTPRequests(ctx, db.CountRecentGuardianOnboardingOTPRequestsParams{OrganizationID: organizationID, SchoolYearID: schoolYearID, RequestedEmailHash: emailHash, CreatedAt: timestamp(since)})
}

func (tx *Tx) ConsumeGuardianOnboardingOTP(ctx context.Context, id ids.XID, verifierHash []byte, now time.Time, maxAttempts int) (GuardianToken, error) {
	row, err := tx.queries.ConsumeGuardianOnboardingOTP(ctx, db.ConsumeGuardianOnboardingOTPParams{ID: id, VerifierHash: verifierHash, ConsumedAt: timestamp(now), Attempts: int32(maxAttempts)})
	if err != nil {
		return GuardianToken{}, err
	}
	return guardianToken(row), nil
}

func (tx *Tx) IncrementGuardianOnboardingOTPAttempts(ctx context.Context, id ids.XID, now time.Time, maxAttempts int) (bool, error) {
	rows, err := tx.queries.IncrementGuardianOnboardingOTPAttempts(ctx, db.IncrementGuardianOnboardingOTPAttemptsParams{ID: id, ExpiresAt: timestamp(now), Attempts: int32(maxAttempts)})
	return rows == 1, err
}

func (tx *Tx) CreateGuardianInvitationContact(ctx context.Context, schoolYearID, invitationTokenID ids.XID, email string) (GuardianInvitationContact, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return GuardianInvitationContact{}, errors.New("create guardian invitation contact: email is required")
	}
	row, err := tx.queries.CreateGuardianInvitationContact(ctx, db.CreateGuardianInvitationContactParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, InvitationTokenID: invitationTokenID, Email: email})
	if err != nil {
		return GuardianInvitationContact{}, err
	}
	return guardianInvitationContact(row), nil
}

func (tx *Tx) GetGuardianInvitationContactByEmail(ctx context.Context, schoolYearID ids.XID, email string) (GuardianInvitationContact, error) {
	row, err := tx.queries.GetGuardianInvitationContactByEmail(ctx, db.GetGuardianInvitationContactByEmailParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Lower: strings.ToLower(strings.TrimSpace(email))})
	if err != nil {
		return GuardianInvitationContact{}, err
	}
	return guardianInvitationContact(row), nil
}

func (tx *Tx) GetGuardianInvitationContactByID(ctx context.Context, schoolYearID, id ids.XID) (GuardianInvitationContact, error) {
	row, err := tx.queries.GetGuardianInvitationContactByID(ctx, db.GetGuardianInvitationContactByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GuardianInvitationContact{}, err
	}
	return guardianInvitationContact(row), nil
}

func (tx *Tx) GetGuardianInvitationContactByTokenID(ctx context.Context, schoolYearID, tokenID ids.XID) (GuardianInvitationContact, error) {
	row, err := tx.queries.GetGuardianInvitationContactByTokenID(ctx, db.GetGuardianInvitationContactByTokenIDParams{InvitationTokenID: tokenID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GuardianInvitationContact{}, err
	}
	return guardianInvitationContact(row), nil
}

func (tx *Tx) UpdateGuardianInvitationContactToken(ctx context.Context, schoolYearID, id, invitationTokenID ids.XID) (GuardianInvitationContact, error) {
	row, err := tx.queries.UpdateGuardianInvitationContactToken(ctx, db.UpdateGuardianInvitationContactTokenParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, InvitationTokenID: invitationTokenID})
	if err != nil {
		return GuardianInvitationContact{}, err
	}
	return guardianInvitationContact(row), nil
}

func (tx *Tx) LinkGuardianInvitationContactConsent(ctx context.Context, schoolYearID ids.XID, email string, consentID ids.XID) (bool, error) {
	rows, err := tx.queries.LinkGuardianInvitationContactConsent(ctx, db.LinkGuardianInvitationContactConsentParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Lower: strings.ToLower(strings.TrimSpace(email)), AcceptedConsentID: &consentID})
	return rows == 1, err
}

func (tx *Tx) ListGuardianInvitationContacts(ctx context.Context, schoolYearID ids.XID) ([]GuardianInvitationContactState, error) {
	rows, err := tx.queries.ListGuardianInvitationContacts(ctx, db.ListGuardianInvitationContactsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return nil, err
	}
	result := make([]GuardianInvitationContactState, 0, len(rows))
	for _, row := range rows {
		contact, err := guardianInvitationContactState(row)
		if err != nil {
			return nil, err
		}
		result = append(result, contact)
	}
	return result, nil
}

func (tx *Tx) TouchGuardianInvitationContactForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.TouchGuardianInvitationContactForRegistry(ctx, db.TouchGuardianInvitationContactForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) DeleteGuardianInvitationContact(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteGuardianInvitationContact(ctx, db.DeleteGuardianInvitationContactParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	return rows == 1, err
}

func (tx *Tx) ListAllGuardianInvitationContactsForRegistry(ctx context.Context) ([]GuardianInvitationContact, error) {
	rows, err := tx.queries.ListAllGuardianInvitationContactsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]GuardianInvitationContact, 0, len(rows))
	for _, row := range rows {
		result = append(result, guardianInvitationContact(row))
	}
	return result, nil
}

func (tx *Tx) FindGuardianInvitationContactForRegistry(ctx context.Context, id ids.XID) (GuardianInvitationContact, error) {
	row, err := tx.queries.FindGuardianInvitationContactForRegistry(ctx, db.FindGuardianInvitationContactForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return GuardianInvitationContact{}, nil
	}
	if err != nil {
		return GuardianInvitationContact{}, err
	}
	return guardianInvitationContact(row), nil
}

func (tx *Tx) CreateGuardianOnboardingConsent(ctx context.Context, schoolYearID, sessionTokenID ids.XID, verifiedEmail, termsVersion, privacyVersion string, signupNoticeVersion *int, signupNoticeHash []byte, acceptedAt time.Time, sourceSurface string) (GuardianOnboardingConsent, error) {
	if strings.TrimSpace(verifiedEmail) == "" || strings.TrimSpace(termsVersion) == "" || strings.TrimSpace(privacyVersion) == "" || strings.TrimSpace(sourceSurface) == "" {
		return GuardianOnboardingConsent{}, errors.New("create guardian onboarding consent: required provenance is missing")
	}
	var noticeVersion pgtype.Int4
	if signupNoticeVersion != nil {
		noticeVersion = pgtype.Int4{Int32: int32(*signupNoticeVersion), Valid: true}
	}
	row, err := tx.queries.CreateGuardianOnboardingConsent(ctx, db.CreateGuardianOnboardingConsentParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, SessionTokenID: sessionTokenID, VerifiedEmail: strings.ToLower(strings.TrimSpace(verifiedEmail)), TermsVersion: strings.TrimSpace(termsVersion), PrivacyVersion: strings.TrimSpace(privacyVersion), SignupNoticeVersion: noticeVersion, SignupNoticeHash: signupNoticeHash, AcceptedAt: timestamp(acceptedAt), SourceSurface: strings.TrimSpace(sourceSurface)})
	if err != nil {
		return GuardianOnboardingConsent{}, err
	}
	return guardianOnboardingConsent(row), nil
}

func (tx *Tx) GetGuardianOnboardingConsent(ctx context.Context, schoolYearID, sessionTokenID ids.XID) (GuardianOnboardingConsent, error) {
	row, err := tx.queries.GetGuardianOnboardingConsent(ctx, db.GetGuardianOnboardingConsentParams{SessionTokenID: sessionTokenID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GuardianOnboardingConsent{}, err
	}
	return guardianOnboardingConsent(row), nil
}

func (tx *Tx) ListAllGuardianOnboardingConsentsForRegistry(ctx context.Context) ([]GuardianOnboardingConsent, error) {
	rows, err := tx.queries.ListAllGuardianOnboardingConsentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]GuardianOnboardingConsent, 0, len(rows))
	for _, row := range rows {
		result = append(result, guardianOnboardingConsent(row))
	}
	return result, nil
}

func (tx *Tx) FindGuardianOnboardingConsentForRegistry(ctx context.Context, id ids.XID) (GuardianOnboardingConsent, error) {
	row, err := tx.queries.FindGuardianOnboardingConsentForRegistry(ctx, db.FindGuardianOnboardingConsentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return GuardianOnboardingConsent{}, nil
	}
	if err != nil {
		return GuardianOnboardingConsent{}, err
	}
	return guardianOnboardingConsent(row), nil
}

func (tx *Tx) TouchGuardianOnboardingConsentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.TouchGuardianOnboardingConsentForRegistry(ctx, db.TouchGuardianOnboardingConsentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) DeleteGuardianOnboardingConsentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteGuardianOnboardingConsentForRegistry(ctx, db.DeleteGuardianOnboardingConsentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) GetGuardianSignupNotice(ctx context.Context) (GuardianSignupNotice, error) {
	row, err := tx.queries.GetGuardianSignupNotice(ctx, tx.organizationID)
	if err != nil {
		return GuardianSignupNotice{}, err
	}
	var content *string
	if row.GuardianSignupNotice.Valid {
		value := row.GuardianSignupNotice.String
		content = &value
	}
	return GuardianSignupNotice{OrganizationID: row.ID, Content: content, Version: int(row.GuardianSignupNoticeVersion), ContentHash: append([]byte(nil), row.GuardianSignupNoticeHash...)}, nil
}

func (tx *Tx) UpdateGuardianSignupNotice(ctx context.Context, content *string, version int, contentHash []byte) (GuardianSignupNotice, error) {
	var notice pgtype.Text
	if content != nil && strings.TrimSpace(*content) != "" {
		notice = pgtype.Text{String: strings.TrimSpace(*content), Valid: true}
	}
	row, err := tx.queries.UpdateGuardianSignupNotice(ctx, db.UpdateGuardianSignupNoticeParams{ID: tx.organizationID, GuardianSignupNotice: notice, GuardianSignupNoticeVersion: int32(version), GuardianSignupNoticeHash: contentHash})
	if err != nil {
		return GuardianSignupNotice{}, err
	}
	var value *string
	if row.GuardianSignupNotice.Valid {
		contentValue := row.GuardianSignupNotice.String
		value = &contentValue
	}
	return GuardianSignupNotice{OrganizationID: row.ID, Content: value, Version: int(row.GuardianSignupNoticeVersion), ContentHash: append([]byte(nil), row.GuardianSignupNoticeHash...)}, nil
}

func guardianToken(row db.AccessToken) GuardianToken {
	return GuardianToken{ID: row.ID, Purpose: string(row.Purpose), ExpiresAt: row.ExpiresAt.Time, Generation: int(row.Generation), OrganizationID: valueID(row.OrganizationID), SchoolYearID: valueID(row.SchoolYearID), ParentTokenID: row.ParentTokenID, RequestedEmailHash: append([]byte(nil), row.RequestedEmailHash...), MailboxVerifiedAt: nullableTime(row.MailboxVerifiedAt), IdleExpiresAt: nullableTime(row.IdleExpiresAt)}
}

func guardianInvitationContact(row db.GuardianInvitationContact) GuardianInvitationContact {
	return GuardianInvitationContact{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, InvitationTokenID: row.InvitationTokenID, Email: row.Email, AcceptedConsentID: row.AcceptedConsentID, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}

func guardianInvitationContactState(row db.ListGuardianInvitationContactsRow) (GuardianInvitationContactState, error) {
	if !row.ExpiresAt.Valid || !row.CreatedAt.Valid || !row.UpdatedAt.Valid {
		return GuardianInvitationContactState{}, fmt.Errorf("guardian invitation contact: required timestamp is null")
	}
	return GuardianInvitationContactState{GuardianInvitationContact: GuardianInvitationContact{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, InvitationTokenID: row.InvitationTokenID, Email: row.Email, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}, ExpiresAt: row.ExpiresAt.Time, RevokedAt: nullableTime(row.RevokedAt), ConsumedAt: nullableTime(row.ConsumedAt), Generation: int(row.Generation)}, nil
}

func guardianOnboardingConsent(row db.GuardianOnboardingConsent) GuardianOnboardingConsent {
	var noticeVersion *int
	if row.SignupNoticeVersion.Valid {
		value := int(row.SignupNoticeVersion.Int32)
		noticeVersion = &value
	}
	return GuardianOnboardingConsent{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, SessionTokenID: row.SessionTokenID, VerifiedEmail: row.VerifiedEmail, TermsVersion: row.TermsVersion, PrivacyVersion: row.PrivacyVersion, SignupNoticeVersion: noticeVersion, SignupNoticeHash: append([]byte(nil), row.SignupNoticeHash...), AcceptedAt: row.AcceptedAt.Time, SourceSurface: row.SourceSurface}
}

func valueID(value *ids.XID) ids.XID {
	if value == nil {
		return ""
	}
	return *value
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
