package people

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var ErrHouseholdNoChanges = errors.New("household update has no changes")

type HouseholdCreateInput struct{ DisplayName string }

type HouseholdUpdateInput struct{ DisplayName *string }

type GuardianRelationshipCreateInput struct {
	AdultID          ids.XID
	StudentID        ids.XID
	RelationshipType data.GuardianRelationshipType
}

type GuardianRelationshipUpdateInput struct {
	RelationshipType *data.GuardianRelationshipType
}

func (s *Service) CreateHousehold(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, input HouseholdCreateInput) (data.Household, error) {
	if s == nil || s.database == nil {
		return data.Household{}, errors.New("create household: data service is nil")
	}
	var result data.Household
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		created, err := tx.CreateHousehold(ctx, schoolYearID, input.DisplayName)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "household", ObjectID: &id, SchoolYearID: &year, ChangeSummary: householdSummary(nil, &created)})
	})
	if err != nil {
		return data.Household{}, fmt.Errorf("create household: %w", err)
	}
	return result, nil
}

func (s *Service) ListHouseholds(ctx context.Context, organizationID string, schoolYearID ids.XID, includeDeleted bool) ([]data.Household, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list households: data service is nil")
	}
	var result []data.Household
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListHouseholds(ctx, schoolYearID, includeDeleted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list households: %w", err)
	}
	return result, nil
}

func (s *Service) RestoreHousehold(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, reason string) (data.Household, error) {
	if s == nil || s.database == nil {
		return data.Household{}, errors.New("restore household: data service is nil")
	}
	reason, err := restoreReason(reason)
	if err != nil {
		return data.Household{}, err
	}
	var result data.Household
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetHouseholdByIDIncludingDeleted(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if current.DeletedAt == nil {
			return ErrRestoreNotDeleted
		}
		restored, err := tx.RestoreHousehold(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		result = restored
		year := restored.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionRestore, ObjectType: "household", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: householdSummary(&current, &restored)})
	})
	if err != nil {
		return data.Household{}, fmt.Errorf("restore household: %w", err)
	}
	return result, nil
}

func (s *Service) GetHousehold(ctx context.Context, organizationID string, schoolYearID, id ids.XID) (data.Household, error) {
	if s == nil || s.database == nil {
		return data.Household{}, errors.New("get household: data service is nil")
	}
	var result data.Household
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetHouseholdByID(ctx, schoolYearID, id)
		return err
	})
	if err != nil {
		return data.Household{}, fmt.Errorf("get household: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateHousehold(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input HouseholdUpdateInput) (data.Household, error) {
	if s == nil || s.database == nil {
		return data.Household{}, errors.New("update household: data service is nil")
	}
	var result data.Household
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetHouseholdByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if input.DisplayName == nil {
			return ErrHouseholdNoChanges
		}
		name := strings.TrimSpace(*input.DisplayName)
		if name == current.DisplayName {
			return ErrHouseholdNoChanges
		}
		updated, err := tx.UpdateHousehold(ctx, schoolYearID, id, name)
		if err != nil {
			return err
		}
		result = updated
		year := updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "household", ObjectID: &id, SchoolYearID: &year, ChangeSummary: householdSummary(&current, &updated)})
	})
	if err != nil {
		return data.Household{}, fmt.Errorf("update household: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteHousehold(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("delete household: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetHouseholdByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		deleted, err := tx.SoftDeleteHousehold(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		year := current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSoftDelete, ObjectType: "household", ObjectID: &id, SchoolYearID: &year, ChangeSummary: householdSummary(&current, nil)})
	})
	if err != nil {
		return fmt.Errorf("delete household: %w", err)
	}
	return nil
}

func (s *Service) ListHouseholdStudents(ctx context.Context, organizationID string, schoolYearID, householdID ids.XID) ([]data.HouseholdStudent, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list household students: data service is nil")
	}
	var result []data.HouseholdStudent
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetHouseholdByID(ctx, schoolYearID, householdID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListHouseholdStudents(ctx, schoolYearID, householdID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list household students: %w", err)
	}
	return result, nil
}

// HouseholdMembership is a whole school year's membership rows. It is one
// answer rather than two listings because every caller that needs one needs the
// other: a roster surface indexes both to name a person's households, and the
// household list derives both member counts from it.
type HouseholdMembership struct {
	Students []data.HouseholdStudent
	Adults   []data.HouseholdAdult
}

// ListHouseholdMembership reads the year's membership in a bounded number of
// queries. Reading it per household instead cost one request per household in
// the year on every roster surface.
func (s *Service) ListHouseholdMembership(ctx context.Context, organizationID string, schoolYearID ids.XID) (HouseholdMembership, error) {
	if s == nil || s.database == nil {
		return HouseholdMembership{}, errors.New("list household membership: data service is nil")
	}
	var result HouseholdMembership
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		students, err := tx.ListHouseholdStudentsForSchoolYear(ctx, schoolYearID)
		if err != nil {
			return err
		}
		adults, err := tx.ListHouseholdAdultsForSchoolYear(ctx, schoolYearID)
		if err != nil {
			return err
		}
		result = HouseholdMembership{Students: students, Adults: adults}
		return nil
	})
	if err != nil {
		return HouseholdMembership{}, fmt.Errorf("list household membership: %w", err)
	}
	return result, nil
}

func (s *Service) AddStudentToHousehold(ctx context.Context, organizationID string, schoolYearID, householdID, studentID ids.XID, actor audit.Actor) (data.HouseholdStudent, error) {
	if s == nil || s.database == nil {
		return data.HouseholdStudent{}, errors.New("add household student: data service is nil")
	}
	var result data.HouseholdStudent
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetHouseholdByID(ctx, schoolYearID, householdID); err != nil {
			return err
		}
		if _, err := tx.GetStudentByID(ctx, schoolYearID, studentID); err != nil {
			return err
		}
		created, err := tx.CreateHouseholdStudent(ctx, schoolYearID, householdID, studentID)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "household_student", ObjectID: &id, SchoolYearID: &year, ChangeSummary: membershipSummary("student", householdID, studentID, true)})
	})
	if err != nil {
		return data.HouseholdStudent{}, fmt.Errorf("add household student: %w", err)
	}
	return result, nil
}

func (s *Service) RemoveStudentFromHousehold(ctx context.Context, organizationID string, schoolYearID, householdID, studentID ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("remove household student: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetHouseholdByID(ctx, schoolYearID, householdID); err != nil {
			return fmt.Errorf("find household: %w", err)
		}
		// Looked up directly rather than scanned out of the household's listing:
		// that listing now hides a soft-deleted student (SPEC §21.3), and a stale
		// membership must stay removable.
		membership, err := tx.GetHouseholdStudent(ctx, schoolYearID, householdID, studentID)
		if err != nil {
			return fmt.Errorf("find household student membership for student %s: %w", studentID, err)
		}
		deleted, err := tx.DeleteHouseholdStudent(ctx, membership.SchoolYearID, membership.HouseholdID, membership.StudentID)
		if err != nil {
			return fmt.Errorf("delete household student membership %s: %w", membership.ID, err)
		}
		if !deleted {
			return fmt.Errorf("delete household student membership %s: %w", membership.ID, pgx.ErrNoRows)
		}
		id, year := membership.ID, membership.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionHardDelete, ObjectType: "household_student", ObjectID: &id, SchoolYearID: &year, ChangeSummary: membershipSummary("student", householdID, studentID, false)})
	})
	if err != nil {
		return fmt.Errorf("remove household student: %w", err)
	}
	return nil
}

func (s *Service) ListHouseholdAdults(ctx context.Context, organizationID string, schoolYearID, householdID ids.XID) ([]data.HouseholdAdult, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list household adults: data service is nil")
	}
	var result []data.HouseholdAdult
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetHouseholdByID(ctx, schoolYearID, householdID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListHouseholdAdults(ctx, schoolYearID, householdID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list household adults: %w", err)
	}
	return result, nil
}

func (s *Service) AddAdultToHousehold(ctx context.Context, organizationID string, schoolYearID, householdID, adultID ids.XID, actor audit.Actor) (data.HouseholdAdult, error) {
	if s == nil || s.database == nil {
		return data.HouseholdAdult{}, errors.New("add household adult: data service is nil")
	}
	var result data.HouseholdAdult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetHouseholdByID(ctx, schoolYearID, householdID); err != nil {
			return err
		}
		if _, err := tx.GetAdultByID(ctx, schoolYearID, adultID); err != nil {
			return err
		}
		created, err := tx.CreateHouseholdAdult(ctx, schoolYearID, householdID, adultID)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "household_adult", ObjectID: &id, SchoolYearID: &year, ChangeSummary: membershipSummary("adult", householdID, adultID, true)})
	})
	if err != nil {
		return data.HouseholdAdult{}, fmt.Errorf("add household adult: %w", err)
	}
	return result, nil
}

func (s *Service) RemoveAdultFromHousehold(ctx context.Context, organizationID string, schoolYearID, householdID, adultID ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("remove household adult: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetHouseholdByID(ctx, schoolYearID, householdID); err != nil {
			return fmt.Errorf("find household: %w", err)
		}
		membership, err := tx.GetHouseholdAdult(ctx, schoolYearID, householdID, adultID)
		if err != nil {
			return fmt.Errorf("find household adult membership for adult %s: %w", adultID, err)
		}
		deleted, err := tx.DeleteHouseholdAdult(ctx, membership.SchoolYearID, membership.HouseholdID, membership.AdultID)
		if err != nil {
			return fmt.Errorf("delete household adult membership %s: %w", membership.ID, err)
		}
		if !deleted {
			return fmt.Errorf("delete household adult membership %s: %w", membership.ID, pgx.ErrNoRows)
		}
		id, year := membership.ID, membership.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionHardDelete, ObjectType: "household_adult", ObjectID: &id, SchoolYearID: &year, ChangeSummary: membershipSummary("adult", householdID, adultID, false)})
	})
	if err != nil {
		return fmt.Errorf("remove household adult: %w", err)
	}
	return nil
}

func (s *Service) ListGuardianRelationships(ctx context.Context, organizationID string, schoolYearID ids.XID, filter data.GuardianRelationshipFilter) ([]data.GuardianRelationship, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list guardian relationships: data service is nil")
	}
	var result []data.GuardianRelationship
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListGuardianRelationships(ctx, schoolYearID, filter)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list guardian relationships: %w", err)
	}
	return result, nil
}

func (s *Service) CreateGuardianRelationship(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, input GuardianRelationshipCreateInput) (data.GuardianRelationship, error) {
	if s == nil || s.database == nil {
		return data.GuardianRelationship{}, errors.New("create guardian relationship: data service is nil")
	}
	var result data.GuardianRelationship
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetAdultByID(ctx, schoolYearID, input.AdultID); err != nil {
			return err
		}
		if _, err := tx.GetStudentByID(ctx, schoolYearID, input.StudentID); err != nil {
			return err
		}
		created, err := tx.CreateGuardianRelationship(ctx, schoolYearID, input.AdultID, input.StudentID, input.RelationshipType)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "guardian_relationship", ObjectID: &id, SchoolYearID: &year, ChangeSummary: guardianRelationshipSummary(nil, &created)})
	})
	if err != nil {
		return data.GuardianRelationship{}, fmt.Errorf("create guardian relationship: %w", err)
	}
	return result, nil
}

func (s *Service) GetGuardianRelationship(ctx context.Context, organizationID string, schoolYearID, id ids.XID) (data.GuardianRelationship, error) {
	if s == nil || s.database == nil {
		return data.GuardianRelationship{}, errors.New("get guardian relationship: data service is nil")
	}
	var result data.GuardianRelationship
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetGuardianRelationshipByID(ctx, schoolYearID, id)
		return err
	})
	if err != nil {
		return data.GuardianRelationship{}, fmt.Errorf("get guardian relationship: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateGuardianRelationship(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input GuardianRelationshipUpdateInput) (data.GuardianRelationship, error) {
	if s == nil || s.database == nil {
		return data.GuardianRelationship{}, errors.New("update guardian relationship: data service is nil")
	}
	var result data.GuardianRelationship
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetGuardianRelationshipByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if input.RelationshipType == nil || *input.RelationshipType == current.RelationshipType {
			return ErrGuardianRelationshipNoChanges
		}
		updated, err := tx.UpdateGuardianRelationship(ctx, schoolYearID, id, *input.RelationshipType)
		if err != nil {
			return err
		}
		result = updated
		year := updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "guardian_relationship", ObjectID: &id, SchoolYearID: &year, ChangeSummary: guardianRelationshipSummary(&current, &updated)})
	})
	if err != nil {
		return data.GuardianRelationship{}, fmt.Errorf("update guardian relationship: %w", err)
	}
	return result, nil
}

var ErrGuardianRelationshipNoChanges = errors.New("guardian relationship update has no changes")

func (s *Service) DeleteGuardianRelationship(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("delete guardian relationship: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetGuardianRelationshipByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		deleted, err := tx.DeleteGuardianRelationship(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		year := current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionHardDelete, ObjectType: "guardian_relationship", ObjectID: &id, SchoolYearID: &year, ChangeSummary: guardianRelationshipSummary(&current, nil)})
	})
	if err != nil {
		return fmt.Errorf("delete guardian relationship: %w", err)
	}
	return nil
}

func householdSummary(before, after *data.Household) json.RawMessage {
	value := map[string]any{}
	if before != nil {
		value["before"] = map[string]any{"display_name": before.DisplayName}
	}
	if after != nil {
		value["after"] = map[string]any{"display_name": after.DisplayName}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode household audit summary"}`)
	}
	return encoded
}

func membershipSummary(kind string, householdID, memberID ids.XID, present bool) json.RawMessage {
	encoded, err := json.Marshal(map[string]any{"member_type": kind, "household_id": householdID, "member_id": memberID, "present": present})
	if err != nil {
		return json.RawMessage(`{"error":"could not encode membership audit summary"}`)
	}
	return encoded
}

func guardianRelationshipSummary(before, after *data.GuardianRelationship) json.RawMessage {
	value := map[string]any{}
	if before != nil {
		value["before"] = map[string]any{"adult_id": before.AdultID, "student_id": before.StudentID, "relationship_type": before.RelationshipType}
	}
	if after != nil {
		value["after"] = map[string]any{"adult_id": after.AdultID, "student_id": after.StudentID, "relationship_type": after.RelationshipType}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode guardian relationship audit summary"}`)
	}
	return encoded
}
