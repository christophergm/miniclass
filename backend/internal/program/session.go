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
	"github.com/chrismott/miniclass/internal/preference"
)

func (s *Service) RegenerateRankedChoiceAccessCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, reason string) ([]preference.RankedChoiceAccessCode, error) {
	return preference.New(s.database).RegenerateRankedChoiceAccessCodes(ctx, organizationID, actor, schoolYearID, programID, sessionID, reason)
}

func (s *Service) RevokeRankedChoiceAccessCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, reason string) error {
	return preference.New(s.database).RevokeRankedChoiceAccessCodes(ctx, organizationID, actor, schoolYearID, programID, sessionID, reason)
}

var (
	ErrSessionNoChanges                = errors.New("session update has no changes")
	ErrMeetingDateNoChanges            = errors.New("meeting date update has no changes")
	ErrSessionRequiresMeetingDate      = errors.New("a session must have at least one meeting date")
	ErrRankedChoiceConfigurationLocked = errors.New("ranked-choice configuration is locked after voting opens")
	ErrRankedChoiceRankDepthInvalid    = errors.New("ranked-choice rank depth must be positive")
	ErrRankedChoiceDeadlineRequired    = errors.New("ranked-choice voting deadline is required")
	ErrRankedChoiceDeadlineInvalid     = errors.New("ranked-choice voting deadline must be in the future")
)

type SessionUpdate struct {
	Name         *string
	Dates        *[]time.Time
	RankedChoice *data.RankedChoiceConfiguration
}

func (s *Service) CreateSession(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID ids.XID, name string, dates []time.Time) (data.Session, error) {
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
		created, err := tx.CreateSession(ctx, schoolYearID, programID, name)
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
		name := current.Name
		changed := false
		if input.Name != nil && strings.TrimSpace(*input.Name) != current.Name {
			name, changed = *input.Name, true
		}
		currentDates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if input.Dates != nil {
			if len(*input.Dates) == 0 {
				return ErrSessionRequiresMeetingDate
			}
			changed = !sameMeetingDates(currentDates, *input.Dates) || changed
		}
		rankedChoice := current.RankedChoice
		if input.RankedChoice != nil {
			if current.State != data.SessionPlanning && current.State != data.SessionCatalogPublished {
				return ErrRankedChoiceConfigurationLocked
			}
			if err := validateRankedChoiceConfiguration(input.RankedChoice, time.Now().UTC()); err != nil {
				return err
			}
			rankedChoice = cloneRankedChoiceConfiguration(input.RankedChoice)
			changed = !sameRankedChoiceConfiguration(current.RankedChoice, rankedChoice) || changed
		}
		if !changed {
			return ErrSessionNoChanges
		}
		result, err = tx.UpdateSession(ctx, schoolYearID, programID, sessionID, name, rankedChoice)
		if err != nil {
			return err
		}
		if input.Dates != nil && !sameMeetingDates(currentDates, *input.Dates) {
			for _, date := range currentDates {
				if _, err := tx.DeleteMeetingDate(ctx, schoolYearID, programID, sessionID, date.ID); err != nil {
					return err
				}
				dateID, year := date.ID, result.SchoolYearID
				if err := tx.Record(ctx, audit.Entry{Action: audit.ActionMeetingDateChange, ObjectType: "meeting_date", ObjectID: &dateID, SchoolYearID: &year, ChangeSummary: meetingDateSummary(&date, data.MeetingDate{})}); err != nil {
					return err
				}
			}
			for _, date := range *input.Dates {
				createdDate, err := tx.CreateMeetingDate(ctx, schoolYearID, programID, sessionID, date)
				if err != nil {
					return err
				}
				dateID := createdDate.ID
				if err := tx.Record(ctx, audit.Entry{Action: audit.ActionMeetingDateChange, ObjectType: "meeting_date", ObjectID: &dateID, SchoolYearID: &result.SchoolYearID, ChangeSummary: meetingDateSummary(nil, createdDate)}); err != nil {
					return err
				}
			}
		}
		storedDates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		result.MeetingDates = meetingDateTimes(storedDates)
		id, year := result.ID, result.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSessionChange, ObjectType: "session", ObjectID: &id, SchoolYearID: &year, ChangeSummary: sessionSummary(&current, result)})
	})
	if err != nil {
		return data.Session{}, fmt.Errorf("update session: %w", err)
	}
	return result, nil
}

func validateRankedChoiceConfiguration(config *data.RankedChoiceConfiguration, now time.Time) error {
	if config == nil {
		return nil
	}
	if config.RankDepth < 1 {
		return ErrRankedChoiceRankDepthInvalid
	}
	if config.Deadline == nil {
		return ErrRankedChoiceDeadlineRequired
	}
	if !config.Deadline.After(now) {
		return ErrRankedChoiceDeadlineInvalid
	}
	return nil
}

func cloneRankedChoiceConfiguration(config *data.RankedChoiceConfiguration) *data.RankedChoiceConfiguration {
	if config == nil {
		return nil
	}
	clone := *config
	if config.Deadline != nil {
		deadline := *config.Deadline
		clone.Deadline = &deadline
	}
	return &clone
}

func sameRankedChoiceConfiguration(a, b *data.RankedChoiceConfiguration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.RankDepth != b.RankDepth {
		return false
	}
	if a.Deadline == nil || b.Deadline == nil {
		return a.Deadline == nil && b.Deadline == nil
	}
	return a.Deadline.Equal(*b.Deadline)
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

func sameMeetingDates(current []data.MeetingDate, requested []time.Time) bool {
	if len(current) != len(requested) {
		return false
	}
	counts := make(map[string]int, len(current))
	for _, date := range current {
		counts[date.Date.UTC().Format("2006-01-02")]++
	}
	for _, date := range requested {
		key := date.UTC().Format("2006-01-02")
		if counts[key] == 0 {
			return false
		}
		counts[key]--
	}
	return true
}

func sessionSummary(before *data.Session, after data.Session) json.RawMessage {
	value := map[string]any{}
	if after.ID != "" {
		value["name"] = after.Name
		value["state"] = after.State
		value["draft_assignments_stale"] = after.DraftAssignmentsStale
	} else {
		value["deleted"] = true
	}
	if before != nil {
		value["before"] = map[string]any{"name": before.Name, "state": before.State, "draft_assignments_stale": before.DraftAssignmentsStale}
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
