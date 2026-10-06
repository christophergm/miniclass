package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

// AssignmentExclusion is a session-scoped hard-rule exclusion.
type AssignmentExclusion struct {
	ID, OrganizationID, SchoolYearID, ProgramID, SessionID, StudentID, OfferingID ids.XID
	CreatedAt, UpdatedAt                                                          time.Time
}

// AssignmentOverride records a named human acceptance of one hard-rule
// violation for one placement (SPEC §16.7).
type AssignmentOverride struct {
	ID, OrganizationID, SchoolYearID, ProgramID, SessionID, AssignmentID ids.XID
	Rule, Reason, RecordedBy                                             string
	CreatedAt, UpdatedAt                                                 time.Time
}

func (tx *Tx) CreateAssignmentExclusion(ctx context.Context, schoolYearID, programID, sessionID, studentID, offeringID ids.XID) (AssignmentExclusion, error) {
	row, err := tx.queries.CreateAssignmentExclusion(ctx, db.CreateAssignmentExclusionParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, StudentID: studentID, OfferingID: offeringID})
	if err != nil {
		return AssignmentExclusion{}, fmt.Errorf("create assignment exclusion: %w", err)
	}
	return assignmentExclusion(row)
}

func (tx *Tx) CreateAssignmentOverride(ctx context.Context, schoolYearID, programID, sessionID, assignmentID ids.XID, rule, reason, recordedBy string) (AssignmentOverride, error) {
	rule, reason, recordedBy = strings.TrimSpace(rule), strings.TrimSpace(reason), strings.TrimSpace(recordedBy)
	if rule == "" || reason == "" || recordedBy == "" {
		return AssignmentOverride{}, errors.New("create assignment override: rule, reason, and recorder are required")
	}
	row, err := tx.queries.CreateAssignmentOverride(ctx, db.CreateAssignmentOverrideParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, AssignmentID: assignmentID, Rule: rule, Reason: reason, RecordedBy: recordedBy})
	if err != nil {
		return AssignmentOverride{}, fmt.Errorf("create assignment override: %w", err)
	}
	return assignmentOverride(row)
}

func (tx *Tx) ListAssignmentExclusions(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]AssignmentExclusion, error) {
	rows, err := tx.queries.ListAssignmentExclusions(ctx, db.ListAssignmentExclusionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list assignment exclusions: %w", err)
	}
	result := make([]AssignmentExclusion, 0, len(rows))
	for _, row := range rows {
		value, err := assignmentExclusion(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) ListAssignmentOverrides(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]AssignmentOverride, error) {
	rows, err := tx.queries.ListAssignmentOverrides(ctx, db.ListAssignmentOverridesParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list assignment overrides: %w", err)
	}
	result := make([]AssignmentOverride, 0, len(rows))
	for _, row := range rows {
		value, err := assignmentOverride(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// AdvanceDraftRevision is a compare-and-swap primitive for future editing
// operations. It returns pgx.ErrNoRows when the caller's revision is stale.
func (tx *Tx) AdvanceDraftRevision(ctx context.Context, schoolYearID, programID, sessionID ids.XID, expected int64) (int64, error) {
	revision, err := tx.queries.AdvanceDraftRevision(ctx, db.AdvanceDraftRevisionParams{ID: sessionID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, DraftRevision: expected})
	if err != nil {
		return 0, fmt.Errorf("advance draft revision: %w", err)
	}
	return revision, nil
}

func assignmentExclusion(row db.AssignmentExclusion) (AssignmentExclusion, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return AssignmentExclusion{}, err
	}
	updated, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return AssignmentExclusion{}, err
	}
	return AssignmentExclusion{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, StudentID: row.StudentID, OfferingID: row.OfferingID, CreatedAt: created, UpdatedAt: updated}, nil
}

func assignmentOverride(row db.AssignmentOverride) (AssignmentOverride, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return AssignmentOverride{}, err
	}
	updated, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return AssignmentOverride{}, err
	}
	return AssignmentOverride{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, AssignmentID: row.AssignmentID, Rule: row.Rule, Reason: row.Reason, RecordedBy: row.RecordedBy, CreatedAt: created, UpdatedAt: updated}, nil
}

// The following methods are restricted to the Layer 2 isolation registry.
func (tx *Tx) ListAllAssignmentExclusionsForRegistry(ctx context.Context) ([]AssignmentExclusion, error) {
	rows, err := tx.queries.ListAllAssignmentExclusionsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]AssignmentExclusion, 0, len(rows))
	for _, row := range rows {
		value, err := assignmentExclusion(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func (tx *Tx) FindAssignmentExclusionForRegistry(ctx context.Context, id ids.XID) (AssignmentExclusion, error) {
	row, err := tx.queries.FindAssignmentExclusionForRegistry(ctx, db.FindAssignmentExclusionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return AssignmentExclusion{}, err
	}
	return assignmentExclusion(row)
}
func (tx *Tx) TouchAssignmentExclusionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	n, err := tx.queries.TouchAssignmentExclusionForRegistry(ctx, db.TouchAssignmentExclusionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return n > 0, err
}
func (tx *Tx) DeleteAssignmentExclusionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	n, err := tx.queries.DeleteAssignmentExclusionForRegistry(ctx, db.DeleteAssignmentExclusionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return n > 0, err
}
func (tx *Tx) ListAllAssignmentOverridesForRegistry(ctx context.Context) ([]AssignmentOverride, error) {
	rows, err := tx.queries.ListAllAssignmentOverridesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]AssignmentOverride, 0, len(rows))
	for _, row := range rows {
		value, err := assignmentOverride(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func (tx *Tx) FindAssignmentOverrideForRegistry(ctx context.Context, id ids.XID) (AssignmentOverride, error) {
	row, err := tx.queries.FindAssignmentOverrideForRegistry(ctx, db.FindAssignmentOverrideForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return AssignmentOverride{}, err
	}
	return assignmentOverride(row)
}
func (tx *Tx) TouchAssignmentOverrideForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	n, err := tx.queries.TouchAssignmentOverrideForRegistry(ctx, db.TouchAssignmentOverrideForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return n > 0, err
}
func (tx *Tx) DeleteAssignmentOverrideForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	n, err := tx.queries.DeleteAssignmentOverrideForRegistry(ctx, db.DeleteAssignmentOverrideForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return n > 0, err
}
