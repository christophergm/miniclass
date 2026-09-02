package identity

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	data "github.com/chrismott/miniclass/internal/data"
	identitydata "github.com/chrismott/miniclass/internal/data/identity"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

const (
	adultOTPValidity        = 10 * time.Minute
	adultOTPRateWindow      = 10 * time.Minute
	adultOTPRequestsWindow  = 3
	guardianSessionAbsolute = 8 * time.Hour
	guardianSessionIdle     = 30 * time.Minute
	adminSessionAbsolute    = 8 * time.Hour
	adminSessionIdle        = 30 * time.Minute
)

type otpDeliveryUnavailable struct{}

func (otpDeliveryUnavailable) DeliverOTP(context.Context, string, string) error {
	return auth.ErrOTPUnavailable
}

type otpDeliveryFunc func(context.Context, string, string) error

func (f otpDeliveryFunc) DeliverOTP(ctx context.Context, email, code string) error {
	return f(ctx, email, code)
}

// SMTPOTPDelivery is the transactional-email adapter. The raw code exists
// only for the duration of SendMail; persistence receives only a verifier
// digest.
type SMTPOTPDelivery struct {
	Address  string
	Username string
	Password string
	From     string
}

func (d SMTPOTPDelivery) DeliverOTP(ctx context.Context, email, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(d.Address) == "" || strings.TrimSpace(d.From) == "" {
		return auth.ErrOTPUnavailable
	}
	parsed, err := mail.ParseAddress(email)
	from, fromErr := mail.ParseAddress(d.From)
	if err != nil || parsed.Address != email || fromErr != nil || strings.ContainsAny(email, "\r\n") || strings.ContainsAny(d.From, "\r\n") {
		return auth.ErrOTPUnavailable
	}
	message := []byte("From: " + d.From + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: MiniClass sign-in code\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Your MiniClass sign-in code is " + code + ". It expires in 10 minutes.\r\n")
	var smtpAuth smtp.Auth
	if strings.TrimSpace(d.Username) != "" {
		host := d.Address
		if hostPart, _, splitErr := strings.Cut(d.Address, ":"); splitErr {
			host = hostPart
		}
		smtpAuth = smtp.PlainAuth("", d.Username, d.Password, host)
	}
	if err := smtp.SendMail(d.Address, smtpAuth, from.Address, []string{email}, message); err != nil {
		return fmt.Errorf("%w: SMTP delivery failed", auth.ErrOTPUnavailable)
	}
	return nil
}

// NewStoreWithOTPDelivery is convenient for API composition and tests that
// need to capture the transactional authentication email.
func NewStoreWithOTPDelivery(database *data.DB, authKey []byte, delivery func(context.Context, string, string) error) *Store {
	if delivery == nil {
		return NewStoreWithAuth(database, authKey, nil)
	}
	return NewStoreWithAuth(database, authKey, otpDeliveryFunc(delivery))
}

func (s *Store) RequestAdultOTP(ctx context.Context, input auth.OTPRequest) (auth.OTPRequestResult, error) {
	if s == nil || s.database == nil || s.tenantDatabase == nil {
		return auth.OTPRequestResult{}, errors.New("request adult OTP: identity store is nil")
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.OrganizationID == "" || input.SchoolYearID == "" || input.Email == "" {
		return auth.OTPRequestResult{}, errors.New("request adult OTP: organization, school year, and email are required")
	}
	now := input.Now.UTC()
	if input.Now.IsZero() {
		now = time.Now().UTC()
	}

	var adults []data.Adult
	err := s.tenantDatabase.InTenantRead(ctx, string(input.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		adults, err = tx.FindActiveAdultsByEmail(ctx, input.SchoolYearID, input.Email)
		return err
	})
	if err != nil {
		return auth.OTPRequestResult{}, fmt.Errorf("request adult OTP: find adult: %w", err)
	}
	result := auth.OTPRequestResult{Accepted: true}
	fakeChallenge, err := GenerateAccessToken()
	if err != nil {
		return auth.OTPRequestResult{}, err
	}
	result.ChallengeID = fakeChallenge.Value
	// Unknown, missing, or duplicate email addresses deliberately take the
	// same response path and never trigger delivery.
	if len(adults) != 1 || adults[0].Email == nil {
		return result, nil
	}

	challengeBearer, err := GenerateAccessToken()
	if err != nil {
		return auth.OTPRequestResult{}, err
	}
	code, err := randomDigits(6)
	if err != nil {
		return auth.OTPRequestResult{}, fmt.Errorf("request adult OTP: generate code: %w", err)
	}
	orgID, yearID, adultID := input.OrganizationID, input.SchoolYearID, adults[0].ID
	emailHash := sha256.Sum256([]byte(input.Email))
	verifierHash := s.otpVerifier(code)
	var challenge identitydata.AccessToken
	var rateLimited bool
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		count, err := tx.CountRecentAdultOTPRequests(ctx, &orgID, &yearID, emailHash[:], now.Add(-adultOTPRateWindow))
		if err != nil {
			return err
		}
		if count >= adultOTPRequestsWindow {
			rateLimited = true
			return nil
		}
		challenge, err = tx.CreateAdultOTP(ctx, challengeBearer.Hash, now.Add(adultOTPValidity), &orgID, &yearID, &adultID, verifierHash, emailHash[:])
		return err
	})
	if err != nil {
		return auth.OTPRequestResult{}, fmt.Errorf("request adult OTP: persist challenge: %w", err)
	}
	if rateLimited {
		return result, nil
	}
	result.ChallengeID = challengeBearer.Value
	if err := s.recordAuthAudit(ctx, string(orgID), audit.Entry{Action: audit.ActionOTPRequested, ObjectType: "adult_otp", ObjectID: &challenge.ID, SchoolYearID: &yearID, ChangeSummary: []byte(`{"accepted":true}`)}, now); err != nil {
		return auth.OTPRequestResult{}, err
	}
	delivery := s.otpDelivery
	if delivery == nil {
		delivery = otpDeliveryUnavailable{}
	}
	if err := delivery.DeliverOTP(ctx, input.Email, code); err != nil {
		// Do not turn a delivery outage into an email-existence oracle. The
		// response remains the same neutral acceptance used for unknown and
		// duplicate addresses; this persisted challenge is unusable because the
		// recipient never received its code.
		return result, nil
	}
	return result, nil
}

func (s *Store) VerifyAdultOTP(ctx context.Context, input auth.OTPVerification) (auth.GuardianSession, error) {
	if s == nil || s.database == nil || s.tenantDatabase == nil {
		return auth.GuardianSession{}, errors.New("verify adult OTP: identity store is nil")
	}
	if input.ChallengeID == "" || strings.TrimSpace(input.Code) == "" {
		return auth.GuardianSession{}, auth.ErrOTPInvalid
	}
	now := input.Now.UTC()
	if input.Now.IsZero() {
		now = time.Now().UTC()
	}
	challengeHash, err := HashAccessToken(strings.TrimSpace(input.ChallengeID))
	if err != nil {
		return auth.GuardianSession{}, auth.ErrOTPInvalid
	}
	var challenge identitydata.AccessToken
	err = s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		challenge, err = tx.GetAdultOTPByHash(ctx, challengeHash)
		return err
	})
	if err != nil || challenge.OrganizationID == nil || challenge.SchoolYearID == nil || challenge.AdultID == nil {
		return auth.GuardianSession{}, auth.ErrOTPInvalid
	}
	var scope data.GuardianScope
	err = s.tenantDatabase.InTenantRead(ctx, string(*challenge.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		scope, err = tx.ResolveGuardianScope(ctx, *challenge.SchoolYearID, *challenge.AdultID)
		return err
	})
	if err != nil || len(scope.StudentIDs) == 0 {
		return auth.GuardianSession{}, auth.ErrOTPInvalid
	}

	sessionBearer, err := GenerateAccessToken()
	if err != nil {
		return auth.GuardianSession{}, err
	}
	var session identitydata.AccessToken
	var invalid bool
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		candidate, err := tx.ConsumeAdultOTP(ctx, challenge.ID, s.otpVerifier(strings.TrimSpace(input.Code)), now, auth.GuardianOTPAttempts)
		if err != nil {
			_, incrementErr := tx.IncrementAdultOTPAttempts(ctx, challenge.ID, now, auth.GuardianOTPAttempts)
			if incrementErr != nil {
				return incrementErr
			}
			invalid = true
			return nil
		}
		session, err = tx.CreateGuardianSession(ctx, sessionBearer.Hash, now.Add(guardianSessionAbsolute), challenge.OrganizationID, challenge.SchoolYearID, challenge.AdultID, now.Add(guardianSessionIdle), now)
		_ = candidate
		return err
	})
	if err != nil || invalid {
		return auth.GuardianSession{}, auth.ErrOTPInvalid
	}
	if err := s.recordAuthAudit(ctx, string(*challenge.OrganizationID), audit.Entry{Action: audit.ActionOTPVerified, ObjectType: "guardian_session", ObjectID: &session.ID, SchoolYearID: challenge.SchoolYearID, ChangeSummary: []byte(`{"mode":"guardian"}`)}, now); err != nil {
		return auth.GuardianSession{}, err
	}
	return auth.GuardianSession{Bearer: sessionBearer.Value, SessionID: session.ID, AdultID: *challenge.AdultID, OrganizationID: *challenge.OrganizationID, SchoolYearID: *challenge.SchoolYearID, ExpiresAt: session.ExpiresAt, IdleExpiresAt: dereferenceTime(session.IdleExpiresAt), StudentIDs: scope.StudentIDs}, nil
}

func (s *Store) ResolveSession(ctx context.Context, bearer string) (auth.Principal, error) {
	if s == nil || s.database == nil || s.tenantDatabase == nil {
		return nil, auth.ErrSessionInvalid
	}
	hash, err := HashAccessToken(bearer)
	if err != nil {
		return nil, auth.ErrSessionInvalid
	}
	now := time.Now().UTC()
	var token identitydata.AccessToken
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		token, err = tx.GetActiveSessionByHash(ctx, hash, now)
		if err != nil {
			return err
		}
		_, err = tx.TouchSession(ctx, token.ID, now, now.Add(sessionIdleFor(token.Purpose)))
		return err
	})
	if err != nil {
		return nil, auth.ErrSessionInvalid
	}
	switch token.Purpose {
	case "guardian_session":
		if token.OrganizationID == nil || token.SchoolYearID == nil || token.AdultID == nil {
			return nil, auth.ErrSessionInvalid
		}
		var scope data.GuardianScope
		if err := s.tenantDatabase.InTenantRead(ctx, string(*token.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
			var err error
			scope, err = tx.ResolveGuardianScope(ctx, *token.SchoolYearID, *token.AdultID)
			return err
		}); err != nil {
			return nil, auth.ErrSessionInvalid
		}
		return auth.GuardianPrincipal{AdultID: *token.AdultID, OrganizationID: *token.OrganizationID, SchoolYearID: *token.SchoolYearID, SessionID: token.ID, Email: scope.AdultEmail(), StudentIDs: scope.StudentIDs}, nil
	case "administrative_session":
		if token.UserID == nil || token.MfaGeneration == nil {
			return nil, auth.ErrSessionInvalid
		}
		state, err := s.mfaState(ctx, *token.UserID)
		if err != nil || state.Generation != *token.MfaGeneration {
			return nil, auth.ErrSessionInvalid
		}
		account, err := s.resolveAccountByUserID(ctx, *token.UserID)
		if err != nil {
			return nil, auth.ErrSessionInvalid
		}
		return auth.AccountPrincipal{UserID: account.User.ID, Subject: account.User.ProviderSubject, Email: account.User.Email, OrganizationID: account.Membership.OrganizationID, Organization: account.Membership.OrganizationName, Role: auth.OrganizationRole(account.Membership.Role), MFAAuthenticated: true, SessionID: token.ID}, nil
	default:
		return nil, auth.ErrSessionInvalid
	}
}

func (s *Store) RevokeSession(ctx context.Context, bearer string) error {
	hash, err := HashAccessToken(bearer)
	if err != nil {
		return auth.ErrSessionInvalid
	}
	return s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		token, err := tx.GetActiveSessionByHash(ctx, hash, time.Now().UTC())
		if err != nil {
			return auth.ErrSessionInvalid
		}
		_, err = tx.RevokeSession(ctx, token.ID, time.Now().UTC())
		return err
	})
}

func (s *Store) RevokeSessionByID(ctx context.Context, sessionID ids.XID) error {
	if s == nil || s.database == nil || sessionID == "" {
		return auth.ErrSessionInvalid
	}
	now := time.Now().UTC()
	return s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		ok, err := tx.RevokeSession(ctx, sessionID, now)
		if err != nil {
			return err
		}
		if !ok {
			return auth.ErrSessionInvalid
		}
		return nil
	})
}

func (s *Store) CreateGuardianSessionForAccount(ctx context.Context, userID, organizationID, schoolYearID ids.XID, now time.Time) (auth.GuardianSession, error) {
	if s == nil || s.database == nil || s.tenantDatabase == nil {
		return auth.GuardianSession{}, auth.ErrAdultAccountLinkMissing
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var link data.AdultAccountLink
	var scope data.GuardianScope
	err := s.tenantDatabase.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		link, err = tx.GetAdultAccountLink(ctx, schoolYearID, userID)
		if err != nil {
			return err
		}
		scope, err = tx.ResolveGuardianScope(ctx, schoolYearID, link.AdultID)
		return err
	})
	if err != nil || len(scope.StudentIDs) == 0 {
		return auth.GuardianSession{}, auth.ErrAdultAccountLinkMissing
	}
	bearer, err := GenerateAccessToken()
	if err != nil {
		return auth.GuardianSession{}, err
	}
	var session identitydata.AccessToken
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		session, err = tx.CreateGuardianSession(ctx, bearer.Hash, now.Add(guardianSessionAbsolute), &organizationID, &schoolYearID, &link.AdultID, now.Add(guardianSessionIdle), now)
		return err
	})
	if err != nil {
		return auth.GuardianSession{}, fmt.Errorf("create guardian session: %w", err)
	}
	if err := s.recordAuthAudit(ctx, string(organizationID), audit.Entry{Action: audit.ActionAuthenticationModeChange, ObjectType: "guardian_session", ObjectID: &session.ID, SchoolYearID: &schoolYearID, ChangeSummary: []byte(`{"from":"administration","to":"guardian"}`)}, now, audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: "administration"}); err != nil {
		return auth.GuardianSession{}, err
	}
	return auth.GuardianSession{Bearer: bearer.Value, SessionID: session.ID, AdultID: link.AdultID, OrganizationID: organizationID, SchoolYearID: schoolYearID, ExpiresAt: session.ExpiresAt, IdleExpiresAt: dereferenceTime(session.IdleExpiresAt), StudentIDs: scope.StudentIDs}, nil
}

func (s *Store) VerifyMFAForGuardian(ctx context.Context, guardian auth.GuardianPrincipal, code, recoveryCode string, now time.Time) (auth.AdministrativeSession, error) {
	if s == nil || s.tenantDatabase == nil {
		return auth.AdministrativeSession{}, auth.ErrAdultAccountLinkMissing
	}
	var link data.AdultAccountLink
	err := s.tenantDatabase.InTenantRead(ctx, string(guardian.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		link, err = tx.GetAdultAccountLinkByAdult(ctx, guardian.SchoolYearID, guardian.AdultID)
		return err
	})
	if err != nil {
		return auth.AdministrativeSession{}, auth.ErrAdultAccountLinkMissing
	}
	return s.VerifyMFA(ctx, auth.MFAVerification{UserID: link.UserID, OrganizationID: guardian.OrganizationID, Code: code, RecoveryCode: recoveryCode, Now: now})
}

func (s *Store) EnrollMFA(ctx context.Context, userID, organizationID ids.XID, actor audit.Actor, now time.Time) (auth.MFAEnrollment, error) {
	if s == nil || s.database == nil {
		return auth.MFAEnrollment{}, errors.New("enroll MFA: identity store is nil")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	belongs, err := s.userBelongsToOrganization(ctx, userID, organizationID)
	if err != nil || !belongs {
		return auth.MFAEnrollment{}, auth.ErrMFAInvalid
	}
	rawSecret := make([]byte, 20)
	if _, err := rand.Read(rawSecret); err != nil {
		return auth.MFAEnrollment{}, fmt.Errorf("enroll MFA: generate secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
	ciphertext, err := s.encrypt(rawSecret)
	if err != nil {
		return auth.MFAEnrollment{}, err
	}
	recoveryCodes := make([]string, 8)
	recoveryHashes := make([][]byte, len(recoveryCodes))
	for i := range recoveryCodes {
		recoveryCodes[i], err = randomRecoveryCode()
		if err != nil {
			return auth.MFAEnrollment{}, err
		}
		hash := sha256.Sum256([]byte(recoveryCodes[i]))
		recoveryHashes[i] = hash[:]
	}
	var state identitydata.MFAState
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		current, err := tx.GetMFAState(ctx, userID)
		if err != nil {
			return err
		}
		if len(current.Secret) != 0 {
			return auth.ErrMFAAlreadyEnrolled
		}
		state, err = tx.SetMFASecret(ctx, userID, ciphertext, now)
		if err != nil {
			return err
		}
		if err := tx.DeleteMFARecoveryCodes(ctx, userID); err != nil {
			return err
		}
		for _, hash := range recoveryHashes {
			if err := tx.CreateMFARecoveryCode(ctx, userID, hash); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return auth.MFAEnrollment{}, fmt.Errorf("enroll MFA: %w", err)
	}
	if err := s.recordAuthAudit(ctx, string(organizationID), audit.Entry{Action: audit.ActionMFAEnrolled, ObjectType: "user", ObjectID: &userID, ChangeSummary: []byte(`{"enrolled":true}`)}, now, actor); err != nil {
		return auth.MFAEnrollment{}, err
	}
	return auth.MFAEnrollment{Secret: secret, RecoveryCodes: recoveryCodes, Generation: state.Generation}, nil
}

func (s *Store) VerifyMFA(ctx context.Context, input auth.MFAVerification) (auth.AdministrativeSession, error) {
	if s == nil || s.database == nil {
		return auth.AdministrativeSession{}, errors.New("verify MFA: identity store is nil")
	}
	now := input.Now.UTC()
	if input.Now.IsZero() {
		now = time.Now().UTC()
	}
	belongs, err := s.userBelongsToOrganization(ctx, input.UserID, input.OrganizationID)
	if err != nil || !belongs {
		return auth.AdministrativeSession{}, auth.ErrMFAInvalid
	}
	state, err := s.mfaState(ctx, input.UserID)
	if err != nil {
		return auth.AdministrativeSession{}, auth.ErrMFAInvalid
	}
	if len(state.Secret) == 0 {
		return auth.AdministrativeSession{}, auth.ErrMFANotEnrolled
	}
	secret, err := s.decrypt(state.Secret)
	if err != nil {
		return auth.AdministrativeSession{}, auth.ErrMFAInvalid
	}
	valid := false
	usedRecovery := strings.TrimSpace(input.RecoveryCode) != ""
	if usedRecovery {
		if strings.TrimSpace(input.Code) != "" {
			return auth.AdministrativeSession{}, auth.ErrMFAInvalid
		}
		hash := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(input.RecoveryCode))))
		err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
			ok, err := tx.ConsumeMFARecoveryCode(ctx, input.UserID, hash[:], now)
			valid = ok
			return err
		})
		valid = err == nil && valid
	} else {
		valid = verifyTOTP(secret, strings.TrimSpace(input.Code), now)
	}
	if !valid {
		return auth.AdministrativeSession{}, auth.ErrMFAInvalid
	}
	bearer, err := GenerateAccessToken()
	if err != nil {
		return auth.AdministrativeSession{}, err
	}
	var session identitydata.AccessToken
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		session, err = tx.CreateAdministrativeSession(ctx, bearer.Hash, &input.UserID, now.Add(adminSessionAbsolute), now.Add(adminSessionIdle), now, state.Generation)
		return err
	})
	if err != nil {
		return auth.AdministrativeSession{}, fmt.Errorf("verify MFA: create session: %w", err)
	}
	changeSummary := []byte(`{"recovery_code":false}`)
	if usedRecovery {
		changeSummary = []byte(`{"recovery_code":true}`)
	}
	if err := s.recordAuthAudit(ctx, string(input.OrganizationID), audit.Entry{Action: audit.ActionMFAVerified, ObjectType: "administrative_session", ObjectID: &session.ID, ChangeSummary: changeSummary}, now, audit.Actor{Type: audit.ActorTypeUser, UserID: &input.UserID, Label: "MFA"}); err != nil {
		return auth.AdministrativeSession{}, err
	}
	return auth.AdministrativeSession{Bearer: bearer.Value, SessionID: session.ID, UserID: input.UserID, ExpiresAt: session.ExpiresAt, IdleExpiresAt: dereferenceTime(session.IdleExpiresAt)}, nil
}

func (s *Store) ResetMFA(ctx context.Context, input auth.MFAReset) error {
	if strings.TrimSpace(input.Reason) == "" {
		return auth.ErrMFAResetReasonRequired
	}
	now := input.Now.UTC()
	if input.Now.IsZero() {
		now = time.Now().UTC()
	}
	belongs, err := s.userBelongsToOrganization(ctx, input.TargetUserID, input.OrganizationID)
	if err != nil || !belongs {
		return auth.ErrAdultAccountLinkMissing
	}
	err = s.databaseIdentity().InTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		if _, err := tx.ResetMFASecret(ctx, input.TargetUserID); err != nil {
			return err
		}
		if err := tx.DeleteMFARecoveryCodes(ctx, input.TargetUserID); err != nil {
			return err
		}
		_, err := tx.RevokeAdministrativeSessions(ctx, &input.TargetUserID, now)
		return err
	})
	if err != nil {
		return fmt.Errorf("reset MFA: %w", err)
	}
	return s.recordAuthAudit(ctx, string(input.OrganizationID), audit.Entry{Action: audit.ActionMFAReset, ObjectType: "user", ObjectID: &input.TargetUserID, Reason: input.Reason, ChangeSummary: []byte(`{"sessions_revoked":true}`)}, now, input.Actor)
}

func (s *Store) CreateAdultAccountLink(ctx context.Context, input auth.AdultAccountLinkInput) (auth.AdultAccountLink, error) {
	if s == nil || s.tenantDatabase == nil {
		return auth.AdultAccountLink{}, errors.New("create adult account link: identity store is nil")
	}
	belongs, err := s.userBelongsToOrganization(ctx, input.UserID, input.OrganizationID)
	if err != nil || !belongs {
		return auth.AdultAccountLink{}, auth.ErrAdultAccountLinkMissing
	}
	var link data.AdultAccountLink
	err = s.tenantDatabase.InTenant(ctx, string(input.OrganizationID), input.Actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetAdultByID(ctx, input.SchoolYearID, input.AdultID); err != nil {
			return err
		}
		var err error
		link, err = tx.CreateAdultAccountLink(ctx, input.SchoolYearID, input.AdultID, input.UserID)
		if err != nil {
			return err
		}
		id, year := link.ID, link.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionAdultAccountLink, ObjectType: "adult_account_link", ObjectID: &id, SchoolYearID: &year, ChangeSummary: []byte(`{"linked":true}`)})
	})
	if err != nil {
		return auth.AdultAccountLink{}, err
	}
	return auth.AdultAccountLink{ID: link.ID, OrganizationID: link.OrganizationID, SchoolYearID: link.SchoolYearID, AdultID: link.AdultID, UserID: link.UserID}, nil
}

func (s *Store) DeleteAdultAccountLink(ctx context.Context, input auth.AdultAccountLinkInput, linkID ids.XID) error {
	return s.tenantDatabase.InTenant(ctx, string(input.OrganizationID), input.Actor, func(ctx context.Context, tx *data.Tx) error {
		deleted, err := tx.DeleteAdultAccountLink(ctx, input.SchoolYearID, linkID)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		year := input.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionAdultAccountLink, ObjectType: "adult_account_link", ObjectID: &linkID, SchoolYearID: &year, ChangeSummary: []byte(`{"linked":false}`)})
	})
}

func (s *Store) ListAdultAccountLinks(ctx context.Context, organizationID, schoolYearID ids.XID) ([]auth.AdultAccountLink, error) {
	if s == nil || s.tenantDatabase == nil {
		return nil, auth.ErrAdultAccountLinkMissing
	}
	var links []data.AdultAccountLink
	err := s.tenantDatabase.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		links, err = tx.ListAdultAccountLinks(ctx, schoolYearID)
		return err
	})
	if err != nil {
		return nil, err
	}
	result := make([]auth.AdultAccountLink, 0, len(links))
	for _, link := range links {
		result = append(result, auth.AdultAccountLink{ID: link.ID, OrganizationID: link.OrganizationID, SchoolYearID: link.SchoolYearID, AdultID: link.AdultID, UserID: link.UserID})
	}
	return result, nil
}

func (s *Store) databaseIdentity() *identitydata.DB {
	return s.database
}

func (s *Store) userBelongsToOrganization(ctx context.Context, userID, organizationID ids.XID) (bool, error) {
	var belongs bool
	err := s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		belongs, err = tx.UserBelongsToOrganization(ctx, userID, organizationID)
		return err
	})
	return belongs, err
}

func (s *Store) mfaState(ctx context.Context, userID ids.XID) (identitydata.MFAState, error) {
	var state identitydata.MFAState
	err := s.databaseIdentity().InReadTx(ctx, func(ctx context.Context, tx *identitydata.Tx) error {
		var err error
		state, err = tx.GetMFAState(ctx, userID)
		return err
	})
	return state, err
}

func (s *Store) resolveAccountByUserID(ctx context.Context, userID ids.XID) (identitydata.Account, error) {
	return s.databaseIdentity().ResolveAccountByUserID(ctx, userID)
}

func (s *Store) otpVerifier(code string) []byte {
	mac := hmac.New(sha256.New, s.authKey)
	_, _ = mac.Write([]byte(strings.TrimSpace(code)))
	return mac.Sum(nil)
}

func (s *Store) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.authKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt MFA secret: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encrypt MFA secret: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("encrypt MFA secret: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *Store) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.authKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("decrypt MFA secret: ciphertext is invalid")
	}
	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

func randomDigits(length int) (string, error) {
	if length < 1 {
		return "", errors.New("OTP length must be positive")
	}
	result := make([]byte, length)
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i, value := range raw {
		result[i] = '0' + value%10
	}
	return string(result), nil
}

func randomRecoveryCode() (string, error) {
	raw := make([]byte, 10)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func verifyTOTP(secret []byte, code string, now time.Time) bool {
	if len(secret) == 0 || len(code) != 6 {
		return false
	}
	for offset := int64(-1); offset <= 1; offset++ {
		counter := uint64(now.Unix()/30 + offset)
		var message [8]byte
		for i := range message {
			message[7-i] = byte(counter >> (8 * i))
		}
		mac := hmac.New(sha1.New, secret)
		_, _ = mac.Write(message[:])
		digest := mac.Sum(nil)
		index := digest[len(digest)-1] & 0x0f
		value := (uint32(digest[index])&0x7f)<<24 | uint32(digest[index+1])<<16 | uint32(digest[index+2])<<8 | uint32(digest[index+3])
		candidate := fmt.Sprintf("%06d", value%1000000)
		if hmac.Equal([]byte(candidate), []byte(code)) {
			return true
		}
	}
	return false
}

func sessionIdleFor(purpose string) time.Duration {
	if purpose == "administrative_session" {
		return adminSessionIdle
	}
	return guardianSessionIdle
}

func dereferenceTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func (s *Store) recordAuthAudit(ctx context.Context, organizationID string, entry audit.Entry, now time.Time, actors ...audit.Actor) error {
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "adult authentication"}
	if len(actors) > 0 {
		actor = actors[0]
	}
	if err := s.tenantDatabase.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		return tx.Record(ctx, entry)
	}); err != nil {
		return fmt.Errorf("record authentication audit at %s: %w", now.Format(time.RFC3339), err)
	}
	return nil
}
