package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	solverservice "github.com/chrismott/miniclass/internal/solver"
	"github.com/chrismott/miniclass/internal/solverclient"
	"github.com/jackc/pgx/v5"
)

type SolveRunResponse struct {
	ID                       string          `json:"id"`
	SchoolYearID             string          `json:"school_year_id"`
	ProgramID                string          `json:"program_id"`
	SessionID                string          `json:"session_id"`
	RerunOfSolveRunID        *string         `json:"rerun_of_solve_run_id,omitempty"`
	ContractVersion          string          `json:"contract_version"`
	Seed                     int64           `json:"seed"`
	InputFingerprint         string          `json:"input_fingerprint"`
	RequestDocument          json.RawMessage `json:"request_document"`
	ResponseDocument         json.RawMessage `json:"response_document"`
	EffectiveWeightsDocument json.RawMessage `json:"effective_weights_document"`
	MetricsDocument          json.RawMessage `json:"metrics_document"`
	SolverStatus             string          `json:"solver_status"`
	ApplicationStatus        string          `json:"application_status"`
	DeterministicDuration    float64         `json:"deterministic_duration"`
	CreatedAt                time.Time       `json:"created_at"`
}
type SolveRunOutput struct{ Body SolveRunResponse }
type SolveRunPathInput struct {
	SessionPathInput
	RunID string `path:"runID" minLength:"1"`
}
type StartSolveRunInput struct {
	SessionPathInput
	Body struct {
		Seed *int64 `json:"seed,omitempty"`
	}
}
type RerunSolveRunInput struct {
	SessionPathInput
	RunID string `path:"runID" minLength:"1"`
}

type SolveRunHandler struct{ service SolveRunService }

func NewSolveRunHandler(service SolveRunService) *SolveRunHandler {
	return &SolveRunHandler{service: service}
}

func (h *SolveRunHandler) Start(ctx context.Context, input *StartSolveRunInput) (*SolveRunOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusServiceUnavailable, problems.SolverUnavailable, "solver is not configured")
	}
	row, err := h.service.StartAuthoritative(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), input.Body.Seed)
	if err != nil {
		return nil, solveRunProblem(err)
	}
	return &SolveRunOutput{Body: solveRunResponse(row)}, nil
}
func (h *SolveRunHandler) Get(ctx context.Context, input *SolveRunPathInput) (*SolveRunOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusServiceUnavailable, problems.SolverUnavailable, "solver is not configured")
	}
	row, err := h.service.Get(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.RunID))
	if err != nil {
		return nil, solveRunProblem(err)
	}
	return &SolveRunOutput{Body: solveRunResponse(row)}, nil
}
func (h *SolveRunHandler) Rerun(ctx context.Context, input *RerunSolveRunInput) (*SolveRunOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusServiceUnavailable, problems.SolverUnavailable, "solver is not configured")
	}
	row, err := h.service.RerunAuthoritative(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.RunID))
	if err != nil {
		return nil, solveRunProblem(err)
	}
	return &SolveRunOutput{Body: solveRunResponse(row)}, nil
}
func solveRunResponse(row data.SolveRun) SolveRunResponse {
	var source *string
	if row.RerunOfSolveRunID != nil {
		value := string(*row.RerunOfSolveRunID)
		source = &value
	}
	return SolveRunResponse{ID: string(row.ID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), SessionID: string(row.SessionID), RerunOfSolveRunID: source, ContractVersion: row.ContractVersion, Seed: row.Seed, InputFingerprint: row.InputFingerprint, RequestDocument: row.RequestDocument, ResponseDocument: row.ResponseDocument, EffectiveWeightsDocument: row.EffectiveWeightsDocument, MetricsDocument: row.MetricsDocument, SolverStatus: row.SolverStatus, ApplicationStatus: row.ApplicationStatus, DeterministicDuration: row.DeterministicDuration, CreatedAt: row.CreatedAt}
}
func solveRunProblem(err error) error {
	if errors.Is(err, solverclient.ErrUnavailable) {
		return problems.New(http.StatusServiceUnavailable, problems.SolverUnavailable, "solver sidecar is unavailable")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "solve run not found")
	}
	if errors.Is(err, solverservice.ErrInputFingerprintMismatch) {
		return problems.New(http.StatusConflict, problems.SolveRunInputMismatch, "the current solver inputs differ from the recorded run")
	}
	if errors.Is(err, solverservice.ErrDraftRevisionChanged) {
		return problems.New(http.StatusConflict, problems.SolveRunInputMismatch, "the session draft changed while the solver was running")
	}
	return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to process solve run")
}
