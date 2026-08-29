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

// GuardianRelationshipType is the relationship vocabulary from SPEC §8.2.
type GuardianRelationshipType string

const (
	GuardianRelationshipParent      GuardianRelationshipType = "parent"
	GuardianRelationshipGuardian    GuardianRelationshipType = "guardian"
	GuardianRelationshipGrandparent GuardianRelationshipType = "grandparent"
	GuardianRelationshipOther       GuardianRelationshipType = "other"
)

type GuardianRelationship struct {
	ID               ids.XID
	OrganizationID   ids.XID
	SchoolYearID     ids.XID
	AdultID          ids.XID
	StudentID        ids.XID
	RelationshipType GuardianRelationshipType
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (tx *Tx) CreateGuardianRelationship(ctx context.Context, schoolYearID, adultID, studentID ids.XID, relationshipType GuardianRelationshipType) (GuardianRelationship, error) {
	if !validGuardianRelationshipType(relationshipType) {
		return GuardianRelationship{}, fmt.Errorf("create guardian relationship: invalid relationship type %q", relationshipType)
	}
	row, err := tx.queries.CreateGuardianRelationship(ctx, db.CreateGuardianRelationshipParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, AdultID: adultID, StudentID: studentID,
		RelationshipType: db.GuardianRelationshipType(relationshipType),
	})
	if err != nil {
		return GuardianRelationship{}, wrapGuardianRelationshipMutationError("create guardian relationship", err)
	}
	return guardianRelationship(row)
}

// GuardianRelationshipFilter narrows a listing to one side of the link. A zero
// identifier leaves that side unconstrained, so the zero filter lists the whole
// school year.
type GuardianRelationshipFilter struct {
	AdultID   ids.XID
	StudentID ids.XID
}

func (tx *Tx) ListGuardianRelationships(ctx context.Context, schoolYearID ids.XID, filter GuardianRelationshipFilter) ([]GuardianRelationship, error) {
	rows, err := tx.queries.ListGuardianRelationships(ctx, db.ListGuardianRelationshipsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID,
		AdultID: nullableGuardianRelationshipXID(filter.AdultID), StudentID: nullableGuardianRelationshipXID(filter.StudentID),
	})
	if err != nil {
		return nil, fmt.Errorf("list guardian relationships: %w", err)
	}
	result := make([]GuardianRelationship, 0, len(rows))
	for _, row := range rows {
		value, err := guardianRelationship(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetGuardianRelationshipByID(ctx context.Context, schoolYearID, id ids.XID) (GuardianRelationship, error) {
	row, err := tx.queries.GetGuardianRelationshipByID(ctx, db.GetGuardianRelationshipByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GuardianRelationship{}, fmt.Errorf("get guardian relationship: %w", err)
	}
	return guardianRelationship(row)
}

func (tx *Tx) UpdateGuardianRelationship(ctx context.Context, schoolYearID, id ids.XID, relationshipType GuardianRelationshipType) (GuardianRelationship, error) {
	if !validGuardianRelationshipType(relationshipType) {
		return GuardianRelationship{}, fmt.Errorf("update guardian relationship: invalid relationship type %q", relationshipType)
	}
	row, err := tx.queries.UpdateGuardianRelationship(ctx, db.UpdateGuardianRelationshipParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, RelationshipType: db.GuardianRelationshipType(relationshipType),
	})
	if err != nil {
		return GuardianRelationship{}, wrapGuardianRelationshipMutationError("update guardian relationship", err)
	}
	return guardianRelationship(row)
}

func (tx *Tx) DeleteGuardianRelationship(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteGuardianRelationship(ctx, db.DeleteGuardianRelationshipParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return false, wrapGuardianRelationshipMutationError("delete guardian relationship", err)
	}
	return rows == 1, nil
}

func (tx *Tx) ListAllGuardianRelationshipsForRegistry(ctx context.Context) ([]GuardianRelationship, error) {
	rows, err := tx.queries.ListAllGuardianRelationshipsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list guardian relationships for registry: %w", err)
	}
	result := make([]GuardianRelationship, 0, len(rows))
	for _, row := range rows {
		value, err := guardianRelationship(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindGuardianRelationshipForRegistry(ctx context.Context, id ids.XID) (GuardianRelationship, ids.XID, error) {
	row, err := tx.queries.FindGuardianRelationshipForRegistry(ctx, db.FindGuardianRelationshipForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GuardianRelationship{}, "", nil
		}
		return GuardianRelationship{}, "", fmt.Errorf("find guardian relationship for registry: %w", err)
	}
	value, err := guardianRelationship(row)
	return value, value.SchoolYearID, err
}

func validGuardianRelationshipType(value GuardianRelationshipType) bool {
	switch value {
	case GuardianRelationshipParent, GuardianRelationshipGuardian, GuardianRelationshipGrandparent, GuardianRelationshipOther:
		return true
	default:
		return false
	}
}

func guardianRelationship(row db.GuardianRelationship) (GuardianRelationship, error) {
	createdAt, err := guardianRelationshipTime(row.CreatedAt, "created_at")
	if err != nil {
		return GuardianRelationship{}, err
	}
	updatedAt, err := guardianRelationshipTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return GuardianRelationship{}, err
	}
	return GuardianRelationship{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, AdultID: row.AdultID, StudentID: row.StudentID,
		RelationshipType: GuardianRelationshipType(row.RelationshipType), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func guardianRelationshipTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("guardian relationship row: %s is null", name)
	}
	return value.Time, nil
}

// nullableGuardianRelationshipXID turns an unset identifier into a SQL null, so
// an optional filter predicate is skipped rather than matched against the empty
// string.
func nullableGuardianRelationshipXID(value ids.XID) *ids.XID {
	if value == "" {
		return nil
	}
	result := value
	return &result
}

func wrapGuardianRelationshipMutationError(operation string, err error) error {
	if isClosedYearDatabaseError(err) {
		return fmt.Errorf("%w: %v", ErrSchoolYearClosed, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
