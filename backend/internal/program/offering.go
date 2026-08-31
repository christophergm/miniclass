package program

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

var (
	ErrOfferingNoChanges  = errors.New("offering update has no changes")
	ErrOfferingGradeOrder = errors.New("offering grade window minimum must not be above maximum")
)

type OfferingUpdate struct {
	Name                    *string
	Description             *string
	MinimumViableEnrollment *int
	Capacity                *int
	MinGradeLevelID         *ids.XID
	MaxGradeLevelID         *ids.XID
	Location                *string
	MeetingPoint            *string
	MeetingInstructions     *string
	InterestAreaID          *ids.XID
}

func (s *Service) CreateOffering(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, name, description string, minimumViableEnrollment *int, capacity int, minGradeLevelID, maxGradeLevelID ids.XID, location, meetingPoint, meetingInstructions string, interestAreaID *ids.XID) (data.Offering, error) {
	if s == nil || s.database == nil {
		return data.Offering{}, errors.New("create offering: data service is nil")
	}
	var result data.Offering
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if err := validateOfferingReferences(ctx, tx, schoolYearID, programID, sessionID, minGradeLevelID, maxGradeLevelID, interestAreaID); err != nil {
			return err
		}
		created, err := tx.CreateOffering(ctx, schoolYearID, programID, sessionID, name, description, minimumViableEnrollment, capacity, minGradeLevelID, maxGradeLevelID, location, meetingPoint, meetingInstructions, interestAreaID)
		if err != nil {
			return err
		}
		result = created
		return recordOfferingChange(ctx, tx, nil, created)
	})
	if err != nil {
		return data.Offering{}, fmt.Errorf("create offering: %w", err)
	}
	return result, nil
}

func (s *Service) ListOfferings(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) ([]data.Offering, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list offerings: data service is nil")
	}
	var result []data.Offering
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list offerings: %w", err)
	}
	return result, nil
}

func (s *Service) GetOffering(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, offeringID ids.XID) (data.Offering, error) {
	if s == nil || s.database == nil {
		return data.Offering{}, errors.New("get offering: data service is nil")
	}
	var result data.Offering
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetOffering(ctx, schoolYearID, programID, sessionID, offeringID)
		return err
	})
	if err != nil {
		return data.Offering{}, fmt.Errorf("get offering: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateOffering(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, offeringID ids.XID, input OfferingUpdate) (data.Offering, error) {
	if s == nil || s.database == nil {
		return data.Offering{}, errors.New("update offering: data service is nil")
	}
	var result data.Offering
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetOffering(ctx, schoolYearID, programID, sessionID, offeringID)
		if err != nil {
			return err
		}
		name, description := current.Name, current.Description
		minimum, capacity := current.MinimumViableEnrollment, current.Capacity
		minGrade, maxGrade := current.MinGradeLevelID, current.MaxGradeLevelID
		location, meetingPoint, meetingInstructions := current.Location, current.MeetingPoint, current.MeetingInstructions
		interestArea := current.InterestAreaID
		changed := false
		if input.Name != nil && strings.TrimSpace(*input.Name) != current.Name {
			name, changed = *input.Name, true
		}
		if input.Description != nil && *input.Description != current.Description {
			description, changed = *input.Description, true
		}
		if input.MinimumViableEnrollment != nil && !sameOfferingInt(minimum, input.MinimumViableEnrollment) {
			minimum, changed = input.MinimumViableEnrollment, true
		}
		if input.Capacity != nil && *input.Capacity != current.Capacity {
			capacity, changed = *input.Capacity, true
		}
		if input.MinGradeLevelID != nil && *input.MinGradeLevelID != current.MinGradeLevelID {
			minGrade, changed = *input.MinGradeLevelID, true
		}
		if input.MaxGradeLevelID != nil && *input.MaxGradeLevelID != current.MaxGradeLevelID {
			maxGrade, changed = *input.MaxGradeLevelID, true
		}
		if input.Location != nil && *input.Location != current.Location {
			location, changed = *input.Location, true
		}
		if input.MeetingPoint != nil && *input.MeetingPoint != current.MeetingPoint {
			meetingPoint, changed = *input.MeetingPoint, true
		}
		if input.MeetingInstructions != nil && *input.MeetingInstructions != current.MeetingInstructions {
			meetingInstructions, changed = *input.MeetingInstructions, true
		}
		if input.InterestAreaID != nil && !sameOfferingID(interestArea, input.InterestAreaID) {
			interestArea, changed = input.InterestAreaID, true
		}
		if !changed {
			return ErrOfferingNoChanges
		}
		if err := validateOfferingReferences(ctx, tx, schoolYearID, programID, sessionID, minGrade, maxGrade, interestArea); err != nil {
			return err
		}
		result, err = tx.UpdateOffering(ctx, schoolYearID, programID, sessionID, offeringID, name, description, minimum, capacity, minGrade, maxGrade, location, meetingPoint, meetingInstructions, interestArea)
		if err != nil {
			return err
		}
		return recordOfferingChange(ctx, tx, &current, result)
	})
	if err != nil {
		return data.Offering{}, fmt.Errorf("update offering: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteOffering(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, offeringID ids.XID) error {
	if s == nil || s.database == nil {
		return errors.New("delete offering: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetOffering(ctx, schoolYearID, programID, sessionID, offeringID)
		if err != nil {
			return err
		}
		deleted, err := tx.DeleteOffering(ctx, schoolYearID, programID, sessionID, offeringID)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("offering not found")
		}
		return recordOfferingChange(ctx, tx, &current, data.Offering{})
	})
	if err != nil {
		return fmt.Errorf("delete offering: %w", err)
	}
	return nil
}

func validateOfferingReferences(ctx context.Context, tx *data.Tx, schoolYearID, programID, sessionID, minGradeLevelID, maxGradeLevelID ids.XID, interestAreaID *ids.XID) error {
	if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
		return err
	}
	minimum, err := tx.GetGradeLevelByID(ctx, schoolYearID, minGradeLevelID)
	if err != nil {
		return err
	}
	maximum, err := tx.GetGradeLevelByID(ctx, schoolYearID, maxGradeLevelID)
	if err != nil {
		return err
	}
	if minimum.Ordinal > maximum.Ordinal {
		return ErrOfferingGradeOrder
	}
	if interestAreaID != nil {
		if _, err := tx.GetInterestArea(ctx, schoolYearID, programID, *interestAreaID); err != nil {
			return err
		}
	}
	return nil
}

func recordOfferingChange(ctx context.Context, tx *data.Tx, before *data.Offering, after data.Offering) error {
	id, year := after.ID, after.SchoolYearID
	if before != nil {
		id, year = before.ID, before.SchoolYearID
	}
	return tx.Record(ctx, audit.Entry{Action: audit.ActionOfferingEdit, ObjectType: "offering", ObjectID: &id, SchoolYearID: &year, ChangeSummary: offeringSummary(before, after)})
}

func offeringSummary(before *data.Offering, after data.Offering) json.RawMessage {
	value := map[string]any{}
	if after.ID == "" {
		value["deleted"] = true
	} else {
		value["after"] = map[string]any{"name": after.Name, "description": after.Description, "minimum_viable_enrollment": after.MinimumViableEnrollment, "capacity": after.Capacity, "min_grade_level_id": after.MinGradeLevelID, "max_grade_level_id": after.MaxGradeLevelID, "location": after.Location, "meeting_point": after.MeetingPoint, "meeting_instructions": after.MeetingInstructions, "interest_area_id": after.InterestAreaID}
	}
	if before != nil {
		value["before"] = map[string]any{"name": before.Name, "description": before.Description, "minimum_viable_enrollment": before.MinimumViableEnrollment, "capacity": before.Capacity, "min_grade_level_id": before.MinGradeLevelID, "max_grade_level_id": before.MaxGradeLevelID, "location": before.Location, "meeting_point": before.MeetingPoint, "meeting_instructions": before.MeetingInstructions, "interest_area_id": before.InterestAreaID}
	}
	encoded, _ := json.Marshal(value)
	return encoded
}

func sameOfferingInt(current *int, next *int) bool {
	if current == nil || next == nil {
		return current == nil && next == nil
	}
	return *current == *next
}

func sameOfferingID(current, next *ids.XID) bool {
	if current == nil || next == nil {
		return current == nil && next == nil
	}
	return *current == *next
}
