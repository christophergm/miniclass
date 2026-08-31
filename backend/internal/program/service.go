// Package program owns annual programme membership and its known-grade gate.
package program

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

var ErrStudentGradeRequired = errors.New("student grade is required for programme membership")

type Service struct{ database *data.DB }

func New(database *data.DB) *Service { return &Service{database: database} }

func (s *Service) Create(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID ids.XID, name string) (data.Program, error) {
	if s == nil || s.database == nil {
		return data.Program{}, errors.New("create program: data service is nil")
	}
	var result data.Program
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		created, err := tx.CreateProgram(ctx, schoolYearID, name)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionProgramCreate, ObjectType: "program", ObjectID: &id, SchoolYearID: &year, ChangeSummary: programSummary(created)})
	})
	if err != nil {
		return data.Program{}, fmt.Errorf("create program: %w", err)
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, organizationID string, schoolYearID ids.XID) ([]data.Program, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list programs: data service is nil")
	}
	var result []data.Program
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListPrograms(ctx, schoolYearID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	return result, nil
}

func (s *Service) ListMemberships(ctx context.Context, organizationID string, schoolYearID, programID ids.XID) ([]data.ProgramMembership, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list program memberships: data service is nil")
	}
	var result []data.ProgramMembership
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListProgramMemberships(ctx, schoolYearID, programID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list program memberships: %w", err)
	}
	return result, nil
}

func (s *Service) AddMembership(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, studentID ids.XID) (data.ProgramMembership, error) {
	if s == nil || s.database == nil {
		return data.ProgramMembership{}, errors.New("create program membership: data service is nil")
	}
	var result data.ProgramMembership
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
		if err != nil {
			return err
		}
		if student.GradeLevelID == nil {
			return fmt.Errorf("%w: %s %s has no known grade", ErrStudentGradeRequired, student.LegalGivenName, student.LegalFamilyName)
		}
		created, err := tx.CreateProgramMembership(ctx, schoolYearID, programID, studentID)
		if err != nil {
			return err
		}
		memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		for _, membership := range memberships {
			if membership.ID == created.ID {
				result = membership
				break
			}
		}
		if result.ID == "" {
			return pgx.ErrNoRows
		}
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "program_membership", ObjectID: &id, SchoolYearID: &year, ChangeSummary: membershipSummary(created)})
	})
	if err != nil {
		return data.ProgramMembership{}, fmt.Errorf("create program membership: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteMembership(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, membershipID ids.XID) error {
	if s == nil || s.database == nil {
		return errors.New("delete program membership: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		deleted, err := tx.DeleteProgramMembership(ctx, schoolYearID, programID, membershipID)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("program membership not found")
		}
		id, year := membershipID, schoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "program_membership", ObjectID: &id, SchoolYearID: &year, Reason: "organizer removed programme membership", ChangeSummary: json.RawMessage(`{"deleted":true}`)})
	})
	if err != nil {
		return fmt.Errorf("delete program membership: %w", err)
	}
	return nil
}

func (s *Service) CountStudentsWithoutGrade(ctx context.Context, organizationID string, schoolYearID ids.XID) (int64, error) {
	if s == nil || s.database == nil {
		return 0, errors.New("count students without grade: data service is nil")
	}
	var result int64
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		var err error
		result, err = tx.CountStudentsWithoutGrade(ctx, schoolYearID)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("count students without grade: %w", err)
	}
	return result, nil
}

func programSummary(row data.Program) json.RawMessage {
	value, _ := json.Marshal(map[string]any{"name": row.Name})
	return value
}

func membershipSummary(row data.ProgramMembership) json.RawMessage {
	value, _ := json.Marshal(map[string]any{"program_id": row.ProgramID, "student_id": row.StudentID})
	return value
}
