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

var ErrSessionNonParticipationNoChanges = errors.New("session non-participation update has no changes")
var ErrStudentNotProgramMember = errors.New("student is not a member of this programme")

type SessionNonParticipationUpdate struct {
	Reason *string
}

func (s *Service) CreateSessionNonParticipation(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, studentID ids.XID, reason string) (data.SessionNonParticipation, error) {
	if s == nil || s.database == nil {
		return data.SessionNonParticipation{}, errors.New("create session non-participation: data service is nil")
	}
	var result data.SessionNonParticipation
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		if err := ensureProgramMember(ctx, tx, schoolYearID, programID, studentID); err != nil {
			return err
		}
		created, err := tx.CreateSessionNonParticipation(ctx, schoolYearID, programID, sessionID, studentID, reason)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSessionNonParticipation, ObjectType: "session_non_participation", ObjectID: &id, SchoolYearID: &year, Reason: created.Reason, ChangeSummary: sessionNonParticipationSummary(nil, created)})
	})
	if err != nil {
		return data.SessionNonParticipation{}, fmt.Errorf("create session non-participation: %w", err)
	}
	return result, nil
}

func (s *Service) ListSessionNonParticipations(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) ([]data.SessionNonParticipation, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list session non-participations: data service is nil")
	}
	var result []data.SessionNonParticipation
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list session non-participations: %w", err)
	}
	return result, nil
}

func (s *Service) GetSessionNonParticipation(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, id ids.XID) (data.SessionNonParticipation, error) {
	if s == nil || s.database == nil {
		return data.SessionNonParticipation{}, errors.New("get session non-participation: data service is nil")
	}
	var result data.SessionNonParticipation
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetSessionNonParticipation(ctx, schoolYearID, programID, sessionID, id)
		return err
	})
	if err != nil {
		return data.SessionNonParticipation{}, fmt.Errorf("get session non-participation: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateSessionNonParticipation(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, id ids.XID, input SessionNonParticipationUpdate) (data.SessionNonParticipation, error) {
	if s == nil || s.database == nil {
		return data.SessionNonParticipation{}, errors.New("update session non-participation: data service is nil")
	}
	var result data.SessionNonParticipation
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSessionNonParticipation(ctx, schoolYearID, programID, sessionID, id)
		if err != nil {
			return err
		}
		if input.Reason == nil || strings.TrimSpace(*input.Reason) == current.Reason {
			return ErrSessionNonParticipationNoChanges
		}
		result, err = tx.UpdateSessionNonParticipation(ctx, schoolYearID, programID, sessionID, id, *input.Reason)
		if err != nil {
			return err
		}
		year := result.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSessionNonParticipation, ObjectType: "session_non_participation", ObjectID: &id, SchoolYearID: &year, Reason: result.Reason, ChangeSummary: sessionNonParticipationSummary(&current, result)})
	})
	if err != nil {
		return data.SessionNonParticipation{}, fmt.Errorf("update session non-participation: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteSessionNonParticipation(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, id ids.XID) error {
	if s == nil || s.database == nil {
		return errors.New("delete session non-participation: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSessionNonParticipation(ctx, schoolYearID, programID, sessionID, id)
		if err != nil {
			return err
		}
		deleted, err := tx.DeleteSessionNonParticipation(ctx, schoolYearID, programID, sessionID, id)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("session non-participation not found")
		}
		year := current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSessionNonParticipation, ObjectType: "session_non_participation", ObjectID: &id, SchoolYearID: &year, Reason: "organizer removed session non-participation", ChangeSummary: sessionNonParticipationSummary(&current, data.SessionNonParticipation{})})
	})
	if err != nil {
		return fmt.Errorf("delete session non-participation: %w", err)
	}
	return nil
}

// ListParticipatingMemberships is the session-level membership projection
// used by assignment work. It leaves annual programme membership untouched.
func (s *Service) ListParticipatingMemberships(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) ([]data.ProgramMembership, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list participating memberships: data service is nil")
	}
	var result []data.ProgramMembership
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		nonParticipations, err := tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		excluded := make(map[ids.XID]struct{}, len(nonParticipations))
		for _, row := range nonParticipations {
			excluded[row.StudentID] = struct{}{}
		}
		for _, membership := range memberships {
			if _, ok := excluded[membership.StudentID]; !ok {
				result = append(result, membership)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list participating memberships: %w", err)
	}
	return result, nil
}

func ensureProgramMember(ctx context.Context, tx *data.Tx, schoolYearID, programID, studentID ids.XID) error {
	memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
	if err != nil {
		return err
	}
	for _, membership := range memberships {
		if membership.StudentID == studentID {
			return nil
		}
	}
	return ErrStudentNotProgramMember
}

func sessionNonParticipationSummary(before *data.SessionNonParticipation, after data.SessionNonParticipation) json.RawMessage {
	value := map[string]any{}
	if after.ID != "" {
		value["student_id"] = after.StudentID
		value["session_id"] = after.SessionID
		value["reason"] = after.Reason
	} else {
		value["deleted"] = true
	}
	if before != nil {
		value["before"] = map[string]any{"student_id": before.StudentID, "session_id": before.SessionID, "reason": before.Reason}
	}
	encoded, _ := json.Marshal(value)
	return encoded
}
