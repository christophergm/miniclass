package handlers

import (
	"context"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

// SolveRunService is the API boundary for immutable solve-run lifecycle work.
// Requests are compiled from authoritative persisted state, never from a
// browser-supplied solver document.
type SolveRunService interface {
	StartAuthoritative(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, *int64) (data.SolveRun, error)
	Get(context.Context, string, ids.XID, ids.XID, ids.XID, ids.XID) (data.SolveRun, error)
	RerunAuthoritative(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID) (data.SolveRun, error)
}
