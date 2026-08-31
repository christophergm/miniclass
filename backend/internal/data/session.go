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

// SessionState is the persisted lifecycle vocabulary from SPEC §14.3.
type SessionState string

const (
	SessionPlanning         SessionState = "planning"
	SessionCatalogPublished SessionState = "catalog_published"
	SessionVotingOpen       SessionState = "voting_open"
	SessionVotingClosed     SessionState = "voting_closed"
	SessionAssigning        SessionState = "assigning"
	SessionPublished        SessionState = "published"
	SessionComplete         SessionState = "complete"
)

// Session is one ordered unit of programme activity within a school year.
// MeetingDates is populated by the programme service for API responses; dates
// remain a separate table so each date can later carry availability data.
type Session struct {
	ID                    ids.XID
	OrganizationID        ids.XID
	SchoolYearID          ids.XID
	ProgramID             ids.XID
	Name                  string
	Ordinal               int
	State                 SessionState
	DraftAssignmentsStale bool
	MeetingDates          []time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// MeetingDate is a date on which every offering in its session meets.
type MeetingDate struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SessionID      ids.XID
	Date           time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (tx *Tx) CreateSession(ctx context.Context, schoolYearID, programID ids.XID, name string, ordinal int) (Session, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Session{}, errors.New("create session: name is required")
	}
	if ordinal < 1 {
		return Session{}, errors.New("create session: ordinal must be positive")
	}
	row, err := tx.queries.CreateSession(ctx, db.CreateSessionParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, Name: name, Ordinal: int32(ordinal),
	})
	if err != nil {
		return Session{}, wrapProgramMutationError("create session", err)
	}
	return session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) ListSessions(ctx context.Context, schoolYearID, programID ids.XID) ([]Session, error) {
	rows, err := tx.queries.ListSessions(ctx, db.ListSessionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	result := make([]Session, 0, len(rows))
	for _, row := range rows {
		value, err := session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetSession(ctx context.Context, schoolYearID, programID, id ids.XID) (Session, error) {
	row, err := tx.queries.GetSession(ctx, db.GetSessionParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
}

// GetSessionForUpdate serializes lifecycle changes so two organizers cannot
// validate the same old state and then apply conflicting transitions.
func (tx *Tx) GetSessionForUpdate(ctx context.Context, schoolYearID, programID, id ids.XID) (Session, error) {
	row, err := tx.queries.GetSessionForUpdate(ctx, db.GetSessionForUpdateParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return Session{}, fmt.Errorf("get session for update: %w", err)
	}
	return session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateSession(ctx context.Context, schoolYearID, programID, id ids.XID, name string, ordinal int) (Session, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Session{}, errors.New("update session: name is required")
	}
	if ordinal < 1 {
		return Session{}, errors.New("update session: ordinal must be positive")
	}
	row, err := tx.queries.UpdateSession(ctx, db.UpdateSessionParams{ID: id, Name: name, Ordinal: int32(ordinal), OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return Session{}, wrapProgramMutationError("update session", err)
	}
	return session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateSessionLifecycle(ctx context.Context, schoolYearID, programID, id ids.XID, state SessionState, draftAssignmentsStale bool) (Session, error) {
	row, err := tx.queries.UpdateSessionLifecycle(ctx, db.UpdateSessionLifecycleParams{
		ID: id, State: db.SessionState(state), DraftAssignmentsStale: draftAssignmentsStale,
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
	})
	if err != nil {
		return Session{}, wrapProgramMutationError("update session lifecycle", err)
	}
	return session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) DeleteSession(ctx context.Context, schoolYearID, programID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteSession(ctx, db.DeleteSessionParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return false, wrapProgramMutationError("delete session", err)
	}
	return rows == 1, nil
}

func (tx *Tx) CreateMeetingDate(ctx context.Context, schoolYearID, programID, sessionID ids.XID, date time.Time) (MeetingDate, error) {
	if date.IsZero() {
		return MeetingDate{}, errors.New("create meeting date: date is required")
	}
	row, err := tx.queries.CreateMeetingDate(ctx, db.CreateMeetingDateParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, MeetingDate: dateParam(date),
	})
	if err != nil {
		return MeetingDate{}, wrapProgramMutationError("create meeting date", err)
	}
	return meetingDate(row)
}

func (tx *Tx) ListMeetingDates(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]MeetingDate, error) {
	rows, err := tx.queries.ListMeetingDates(ctx, db.ListMeetingDatesParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list meeting dates: %w", err)
	}
	result := make([]MeetingDate, 0, len(rows))
	for _, row := range rows {
		value, err := meetingDate(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetMeetingDate(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (MeetingDate, error) {
	row, err := tx.queries.GetMeetingDate(ctx, db.GetMeetingDateParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return MeetingDate{}, fmt.Errorf("get meeting date: %w", err)
	}
	return meetingDate(row)
}

func (tx *Tx) UpdateMeetingDate(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID, date time.Time) (MeetingDate, error) {
	if date.IsZero() {
		return MeetingDate{}, errors.New("update meeting date: date is required")
	}
	row, err := tx.queries.UpdateMeetingDate(ctx, db.UpdateMeetingDateParams{ID: id, MeetingDate: dateParam(date), OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return MeetingDate{}, wrapProgramMutationError("update meeting date", err)
	}
	return meetingDate(row)
}

func (tx *Tx) DeleteMeetingDate(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteMeetingDate(ctx, db.DeleteMeetingDateParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return false, wrapProgramMutationError("delete meeting date", err)
	}
	return rows == 1, nil
}

// The following methods are used only by the Layer 2 isolation registry.
func (tx *Tx) ListAllSessionsForRegistry(ctx context.Context) ([]Session, error) {
	rows, err := tx.queries.ListAllSessionsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list sessions for registry: %w", err)
	}
	result := make([]Session, 0, len(rows))
	for _, row := range rows {
		value, err := session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindSessionForRegistry(ctx context.Context, id ids.XID) (Session, error) {
	row, err := tx.queries.FindSessionForRegistry(ctx, db.FindSessionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, nil
		}
		return Session{}, fmt.Errorf("find session for registry: %w", err)
	}
	return session(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Ordinal, row.State, row.DraftAssignmentsStale, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateSessionForRegistry(ctx context.Context, id ids.XID, name string) (bool, error) {
	rows, err := tx.queries.UpdateSessionForRegistry(ctx, db.UpdateSessionForRegistryParams{ID: id, OrganizationID: tx.organizationID, Name: name})
	if err != nil {
		return false, wrapProgramMutationError("update session for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteSessionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteSessionForRegistry(ctx, db.DeleteSessionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("delete session for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) ListAllMeetingDatesForRegistry(ctx context.Context) ([]MeetingDate, error) {
	rows, err := tx.queries.ListAllMeetingDatesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list meeting dates for registry: %w", err)
	}
	result := make([]MeetingDate, 0, len(rows))
	for _, row := range rows {
		value, err := meetingDate(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindMeetingDateForRegistry(ctx context.Context, id ids.XID) (MeetingDate, error) {
	row, err := tx.queries.FindMeetingDateForRegistry(ctx, db.FindMeetingDateForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MeetingDate{}, nil
		}
		return MeetingDate{}, fmt.Errorf("find meeting date for registry: %w", err)
	}
	return meetingDate(row)
}

func (tx *Tx) UpdateMeetingDateForRegistry(ctx context.Context, id ids.XID, date time.Time) (bool, error) {
	rows, err := tx.queries.UpdateMeetingDateForRegistry(ctx, db.UpdateMeetingDateForRegistryParams{ID: id, OrganizationID: tx.organizationID, MeetingDate: dateParam(date)})
	if err != nil {
		return false, wrapProgramMutationError("update meeting date for registry", err)
	}
	return rows == 1, nil
}

func (tx *Tx) DeleteMeetingDateForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteMeetingDateForRegistry(ctx, db.DeleteMeetingDateForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return false, wrapProgramMutationError("delete meeting date for registry", err)
	}
	return rows == 1, nil
}

func session(id ids.XID, organizationID, schoolYearID, programID ids.XID, name string, ordinal int32, state db.SessionState, draftAssignmentsStale bool, createdAtValue, updatedAtValue pgtype.Timestamptz) (Session, error) {
	createdAt, err := programTime(createdAtValue, "created_at")
	if err != nil {
		return Session{}, err
	}
	updatedAt, err := programTime(updatedAtValue, "updated_at")
	if err != nil {
		return Session{}, err
	}
	return Session{ID: id, OrganizationID: organizationID, SchoolYearID: schoolYearID, ProgramID: programID, Name: name, Ordinal: int(ordinal), State: SessionState(state), DraftAssignmentsStale: draftAssignmentsStale, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func meetingDate(row db.MeetingDate) (MeetingDate, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return MeetingDate{}, err
	}
	updatedAt, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return MeetingDate{}, err
	}
	if !row.MeetingDate.Valid {
		return MeetingDate{}, errors.New("meeting date row: meeting_date is null")
	}
	return MeetingDate{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, Date: row.MeetingDate.Time, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func dateParam(value time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}
