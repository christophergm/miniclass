package identity

import (
	"context"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgtype"
)

func (tx *Tx) CreateAdultOTP(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID, adultID *ids.XID, verifierHash, emailHash []byte) (AccessToken, error) {
	row, err := tx.queries.CreateAdultOTP(ctx, db.CreateAdultOTPParams{
		TokenHash: tokenHash, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		OrganizationID: organizationID, SchoolYearID: schoolYearID, AdultID: adultID,
		VerifierHash: verifierHash, RequestedEmailHash: emailHash,
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) CountRecentAdultOTPRequests(ctx context.Context, organizationID, schoolYearID *ids.XID, emailHash []byte, since time.Time) (int64, error) {
	return tx.queries.CountRecentAdultOTPRequests(ctx, db.CountRecentAdultOTPRequestsParams{
		OrganizationID: organizationID, SchoolYearID: schoolYearID,
		RequestedEmailHash: emailHash, CreatedAt: pgtype.Timestamptz{Time: since, Valid: true},
	})
}

func (tx *Tx) GetAdultOTP(ctx context.Context, id ids.XID) (AccessToken, error) {
	row, err := tx.queries.GetAdultOTP(ctx, id)
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetAdultOTPByHash(ctx context.Context, tokenHash []byte) (AccessToken, error) {
	row, err := tx.queries.GetAdultOTPByHash(ctx, tokenHash)
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) ConsumeAdultOTP(ctx context.Context, id ids.XID, verifierHash []byte, now time.Time, maxAttempts int) (AccessToken, error) {
	row, err := tx.queries.ConsumeAdultOTP(ctx, db.ConsumeAdultOTPParams{
		ID: id, VerifierHash: verifierHash, ExpiresAt: pgtype.Timestamptz{Time: now, Valid: true}, Attempts: int32(maxAttempts),
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) IncrementAdultOTPAttempts(ctx context.Context, id ids.XID, now time.Time, maxAttempts int) (bool, error) {
	rows, err := tx.queries.IncrementAdultOTPAttempts(ctx, db.IncrementAdultOTPAttemptsParams{
		ID: id, ExpiresAt: pgtype.Timestamptz{Time: now, Valid: true}, Attempts: int32(maxAttempts),
	})
	return rows == 1, err
}

func (tx *Tx) CreateGuardianSession(ctx context.Context, tokenHash []byte, expiresAt time.Time, organizationID, schoolYearID, adultID *ids.XID, idleExpiresAt, lastSeenAt time.Time) (AccessToken, error) {
	row, err := tx.queries.CreateGuardianSession(ctx, db.CreateGuardianSessionParams{
		TokenHash: tokenHash, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		OrganizationID: organizationID, SchoolYearID: schoolYearID, AdultID: adultID,
		IdleExpiresAt: pgtype.Timestamptz{Time: idleExpiresAt, Valid: true},
		LastSeenAt:    pgtype.Timestamptz{Time: lastSeenAt, Valid: true},
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) CreateAdministrativeSession(ctx context.Context, tokenHash []byte, userID *ids.XID, expiresAt, idleExpiresAt, lastSeenAt time.Time, mfaGeneration int) (AccessToken, error) {
	row, err := tx.queries.CreateAdministrativeSession(ctx, db.CreateAdministrativeSessionParams{
		TokenHash: tokenHash, UserID: userID,
		ExpiresAt:     pgtype.Timestamptz{Time: expiresAt, Valid: true},
		IdleExpiresAt: pgtype.Timestamptz{Time: idleExpiresAt, Valid: true},
		LastSeenAt:    pgtype.Timestamptz{Time: lastSeenAt, Valid: true},
		MfaGeneration: pgtype.Int4{Int32: int32(mfaGeneration), Valid: true},
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) GetActiveSessionByHash(ctx context.Context, tokenHash []byte, now time.Time) (AccessToken, error) {
	row, err := tx.queries.GetActiveSessionByHash(ctx, db.GetActiveSessionByHashParams{
		TokenHash: tokenHash, ExpiresAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return AccessToken{}, err
	}
	return accessToken(row)
}

func (tx *Tx) TouchSession(ctx context.Context, id ids.XID, lastSeenAt, idleExpiresAt time.Time) (bool, error) {
	rows, err := tx.queries.TouchSession(ctx, db.TouchSessionParams{
		ID: id, LastSeenAt: pgtype.Timestamptz{Time: lastSeenAt, Valid: true},
		IdleExpiresAt: pgtype.Timestamptz{Time: idleExpiresAt, Valid: true},
	})
	return rows == 1, err
}

func (tx *Tx) RevokeSession(ctx context.Context, id ids.XID, at time.Time) (bool, error) {
	rows, err := tx.queries.RevokeSession(ctx, db.RevokeSessionParams{
		ID: id, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return rows == 1, err
}

func (tx *Tx) RevokeAdministrativeSessions(ctx context.Context, userID *ids.XID, at time.Time) (int64, error) {
	return tx.queries.RevokeAdministrativeSessions(ctx, db.RevokeAdministrativeSessionsParams{
		UserID: userID, RevokedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

type MFAState struct {
	UserID     ids.XID
	Secret     []byte
	EnrolledAt *time.Time
	Generation int
}

func (tx *Tx) UserBelongsToOrganization(ctx context.Context, userID, organizationID ids.XID) (bool, error) {
	return tx.queries.UserBelongsToOrganization(ctx, db.UserBelongsToOrganizationParams{UserID: &userID, OrganizationID: organizationID})
}

func (tx *Tx) GetMFAState(ctx context.Context, userID ids.XID) (MFAState, error) {
	row, err := tx.queries.GetMFAState(ctx, userID)
	if err != nil {
		return MFAState{}, err
	}
	return MFAState{UserID: row.ID, Secret: row.MfaSecretCiphertext, EnrolledAt: nullableTime(row.MfaEnrolledAt), Generation: int(row.MfaGeneration)}, nil
}

func (tx *Tx) SetMFASecret(ctx context.Context, userID ids.XID, secret []byte, enrolledAt time.Time) (MFAState, error) {
	row, err := tx.queries.SetMFASecret(ctx, db.SetMFASecretParams{
		ID: userID, MfaSecretCiphertext: secret, MfaEnrolledAt: pgtype.Timestamptz{Time: enrolledAt, Valid: true},
	})
	if err != nil {
		return MFAState{}, err
	}
	return MFAState{UserID: row.ID, Secret: row.MfaSecretCiphertext, EnrolledAt: nullableTime(row.MfaEnrolledAt), Generation: int(row.MfaGeneration)}, nil
}

func (tx *Tx) ResetMFASecret(ctx context.Context, userID ids.XID) (MFAState, error) {
	row, err := tx.queries.ResetMFASecret(ctx, userID)
	if err != nil {
		return MFAState{}, err
	}
	return MFAState{UserID: row.ID, Secret: row.MfaSecretCiphertext, EnrolledAt: nullableTime(row.MfaEnrolledAt), Generation: int(row.MfaGeneration)}, nil
}

func (tx *Tx) DeleteMFARecoveryCodes(ctx context.Context, userID ids.XID) error {
	return tx.queries.DeleteMFARecoveryCodes(ctx, userID)
}

func (tx *Tx) CreateMFARecoveryCode(ctx context.Context, userID ids.XID, codeHash []byte) error {
	return tx.queries.CreateMFARecoveryCode(ctx, db.CreateMFARecoveryCodeParams{UserID: userID, CodeHash: codeHash})
}

func (tx *Tx) ConsumeMFARecoveryCode(ctx context.Context, userID ids.XID, codeHash []byte, at time.Time) (bool, error) {
	rows, err := tx.queries.ConsumeMFARecoveryCode(ctx, db.ConsumeMFARecoveryCodeParams{
		UserID: userID, CodeHash: codeHash, UsedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	return rows == 1, err
}
