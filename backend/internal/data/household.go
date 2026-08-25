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

// GuardianRelationshipType is the relationship vocabulary from SPEC §8.2.
type GuardianRelationshipType string

const (
	GuardianRelationshipParent      GuardianRelationshipType = "parent"
	GuardianRelationshipGuardian    GuardianRelationshipType = "guardian"
	GuardianRelationshipGrandparent GuardianRelationshipType = "grandparent"
	GuardianRelationshipOther       GuardianRelationshipType = "other"
)

type Household struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	DisplayName    string
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type HouseholdStudent struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	HouseholdID    ids.XID
	StudentID      ids.XID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type HouseholdAdult struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	HouseholdID    ids.XID
	AdultID        ids.XID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

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

func (tx *Tx) CreateHousehold(ctx context.Context, schoolYearID ids.XID, displayName string) (Household, error) {
	displayName = strings.TrimSpace(displayName)
	if strings.TrimSpace(string(schoolYearID)) == "" || displayName == "" {
		return Household{}, errors.New("create household: school year and display name are required")
	}
	row, err := tx.queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, DisplayName: displayName,
	})
	if err != nil {
		return Household{}, wrapHouseholdMutationError("create household", err)
	}
	return household(row)
}

func (tx *Tx) ListHouseholds(ctx context.Context, schoolYearID ids.XID) ([]Household, error) {
	rows, err := tx.queries.ListHouseholds(ctx, db.ListHouseholdsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return nil, fmt.Errorf("list households: %w", err)
	}
	result := make([]Household, 0, len(rows))
	for _, row := range rows {
		value, err := household(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetHouseholdByID(ctx context.Context, schoolYearID, id ids.XID) (Household, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return Household{}, errors.New("get household: ids are required")
	}
	row, err := tx.queries.GetHouseholdByID(ctx, db.GetHouseholdByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Household{}, fmt.Errorf("get household: %w", err)
	}
	return household(row)
}

func (tx *Tx) UpdateHousehold(ctx context.Context, schoolYearID, id ids.XID, displayName string) (Household, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return Household{}, errors.New("update household: display name is required")
	}
	row, err := tx.queries.UpdateHousehold(ctx, db.UpdateHouseholdParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, DisplayName: displayName,
	})
	if err != nil {
		return Household{}, wrapHouseholdMutationError("update household", err)
	}
	return household(row)
}

func (tx *Tx) SoftDeleteHousehold(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	rows, err := tx.queries.SoftDeleteHousehold(ctx, db.SoftDeleteHouseholdParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return false, wrapHouseholdMutationError("delete household", err)
	}
	return rows == 1, nil
}

func (tx *Tx) ListAllActiveHouseholdsForRegistry(ctx context.Context) ([]Household, error) {
	rows, err := tx.queries.ListAllActiveHouseholdsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list households for registry: %w", err)
	}
	result := make([]Household, 0, len(rows))
	for _, row := range rows {
		value, err := household(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindHouseholdForRegistry(ctx context.Context, id ids.XID) (Household, ids.XID, error) {
	row, err := tx.queries.FindHouseholdForRegistry(ctx, db.FindHouseholdForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Household{}, "", nil
		}
		return Household{}, "", fmt.Errorf("find household for registry: %w", err)
	}
	value, err := household(row)
	return value, value.SchoolYearID, err
}

func (tx *Tx) CreateHouseholdStudent(ctx context.Context, schoolYearID, householdID, studentID ids.XID) (HouseholdStudent, error) {
	row, err := tx.queries.CreateHouseholdStudent(ctx, db.CreateHouseholdStudentParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID, StudentID: studentID,
	})
	if err != nil {
		return HouseholdStudent{}, wrapHouseholdMutationError("create household student membership", err)
	}
	return householdStudent(row)
}

func (tx *Tx) ListHouseholdStudents(ctx context.Context, schoolYearID, householdID ids.XID) ([]HouseholdStudent, error) {
	rows, err := tx.queries.ListHouseholdStudents(ctx, db.ListHouseholdStudentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID})
	if err != nil {
		return nil, fmt.Errorf("list household students: %w", err)
	}
	result := make([]HouseholdStudent, 0, len(rows))
	for _, row := range rows {
		value, err := householdStudent(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) DeleteHouseholdStudent(ctx context.Context, schoolYearID, householdID, studentID ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteHouseholdStudentMembership(ctx, db.DeleteHouseholdStudentMembershipParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID, StudentID: studentID,
	})
	if err != nil {
		return false, wrapHouseholdMutationError("delete household student membership", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteHouseholdStudentMembership(ctx context.Context, schoolYearID, householdID, studentID ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteHouseholdStudentMembership(ctx, db.DeleteHouseholdStudentMembershipParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID, StudentID: studentID,
	})
	if err != nil {
		return false, wrapHouseholdMutationError("delete household student membership", err)
	}
	return rows == 1, nil
}

func (tx *Tx) ListAllHouseholdStudentsForRegistry(ctx context.Context) ([]HouseholdStudent, error) {
	rows, err := tx.queries.ListAllHouseholdStudentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list household students for registry: %w", err)
	}
	result := make([]HouseholdStudent, 0, len(rows))
	for _, row := range rows {
		value, err := householdStudent(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindHouseholdStudentForRegistry(ctx context.Context, id ids.XID) (HouseholdStudent, ids.XID, error) {
	row, err := tx.queries.FindHouseholdStudentForRegistry(ctx, db.FindHouseholdStudentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return HouseholdStudent{}, "", nil
		}
		return HouseholdStudent{}, "", fmt.Errorf("find household student for registry: %w", err)
	}
	value, err := householdStudent(row)
	return value, value.SchoolYearID, err
}

func (tx *Tx) TouchHouseholdStudent(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	_, err := tx.queries.TouchHouseholdStudent(ctx, db.TouchHouseholdStudentParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, wrapHouseholdMutationError("touch household student membership", err)
	}
	return true, nil
}

func (tx *Tx) CreateHouseholdAdult(ctx context.Context, schoolYearID, householdID, adultID ids.XID) (HouseholdAdult, error) {
	row, err := tx.queries.CreateHouseholdAdult(ctx, db.CreateHouseholdAdultParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID, AdultID: adultID,
	})
	if err != nil {
		return HouseholdAdult{}, wrapHouseholdMutationError("create household adult membership", err)
	}
	return householdAdult(row)
}

func (tx *Tx) ListHouseholdAdults(ctx context.Context, schoolYearID, householdID ids.XID) ([]HouseholdAdult, error) {
	rows, err := tx.queries.ListHouseholdAdults(ctx, db.ListHouseholdAdultsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID})
	if err != nil {
		return nil, fmt.Errorf("list household adults: %w", err)
	}
	result := make([]HouseholdAdult, 0, len(rows))
	for _, row := range rows {
		value, err := householdAdult(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) DeleteHouseholdAdult(ctx context.Context, schoolYearID, householdID, adultID ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteHouseholdAdultMembership(ctx, db.DeleteHouseholdAdultMembershipParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID, AdultID: adultID,
	})
	if err != nil {
		return false, wrapHouseholdMutationError("delete household adult membership", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteHouseholdAdultMembership(ctx context.Context, schoolYearID, householdID, adultID ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteHouseholdAdultMembership(ctx, db.DeleteHouseholdAdultMembershipParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, HouseholdID: householdID, AdultID: adultID,
	})
	if err != nil {
		return false, wrapHouseholdMutationError("delete household adult membership", err)
	}
	return rows == 1, nil
}

func (tx *Tx) ListAllHouseholdAdultsForRegistry(ctx context.Context) ([]HouseholdAdult, error) {
	rows, err := tx.queries.ListAllHouseholdAdultsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list household adults for registry: %w", err)
	}
	result := make([]HouseholdAdult, 0, len(rows))
	for _, row := range rows {
		value, err := householdAdult(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindHouseholdAdultForRegistry(ctx context.Context, id ids.XID) (HouseholdAdult, ids.XID, error) {
	row, err := tx.queries.FindHouseholdAdultForRegistry(ctx, db.FindHouseholdAdultForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return HouseholdAdult{}, "", nil
		}
		return HouseholdAdult{}, "", fmt.Errorf("find household adult for registry: %w", err)
	}
	value, err := householdAdult(row)
	return value, value.SchoolYearID, err
}

func (tx *Tx) TouchHouseholdAdult(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	_, err := tx.queries.TouchHouseholdAdult(ctx, db.TouchHouseholdAdultParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, wrapHouseholdMutationError("touch household adult membership", err)
	}
	return true, nil
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
		return GuardianRelationship{}, wrapHouseholdMutationError("create guardian relationship", err)
	}
	return guardianRelationship(row)
}

func (tx *Tx) ListGuardianRelationships(ctx context.Context, schoolYearID ids.XID) ([]GuardianRelationship, error) {
	rows, err := tx.queries.ListGuardianRelationships(ctx, db.ListGuardianRelationshipsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
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
		return GuardianRelationship{}, wrapHouseholdMutationError("update guardian relationship", err)
	}
	return guardianRelationship(row)
}

func (tx *Tx) DeleteGuardianRelationship(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteGuardianRelationship(ctx, db.DeleteGuardianRelationshipParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return false, wrapHouseholdMutationError("delete guardian relationship", err)
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

func household(row db.Household) (Household, error) {
	createdAt, err := householdTime(row.CreatedAt, "created_at")
	if err != nil {
		return Household{}, err
	}
	updatedAt, err := householdTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return Household{}, err
	}
	return Household{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, DisplayName: row.DisplayName,
		DeletedAt: nullableHouseholdTime(row.DeletedAt), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func householdStudent(row db.HouseholdStudent) (HouseholdStudent, error) {
	createdAt, err := householdTime(row.CreatedAt, "created_at")
	if err != nil {
		return HouseholdStudent{}, err
	}
	updatedAt, err := householdTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return HouseholdStudent{}, err
	}
	return HouseholdStudent{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, HouseholdID: row.HouseholdID, StudentID: row.StudentID, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func householdAdult(row db.HouseholdAdult) (HouseholdAdult, error) {
	createdAt, err := householdTime(row.CreatedAt, "created_at")
	if err != nil {
		return HouseholdAdult{}, err
	}
	updatedAt, err := householdTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return HouseholdAdult{}, err
	}
	return HouseholdAdult{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, HouseholdID: row.HouseholdID, AdultID: row.AdultID, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func guardianRelationship(row db.GuardianRelationship) (GuardianRelationship, error) {
	createdAt, err := householdTime(row.CreatedAt, "created_at")
	if err != nil {
		return GuardianRelationship{}, err
	}
	updatedAt, err := householdTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return GuardianRelationship{}, err
	}
	return GuardianRelationship{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, AdultID: row.AdultID, StudentID: row.StudentID,
		RelationshipType: GuardianRelationshipType(row.RelationshipType), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func householdTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("household row: %s is null", name)
	}
	return value.Time, nil
}

func nullableHouseholdTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func wrapHouseholdMutationError(operation string, err error) error {
	if isClosedYearDatabaseError(err) {
		return fmt.Errorf("%w: %v", ErrSchoolYearClosed, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
