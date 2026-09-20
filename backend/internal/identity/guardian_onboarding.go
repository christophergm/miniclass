package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	identitydata "github.com/chrismott/miniclass/internal/data/identity"
	"github.com/chrismott/miniclass/internal/guardian"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

const (
	guardianRegistrationValidity      = 7 * 24 * time.Hour
	guardianInvitationValidity        = 7 * 24 * time.Hour
	guardianOnboardingAbsolute        = 2 * time.Hour
	guardianOnboardingIdle            = 30 * time.Minute
	guardianOnboardingOTPValidity     = 10 * time.Minute
	guardianOnboardingOTPWindow       = 10 * time.Minute
	guardianOnboardingOTPRequests     = 3
	guardianOnboardingOTPAttempts     = 5
	guardianOnboardingSessionWindow   = 10 * time.Minute
	guardianOnboardingSessionRequests = 10
	guardianInvitationImportLimit     = 500
)

func (s *Store) CreateRegistrationEntry(ctx context.Context, organizationID, schoolYearID ids.XID, actor audit.Actor, now time.Time) (guardian.RegistrationEntry, error) {
	if s == nil || s.tenantDatabase == nil {
		return guardian.RegistrationEntry{}, errors.New("create guardian registration entry: identity store is nil")
	}
	now = onboardingNow(now)
	bearer, err := GenerateAccessToken()
	if err != nil {
		return guardian.RegistrationEntry{}, err
	}
	expiresAt := now.Add(guardianRegistrationValidity)
	var result guardian.RegistrationEntry
	err = s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		if _, err := tx.RevokeGuardianRegistrationEntries(ctx, organizationID, schoolYearID, now); err != nil {
			return err
		}
		created, err := tx.CreateGuardianRegistrationEntry(ctx, bearer.Hash, expiresAt, organizationID, schoolYearID)
		if err != nil {
			return err
		}
		result = guardian.RegistrationEntry{ID: created.ID, OrganizationID: organizationID, SchoolYearID: schoolYearID, Token: bearer.Value, ExpiresAt: created.ExpiresAt, Generation: created.Generation}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianRegistrationChange, ObjectType: "guardian_registration_entry", ObjectID: &created.ID, SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"issued": true, "generation": created.Generation})})
	})
	if err != nil {
		return guardian.RegistrationEntry{}, fmt.Errorf("create guardian registration entry: %w", err)
	}
	return result, nil
}

func (s *Store) GetRegistrationEntry(ctx context.Context, organizationID, schoolYearID ids.XID) (guardian.RegistrationEntry, error) {
	if s == nil || s.database == nil {
		return guardian.RegistrationEntry{}, errors.New("get guardian registration entry: identity store is nil")
	}
	var entry identitydata.AccessToken
	err := s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		entry, err = tx.GetCurrentGuardianRegistrationEntry(ctx, organizationID, schoolYearID)
		return err
	})
	if err != nil {
		return guardian.RegistrationEntry{}, err
	}
	return guardian.RegistrationEntry{ID: entry.ID, OrganizationID: valueOrEmpty(entry.OrganizationID), SchoolYearID: valueOrEmpty(entry.SchoolYearID), ExpiresAt: entry.ExpiresAt, Generation: entry.Generation}, nil
}

func (s *Store) RevokeRegistrationEntry(ctx context.Context, organizationID, schoolYearID ids.XID, actor audit.Actor, now time.Time) error {
	if s == nil || s.tenantDatabase == nil {
		return errors.New("revoke guardian registration entry: identity store is nil")
	}
	now = onboardingNow(now)
	return s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		count, err := tx.RevokeGuardianRegistrationEntries(ctx, organizationID, schoolYearID, now)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianRegistrationChange, ObjectType: "guardian_registration_entry", SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"revoked": count > 0})})
	})
}

func (s *Store) ImportInvitationContacts(ctx context.Context, organizationID, schoolYearID ids.XID, raw []byte, actor audit.Actor, now time.Time) (guardian.InvitationImportResult, error) {
	if s == nil || s.tenantDatabase == nil {
		return guardian.InvitationImportResult{}, errors.New("import guardian invitations: identity store is nil")
	}
	now = onboardingNow(now)
	rows, err := parseGuardianInvitationCSV(raw)
	if err != nil {
		return guardian.InvitationImportResult{}, err
	}
	var existing []data.GuardianInvitationContactState
	if err := s.tenantDatabase.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		existing, err = tx.ListGuardianInvitationContacts(ctx, schoolYearID)
		return err
	}); err != nil {
		return guardian.InvitationImportResult{}, err
	}
	byEmail := make(map[string]data.GuardianInvitationContactState, len(existing))
	for _, contact := range existing {
		byEmail[strings.ToLower(contact.Email)] = contact
	}
	seen := map[string]bool{}
	result := guardian.InvitationImportResult{Rows: make([]guardian.InvitationImportRow, 0, len(rows))}
	err = s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		for index, value := range rows {
			rowNumber := index + 2
			email, normalizeErr := normalizeGuardianEmail(value)
			if normalizeErr != nil {
				result.Rows = append(result.Rows, guardian.InvitationImportRow{Row: rowNumber, Status: "error", Error: normalizeErr.Error()})
				continue
			}
			if seen[email] {
				result.Rows = append(result.Rows, guardian.InvitationImportRow{Row: rowNumber, Email: email, Status: "duplicate", Error: "the email is duplicated in this file"})
				continue
			}
			seen[email] = true
			current, exists := byEmail[email]
			if exists && current.RevokedAt == nil && current.ConsumedAt == nil && now.Before(current.ExpiresAt) {
				result.Rows = append(result.Rows, guardian.InvitationImportRow{Row: rowNumber, Email: email, Status: "duplicate", ContactID: string(current.ID)})
				continue
			}
			bearer, err := GenerateAccessToken()
			if err != nil {
				return err
			}
			createdToken, err := tx.CreateGuardianInvitationToken(ctx, bearer.Hash, now.Add(guardianInvitationValidity), organizationID, schoolYearID)
			if err != nil {
				return err
			}
			var contact data.GuardianInvitationContact
			if exists {
				if _, err := tx.RevokeGuardianInvitationToken(ctx, current.InvitationTokenID, now); err != nil {
					return err
				}
				contact, err = tx.UpdateGuardianInvitationContactToken(ctx, schoolYearID, current.ID, createdToken.ID)
			} else {
				contact, err = tx.CreateGuardianInvitationContact(ctx, schoolYearID, createdToken.ID, email)
			}
			if err != nil {
				return err
			}
			byEmail[email] = data.GuardianInvitationContactState{GuardianInvitationContact: contact, ExpiresAt: createdToken.ExpiresAt, Generation: createdToken.Generation}
			result.Rows = append(result.Rows, guardian.InvitationImportRow{Row: rowNumber, Email: email, Status: "issued", ContactID: string(contact.ID), Token: bearer.Value})
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianInvitationImport, ObjectType: "guardian_invitation_import", SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"rows": len(result.Rows), "issued": countInvitationRows(result.Rows, "issued"), "duplicates": countInvitationRows(result.Rows, "duplicate"), "errors": countInvitationRows(result.Rows, "error")})})
	})
	if err != nil {
		return guardian.InvitationImportResult{}, fmt.Errorf("import guardian invitations: %w", err)
	}
	return result, nil
}

func (s *Store) ListInvitationContacts(ctx context.Context, organizationID, schoolYearID ids.XID, actor audit.Actor) ([]guardian.InvitationExportRow, error) {
	if s == nil || s.tenantDatabase == nil {
		return nil, errors.New("list guardian invitations: identity store is nil")
	}
	now := time.Now().UTC()
	var contacts []data.GuardianInvitationContactState
	err := s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		var err error
		contacts, err = tx.ListGuardianInvitationContacts(ctx, schoolYearID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianInvitationExport, ObjectType: "guardian_invitation_export", SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"rows": len(contacts)})})
	})
	if err != nil {
		return nil, err
	}
	result := make([]guardian.InvitationExportRow, 0, len(contacts))
	for _, contact := range contacts {
		status := invitationStatus(contact, now)
		result = append(result, guardian.InvitationExportRow{Email: contact.Email, Status: status, ExpiresAt: contact.ExpiresAt, CreatedAt: contact.CreatedAt, ConsumedAt: contact.ConsumedAt})
	}
	return result, nil
}

func (s *Store) RevokeInvitationContact(ctx context.Context, organizationID, schoolYearID, contactID ids.XID, actor audit.Actor, now time.Time) error {
	if s == nil || s.tenantDatabase == nil {
		return errors.New("revoke guardian invitation: identity store is nil")
	}
	now = onboardingNow(now)
	return s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		contact, err := tx.GetGuardianInvitationContactByID(ctx, schoolYearID, contactID)
		if err != nil {
			return err
		}
		if _, err := tx.RevokeGuardianInvitationToken(ctx, contact.InvitationTokenID, now); err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianInvitationChange, ObjectType: "guardian_invitation_contact", ObjectID: &contactID, SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"revoked": true})})
	})
}

func (s *Store) RevokeOnboardingSession(ctx context.Context, organizationID, schoolYearID, sessionID ids.XID, actor audit.Actor, now time.Time) error {
	if s == nil || s.tenantDatabase == nil {
		return errors.New("revoke guardian onboarding session: identity store is nil")
	}
	now = onboardingNow(now)
	err := s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		revoked, err := tx.RevokeGuardianOnboardingSession(ctx, schoolYearID, sessionID, now)
		if err != nil {
			return err
		}
		if !revoked {
			return guardian.ErrOnboardingInvalid
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingRevoked, ObjectType: "guardian_onboarding_session", ObjectID: &sessionID, SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"revoked": true})})
	})
	if err != nil {
		return fmt.Errorf("revoke guardian onboarding session: %w", err)
	}
	return nil
}

func (s *Store) UpdateSignupNotice(ctx context.Context, organizationID ids.XID, content *string, actor audit.Actor, now time.Time) (guardian.Policy, error) {
	if s == nil || s.tenantDatabase == nil {
		return guardian.Policy{}, errors.New("update guardian signup notice: identity store is nil")
	}
	var policy guardian.Policy
	err := s.tenantDatabase.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetGuardianSignupNotice(ctx)
		if err != nil {
			return err
		}
		value := ""
		if content != nil {
			value = strings.TrimSpace(*content)
		}
		var next *string
		version := 0
		var hash []byte
		if value != "" {
			next = &value
			version = current.Version + 1
			digest := sha256.Sum256([]byte(value))
			hash = digest[:]
		}
		updated, err := tx.UpdateGuardianSignupNotice(ctx, next, version, hash)
		if err != nil {
			return err
		}
		policy = policyFromNotice(updated)
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianSignupNoticeChange, ObjectType: "organization_guardian_signup_notice", ChangeSummary: jsonObject(map[string]any{"version": version, "present": value != ""})})
	})
	if err != nil {
		return guardian.Policy{}, fmt.Errorf("update guardian signup notice: %w", err)
	}
	return policy, nil
}

func (s *Store) Begin(ctx context.Context, input guardian.BeginInput) (guardian.Session, error) {
	entry, err := s.lookupRegistrationEntry(ctx, input.EntryToken, input.Now)
	if err != nil || entry.OrganizationID == nil || entry.SchoolYearID == nil {
		return guardian.Session{}, guardian.ErrRegistrationInvalid
	}
	now := onboardingNow(input.Now)
	return s.createOnboardingSession(ctx, *entry.OrganizationID, *entry.SchoolYearID, &entry.ID, "registration-entry", "", now)
}

func (s *Store) Redeem(ctx context.Context, input guardian.RedeemInput) (guardian.Session, error) {
	token, err := s.lookupInvitation(ctx, input.InvitationToken, input.Now)
	if err != nil || token.OrganizationID == nil || token.SchoolYearID == nil {
		return guardian.Session{}, guardian.ErrInvitationInvalid
	}
	now := onboardingNow(input.Now)
	var result guardian.Session
	err = s.tenantDatabase.InTenant(ctx, string(*token.OrganizationID), audit.Actor{Type: audit.ActorTypeLink, Label: "guardian invitation"}, func(ctx context.Context, tx *data.Tx) error {
		contact, err := tx.GetGuardianInvitationContactByTokenID(ctx, *token.SchoolYearID, token.ID)
		if err != nil {
			return guardian.ErrInvitationInvalid
		}
		if _, err := tx.ConsumeGuardianInvitationToken(ctx, token.ID, now); err != nil {
			return guardian.ErrInvitationInvalid
		}
		bearer, err := GenerateAccessToken()
		if err != nil {
			return err
		}
		created, err := tx.CreateGuardianOnboardingSession(ctx, bearer.Hash, now.Add(guardianOnboardingAbsolute), token.OrganizationID, token.SchoolYearID, &token.ID, now, now.Add(guardianOnboardingIdle))
		if err != nil {
			return err
		}
		emailHash := hashEmail(contact.Email)
		verified, err := tx.VerifyGuardianOnboardingSession(ctx, created.ID, emailHash[:], now)
		if err != nil {
			return err
		}
		policy, err := tx.GetGuardianSignupNotice(ctx)
		if err != nil {
			return err
		}
		result = sessionResponse(bearer.Value, verified, policyFromNotice(policy), true, contact.Email)
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianInvitationRedeem, ObjectType: "guardian_invitation_contact", ObjectID: &contact.ID, SchoolYearID: token.SchoolYearID, ChangeSummary: jsonObject(map[string]any{"redeemed": true})})
	})
	if err != nil {
		if errors.Is(err, guardian.ErrInvitationInvalid) {
			return guardian.Session{}, guardian.ErrInvitationInvalid
		}
		return guardian.Session{}, fmt.Errorf("redeem guardian invitation: %w", err)
	}
	return result, nil
}

func (s *Store) RequestOTP(ctx context.Context, input guardian.OTPRequestInput) (guardian.OTPRequestResult, error) {
	now := onboardingNow(input.Now)
	session, err := s.lookupOnboardingSession(ctx, input.SessionToken, now, true)
	if err != nil || session.OrganizationID == nil || session.SchoolYearID == nil {
		return guardian.OTPRequestResult{}, guardian.ErrOnboardingInvalid
	}
	result := guardian.OTPRequestResult{Accepted: true}
	fake, err := GenerateAccessToken()
	if err != nil {
		return guardian.OTPRequestResult{}, err
	}
	result.ChallengeID = fake.Value
	if session.MailboxVerifiedAt != nil {
		return result, nil
	}
	email, err := normalizeGuardianEmail(input.Email)
	if err != nil {
		return guardian.OTPRequestResult{}, err
	}
	emailHash := hashEmail(email)
	code, err := randomDigits(6)
	if err != nil {
		return guardian.OTPRequestResult{}, err
	}
	challengeBearer, err := GenerateAccessToken()
	if err != nil {
		return guardian.OTPRequestResult{}, err
	}
	var challenge data.GuardianToken
	rateLimited := false
	err = s.tenantDatabase.InTenant(ctx, string(*session.OrganizationID), audit.Actor{Type: audit.ActorTypeLink, Label: "guardian onboarding"}, func(ctx context.Context, tx *data.Tx) error {
		count, err := tx.CountRecentGuardianOnboardingOTPRequests(ctx, session.OrganizationID, session.SchoolYearID, emailHash[:], now.Add(-guardianOnboardingOTPWindow))
		if err != nil {
			return err
		}
		if count >= guardianOnboardingOTPRequests {
			rateLimited = true
			tx.NoAuditRequired("guardian onboarding OTP request was rate limited")
			return nil
		}
		challenge, err = tx.CreateGuardianOnboardingOTP(ctx, challengeBearer.Hash, now.Add(guardianOnboardingOTPValidity), *session.OrganizationID, *session.SchoolYearID, session.ID, s.otpVerifier(code), emailHash[:])
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingOTPRequested, ObjectType: "guardian_onboarding_otp", ObjectID: &challenge.ID, SchoolYearID: session.SchoolYearID, ChangeSummary: jsonObject(map[string]any{"accepted": true})})
	})
	if err != nil {
		return guardian.OTPRequestResult{}, fmt.Errorf("request guardian onboarding OTP: %w", err)
	}
	if rateLimited || s.otpDelivery == nil || s.otpDelivery.DeliverOTP(ctx, email, code) != nil {
		return result, nil
	}
	result.ChallengeID = challengeBearer.Value
	return result, nil
}

func (s *Store) VerifyOTP(ctx context.Context, input guardian.OTPVerifyInput) (guardian.Session, error) {
	now := onboardingNow(input.Now)
	challengeHash, err := HashAccessToken(strings.TrimSpace(input.ChallengeID))
	if err != nil {
		return guardian.Session{}, guardian.ErrOTPInvalid
	}
	sessionHash, err := HashAccessToken(strings.TrimSpace(input.SessionToken))
	if err != nil {
		return guardian.Session{}, guardian.ErrOTPInvalid
	}
	var challenge identitydata.AccessToken
	var session identitydata.AccessToken
	err = s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		challenge, err = tx.GetGuardianOnboardingOTPByHash(ctx, challengeHash)
		if err != nil || challenge.ParentTokenID == nil || challenge.OrganizationID == nil || challenge.SchoolYearID == nil {
			return guardian.ErrOTPInvalid
		}
		session, err = tx.GetGuardianOnboardingSessionByHash(ctx, sessionHash, now)
		if err != nil || session.ID != *challenge.ParentTokenID || session.OrganizationID == nil || session.SchoolYearID == nil || *session.OrganizationID != *challenge.OrganizationID || *session.SchoolYearID != *challenge.SchoolYearID {
			return guardian.ErrOTPInvalid
		}
		return nil
	})
	if err != nil {
		return guardian.Session{}, guardian.ErrOTPInvalid
	}
	var invalid bool
	err = s.tenantDatabase.InTenant(ctx, string(*challenge.OrganizationID), audit.Actor{Type: audit.ActorTypeLink, Label: "guardian onboarding"}, func(ctx context.Context, tx *data.Tx) error {
		_, consumeErr := tx.ConsumeGuardianOnboardingOTP(ctx, challenge.ID, s.otpVerifier(strings.TrimSpace(input.Code)), now, guardianOnboardingOTPAttempts)
		if consumeErr != nil {
			incremented, err := tx.IncrementGuardianOnboardingOTPAttempts(ctx, challenge.ID, now, guardianOnboardingOTPAttempts)
			if err != nil {
				return err
			}
			invalid = true
			if incremented {
				return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingOTPFailed, ObjectType: "guardian_onboarding_otp", ObjectID: &challenge.ID, SchoolYearID: challenge.SchoolYearID, ChangeSummary: jsonObject(map[string]any{"accepted": false})})
			}
			tx.NoAuditRequired("guardian onboarding OTP was already expired or consumed")
			return nil
		}
		if _, err := tx.VerifyGuardianOnboardingSession(ctx, session.ID, challenge.RequestedEmailHash, now); err != nil {
			return err
		}
		contacts, err := tx.ListGuardianInvitationContacts(ctx, *challenge.SchoolYearID)
		if err != nil {
			return err
		}
		for _, contact := range contacts {
			contactHash := hashEmail(contact.Email)
			if !bytes.Equal(contactHash[:], challenge.RequestedEmailHash) || contact.RevokedAt != nil || contact.ConsumedAt != nil || !now.Before(contact.ExpiresAt) {
				continue
			}
			_, _ = tx.ConsumeGuardianInvitationToken(ctx, contact.InvitationTokenID, now)
			break
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingOTPVerified, ObjectType: "guardian_onboarding_session", ObjectID: &session.ID, SchoolYearID: challenge.SchoolYearID, ChangeSummary: jsonObject(map[string]any{"mode": "guardian_onboarding"})})
	})
	if err != nil || invalid {
		return guardian.Session{}, guardian.ErrOTPInvalid
	}
	return s.GetSession(ctx, input.SessionToken, now)
}

func (s *Store) AcceptConsent(ctx context.Context, input guardian.ConsentInput) (guardian.Session, error) {
	now := onboardingNow(input.Now)
	session, err := s.lookupOnboardingSession(ctx, input.SessionToken, now, true)
	if err != nil || session.OrganizationID == nil || session.SchoolYearID == nil {
		return guardian.Session{}, guardian.ErrOnboardingInvalid
	}
	if session.MailboxVerifiedAt == nil || len(session.RequestedEmailHash) != sha256.Size {
		return guardian.Session{}, guardian.ErrMailboxUnverified
	}
	email, err := normalizeGuardianEmail(input.Email)
	if err != nil {
		return guardian.Session{}, err
	}
	emailHash := hashEmail(email)
	if !bytes.Equal(emailHash[:], session.RequestedEmailHash) {
		return guardian.Session{}, guardian.ErrConsentInvalid
	}
	var policy guardian.Policy
	err = s.tenantDatabase.InTenantRead(ctx, string(*session.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		notice, err := tx.GetGuardianSignupNotice(ctx)
		if err != nil {
			return err
		}
		policy = policyFromNotice(notice)
		return nil
	})
	if err != nil {
		return guardian.Session{}, err
	}
	if input.TermsVersion != guardian.TermsVersion || input.PrivacyVersion != guardian.PrivacyVersion {
		return guardian.Session{}, guardian.ErrConsentInvalid
	}
	var noticeVersion *int
	if policy.SignupNotice != nil {
		if input.SignupNoticeVersion == nil || *input.SignupNoticeVersion != policy.SignupNotice.Version || !bytes.Equal(input.SignupNoticeHash, policy.SignupNotice.Hash) {
			return guardian.Session{}, guardian.ErrSignupNoticeInvalid
		}
		noticeVersion = input.SignupNoticeVersion
	} else if input.SignupNoticeVersion != nil || len(input.SignupNoticeHash) > 0 {
		return guardian.Session{}, guardian.ErrSignupNoticeInvalid
	}
	source := strings.TrimSpace(input.SourceSurface)
	if source == "" {
		source = "guardian_onboarding_web"
	}
	err = s.tenantDatabase.InTenant(ctx, string(*session.OrganizationID), audit.Actor{Type: audit.ActorTypeLink, Label: email}, func(ctx context.Context, tx *data.Tx) error {
		if existing, err := tx.GetGuardianOnboardingConsent(ctx, *session.SchoolYearID, session.ID); err == nil {
			if existing.TermsVersion != guardian.TermsVersion || existing.PrivacyVersion != guardian.PrivacyVersion || existing.SignupNoticeVersion == nil && policy.SignupNotice != nil || existing.SignupNoticeVersion != nil && policy.SignupNotice == nil {
				return guardian.ErrConsentRequired
			}
			if policy.SignupNotice != nil && (existing.SignupNoticeVersion == nil || *existing.SignupNoticeVersion != policy.SignupNotice.Version || !bytes.Equal(existing.SignupNoticeHash, policy.SignupNotice.Hash)) {
				return guardian.ErrConsentRequired
			}
			invitationAccepted, err := tx.LinkGuardianInvitationContactConsent(ctx, *session.SchoolYearID, email, existing.ID)
			if err != nil {
				return err
			}
			if invitationAccepted {
				return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianTermsAccepted, ObjectType: "guardian_onboarding_consent", ObjectID: &existing.ID, SchoolYearID: session.SchoolYearID, Reason: source, ChangeSummary: jsonObject(map[string]any{"invitation_accepted": true, "repaired": true})})
			}
			tx.NoAuditRequired("guardian onboarding consent was already recorded")
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		consent, err := tx.CreateGuardianOnboardingConsent(ctx, *session.SchoolYearID, session.ID, email, guardian.TermsVersion, guardian.PrivacyVersion, noticeVersion, input.SignupNoticeHash, now, source)
		if err != nil {
			return err
		}
		invitationAccepted, err := tx.LinkGuardianInvitationContactConsent(ctx, *session.SchoolYearID, email, consent.ID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianTermsAccepted, ObjectType: "guardian_onboarding_consent", ObjectID: &consent.ID, SchoolYearID: session.SchoolYearID, Reason: source, ChangeSummary: jsonObject(map[string]any{"terms_version": guardian.TermsVersion, "privacy_version": guardian.PrivacyVersion, "signup_notice_version": noticeVersion, "invitation_accepted": invitationAccepted})})
	})
	if err != nil {
		return guardian.Session{}, fmt.Errorf("accept guardian consent: %w", err)
	}
	result, err := s.GetSession(ctx, input.SessionToken, now)
	if err != nil {
		return guardian.Session{}, err
	}
	result.Email = email
	result.Consented = true
	result.Policy = policy
	return result, nil
}

func (s *Store) Complete(ctx context.Context, input guardian.CompleteInput) (guardian.Completion, error) {
	now := onboardingNow(input.Now)
	session, err := s.lookupOnboardingSession(ctx, input.SessionToken, now, true)
	if err != nil || session.OrganizationID == nil || session.SchoolYearID == nil {
		return guardian.Completion{}, guardian.ErrOnboardingInvalid
	}
	if session.MailboxVerifiedAt == nil {
		return guardian.Completion{}, guardian.ErrMailboxUnverified
	}
	var result guardian.Completion
	err = s.tenantDatabase.InTenant(ctx, string(*session.OrganizationID), audit.Actor{Type: audit.ActorTypeLink, Label: "guardian onboarding"}, func(ctx context.Context, tx *data.Tx) error {
		consent, err := tx.GetGuardianOnboardingConsent(ctx, *session.SchoolYearID, session.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return guardian.ErrConsentRequired
		}
		if err != nil {
			return err
		}
		currentNotice, err := tx.GetGuardianSignupNotice(ctx)
		if err != nil {
			return err
		}
		if consent.TermsVersion != guardian.TermsVersion || consent.PrivacyVersion != guardian.PrivacyVersion {
			return guardian.ErrConsentRequired
		}
		if currentNotice.Content != nil {
			if consent.SignupNoticeVersion == nil || *consent.SignupNoticeVersion != currentNotice.Version || !bytes.Equal(consent.SignupNoticeHash, currentNotice.ContentHash) {
				return guardian.ErrConsentRequired
			}
		} else if consent.SignupNoticeVersion != nil || len(consent.SignupNoticeHash) > 0 {
			return guardian.ErrConsentRequired
		}
		if input.GradeLevelID == "" || input.HomeroomID == "" {
			return guardian.ErrStudentAttributesRequired
		}
		if err := tx.LockGuardianOnboardingEmail(ctx, *session.SchoolYearID, consent.VerifiedEmail); err != nil {
			return err
		}
		if _, err := tx.GetGradeLevelByID(ctx, *session.SchoolYearID, input.GradeLevelID); err != nil {
			return err
		}
		if _, err := tx.GetHomeroomByID(ctx, *session.SchoolYearID, input.HomeroomID); err != nil {
			return err
		}
		adults, err := tx.FindActiveAdultsByEmail(ctx, *session.SchoolYearID, consent.VerifiedEmail)
		if err != nil {
			return err
		}
		var adult data.Adult
		switch len(adults) {
		case 0:
			intent := data.AdultParticipationUnavailable
			adult, err = tx.CreateAdult(ctx, *session.SchoolYearID, input.AdultGivenName, input.AdultFamilyName, nil, stringPtr(consent.VerifiedEmail), nil, nil, &intent)
			if err != nil {
				return err
			}
		case 1:
			adult = adults[0]
		default:
			return guardian.ErrOnboardingEmailConflict
		}
		student, err := tx.CreateStudentWithMetadata(ctx, *session.SchoolYearID, &input.GradeLevelID, input.HomeroomID, input.StudentGivenName, input.StudentFamilyName, nil, nil, false, "guardian")
		if err != nil {
			return err
		}
		relationship, err := tx.CreateGuardianRelationship(ctx, *session.SchoolYearID, adult.ID, student.ID, input.RelationshipType)
		if err != nil {
			return err
		}
		completed, err := tx.CompleteGuardianOnboardingSession(ctx, session.ID, now)
		if err != nil {
			return err
		}
		if !completed {
			return guardian.ErrOnboardingInvalid
		}
		result = guardian.Completion{AdultID: adult.ID, StudentID: student.ID, RelationshipID: relationship.ID, OrganizationID: *session.OrganizationID, SchoolYearID: *session.SchoolYearID}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingCompleted, ObjectType: "guardian_onboarding_session", ObjectID: &session.ID, SchoolYearID: session.SchoolYearID, Reason: consent.SourceSurface, ChangeSummary: jsonObject(map[string]any{"adult_id": adult.ID, "student_id": student.ID, "relationship_id": relationship.ID, "consent_id": consent.ID, "mailbox_verified": true})})
	})
	if err != nil {
		return guardian.Completion{}, fmt.Errorf("complete guardian onboarding: %w", err)
	}
	return result, nil
}

func (s *Store) GetSession(ctx context.Context, bearer string, now time.Time) (guardian.Session, error) {
	now = onboardingNow(now)
	token, err := s.lookupOnboardingSession(ctx, bearer, now, true)
	if err != nil || token.OrganizationID == nil || token.SchoolYearID == nil {
		return guardian.Session{}, guardian.ErrOnboardingInvalid
	}
	var policy guardian.Policy
	var consent data.GuardianOnboardingConsent
	consented := false
	err = s.tenantDatabase.InTenantRead(ctx, string(*token.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		notice, err := tx.GetGuardianSignupNotice(ctx)
		if err != nil {
			return err
		}
		policy = policyFromNotice(notice)
		consent, err = tx.GetGuardianOnboardingConsent(ctx, *token.SchoolYearID, token.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		consented = true
		return nil
	})
	if err != nil {
		return guardian.Session{}, err
	}
	email := ""
	if consented {
		email = consent.VerifiedEmail
	}
	return sessionResponse(bearer, guardianTokenFromIdentity(token), policy, consented, email), nil
}

func (s *Store) lookupRegistrationEntry(ctx context.Context, bearer string, now time.Time) (identitydata.AccessToken, error) {
	hash, err := HashAccessToken(strings.TrimSpace(bearer))
	if err != nil {
		return identitydata.AccessToken{}, guardian.ErrRegistrationInvalid
	}
	now = onboardingNow(now)
	var token identitydata.AccessToken
	err = s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		token, err = tx.GetGuardianRegistrationEntryByHash(ctx, hash, now)
		return err
	})
	return token, err
}

func (s *Store) lookupInvitation(ctx context.Context, bearer string, now time.Time) (identitydata.AccessToken, error) {
	hash, err := HashAccessToken(strings.TrimSpace(bearer))
	if err != nil {
		return identitydata.AccessToken{}, guardian.ErrInvitationInvalid
	}
	now = onboardingNow(now)
	var token identitydata.AccessToken
	err = s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		token, err = tx.GetGuardianInvitationTokenByHash(ctx, hash, now)
		return err
	})
	return token, err
}

func (s *Store) lookupOnboardingSession(ctx context.Context, bearer string, now time.Time, touch bool) (identitydata.AccessToken, error) {
	hash, err := HashAccessToken(strings.TrimSpace(bearer))
	if err != nil {
		return identitydata.AccessToken{}, guardian.ErrOnboardingInvalid
	}
	now = onboardingNow(now)
	var token identitydata.AccessToken
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		token, err = tx.GetGuardianOnboardingSessionByHash(ctx, hash, now)
		if err != nil {
			return err
		}
		if token.ParentTokenID != nil {
			parent, err := tx.GetAccessTokenByID(ctx, *token.ParentTokenID)
			if err != nil {
				return err
			}
			if parent.Purpose == "guardian_registration_entry" && (parent.RevokedAt != nil || parent.ConsumedAt != nil || !now.Before(parent.ExpiresAt)) {
				return guardian.ErrOnboardingInvalid
			}
		}
		if touch {
			if _, err := tx.TouchGuardianOnboardingSession(ctx, token.ID, now, now.Add(guardianOnboardingIdle)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return identitydata.AccessToken{}, err
	}
	return token, nil
}

func (s *Store) createOnboardingSession(ctx context.Context, organizationID, schoolYearID ids.XID, parentTokenID *ids.XID, source, email string, now time.Time) (guardian.Session, error) {
	bearer, err := GenerateAccessToken()
	if err != nil {
		return guardian.Session{}, err
	}
	var result guardian.Session
	rateLimited := false
	err = s.tenantDatabase.InTenant(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeLink, Label: "guardian onboarding"}, func(ctx context.Context, tx *data.Tx) error {
		if source == "registration-entry" && parentTokenID != nil {
			if err := tx.LockGuardianRegistrationEntry(ctx, *parentTokenID); err != nil {
				return err
			}
			count, err := tx.CountRecentGuardianOnboardingSessionsForParent(ctx, *parentTokenID, now.Add(-guardianOnboardingSessionWindow))
			if err != nil {
				return err
			}
			if count >= guardianOnboardingSessionRequests {
				rateLimited = true
				return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingRateLimited, ObjectType: "guardian_registration_entry", ObjectID: parentTokenID, SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"surface": "registration-entry", "limit": guardianOnboardingSessionRequests, "window_seconds": int(guardianOnboardingSessionWindow.Seconds())})})
			}
		}
		created, err := tx.CreateGuardianOnboardingSession(ctx, bearer.Hash, now.Add(guardianOnboardingAbsolute), &organizationID, &schoolYearID, parentTokenID, now, now.Add(guardianOnboardingIdle))
		if err != nil {
			return err
		}
		if email != "" {
			digest := hashEmail(email)
			if _, err := tx.VerifyGuardianOnboardingSession(ctx, created.ID, digest[:], now); err != nil {
				return err
			}
		}
		notice, err := tx.GetGuardianSignupNotice(ctx)
		if err != nil {
			return err
		}
		result = sessionResponse(bearer.Value, created, policyFromNotice(notice), false, email)
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianOnboardingStarted, ObjectType: "guardian_onboarding_session", ObjectID: &created.ID, SchoolYearID: &schoolYearID, ChangeSummary: jsonObject(map[string]any{"source": source})})
	})
	if err != nil {
		return guardian.Session{}, fmt.Errorf("create guardian onboarding session: %w", err)
	}
	if rateLimited {
		return guardian.Session{}, guardian.ErrOnboardingRateLimit
	}
	return result, nil
}

func sessionResponse(bearer string, token data.GuardianToken, policy guardian.Policy, consented bool, email string) guardian.Session {
	verified := token.MailboxVerifiedAt != nil || len(token.RequestedEmailHash) == sha256.Size
	idleExpiresAt := token.ExpiresAt
	if token.IdleExpiresAt != nil {
		idleExpiresAt = *token.IdleExpiresAt
	}
	return guardian.Session{Token: bearer, ID: token.ID, OrganizationID: token.OrganizationID, SchoolYearID: token.SchoolYearID, Email: email, Verified: verified, Consented: consented, ExpiresAt: token.ExpiresAt, IdleExpiresAt: idleExpiresAt, Policy: policy}
}

func guardianTokenFromIdentity(token identitydata.AccessToken) data.GuardianToken {
	return data.GuardianToken{ID: token.ID, Purpose: token.Purpose, ExpiresAt: token.ExpiresAt, OrganizationID: valueOrEmpty(token.OrganizationID), SchoolYearID: valueOrEmpty(token.SchoolYearID), ParentTokenID: token.ParentTokenID, RequestedEmailHash: append([]byte(nil), token.RequestedEmailHash...), MailboxVerifiedAt: token.MailboxVerifiedAt, IdleExpiresAt: token.IdleExpiresAt, Generation: token.Generation}
}

func policyFromNotice(notice data.GuardianSignupNotice) guardian.Policy {
	policy := guardian.Policy{TermsVersion: guardian.TermsVersion, TermsNotice: guardian.TermsNotice, PrivacyVersion: guardian.PrivacyVersion, PrivacyNotice: guardian.PrivacyNotice}
	if notice.Content != nil && notice.Version > 0 && len(notice.ContentHash) == sha256.Size {
		policy.SignupNotice = &guardian.SignupNotice{Content: *notice.Content, Version: notice.Version, Hash: append([]byte(nil), notice.ContentHash...)}
	}
	return policy
}

func parseGuardianInvitationCSV(raw []byte) ([]string, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("invitation CSV must contain an email header")
	}
	emailColumn := -1
	for index, column := range header {
		if strings.EqualFold(strings.TrimSpace(column), "email") {
			emailColumn = index
			break
		}
	}
	if emailColumn < 0 {
		return nil, errors.New("invitation CSV must contain an email column")
	}
	values := make([]string, 0)
	for len(values) < guardianInvitationImportLimit {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read invitation CSV: %w", err)
		}
		if emailColumn >= len(row) {
			values = append(values, "")
			continue
		}
		values = append(values, row[emailColumn])
	}
	if _, err := reader.Read(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invitation CSV exceeds %d rows", guardianInvitationImportLimit)
	}
	return values, nil
}

func normalizeGuardianEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || strings.ContainsAny(value, "\r\n") || !strings.Contains(value, "@") {
		return "", errors.New("email is invalid")
	}
	return value, nil
}

func hashEmail(email string) [sha256.Size]byte {
	return sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
}

func invitationStatus(contact data.GuardianInvitationContactState, now time.Time) string {
	switch {
	case contact.RevokedAt != nil:
		return "revoked"
	case contact.ConsumedAt != nil:
		return "redeemed"
	case !now.Before(contact.ExpiresAt):
		return "expired"
	default:
		return "pending"
	}
}

func countInvitationRows(rows []guardian.InvitationImportRow, status string) int {
	count := 0
	for _, row := range rows {
		if row.Status == status {
			count++
		}
	}
	return count
}

func onboardingNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func valueOrEmpty(value *ids.XID) ids.XID {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtr(value string) *string { return &value }

func jsonObject(value map[string]any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"error":"could not encode guardian onboarding audit summary"}`)
	}
	return encoded
}
