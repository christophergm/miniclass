package people

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

type GuardianRelationshipCreateInput struct {
	AdultID          ids.XID
	StudentID        ids.XID
	RelationshipType data.GuardianRelationshipType
}

type GuardianRelationshipUpdateInput struct {
	RelationshipType *data.GuardianRelationshipType
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
