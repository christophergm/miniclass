package identity

import (
	"context"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgtype"
)

func (tx *Tx) CreateGuardianRegistrationEntry(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID ids.XID) (AccessToken, error) {
	row, err := tx.queries.CreateGuardianRegistrationEntry(ctx, db.CreateGuardianRegistrationEntryParams{
		TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: &organizationID, SchoolYearID: &schoolYearID,
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetGuardianRegistrationEntryByHash(ctx context.Context, tokenHash []byte, now time.Time) (AccessToken, error) {
	row, err := tx.queries.GetGuardianRegistrationEntryByHash(ctx, db.GetGuardianRegistrationEntryByHashParams{TokenHash: tokenHash, ExpiresAt: timestamp(now)})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetCurrentGuardianRegistrationEntry(ctx context.Context, organizationID, schoolYearID ids.XID) (AccessToken, error) {
	row, err := tx.queries.GetCurrentGuardianRegistrationEntry(ctx, db.GetCurrentGuardianRegistrationEntryParams{OrganizationID: &organizationID, SchoolYearID: &schoolYearID})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) RevokeGuardianRegistrationEntries(ctx context.Context, organizationID, schoolYearID ids.XID, at time.Time) (int64, error) {
	return tx.queries.RevokeGuardianRegistrationEntries(ctx, db.RevokeGuardianRegistrationEntriesParams{OrganizationID: &organizationID, SchoolYearID: &schoolYearID, RevokedAt: timestamp(at)})
}

func (tx *Tx) CreateGuardianInvitationToken(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID ids.XID) (AccessToken, error) {
	row, err := tx.queries.CreateGuardianInvitationToken(ctx, db.CreateGuardianInvitationTokenParams{
		TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: &organizationID, SchoolYearID: &schoolYearID,
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetGuardianInvitationTokenByHash(ctx context.Context, tokenHash []byte, now time.Time) (AccessToken, error) {
	row, err := tx.queries.GetGuardianInvitationTokenByHash(ctx, db.GetGuardianInvitationTokenByHashParams{TokenHash: tokenHash, ExpiresAt: timestamp(now)})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) RevokeGuardianInvitationToken(ctx context.Context, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.RevokeGuardianInvitationToken(ctx, db.RevokeGuardianInvitationTokenParams{ID: id, RevokedAt: timestamp(at)})
	return rows == 1, err
}

func (tx *Tx) ConsumeGuardianInvitationToken(ctx context.Context, id ids.XID, at time.Time) (AccessToken, error) {
	row, err := tx.queries.ConsumeGuardianInvitationToken(ctx, db.ConsumeGuardianInvitationTokenParams{ID: id, ConsumedAt: timestamp(at)})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) CreateGuardianOnboardingSession(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID, parentTokenID *ids.XID, now time.Time, idleExpiresAt time.Time) (AccessToken, error) {
	row, err := tx.queries.CreateGuardianOnboardingSession(ctx, db.CreateGuardianOnboardingSessionParams{
		TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: organizationID, SchoolYearID: schoolYearID,
		ParentTokenID: parentTokenID, LastSeenAt: timestamp(now), IdleExpiresAt: timestamp(idleExpiresAt),
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetGuardianOnboardingSessionByHash(ctx context.Context, tokenHash []byte, now time.Time) (AccessToken, error) {
	row, err := tx.queries.GetGuardianOnboardingSessionByHash(ctx, db.GetGuardianOnboardingSessionByHashParams{TokenHash: tokenHash, ExpiresAt: timestamp(now)})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) TouchGuardianOnboardingSession(ctx context.Context, id ids.XID, now, idleExpiresAt time.Time) (bool, error) {
	rows, err := tx.queries.TouchGuardianOnboardingSession(ctx, db.TouchGuardianOnboardingSessionParams{ID: id, LastSeenAt: timestamp(now), IdleExpiresAt: timestamp(idleExpiresAt)})
	return rows == 1, err
}

func (tx *Tx) VerifyGuardianOnboardingSession(ctx context.Context, id ids.XID, emailHash []byte, verifiedAt time.Time) (AccessToken, error) {
	row, err := tx.queries.VerifyGuardianOnboardingSession(ctx, db.VerifyGuardianOnboardingSessionParams{ID: id, RequestedEmailHash: emailHash, MailboxVerifiedAt: timestamp(verifiedAt)})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) CompleteGuardianOnboardingSession(ctx context.Context, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.CompleteGuardianOnboardingSession(ctx, db.CompleteGuardianOnboardingSessionParams{ID: id, ConsumedAt: timestamp(at)})
	return rows == 1, err
}

func (tx *Tx) RevokeGuardianOnboardingSession(ctx context.Context, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.RevokeGuardianOnboardingSession(ctx, db.RevokeGuardianOnboardingSessionParams{ID: id, RevokedAt: timestamp(at)})
	return rows == 1, err
}

func (tx *Tx) CreateGuardianOnboardingOTP(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID, sessionID ids.XID, verifierHash, emailHash []byte) (AccessToken, error) {
	row, err := tx.queries.CreateGuardianOnboardingOTP(ctx, db.CreateGuardianOnboardingOTPParams{
		TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt), OrganizationID: &organizationID, SchoolYearID: &schoolYearID,
		ParentTokenID: &sessionID, VerifierHash: verifierHash, RequestedEmailHash: emailHash,
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetGuardianOnboardingOTPByHash(ctx context.Context, tokenHash []byte) (AccessToken, error) {
	row, err := tx.queries.GetGuardianOnboardingOTPByHash(ctx, tokenHash)
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) CountRecentGuardianOnboardingOTPRequests(ctx context.Context, organizationID, schoolYearID *ids.XID, emailHash []byte, since time.Time) (int64, error) {
	return tx.queries.CountRecentGuardianOnboardingOTPRequests(ctx, db.CountRecentGuardianOnboardingOTPRequestsParams{OrganizationID: organizationID, SchoolYearID: schoolYearID, RequestedEmailHash: emailHash, CreatedAt: timestamp(since)})
}

func (tx *Tx) ConsumeGuardianOnboardingOTP(ctx context.Context, id ids.XID, verifierHash []byte, now time.Time, maxAttempts int) (AccessToken, error) {
	row, err := tx.queries.ConsumeGuardianOnboardingOTP(ctx, db.ConsumeGuardianOnboardingOTPParams{ID: id, VerifierHash: verifierHash, ConsumedAt: timestamp(now), Attempts: int32(maxAttempts)})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) IncrementGuardianOnboardingOTPAttempts(ctx context.Context, id ids.XID, now time.Time, maxAttempts int) (bool, error) {
	rows, err := tx.queries.IncrementGuardianOnboardingOTPAttempts(ctx, db.IncrementGuardianOnboardingOTPAttemptsParams{ID: id, ExpiresAt: timestamp(now), Attempts: int32(maxAttempts)})
	return rows == 1, err
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
