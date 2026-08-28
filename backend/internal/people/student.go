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

var ErrStudentNoChanges = errors.New("student update has no changes")

type StudentCreateInput struct {
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	GradeLevelID       ids.XID
	HomeroomID         ids.XID
	ExternalIdentifier *string
	PriorYearStudentID *ids.XID
}

type StudentUpdateInput struct {
	LegalGivenName     *string
	LegalFamilyName    *string
	PreferredGivenName **string
	GradeLevelID       *ids.XID
	HomeroomID         *ids.XID
	ExternalIdentifier **string
	PriorYearStudentID **ids.XID
}

// CreateStudent creates a year-scoped student and records the roster change.
func (s *Service) CreateStudent(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, input StudentCreateInput) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("create student: data service is nil")
	}
	var result data.Student
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		created, err := tx.CreateStudent(ctx, schoolYearID, input.GradeLevelID, input.HomeroomID, input.LegalGivenName, input.LegalFamilyName, input.PreferredGivenName, input.ExternalIdentifier, input.PriorYearStudentID)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, ChangeSummary: studentSummary(nil, &created)})
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("create student: %w", err)
	}
	return result, nil
}

// ListStudents returns students for one school year. Deleted rows are opt-in.
func (s *Service) ListStudents(ctx context.Context, organizationID string, schoolYearID ids.XID, includeDeleted bool) ([]data.Student, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list students: data service is nil")
	}
	var result []data.Student
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListStudents(ctx, schoolYearID, includeDeleted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	return result, nil
}

func (s *Service) RestoreStudent(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, reason string) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("restore student: data service is nil")
	}
	reason, err := restoreReason(reason)
	if err != nil {
		return data.Student{}, err
	}
	var result data.Student
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByIDIncludingDeleted(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if current.DeletedAt == nil {
			return ErrRestoreNotDeleted
		}
		restored, err := tx.RestoreStudent(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		result = restored
		year := restored.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionRestore, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: studentSummary(&current, &restored)})
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("restore student: %w", err)
	}
	return result, nil
}

// GetStudent returns one active student in the requested year.
func (s *Service) GetStudent(ctx context.Context, organizationID string, schoolYearID, id ids.XID) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("get student: data service is nil")
	}
	var result data.Student
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetStudentByID(ctx, schoolYearID, id)
		return err
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("get student: %w", err)
	}
	return result, nil
}

// UpdateStudent edits an active student and records the before/after values.
func (s *Service) UpdateStudent(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input StudentUpdateInput) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("update student: data service is nil")
	}
	var result data.Student
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		updatedInput, changed := applyStudentUpdate(current, input)
		if !changed {
			return ErrStudentNoChanges
		}
		updated, err := tx.UpdateStudent(ctx, schoolYearID, id, updatedInput.LegalGivenName, updatedInput.LegalFamilyName, updatedInput.PreferredGivenName, updatedInput.GradeLevelID, updatedInput.HomeroomID, updatedInput.ExternalIdentifier, updatedInput.PriorYearStudentID)
		if err != nil {
			return err
		}
		result = updated
		year := updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, ChangeSummary: studentSummary(&current, &updated)})
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("update student: %w", err)
	}
	return result, nil
}

// DeleteStudent soft-deletes an active student and records the change.
func (s *Service) DeleteStudent(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("delete student: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		deleted, err := tx.SoftDeleteStudent(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		year := current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSoftDelete, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, ChangeSummary: studentSummary(&current, nil)})
	})
	if err != nil {
		return fmt.Errorf("delete student: %w", err)
	}
	return nil
}

type studentUpdateValues struct {
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	GradeLevelID       ids.XID
	HomeroomID         ids.XID
	ExternalIdentifier *string
	PriorYearStudentID *ids.XID
}

func applyStudentUpdate(current data.Student, input StudentUpdateInput) (studentUpdateValues, bool) {
	result := studentUpdateValues{
		LegalGivenName: current.LegalGivenName, LegalFamilyName: current.LegalFamilyName,
		PreferredGivenName: current.PreferredGivenName, GradeLevelID: current.GradeLevelID,
		HomeroomID: current.HomeroomID, ExternalIdentifier: current.ExternalIdentifier,
		PriorYearStudentID: current.PriorYearStudentID,
	}
	changed := false
	if input.LegalGivenName != nil && strings.TrimSpace(*input.LegalGivenName) != current.LegalGivenName {
		result.LegalGivenName, changed = *input.LegalGivenName, true
	}
	if input.LegalFamilyName != nil && strings.TrimSpace(*input.LegalFamilyName) != current.LegalFamilyName {
		result.LegalFamilyName, changed = *input.LegalFamilyName, true
	}
	if input.PreferredGivenName != nil && !sameStudentOptionalString(result.PreferredGivenName, *input.PreferredGivenName) {
		result.PreferredGivenName, changed = *input.PreferredGivenName, true
	}
	if input.GradeLevelID != nil && *input.GradeLevelID != result.GradeLevelID {
		result.GradeLevelID, changed = *input.GradeLevelID, true
	}
	if input.HomeroomID != nil && *input.HomeroomID != result.HomeroomID {
		result.HomeroomID, changed = *input.HomeroomID, true
	}
	if input.ExternalIdentifier != nil && !sameStudentOptionalString(result.ExternalIdentifier, *input.ExternalIdentifier) {
		result.ExternalIdentifier, changed = *input.ExternalIdentifier, true
	}
	if input.PriorYearStudentID != nil && !sameStudentOptionalID(result.PriorYearStudentID, *input.PriorYearStudentID) {
		result.PriorYearStudentID, changed = *input.PriorYearStudentID, true
	}
	return result, changed
}

func sameStudentOptionalString(current, next *string) bool {
	currentValue, nextValue := "", ""
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}
	if next != nil {
		nextValue = strings.TrimSpace(*next)
	}
	return currentValue == nextValue
}

func sameStudentOptionalID(current, next *ids.XID) bool {
	if current == nil || next == nil {
		return current == nil && next == nil
	}
	return *current == *next
}

func studentSummary(before, after *data.Student) json.RawMessage {
	value := map[string]any{}
	if before != nil {
		value["before"] = studentSummaryFields(before)
	}
	if after != nil {
		value["after"] = studentSummaryFields(after)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode student audit summary"}`)
	}
	return encoded
}

func studentSummaryFields(student *data.Student) map[string]any {
	return map[string]any{
		"legal_given_name": student.LegalGivenName, "legal_family_name": student.LegalFamilyName,
		"preferred_given_name": student.PreferredGivenName, "grade_level_id": student.GradeLevelID,
		"homeroom_id": student.HomeroomID, "external_identifier": student.ExternalIdentifier,
		"prior_year_student_id": student.PriorYearStudentID,
	}
}
