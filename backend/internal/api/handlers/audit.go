package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultAuditPageSize int32 = 50
	maxAuditPageSize     int32 = 100
)

// AuditLogReader is the tenant-scoped data operation needed by the endpoint.
type AuditLogReader interface {
	ListAuditLog(context.Context, string, data.AuditLogFilter) ([]data.AuditLogEntry, error)
}

type AuditLogInput struct {
	ObjectType string `query:"object_type" doc:"Filter by affected object type."`
	Cursor     string `query:"cursor" doc:"Opaque cursor returned by the previous page."`
	Limit      int32  `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Number of entries to return."`
}

type AuditLogActor struct {
	Type  string `json:"type" doc:"Actor principal type."`
	ID    string `json:"id,omitempty" doc:"Actor user identifier, when present."`
	Label string `json:"label" doc:"Actor label captured at the time of the action."`
}

type AuditLogEntry struct {
	ID            string          `json:"id" doc:"Opaque audit entry identifier."`
	OccurredAt    string          `json:"occurred_at" doc:"Time the action occurred."`
	Actor         AuditLogActor   `json:"actor"`
	Action        string          `json:"action" enum:"create,edit,soft_delete,hard_delete,import_commit,school_year_create,program_create,membership_change,session_non_participation,session_state_transition,offering_edit_after_publish,tag_definition_change,tag_assignment_change,pairing_change,exclusion_change,vocabulary_change,solve_run,manual_operation,override,publish,republish,link_generate,link_regenerate,link_revoke,permission_change,administrator_add,administrator_remove"`
	ObjectType    string          `json:"object_type"`
	ObjectID      string          `json:"object_id,omitempty"`
	ChangeSummary json.RawMessage `json:"change_summary"`
	Reason        string          `json:"reason,omitempty"`
}

type AuditLogOutput struct {
	Body struct {
		Entries    []AuditLogEntry `json:"entries"`
		NextCursor string          `json:"next_cursor,omitempty" doc:"Cursor for the next page, when more entries exist."`
	}
}

type AuditLogHandler struct{ reader AuditLogReader }

func NewAuditLogHandler(reader AuditLogReader) *AuditLogHandler {
	return &AuditLogHandler{reader: reader}
}

func (h *AuditLogHandler) Handle(ctx context.Context, input *AuditLogInput) (*AuditLogOutput, error) {
	if h == nil || h.reader == nil {
		return nil, problems.New(http.StatusInternalServerError, problems.DatabaseUnavailable, "audit log is not configured")
	}
	principal, ok := authPrincipal(ctx)
	if !ok {
		return nil, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	pageSize := defaultAuditPageSize
	if input != nil && input.Limit != 0 {
		pageSize = input.Limit
	}
	if pageSize < 1 || pageSize > maxAuditPageSize {
		return nil, problems.New(http.StatusBadRequest, problems.InvalidAuditCursor, "limit must be between 1 and 100")
	}
	filter := data.AuditLogFilter{PageSize: pageSize}
	if input != nil && strings.TrimSpace(input.ObjectType) != "" {
		objectType := strings.TrimSpace(input.ObjectType)
		filter.ObjectType = &objectType
	}
	if input != nil && strings.TrimSpace(input.Cursor) != "" {
		occurredAt, id, err := decodeAuditCursor(input.Cursor)
		if err != nil {
			return nil, problems.New(http.StatusBadRequest, problems.InvalidAuditCursor, "cursor is invalid")
		}
		filter.CursorOccurredAt, filter.CursorID = &occurredAt, &id
	}
	rows, err := h.reader.ListAuditLog(ctx, string(principal.OrganizationID), filter)
	if err != nil {
		return nil, problems.New(http.StatusInternalServerError, problems.InternalError, "unable to read audit log")
	}
	output := &AuditLogOutput{}
	output.Body.Entries = make([]AuditLogEntry, len(rows))
	for i, row := range rows {
		output.Body.Entries[i] = auditResponse(row)
	}
	if len(rows) == int(pageSize) {
		output.Body.NextCursor = encodeAuditCursor(rows[len(rows)-1])
	}
	return output, nil
}

func authPrincipal(ctx context.Context) (auth.AccountPrincipal, bool) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.AccountPrincipal{}, false
	}
	account, ok := principal.(auth.AccountPrincipal)
	return account, ok
}

func auditResponse(row data.AuditLogEntry) AuditLogEntry {
	response := AuditLogEntry{ID: string(row.ID), OccurredAt: row.OccurredAt.Time.UTC().Format(time.RFC3339Nano), Actor: AuditLogActor{Type: row.ActorType, Label: row.ActorLabel}, Action: row.Action, ObjectType: row.ObjectType, ChangeSummary: row.ChangeSummary}
	if row.ActorUserID != nil {
		response.Actor.ID = string(*row.ActorUserID)
	}
	if row.ObjectID != nil {
		response.ObjectID = string(*row.ObjectID)
	}
	if row.Reason.Valid {
		response.Reason = row.Reason.String
	}
	return response
}

type auditCursor struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         ids.XID   `json:"id"`
}

func encodeAuditCursor(row data.AuditLogEntry) string {
	payload, _ := json.Marshal(auditCursor{OccurredAt: row.OccurredAt.Time.UTC(), ID: row.ID})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeAuditCursor(value string) (pgtype.Timestamptz, ids.XID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return pgtype.Timestamptz{}, "", err
	}
	var cursor auditCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.ID == "" || cursor.OccurredAt.IsZero() {
		return pgtype.Timestamptz{}, "", errors.New("invalid cursor")
	}
	return pgtype.Timestamptz{Time: cursor.OccurredAt, Valid: true}, cursor.ID, nil
}
