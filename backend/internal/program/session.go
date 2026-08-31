package program

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

var (
	ErrSessionNoChanges           = errors.New("session update has no changes")
	ErrMeetingDateNoChanges       = errors.New("meeting date update has no changes")
	ErrSessionRequiresMeetingDate = errors.New("a session must have at least one meeting date")
)

type SessionUpdate struct {
	Name    *string
	Ordinal *int
}

func (s *Service) CreateSession(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID ids.XID, name string, ordinal int, dates []time.Time) (data.Session, error) {
	if s == nil || s.database == nil {
		return data.Session{}, errors.New("create session: data service is nil")
	}
	if len(dates) == 0 {
		return data.Session{}, ErrSessionRequiresMeetingDate
	}
	var result data.Session
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		created, err := tx.CreateSession(ctx, schoolYearID, programID, name, ordinal)
		if err != nil {
			return err
		}
		id, year := created.ID, created.SchoolYearID
		if err := tx.Record(ctx, audit.Entry{Action: audit.ActionSessionCreate, ObjectType: "session", ObjectID: &id, SchoolYearID: &year, ChangeSummary: sessionSummary(nil, created)}); err != nil {
			return err
		}
		result = created
		for _, date := range dates {
			meetingDate, err := tx.CreateMeetingDate(ctx, schoolYearID, programID, created.ID, date)
			if err != nil {
				return err
			}
			meetingID := meetingDate.ID
			if err := tx.Record(ctx, audit.Entry{Action: audit.ActionMeetingDateChange, ObjectType: "meeting_date", ObjectID: &meetingID, SchoolYearID: &year, ChangeSummary: meetingDateSummary(nil, meetingDate)}); err != nil {
				return err
			}
		}
		storedDates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, created.ID)
		if err != nil {
			return err
		}
		result.MeetingDates = meetingDateTimes(storedDates)
		return nil
	})
	if err != nil {
		return data.Session{}, fmt.Errorf("create session: %w", err)
	}
	return result, nil
}

func (s *Service) ListSessions(ctx context.Context, organizationID string, schoolYearID, programID ids.XID) ([]data.Session, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list sessions: data service is nil")
	}
	var result []data.Session
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		rows, err := tx.ListSessions(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		if err := s.loadSessionDates(ctx, tx, rows); err != nil {
			return err
		}
		result = rows
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return result, nil
}

func (s *Service) GetSession(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (data.Session, error) {
	if s == nil || s.database == nil {
		return data.Session{}, errors.New("get session: data service is nil")
	}
	var result data.Session
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		row, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		rows, err := tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		row.MeetingDates = meetingDateTimes(rows)
		result = row
		return nil
	})
	if err != nil {
		return data.Session{}, fmt.Errorf("get session: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateSession(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, input SessionUpdate) (data.Session, error) {
	if s == nil || s.database == nil {
		return data.Session{}, errors.New("update session: data service is nil")
	}
	var result data.Session
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if err := ensureSessionMutable(current); err != nil {
			return err
		}
		name, ordinal := current.Name, current.Ordinal
		changed := false
		if input.Name != nil && strings.TrimSpace(*input.Name) != current.Name {
			name, changed = *input.Name, true
		}
		if input.Ordinal != nil && *input.Ordinal != current.Ordinal {
			ordinal, changed = *input.Ordinal, true
		}
		if !changed {
			return ErrSessionNoChanges
		}
		result, err = tx.UpdateSession(ctx, schoolYearID, programID, sessionID, name, ordinal)
		if err != nil {
			return err
		}
		id, year := result.ID, result.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSessionChange, ObjectType: "session", ObjectID: &id, SchoolYearID: &year, ChangeSummary: sessionSummary(&current, result)})
	})
	if err != nil {
		return data.Session{}, fmt.Errorf("update session: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteSession(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID) error {
	if s == nil || s.database == nil {
		return errors.New("delete session: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if err := ensureSessionMutable(current); err != nil {
			return err
		}
		deleted, err := tx.DeleteSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("session not found")
		}
		id, year := current.ID, current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSessionChange, ObjectType: "session", ObjectID: &id, SchoolYearID: &year, ChangeSummary: sessionSummary(&current, data.Session{})})
	})
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Service) ListMeetingDates(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) ([]data.MeetingDate, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list meeting dates: data service is nil")
	}
	var result []data.MeetingDate
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list meeting dates: %w", err)
	}
	return result, nil
}

func (s *Service) GetMeetingDate(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, meetingDateID ids.XID) (data.MeetingDate, error) {
	if s == nil || s.database == nil {
		return data.MeetingDate{}, errors.New("get meeting date: data service is nil")
	}
	var result data.MeetingDate
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetMeetingDate(ctx, schoolYearID, programID, sessionID, meetingDateID)
		return err
	})
	if err != nil {
		return data.MeetingDate{}, fmt.Errorf("get meeting date: %w", err)
	}
	return result, nil
}

func (s *Service) CreateMeetingDate(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, date time.Time) (data.MeetingDate, error) {
	if s == nil || s.database == nil {
		return data.MeetingDate{}, errors.New("create meeting date: data service is nil")
	}
	var result data.MeetingDate
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if err := ensureSessionMutable(session); err != nil {
			return err
		}
		result, err = tx.CreateMeetingDate(ctx, schoolYearID, programID, sessionID, date)
		if err != nil {
			return err
		}
		id, year := result.ID, result.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMeetingDateChange, ObjectType: "meeting_date", ObjectID: &id, SchoolYearID: &year, ChangeSummary: meetingDateSummary(nil, result)})
	})
	if err != nil {
		return data.MeetingDate{}, fmt.Errorf("create meeting date: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateMeetingDate(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, meetingDateID ids.XID, date time.Time) (data.MeetingDate, error) {
	if s == nil || s.database == nil {
		return data.MeetingDate{}, errors.New("update meeting date: data service is nil")
	}
	var result data.MeetingDate
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if err := ensureSessionMutable(session); err != nil {
			return err
		}
		current, err := tx.GetMeetingDate(ctx, schoolYearID, programID, sessionID, meetingDateID)
		if err != nil {
			return err
		}
		if current.Date.Equal(date) {
			return ErrMeetingDateNoChanges
		}
		result, err = tx.UpdateMeetingDate(ctx, schoolYearID, programID, sessionID, meetingDateID, date)
		if err != nil {
			return err
		}
		id, year := result.ID, result.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMeetingDateChange, ObjectType: "meeting_date", ObjectID: &id, SchoolYearID: &year, ChangeSummary: meetingDateSummary(&current, result)})
	})
	if err != nil {
		return data.MeetingDate{}, fmt.Errorf("update meeting date: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteMeetingDate(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, meetingDateID ids.XID) error {
	if s == nil || s.database == nil {
		return errors.New("delete meeting date: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if err := ensureSessionMutable(session); err != nil {
			return err
		}
		current, err := tx.GetMeetingDate(ctx, schoolYearID, programID, sessionID, meetingDateID)
		if err != nil {
			return err
		}
		dates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if len(dates) <= 1 {
			return ErrSessionRequiresMeetingDate
		}
		deleted, err := tx.DeleteMeetingDate(ctx, schoolYearID, programID, sessionID, meetingDateID)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("meeting date not found")
		}
		id, year := current.ID, current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionMeetingDateChange, ObjectType: "meeting_date", ObjectID: &id, SchoolYearID: &year, ChangeSummary: meetingDateSummary(&current, data.MeetingDate{})})
	})
	if err != nil {
		return fmt.Errorf("delete meeting date: %w", err)
	}
	return nil
}

func (s *Service) loadSessionDates(ctx context.Context, tx *data.Tx, rows []data.Session) error {
	for index := range rows {
		dates, err := tx.ListMeetingDates(ctx, rows[index].SchoolYearID, rows[index].ProgramID, rows[index].ID)
		if err != nil {
			return err
		}
		rows[index].MeetingDates = meetingDateTimes(dates)
	}
	return nil
}

func meetingDateTimes(rows []data.MeetingDate) []time.Time {
	result := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.Date)
	}
	return result
}

func sessionSummary(before *data.Session, after data.Session) json.RawMessage {
	value := map[string]any{}
	if after.ID != "" {
		value["name"] = after.Name
		value["ordinal"] = after.Ordinal
		value["state"] = after.State
		value["draft_assignments_stale"] = after.DraftAssignmentsStale
	} else {
		value["deleted"] = true
	}
	if before != nil {
		value["before"] = map[string]any{"name": before.Name, "ordinal": before.Ordinal, "state": before.State, "draft_assignments_stale": before.DraftAssignmentsStale}
	}
	return mustJSON(value)
}

func meetingDateSummary(before *data.MeetingDate, after data.MeetingDate) json.RawMessage {
	value := map[string]any{}
	if after.ID != "" {
		value["meeting_date"] = after.Date.Format("2006-01-02")
	} else {
		value["deleted"] = true
	}
	if before != nil {
		value["before"] = before.Date.Format("2006-01-02")
	}
	return mustJSON(value)
}

func mustJSON(value any) json.RawMessage {
	result, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return result
}
