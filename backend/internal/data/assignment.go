package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

// Assignment is the current, replaceable session draft. Its quality is the
// historical result recorded by the solver rather than a live preference read
// (SPEC §8.6).
type Assignment struct {
	ID              ids.XID   `json:"id"`
	OrganizationID  ids.XID   `json:"organization_id"`
	SchoolYearID    ids.XID   `json:"school_year_id"`
	ProgramID       ids.XID   `json:"program_id"`
	SessionID       ids.XID   `json:"session_id"`
	StudentID       ids.XID   `json:"student_id"`
	OfferingID      ids.XID   `json:"offering_id"`
	SolveRunID      ids.XID   `json:"solve_run_id"`
	Origin          string    `json:"origin"`
	Pinned          bool      `json:"pinned"`
	RealizedQuality string    `json:"realized_quality"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateAssignmentInput struct {
	SchoolYearID    ids.XID
	ProgramID       ids.XID
	SessionID       ids.XID
	StudentID       ids.XID
	OfferingID      ids.XID
	SolveRunID      ids.XID
	Origin          string
	Pinned          bool
	RealizedQuality string
}

func (tx *Tx) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (Assignment, error) {
	if tx == nil || tx.queries == nil {
		return Assignment{}, errors.New("create assignment: transaction is nil")
	}
	if input.SchoolYearID == "" || input.ProgramID == "" || input.SessionID == "" || input.StudentID == "" || input.OfferingID == "" || input.SolveRunID == "" || input.Origin == "" || input.RealizedQuality == "" {
		return Assignment{}, errors.New("create assignment: required assignment fields are missing")
	}
	row, err := tx.queries.CreateAssignment(ctx, db.CreateAssignmentParams{
		OrganizationID: tx.organizationID, SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID,
		StudentID: input.StudentID, OfferingID: input.OfferingID, SolveRunID: input.SolveRunID, Origin: input.Origin,
		Pinned: input.Pinned, RealizedQuality: input.RealizedQuality,
	})
	if err != nil {
		return Assignment{}, fmt.Errorf("create assignment: %w", err)
	}
	return assignment(row)
}

// ReplaceDraftAssignments clears exactly one session's current draft before
// writing its complete replacement in the same caller-owned transaction.
func (tx *Tx) ReplaceDraftAssignments(ctx context.Context, schoolYearID, programID, sessionID ids.XID, inputs []CreateAssignmentInput) ([]Assignment, error) {
	if tx == nil || tx.queries == nil {
		return nil, errors.New("replace draft assignments: transaction is nil")
	}
	if schoolYearID == "" || programID == "" || sessionID == "" {
		return nil, errors.New("replace draft assignments: session scope is required")
	}
	if _, err := tx.queries.DeleteDraftAssignments(ctx, db.DeleteDraftAssignmentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID}); err != nil {
		return nil, fmt.Errorf("replace draft assignments: delete current draft: %w", err)
	}
	assignments := make([]Assignment, 0, len(inputs))
	for _, input := range inputs {
		if input.SchoolYearID != schoolYearID || input.ProgramID != programID || input.SessionID != sessionID {
			return nil, errors.New("replace draft assignments: assignment scope does not match session")
		}
		assignment, err := tx.CreateAssignment(ctx, input)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

func (tx *Tx) ListAssignments(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]Assignment, error) {
	if tx == nil || tx.queries == nil {
		return nil, errors.New("list assignments: transaction is nil")
	}
	rows, err := tx.queries.ListAssignments(ctx, db.ListAssignmentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	result := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		value, err := assignment(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// ListAllAssignmentsForRegistry is restricted to the Layer 2 tenancy harness.
func (tx *Tx) ListAllAssignmentsForRegistry(ctx context.Context) ([]Assignment, error) {
	rows, err := tx.queries.ListAllAssignmentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list assignments for registry: %w", err)
	}
	result := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		value, err := assignment(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// FindAssignmentForRegistry is restricted to the Layer 2 tenancy harness.
func (tx *Tx) FindAssignmentForRegistry(ctx context.Context, id ids.XID) (Assignment, error) {
	row, err := tx.queries.FindAssignmentForRegistry(ctx, db.FindAssignmentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return Assignment{}, err
	}
	return assignment(row)
}

func (tx *Tx) TouchAssignmentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.TouchAssignmentForRegistry(ctx, db.TouchAssignmentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (tx *Tx) DeleteAssignmentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteAssignmentForRegistry(ctx, db.DeleteAssignmentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func assignment(row db.Assignment) (Assignment, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return Assignment{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return Assignment{}, err
	}
	return Assignment{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID,
		SessionID: row.SessionID, StudentID: row.StudentID, OfferingID: row.OfferingID, SolveRunID: row.SolveRunID,
		Origin: row.Origin, Pinned: row.Pinned, RealizedQuality: row.RealizedQuality, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}
