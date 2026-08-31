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

// Offering is one class in one programme session. Its grade references retain
// the school-year vocabulary identity rather than copying labels into the
// catalog.
type Offering struct {
	ID                      ids.XID
	OrganizationID          ids.XID
	SchoolYearID            ids.XID
	ProgramID               ids.XID
	SessionID               ids.XID
	Name                    string
	Description             string
	MinimumViableEnrollment *int
	Capacity                int
	MinGradeLevelID         ids.XID
	MaxGradeLevelID         ids.XID
	Location                string
	MeetingPoint            string
	MeetingInstructions     string
	InterestAreaID          *ids.XID
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// CreateOffering writes a fully validated row within the caller's tenant
// transaction. Parent and vocabulary checks remain explicit so a foreign
// parent cannot be smuggled through a composite identifier.
func (tx *Tx) CreateOffering(ctx context.Context, schoolYearID, programID, sessionID ids.XID, name, description string, minimumViableEnrollment *int, capacity int, minGradeLevelID, maxGradeLevelID ids.XID, location, meetingPoint, meetingInstructions string, interestAreaID *ids.XID) (Offering, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Offering{}, errors.New("create offering: name is required")
	}
	if capacity < 1 {
		return Offering{}, errors.New("create offering: capacity must be positive")
	}
	if minimumViableEnrollment != nil && (*minimumViableEnrollment < 0 || *minimumViableEnrollment > capacity) {
		return Offering{}, errors.New("create offering: minimum viable enrollment must be between zero and capacity")
	}
	row, err := tx.queries.CreateOffering(ctx, db.CreateOfferingParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
		Name: name, Description: description, MinimumViableEnrollment: nullableOfferingInt(minimumViableEnrollment), Capacity: int32(capacity),
		MinGradeLevelID: minGradeLevelID, MaxGradeLevelID: maxGradeLevelID, Location: location, MeetingPoint: meetingPoint,
		MeetingInstructions: meetingInstructions, InterestAreaID: interestAreaID,
	})
	if err != nil {
		return Offering{}, wrapProgramMutationError("create offering", err)
	}
	return offering(row)
}

func (tx *Tx) ListOfferings(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]Offering, error) {
	rows, err := tx.queries.ListOfferings(ctx, db.ListOfferingsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list offerings: %w", err)
	}
	result := make([]Offering, 0, len(rows))
	for _, row := range rows {
		value, err := offering(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetOffering(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (Offering, error) {
	row, err := tx.queries.GetOffering(ctx, db.GetOfferingParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return Offering{}, fmt.Errorf("get offering: %w", err)
	}
	return offering(row)
}

func (tx *Tx) UpdateOffering(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID, name, description string, minimumViableEnrollment *int, capacity int, minGradeLevelID, maxGradeLevelID ids.XID, location, meetingPoint, meetingInstructions string, interestAreaID *ids.XID) (Offering, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Offering{}, errors.New("update offering: name is required")
	}
	if capacity < 1 {
		return Offering{}, errors.New("update offering: capacity must be positive")
	}
	if minimumViableEnrollment != nil && (*minimumViableEnrollment < 0 || *minimumViableEnrollment > capacity) {
		return Offering{}, errors.New("update offering: minimum viable enrollment must be between zero and capacity")
	}
	row, err := tx.queries.UpdateOffering(ctx, db.UpdateOfferingParams{
		ID: id, Name: name, Description: description, MinimumViableEnrollment: nullableOfferingInt(minimumViableEnrollment), Capacity: int32(capacity),
		MinGradeLevelID: minGradeLevelID, MaxGradeLevelID: maxGradeLevelID, Location: location, MeetingPoint: meetingPoint,
		MeetingInstructions: meetingInstructions, InterestAreaID: interestAreaID, OrganizationID: tx.organizationID,
		SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
	})
	if err != nil {
		return Offering{}, wrapProgramMutationError("update offering", err)
	}
	return offering(row)
}

func (tx *Tx) DeleteOffering(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteOffering(ctx, db.DeleteOfferingParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return false, wrapProgramMutationError("delete offering", err)
	}
	return rows == 1, nil
}

// The following methods are used only by the Layer 2 isolation registry.
func (tx *Tx) ListAllOfferingsForRegistry(ctx context.Context) ([]Offering, error) {
	rows, err := tx.queries.ListAllOfferingsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list offerings for registry: %w", err)
	}
	result := make([]Offering, 0, len(rows))
	for _, row := range rows {
		value, err := offering(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindOfferingForRegistry(ctx context.Context, id ids.XID) (Offering, error) {
	row, err := tx.queries.FindOfferingForRegistry(ctx, db.FindOfferingForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Offering{}, nil
		}
		return Offering{}, fmt.Errorf("find offering for registry: %w", err)
	}
	return offering(row)
}

func (tx *Tx) TouchOfferingForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.TouchOfferingForRegistry(ctx, db.TouchOfferingForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("update offering for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteOfferingForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteOfferingForRegistry(ctx, db.DeleteOfferingForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("delete offering for registry", err)
	}
	return rows == 1, nil
}

func nullableOfferingInt(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func offering(row db.Offering) (Offering, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return Offering{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return Offering{}, err
	}
	var minimum *int
	if row.MinimumViableEnrollment.Valid {
		value := int(row.MinimumViableEnrollment.Int32)
		minimum = &value
	}
	return Offering{
		ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID,
		Name: row.Name, Description: row.Description, MinimumViableEnrollment: minimum, Capacity: int(row.Capacity),
		MinGradeLevelID: row.MinGradeLevelID, MaxGradeLevelID: row.MaxGradeLevelID, Location: row.Location,
		MeetingPoint: row.MeetingPoint, MeetingInstructions: row.MeetingInstructions, InterestAreaID: row.InterestAreaID,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}
