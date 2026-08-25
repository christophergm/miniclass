package data

import (
	"context"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuditLogFilter describes the only filters supported by the Phase 1 audit
// endpoint. The cursor is the last row from the previous page.
type AuditLogFilter struct {
	ObjectType       *string
	CursorOccurredAt *pgtype.Timestamptz
	CursorID         *ids.XID
	PageSize         int32
}

// AuditLogEntry is the data-layer representation exposed to API adapters.
// Keeping generated SQL types here prevents API packages from bypassing the
// tenant data boundary.
type AuditLogEntry struct {
	ID            ids.XID
	OccurredAt    pgtype.Timestamptz
	ActorType     string
	ActorUserID   *ids.XID
	ActorLabel    string
	Action        string
	ObjectType    string
	ObjectID      *ids.XID
	ChangeSummary []byte
	Reason        pgtype.Text
	SchoolYearID  *ids.XID
	RequestID     pgtype.Text
}

// ListAuditLog reads one tenant-scoped page using keyset pagination.
func (d *DB) ListAuditLog(ctx context.Context, organizationID string, filter AuditLogFilter) ([]AuditLogEntry, error) {
	var entries []AuditLogEntry
	err := d.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *Tx) error {
		var objectType pgtype.Text
		if filter.ObjectType != nil {
			objectType = pgtype.Text{String: *filter.ObjectType, Valid: true}
		}
		var occurredAt pgtype.Timestamptz
		if filter.CursorOccurredAt != nil {
			occurredAt = *filter.CursorOccurredAt
		}
		var cursorID ids.XID
		if filter.CursorID != nil {
			cursorID = *filter.CursorID
		}
		rows, err := tx.Queries().ListAuditLog(ctx, db.ListAuditLogParams{
			ObjectType:       objectType,
			CursorOccurredAt: occurredAt,
			CursorID:         cursorID,
			PageSize:         filter.PageSize,
		})
		if err != nil {
			return err
		}
		entries = make([]AuditLogEntry, len(rows))
		for i, row := range rows {
			entries[i] = AuditLogEntry{
				ID: row.ID, OccurredAt: row.OccurredAt, ActorType: string(row.ActorType),
				ActorUserID: row.ActorUserID, ActorLabel: row.ActorLabel, Action: row.Action,
				ObjectType: row.ObjectType, ObjectID: row.ObjectID, ChangeSummary: row.ChangeSummary,
				Reason: row.Reason, SchoolYearID: row.SchoolYearID, RequestID: row.RequestID,
			}
		}
		return nil
	})
	return entries, err
}
