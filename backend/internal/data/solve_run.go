package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

// SolveRun is the immutable persisted record required by SPEC §20.2. Documents
// are canonical solver-contract JSON, not database-backed domain objects.
type SolveRun struct {
	ID                    ids.XID         `json:"id"`
	OrganizationID        ids.XID         `json:"organization_id"`
	SchoolYearID          ids.XID         `json:"school_year_id"`
	ProgramID             ids.XID         `json:"program_id"`
	SessionID             ids.XID         `json:"session_id"`
	RerunOfSolveRunID     *ids.XID        `json:"rerun_of_solve_run_id,omitempty"`
	ContractVersion       string          `json:"contract_version"`
	Seed                  int64           `json:"seed"`
	InputFingerprint      string          `json:"input_fingerprint"`
	RequestDocument       json.RawMessage `json:"request_document"`
	ResponseDocument      json.RawMessage `json:"response_document"`
	SolverStatus          string          `json:"solver_status"`
	DeterministicDuration float64         `json:"deterministic_duration"`
	CreatedAt             time.Time       `json:"created_at"`
}

type CreateSolveRunInput struct {
	SchoolYearID          ids.XID
	ProgramID             ids.XID
	SessionID             ids.XID
	RerunOfSolveRunID     *ids.XID
	ContractVersion       string
	Seed                  int64
	InputFingerprint      string
	RequestDocument       json.RawMessage
	ResponseDocument      json.RawMessage
	SolverStatus          string
	DeterministicDuration float64
}

func (tx *Tx) CreateSolveRun(ctx context.Context, input CreateSolveRunInput) (SolveRun, error) {
	if tx == nil || tx.queries == nil {
		return SolveRun{}, errors.New("create solve run: transaction is nil")
	}
	if input.SchoolYearID == "" || input.ProgramID == "" || input.SessionID == "" || input.ContractVersion == "" || input.InputFingerprint == "" || input.SolverStatus == "" || input.DeterministicDuration < 0 {
		return SolveRun{}, errors.New("create solve run: required solve-run fields are missing or invalid")
	}
	if !json.Valid(input.RequestDocument) || !json.Valid(input.ResponseDocument) {
		return SolveRun{}, errors.New("create solve run: request and response documents must be valid JSON")
	}
	row, err := tx.queries.CreateSolveRun(ctx, db.CreateSolveRunParams{
		OrganizationID: tx.organizationID, SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID,
		RerunOfSolveRunID: input.RerunOfSolveRunID, ContractVersion: input.ContractVersion, Seed: input.Seed,
		InputFingerprint: input.InputFingerprint, RequestDocument: input.RequestDocument, ResponseDocument: input.ResponseDocument,
		SolverStatus: input.SolverStatus, DeterministicDuration: input.DeterministicDuration,
	})
	if err != nil {
		return SolveRun{}, fmt.Errorf("create solve run: %w", err)
	}
	return solveRun(row)
}

func (tx *Tx) GetSolveRun(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (SolveRun, error) {
	if tx == nil || tx.queries == nil {
		return SolveRun{}, errors.New("get solve run: transaction is nil")
	}
	row, err := tx.queries.GetSolveRun(ctx, db.GetSolveRunParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return SolveRun{}, fmt.Errorf("get solve run: %w", err)
	}
	return solveRun(row)
}

func (tx *Tx) ListSolveRuns(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]SolveRun, error) {
	if tx == nil || tx.queries == nil {
		return nil, errors.New("list solve runs: transaction is nil")
	}
	rows, err := tx.queries.ListSolveRuns(ctx, db.ListSolveRunsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list solve runs: %w", err)
	}
	result := make([]SolveRun, 0, len(rows))
	for _, row := range rows {
		value, err := solveRun(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// ListAllSolveRunsForRegistry is restricted to the Layer 2 tenancy harness.
func (tx *Tx) ListAllSolveRunsForRegistry(ctx context.Context) ([]SolveRun, error) {
	rows, err := tx.queries.ListAllSolveRunsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list solve runs for registry: %w", err)
	}
	result := make([]SolveRun, 0, len(rows))
	for _, row := range rows {
		value, err := solveRun(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// FindSolveRunForRegistry is restricted to the Layer 2 tenancy harness.
func (tx *Tx) FindSolveRunForRegistry(ctx context.Context, id ids.XID) (SolveRun, error) {
	row, err := tx.queries.FindSolveRunForRegistry(ctx, db.FindSolveRunForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return SolveRun{}, err
	}
	return solveRun(row)
}

func solveRun(row db.SolveRun) (SolveRun, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return SolveRun{}, err
	}
	return SolveRun{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID,
		RerunOfSolveRunID: row.RerunOfSolveRunID, ContractVersion: row.ContractVersion, Seed: row.Seed, InputFingerprint: row.InputFingerprint,
		RequestDocument: json.RawMessage(row.RequestDocument), ResponseDocument: json.RawMessage(row.ResponseDocument), SolverStatus: row.SolverStatus,
		DeterministicDuration: row.DeterministicDuration, CreatedAt: createdAt}, nil
}
