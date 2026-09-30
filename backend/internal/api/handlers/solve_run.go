package handlers

import (
	"context"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	solverservice "github.com/chrismott/miniclass/internal/solver"
	"github.com/chrismott/miniclass/internal/solvercontract"
)

// SolveRunService is the API boundary for immutable solve-run lifecycle work.
// The prepared contract request is deliberately supplied by the caller: live
// feasibility-model construction belongs to issue #237.
type SolveRunService interface {
	Start(context.Context, string, audit.Actor, solverservice.StartInput) (data.SolveRun, error)
	Get(context.Context, string, ids.XID, ids.XID, ids.XID, ids.XID) (data.SolveRun, error)
	Rerun(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID, solvercontract.Request) (data.SolveRun, error)
}
