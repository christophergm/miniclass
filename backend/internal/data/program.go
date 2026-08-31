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

// Program is a named, year-scoped body of activity owned by one organization.
type Program struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// InterestArea is an ordered, programme-owned vocabulary entry. Its ID is
// stable across label edits and retirement so later profile and placement
// records can continue to refer to the same area.
type InterestArea struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	Label          string
	Ordinal        int
	RetiredAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ProgramMembership is an explicit annual membership. GradeMissing is
// derived from the current roster row so clearing a grade flags the member
// without silently removing the membership.
type ProgramMembership struct {
	ID              ids.XID
	OrganizationID  ids.XID
	SchoolYearID    ids.XID
	ProgramID       ids.XID
	StudentID       ids.XID
	GradeLevelID    *ids.XID
	LegalGivenName  string
	LegalFamilyName string
	GradeMissing    bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (tx *Tx) CreateProgram(ctx context.Context, schoolYearID ids.XID, name string) (Program, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Program{}, errors.New("create program: name is required")
	}
	row, err := tx.queries.CreateProgram(ctx, db.CreateProgramParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Name: name,
	})
	if err != nil {
		return Program{}, wrapProgramMutationError("create program", err)
	}
	return program(row)
}

func (tx *Tx) ListPrograms(ctx context.Context, schoolYearID ids.XID) ([]Program, error) {
	rows, err := tx.queries.ListPrograms(ctx, db.ListProgramsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	result := make([]Program, 0, len(rows))
	for _, row := range rows {
		value, err := program(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetProgram(ctx context.Context, schoolYearID, id ids.XID) (Program, error) {
	row, err := tx.queries.GetProgram(ctx, db.GetProgramParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Program{}, fmt.Errorf("get program: %w", err)
	}
	return program(row)
}

func (tx *Tx) CreateInterestArea(ctx context.Context, schoolYearID, programID ids.XID, label string, ordinal int) (InterestArea, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return InterestArea{}, errors.New("create interest area: label is required")
	}
	if ordinal < 1 {
		return InterestArea{}, errors.New("create interest area: ordinal must be positive")
	}
	row, err := tx.queries.CreateInterestArea(ctx, db.CreateInterestAreaParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, Label: label, Ordinal: int32(ordinal),
	})
	if err != nil {
		return InterestArea{}, wrapProgramMutationError("create interest area", err)
	}
	return interestArea(row)
}

func (tx *Tx) NextInterestAreaOrdinal(ctx context.Context, schoolYearID, programID ids.XID) (int, error) {
	ordinal, err := tx.queries.ListInterestAreasForProgramOrdinal(ctx, db.ListInterestAreasForProgramOrdinalParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
	})
	if err != nil {
		return 0, fmt.Errorf("next interest area ordinal: %w", err)
	}
	return int(ordinal), nil
}

func (tx *Tx) ListInterestAreas(ctx context.Context, schoolYearID, programID ids.XID, includeRetired bool) ([]InterestArea, error) {
	var rows []db.InterestArea
	var err error
	params := db.ListInterestAreasParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID}
	if includeRetired {
		rows, err = tx.queries.ListAllInterestAreas(ctx, db.ListAllInterestAreasParams(params))
	} else {
		rows, err = tx.queries.ListInterestAreas(ctx, params)
	}
	if err != nil {
		return nil, fmt.Errorf("list interest areas: %w", err)
	}
	result := make([]InterestArea, 0, len(rows))
	for _, row := range rows {
		value, err := interestArea(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetInterestArea(ctx context.Context, schoolYearID, programID, id ids.XID) (InterestArea, error) {
	row, err := tx.queries.GetInterestArea(ctx, db.GetInterestAreaParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return InterestArea{}, fmt.Errorf("get interest area: %w", err)
	}
	return interestArea(row)
}

func (tx *Tx) UpdateInterestArea(ctx context.Context, schoolYearID, programID, id ids.XID, label string) (InterestArea, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return InterestArea{}, errors.New("update interest area: label is required")
	}
	row, err := tx.queries.UpdateInterestArea(ctx, db.UpdateInterestAreaParams{ID: id, Label: label, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return InterestArea{}, wrapProgramMutationError("update interest area", err)
	}
	return interestArea(row)
}

func (tx *Tx) SetInterestAreaRetired(ctx context.Context, schoolYearID, programID, id ids.XID, retired bool) (InterestArea, error) {
	row, err := tx.queries.SetInterestAreaRetired(ctx, db.SetInterestAreaRetiredParams{ID: id, Column2: retired, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return InterestArea{}, wrapProgramMutationError("set interest area retirement", err)
	}
	return interestArea(row)
}

func (tx *Tx) CreateProgramMembership(ctx context.Context, schoolYearID, programID, studentID ids.XID) (ProgramMembership, error) {
	row, err := tx.queries.CreateProgramMembership(ctx, db.CreateProgramMembershipParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, StudentID: studentID,
	})
	if err != nil {
		return ProgramMembership{}, wrapProgramMutationError("create program membership", err)
	}
	return programMembershipFromCreate(row)
}

func (tx *Tx) ListProgramMemberships(ctx context.Context, schoolYearID, programID ids.XID) ([]ProgramMembership, error) {
	rows, err := tx.queries.ListProgramMemberships(ctx, db.ListProgramMembershipsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return nil, fmt.Errorf("list program memberships: %w", err)
	}
	result := make([]ProgramMembership, 0, len(rows))
	for _, row := range rows {
		value, err := programMembership(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) DeleteProgramMembership(ctx context.Context, schoolYearID, programID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteProgramMembership(ctx, db.DeleteProgramMembershipParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return false, wrapProgramMutationError("delete program membership", err)
	}
	return rows == 1, nil
}

func (tx *Tx) CountStudentsWithoutGrade(ctx context.Context, schoolYearID ids.XID) (int64, error) {
	count, err := tx.queries.CountStudentsWithoutGrade(ctx, db.CountStudentsWithoutGradeParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return 0, fmt.Errorf("count students without grade: %w", err)
	}
	return count, nil
}

// The following methods are used only by the Layer 2 isolation registry.
func (tx *Tx) ListAllProgramsForRegistry(ctx context.Context) ([]Program, error) {
	rows, err := tx.queries.ListAllProgramsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list programs for registry: %w", err)
	}
	result := make([]Program, 0, len(rows))
	for _, row := range rows {
		value, err := program(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindProgramForRegistry(ctx context.Context, id ids.XID) (Program, error) {
	row, err := tx.queries.FindProgramForRegistry(ctx, db.FindProgramForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Program{}, nil
		}
		return Program{}, fmt.Errorf("find program for registry: %w", err)
	}
	return program(row)
}

func (tx *Tx) UpdateProgramForRegistry(ctx context.Context, id ids.XID, name string) (bool, error) {
	rows, err := tx.queries.UpdateProgramForRegistry(ctx, db.UpdateProgramForRegistryParams{ID: id, OrganizationID: tx.organizationID, Name: name})
	if err != nil {
		return false, wrapProgramMutationError("update program for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteProgramForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteProgramForRegistry(ctx, db.DeleteProgramForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("delete program for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) ListAllProgramMembershipsForRegistry(ctx context.Context) ([]ProgramMembership, error) {
	rows, err := tx.queries.ListAllProgramMembershipsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list program memberships for registry: %w", err)
	}
	result := make([]ProgramMembership, 0, len(rows))
	for _, row := range rows {
		value, err := programMembershipFromRegistry(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) ListAllInterestAreasForRegistry(ctx context.Context) ([]InterestArea, error) {
	rows, err := tx.queries.ListAllInterestAreasForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list interest areas for registry: %w", err)
	}
	result := make([]InterestArea, 0, len(rows))
	for _, row := range rows {
		value, err := interestArea(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestAreaForRegistry(ctx context.Context, id ids.XID) (InterestArea, error) {
	row, err := tx.queries.FindInterestAreaForRegistry(ctx, db.FindInterestAreaForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InterestArea{}, nil
		}
		return InterestArea{}, fmt.Errorf("find interest area for registry: %w", err)
	}
	return interestArea(row)
}

func (tx *Tx) UpdateInterestAreaForRegistry(ctx context.Context, id ids.XID, label string) (bool, error) {
	rows, err := tx.queries.UpdateInterestAreaForRegistry(ctx, db.UpdateInterestAreaForRegistryParams{ID: id, OrganizationID: tx.organizationID, Label: label})
	if err != nil {
		return false, wrapProgramMutationError("update interest area for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) RetireInterestAreaForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.RetireInterestAreaForRegistry(ctx, db.RetireInterestAreaForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("retire interest area for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) FindProgramMembershipForRegistry(ctx context.Context, id ids.XID) (ProgramMembership, error) {
	row, err := tx.queries.FindProgramMembershipForRegistry(ctx, db.FindProgramMembershipForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProgramMembership{}, nil
		}
		return ProgramMembership{}, fmt.Errorf("find program membership for registry: %w", err)
	}
	return programMembershipFromRegistry(row)
}

func (tx *Tx) TouchProgramMembershipForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.TouchProgramMembershipForRegistry(ctx, db.TouchProgramMembershipForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("update program membership for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteProgramMembershipForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteProgramMembershipForRegistry(ctx, db.DeleteProgramMembershipForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("delete program membership for registry", err)
	}
	return rows == 1, nil
}

func program(row db.Program) (Program, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return Program{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return Program{}, err
	}
	return Program{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, Name: row.Name, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func interestArea(row db.InterestArea) (InterestArea, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestArea{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return InterestArea{}, err
	}
	var retiredAt *time.Time
	if row.RetiredAt.Valid {
		retiredAt = &row.RetiredAt.Time
	}
	return InterestArea{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, Label: row.Label, Ordinal: int(row.Ordinal), RetiredAt: retiredAt, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func programMembershipFromCreate(row db.ProgramMembership) (ProgramMembership, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return ProgramMembership{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return ProgramMembership{}, err
	}
	return ProgramMembership{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, StudentID: row.StudentID, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func programMembership(row db.ListProgramMembershipsRow) (ProgramMembership, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return ProgramMembership{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return ProgramMembership{}, err
	}
	return ProgramMembership{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, StudentID: row.StudentID, GradeLevelID: row.GradeLevelID, LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName, GradeMissing: row.GradeLevelID == nil, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func programMembershipFromRegistry(row db.ProgramMembership) (ProgramMembership, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return ProgramMembership{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return ProgramMembership{}, err
	}
	return ProgramMembership{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, StudentID: row.StudentID, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func programTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("program row: %s is null", name)
	}
	return value.Time, nil
}

func wrapProgramMutationError(operation string, err error) error {
	if isClosedYearDatabaseError(err) {
		return fmt.Errorf("%w: %v", ErrSchoolYearClosed, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
