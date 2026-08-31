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
)

// SessionNonParticipation records that a programme member is excluded from
// one session. It is deliberately independent from the annual membership.
type SessionNonParticipation struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SessionID      ids.XID
	StudentID      ids.XID
	Reason         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (tx *Tx) CreateSessionNonParticipation(ctx context.Context, schoolYearID, programID, sessionID, studentID ids.XID, reason string) (SessionNonParticipation, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return SessionNonParticipation{}, errors.New("create session non-participation: reason is required")
	}
	row, err := tx.queries.CreateSessionNonParticipation(ctx, db.CreateSessionNonParticipationParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
		SessionID: sessionID, StudentID: studentID, Reason: reason,
	})
	if err != nil {
		return SessionNonParticipation{}, wrapProgramMutationError("create session non-participation", err)
	}
	return sessionNonParticipation(row)
}

func (tx *Tx) ListSessionNonParticipations(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]SessionNonParticipation, error) {
	rows, err := tx.queries.ListSessionNonParticipations(ctx, db.ListSessionNonParticipationsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("list session non-participations: %w", err)
	}
	result := make([]SessionNonParticipation, 0, len(rows))
	for _, row := range rows {
		value, err := sessionNonParticipation(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetSessionNonParticipation(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (SessionNonParticipation, error) {
	row, err := tx.queries.GetSessionNonParticipation(ctx, db.GetSessionNonParticipationParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return SessionNonParticipation{}, fmt.Errorf("get session non-participation: %w", err)
	}
	return sessionNonParticipation(row)
}

func (tx *Tx) UpdateSessionNonParticipation(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID, reason string) (SessionNonParticipation, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return SessionNonParticipation{}, errors.New("update session non-participation: reason is required")
	}
	row, err := tx.queries.UpdateSessionNonParticipation(ctx, db.UpdateSessionNonParticipationParams{
		ID: id, Reason: reason, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return SessionNonParticipation{}, wrapProgramMutationError("update session non-participation", err)
	}
	return sessionNonParticipation(row)
}

func (tx *Tx) DeleteSessionNonParticipation(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteSessionNonParticipation(ctx, db.DeleteSessionNonParticipationParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return false, wrapProgramMutationError("delete session non-participation", err)
	}
	return rows == 1, nil
}

// The following methods are used only by the Layer 2 isolation registry.
func (tx *Tx) ListAllSessionNonParticipationsForRegistry(ctx context.Context) ([]SessionNonParticipation, error) {
	rows, err := tx.queries.ListAllSessionNonParticipationsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list session non-participations for registry: %w", err)
	}
	result := make([]SessionNonParticipation, 0, len(rows))
	for _, row := range rows {
		value, err := sessionNonParticipation(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindSessionNonParticipationForRegistry(ctx context.Context, id ids.XID) (SessionNonParticipation, error) {
	row, err := tx.queries.FindSessionNonParticipationForRegistry(ctx, db.FindSessionNonParticipationForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionNonParticipation{}, nil
		}
		return SessionNonParticipation{}, fmt.Errorf("find session non-participation for registry: %w", err)
	}
	return sessionNonParticipation(row)
}

func (tx *Tx) UpdateSessionNonParticipationForRegistry(ctx context.Context, id ids.XID, reason string) (bool, error) {
	rows, err := tx.queries.UpdateSessionNonParticipationForRegistry(ctx, db.UpdateSessionNonParticipationForRegistryParams{ID: id, OrganizationID: tx.organizationID, Reason: reason})
	if err != nil {
		return false, wrapProgramMutationError("update session non-participation for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteSessionNonParticipationForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteSessionNonParticipationForRegistry(ctx, db.DeleteSessionNonParticipationForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("delete session non-participation for registry", err)
	}
	return rows == 1, nil
}

func sessionNonParticipation(row db.SessionNonParticipation) (SessionNonParticipation, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return SessionNonParticipation{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return SessionNonParticipation{}, err
	}
	return SessionNonParticipation{
		ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
		ProgramID: row.ProgramID, SessionID: row.SessionID, StudentID: row.StudentID,
		Reason: row.Reason, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}
