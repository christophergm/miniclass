package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrSchoolYearClosed identifies the database-enforced immutability error
// shared by school-year-scoped tables.
var ErrSchoolYearClosed = errors.New("school year is closed")

// SchoolYearState is the application view of the database enum.
type SchoolYearState string

const (
	SchoolYearSetup  SchoolYearState = "setup"
	SchoolYearActive SchoolYearState = "active"
	SchoolYearClosed SchoolYearState = "closed"
)

// SchoolYear is the tenant-safe application representation of a school year.
type SchoolYear struct {
	ID             ids.XID
	OrganizationID ids.XID
	Label          string
	State          SchoolYearState
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateSchoolYear creates a setup year for the transaction tenant.
func (tx *Tx) CreateSchoolYear(ctx context.Context, label string) (SchoolYear, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return SchoolYear{}, errors.New("create school year: label is empty")
	}
	row, err := tx.queries.CreateSchoolYear(ctx, db.CreateSchoolYearParams{
		OrganizationID: tx.organizationID,
		Label:          label,
	})
	if err != nil {
		return SchoolYear{}, fmt.Errorf("create school year: %w", err)
	}
	return schoolYear(row)
}

// ListSchoolYears lists only the transaction tenant's years. RLS supplies the
// tenant predicate; callers do not repeat organization filters in SQL.
func (tx *Tx) ListSchoolYears(ctx context.Context) ([]SchoolYear, error) {
	rows, err := tx.queries.ListSchoolYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("list school years: %w", err)
	}
	result := make([]SchoolYear, 0, len(rows))
	for _, row := range rows {
		value, err := schoolYear(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// GetSchoolYearByID fetches one year in the transaction tenant.
func (tx *Tx) GetSchoolYearByID(ctx context.Context, id ids.XID) (SchoolYear, error) {
	if strings.TrimSpace(string(id)) == "" {
		return SchoolYear{}, errors.New("get school year: id is empty")
	}
	row, err := tx.queries.GetSchoolYearByID(ctx, id)
	if err != nil {
		return SchoolYear{}, fmt.Errorf("get school year: %w", err)
	}
	return schoolYear(row)
}

// UpdateSchoolYearLabel changes a year label. Closed-year errors are retained
// as a sentinel so the API can map the trigger to its stable 409 problem.
func (tx *Tx) UpdateSchoolYearLabel(ctx context.Context, id ids.XID, label string) (SchoolYear, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return SchoolYear{}, errors.New("update school year: label is empty")
	}
	row, err := tx.queries.UpdateSchoolYearLabel(ctx, db.UpdateSchoolYearLabelParams{ID: id, Label: label})
	if err != nil {
		return SchoolYear{}, wrapSchoolYearError("update school year", err)
	}
	return schoolYear(row)
}

// UpdateSchoolYearState changes a year state. The lifecycle service validates
// the transition before calling this storage primitive.
func (tx *Tx) UpdateSchoolYearState(ctx context.Context, id ids.XID, state SchoolYearState) (SchoolYear, error) {
	if !validSchoolYearState(state) {
		return SchoolYear{}, fmt.Errorf("update school year: invalid state %q", state)
	}
	row, err := tx.queries.UpdateSchoolYearState(ctx, db.UpdateSchoolYearStateParams{
		ID: id, State: db.SchoolYearState(state),
	})
	if err != nil {
		return SchoolYear{}, wrapSchoolYearError("update school year state", err)
	}
	return schoolYear(row)
}

// DeleteSchoolYear removes a non-closed year and returns the affected-row
// count. A zero count is the tenant-safe not-found result.
func (tx *Tx) DeleteSchoolYear(ctx context.Context, id ids.XID) (int64, error) {
	rows, err := tx.queries.DeleteSchoolYear(ctx, id)
	if err != nil {
		return 0, wrapSchoolYearError("delete school year", err)
	}
	return rows, nil
}

// PrepareSchoolYearReopen arms the shared trigger for the one audited,
// Owner-only closed-to-active transition. The setting is LOCAL to this unit
// of work and a reason is required by both the service and the trigger.
func (tx *Tx) PrepareSchoolYearReopen(ctx context.Context, id ids.XID, reason string) error {
	if tx == nil || tx.raw == nil {
		return errors.New("prepare school year reopen: transaction is nil")
	}
	if tx.readOnly {
		return errors.New("prepare school year reopen: transaction is read-only")
	}
	if strings.TrimSpace(string(id)) == "" {
		return errors.New("prepare school year reopen: school year id is empty")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("prepare school year reopen: reason is required")
	}
	if _, err := tx.raw.Exec(ctx, "select set_config('app.school_year_reopen', 'true', true), set_config('app.school_year_reopen_id', $1, true), set_config('app.school_year_reopen_reason', $2, true)", string(id), reason); err != nil {
		return fmt.Errorf("prepare school year reopen: %w", err)
	}
	return nil
}

func validSchoolYearState(state SchoolYearState) bool {
	switch state {
	case SchoolYearSetup, SchoolYearActive, SchoolYearClosed:
		return true
	default:
		return false
	}
}

func IsSchoolYearClosed(err error) bool {
	return errors.Is(err, ErrSchoolYearClosed)
}

func wrapSchoolYearError(operation string, err error) error {
	if isClosedYearDatabaseError(err) {
		return fmt.Errorf("%w: %v", ErrSchoolYearClosed, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func isClosedYearDatabaseError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "P0001" && pgErr.Message == "school year is closed"
}

func schoolYear(row db.SchoolYear) (SchoolYear, error) {
	createdAt, err := schoolYearTime(row.CreatedAt, "created_at")
	if err != nil {
		return SchoolYear{}, err
	}
	updatedAt, err := schoolYearTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return SchoolYear{}, err
	}
	return SchoolYear{
		ID: row.ID, OrganizationID: row.OrganizationID, Label: row.Label,
		State: SchoolYearState(row.State), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func schoolYearTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("school year row: %s is null", name)
	}
	return value.Time, nil
}
