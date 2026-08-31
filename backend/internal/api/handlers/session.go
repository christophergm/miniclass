package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SessionResponse struct {
	ID             string    `json:"id" doc:"Opaque session identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID      string    `json:"program_id" doc:"Opaque program identifier."`
	Name           string    `json:"name"`
	Ordinal        int       `json:"ordinal" doc:"Explicit session order; never inferred from dates."`
	State          string    `json:"state" enum:"planning,catalog_published,voting_open,voting_closed,assigning,published,complete"`
	MeetingDates   []string  `json:"meeting_dates" format:"date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MeetingDateResponse struct {
	ID             string    `json:"id" doc:"Opaque meeting-date identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID      string    `json:"program_id" doc:"Opaque program identifier."`
	SessionID      string    `json:"session_id" doc:"Opaque session identifier."`
	Date           string    `json:"meeting_date" format:"date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SessionListOutput struct{ Body []SessionResponse }
type SessionOutput struct{ Body SessionResponse }
type MeetingDateListOutput struct{ Body []MeetingDateResponse }
type MeetingDateOutput struct{ Body MeetingDateResponse }
type SessionPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	ProgramID    string `path:"programID" minLength:"1" doc:"Opaque program identifier."`
	SessionID    string `path:"sessionID" minLength:"1" doc:"Opaque session identifier."`
}
type SessionCollectionInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	ProgramID    string `path:"programID" minLength:"1" doc:"Opaque program identifier."`
}
type CreateSessionInput struct {
	SessionCollectionInput
	Body struct {
		Name         string   `json:"name" minLength:"1"`
		Ordinal      int      `json:"ordinal" minimum:"1"`
		MeetingDates []string `json:"meeting_dates" minItems:"1" format:"date"`
	}
}
type UpdateSessionInput struct {
	SessionPathInput
	Body struct {
		Name    *string `json:"name,omitempty" minLength:"1"`
		Ordinal *int    `json:"ordinal,omitempty" minimum:"1"`
	}
}
type MeetingDatePathInput struct {
	SessionPathInput
	MeetingDateID string `path:"meetingDateID" minLength:"1" doc:"Opaque meeting-date identifier."`
}
type CreateMeetingDateInput struct {
	SessionPathInput
	Body struct {
		Date string `json:"meeting_date" format:"date" minLength:"10"`
	}
}
type UpdateMeetingDateInput struct {
	MeetingDatePathInput
	Body struct {
		Date string `json:"meeting_date" format:"date" minLength:"10"`
	}
}

func (h *ProgramHandler) ListSessions(ctx context.Context, input *SessionCollectionInput) (*SessionListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	rows, err := h.service.ListSessions(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID))
	if err != nil {
		return nil, sessionProblem(err)
	}
	result := make([]SessionResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, sessionResponse(row))
	}
	return &SessionListOutput{Body: result}, nil
}

func (h *ProgramHandler) CreateSession(ctx context.Context, input *CreateSessionInput) (*SessionOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	dates, err := parseMeetingDates(input.Body.MeetingDates)
	if err != nil {
		return nil, sessionProblem(err)
	}
	row, err := h.service.CreateSession(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), input.Body.Name, input.Body.Ordinal, dates)
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &SessionOutput{Body: sessionResponse(row)}, nil
}

func (h *ProgramHandler) GetSession(ctx context.Context, input *SessionPathInput) (*SessionOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	row, err := h.service.GetSession(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &SessionOutput{Body: sessionResponse(row)}, nil
}

func (h *ProgramHandler) UpdateSession(ctx context.Context, input *UpdateSessionInput) (*SessionOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	row, err := h.service.UpdateSession(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), programservice.SessionUpdate{Name: input.Body.Name, Ordinal: input.Body.Ordinal})
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &SessionOutput{Body: sessionResponse(row)}, nil
}

func (h *ProgramHandler) DeleteSession(ctx context.Context, input *SessionPathInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	if err := h.service.DeleteSession(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID)); err != nil {
		return nil, sessionProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func (h *ProgramHandler) ListMeetingDates(ctx context.Context, input *SessionPathInput) (*MeetingDateListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, meetingDateNotFound()
	}
	rows, err := h.service.ListMeetingDates(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, sessionProblem(err)
	}
	result := make([]MeetingDateResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, meetingDateResponse(row))
	}
	return &MeetingDateListOutput{Body: result}, nil
}

func (h *ProgramHandler) GetMeetingDate(ctx context.Context, input *MeetingDatePathInput) (*MeetingDateOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, meetingDateNotFound()
	}
	row, err := h.service.GetMeetingDate(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.MeetingDateID))
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &MeetingDateOutput{Body: meetingDateResponse(row)}, nil
}

func (h *ProgramHandler) CreateMeetingDate(ctx context.Context, input *CreateMeetingDateInput) (*MeetingDateOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, meetingDateNotFound()
	}
	date, err := parseMeetingDate(input.Body.Date)
	if err != nil {
		return nil, sessionProblem(err)
	}
	row, err := h.service.CreateMeetingDate(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), date)
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &MeetingDateOutput{Body: meetingDateResponse(row)}, nil
}

func (h *ProgramHandler) UpdateMeetingDate(ctx context.Context, input *UpdateMeetingDateInput) (*MeetingDateOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, meetingDateNotFound()
	}
	date, err := parseMeetingDate(input.Body.Date)
	if err != nil {
		return nil, sessionProblem(err)
	}
	row, err := h.service.UpdateMeetingDate(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.MeetingDateID), date)
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &MeetingDateOutput{Body: meetingDateResponse(row)}, nil
}

func (h *ProgramHandler) DeleteMeetingDate(ctx context.Context, input *MeetingDatePathInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, meetingDateNotFound()
	}
	if err := h.service.DeleteMeetingDate(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.MeetingDateID)); err != nil {
		return nil, sessionProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func sessionResponse(row data.Session) SessionResponse {
	dates := make([]string, 0, len(row.MeetingDates))
	for _, date := range row.MeetingDates {
		dates = append(dates, date.Format("2006-01-02"))
	}
	return SessionResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), Name: row.Name, Ordinal: row.Ordinal, State: string(row.State), MeetingDates: dates, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func meetingDateResponse(row data.MeetingDate) MeetingDateResponse {
	return MeetingDateResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), SessionID: string(row.SessionID), Date: row.Date.Format("2006-01-02"), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func parseMeetingDates(values []string) ([]time.Time, error) {
	if len(values) == 0 {
		return nil, programservice.ErrSessionRequiresMeetingDate
	}
	result := make([]time.Time, 0, len(values))
	for _, value := range values {
		date, err := parseMeetingDate(value)
		if err != nil {
			return nil, err
		}
		result = append(result, date)
	}
	return result, nil
}

func parseMeetingDate(value string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, errors.New("meeting date must be a valid date in YYYY-MM-DD format")
	}
	return date, nil
}

func sessionNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "session not found")
}

func meetingDateNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "meeting date not found")
}

func sessionProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), strings.Contains(err.Error(), "session not found"):
		return sessionNotFound()
	case strings.Contains(err.Error(), "meeting date not found"):
		return meetingDateNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the session or meeting date already exists in this program")
	case errors.Is(err, programservice.ErrSessionNoChanges), errors.Is(err, programservice.ErrMeetingDateNoChanges), errors.Is(err, programservice.ErrSessionRequiresMeetingDate), strings.Contains(err.Error(), "name is required"), strings.Contains(err.Error(), "ordinal must be positive"), strings.Contains(err.Error(), "valid date"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change session data")
	}
}
