package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RankedChoiceAccessCode is the persisted grant for one student in one
// session. The plaintext code is intentionally never part of this value.
type RankedChoiceAccessCode struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SessionID      ids.XID
	StudentID      ids.XID
	IssuedAt       time.Time
	RevokedAt      *time.Time
}

func (tx *Tx) CreateRankedChoiceAccessCode(ctx context.Context, schoolYearID, programID, sessionID, studentID ids.XID, codeHash string) (RankedChoiceAccessCode, error) {
	row, err := tx.queries.CreateRankedChoiceAccessCode(ctx, db.CreateRankedChoiceAccessCodeParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
		SessionID: sessionID, StudentID: studentID, CodeHash: codeHash,
	})
	if err != nil {
		return RankedChoiceAccessCode{}, fmt.Errorf("create ranked-choice access code: %w", err)
	}
	return rankedChoiceAccessCode(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.SessionID, row.StudentID, row.IssuedAt, row.RevokedAt)
}

func (tx *Tx) ListActiveRankedChoiceAccessCodes(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]RankedChoiceAccessCode, error) {
	rows, err := tx.queries.ListActiveRankedChoiceAccessCodes(ctx, db.ListActiveRankedChoiceAccessCodesParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("list active ranked-choice access codes: %w", err)
	}
	result := make([]RankedChoiceAccessCode, 0, len(rows))
	for _, row := range rows {
		value, err := rankedChoiceAccessCode(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.SessionID, row.StudentID, row.IssuedAt, row.RevokedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindActiveRankedChoiceAccessCode(ctx context.Context, schoolYearID, programID, sessionID ids.XID, codeHash string) (ids.XID, error) {
	row, err := tx.queries.FindActiveRankedChoiceAccessCode(ctx, db.FindActiveRankedChoiceAccessCodeParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, CodeHash: codeHash,
	})
	if err != nil {
		return "", fmt.Errorf("find active ranked-choice access code: %w", err)
	}
	return row, nil
}

func (tx *Tx) RevokeRankedChoiceAccessCodes(ctx context.Context, schoolYearID, programID, sessionID ids.XID) (int64, error) {
	count, err := tx.queries.RevokeRankedChoiceAccessCodes(ctx, db.RevokeRankedChoiceAccessCodesParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return 0, fmt.Errorf("revoke ranked-choice access codes: %w", err)
	}
	return count, nil
}

// ListParticipatingStudentIDs returns the current session participation
// projection used when first opening ranked-choice voting.
func (tx *Tx) ListParticipatingStudentIDs(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]ids.XID, error) {
	memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
	if err != nil {
		return nil, err
	}
	nonParticipations, err := tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
	if err != nil {
		return nil, err
	}
	excluded := make(map[ids.XID]struct{}, len(nonParticipations))
	for _, row := range nonParticipations {
		excluded[row.StudentID] = struct{}{}
	}
	result := make([]ids.XID, 0, len(memberships))
	for _, membership := range memberships {
		if _, ok := excluded[membership.StudentID]; !ok {
			result = append(result, membership.StudentID)
		}
	}
	return result, nil
}

// The following methods are used only by the Layer 2 isolation registry.
func (tx *Tx) ListAllRankedChoiceAccessCodesForRegistry(ctx context.Context) ([]RankedChoiceAccessCode, error) {
	rows, err := tx.queries.ListAllRankedChoiceAccessCodesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list ranked-choice access codes for registry: %w", err)
	}
	result := make([]RankedChoiceAccessCode, 0, len(rows))
	for _, row := range rows {
		value, err := rankedChoiceAccessCode(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.SessionID, row.StudentID, row.IssuedAt, row.RevokedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindRankedChoiceAccessCodeForRegistry(ctx context.Context, id ids.XID) (RankedChoiceAccessCode, error) {
	row, err := tx.queries.FindRankedChoiceAccessCodeForRegistry(ctx, db.FindRankedChoiceAccessCodeForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return RankedChoiceAccessCode{}, nil
	}
	if err != nil {
		return RankedChoiceAccessCode{}, fmt.Errorf("find ranked-choice access code for registry: %w", err)
	}
	return rankedChoiceAccessCode(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.SessionID, row.StudentID, row.IssuedAt, row.RevokedAt)
}

func (tx *Tx) RevokeRankedChoiceAccessCodeForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.RevokeRankedChoiceAccessCodeForRegistry(ctx, db.RevokeRankedChoiceAccessCodeForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, fmt.Errorf("revoke ranked-choice access code for registry: %w", err)
	}
	return rows == 1, nil
}

func rankedChoiceAccessCode(id, organizationID, schoolYearID, programID, sessionID, studentID ids.XID, issuedAtValue, revokedAtValue pgtype.Timestamptz) (RankedChoiceAccessCode, error) {
	issuedAt, err := programTime(issuedAtValue, "issued_at")
	if err != nil {
		return RankedChoiceAccessCode{}, err
	}
	var revokedAt *time.Time
	if revokedAtValue.Valid {
		value := revokedAtValue.Time
		revokedAt = &value
	}
	return RankedChoiceAccessCode{
		ID: id, OrganizationID: organizationID, SchoolYearID: schoolYearID,
		ProgramID: programID, SessionID: sessionID, StudentID: studentID,
		IssuedAt: issuedAt, RevokedAt: revokedAt,
	}, nil
}
