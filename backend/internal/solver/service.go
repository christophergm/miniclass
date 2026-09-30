// Package solver owns solve-run execution and immutable run persistence.
package solver

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/solverclient"
	"github.com/chrismott/miniclass/internal/solvercontract"
)

type Service struct {
	database *data.DB
	client   solverclient.Client
}

func New(database *data.DB, client solverclient.Client) *Service {
	return &Service{database: database, client: client}
}

type StartInput struct {
	SchoolYearID      ids.XID
	ProgramID         ids.XID
	SessionID         ids.XID
	RerunOfSolveRunID *ids.XID
	Request           solvercontract.Request
	Seed              *int64
}

// Start calls the stateless sidecar before atomically persisting its canonical
// input/output with the audit event required by SPEC §20.1–§20.2.
func (s *Service) Start(ctx context.Context, organizationID string, actor audit.Actor, input StartInput) (data.SolveRun, error) {
	if s == nil || s.database == nil || s.client == nil {
		return data.SolveRun{}, errors.New("start solve run: service is not configured")
	}
	if input.Seed == nil {
		seed, err := generatedSeed()
		if err != nil {
			return data.SolveRun{}, err
		}
		input.Request.Seed = seed
	} else {
		input.Request.Seed = *input.Seed
	}
	requestDocument, err := solvercontract.CanonicalJSON(input.Request)
	if err != nil {
		return data.SolveRun{}, fmt.Errorf("start solve run: %w", err)
	}
	fingerprint, err := solvercontract.Fingerprint(input.Request)
	if err != nil {
		return data.SolveRun{}, fmt.Errorf("start solve run: %w", err)
	}
	response, err := s.client.Solve(ctx, input.Request)
	if err != nil {
		return data.SolveRun{}, err
	}
	responseDocument, err := solvercontract.CanonicalResponseJSON(response)
	if err != nil {
		return data.SolveRun{}, fmt.Errorf("start solve run: %w", err)
	}
	var result data.SolveRun
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		result, err = tx.CreateSolveRun(ctx, data.CreateSolveRunInput{SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID,
			RerunOfSolveRunID: input.RerunOfSolveRunID, ContractVersion: input.Request.Version, Seed: input.Request.Seed, InputFingerprint: fingerprint,
			RequestDocument: json.RawMessage(requestDocument), ResponseDocument: json.RawMessage(responseDocument), SolverStatus: response.Status, DeterministicDuration: 0})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSolveRun, ObjectType: "solve_run", ObjectID: &result.ID, SchoolYearID: &input.SchoolYearID,
			ChangeSummary: json.RawMessage(fmt.Sprintf(`{"contract_version":%q,"seed":%d,"input_fingerprint":%q,"solver_status":%q}`, input.Request.Version, input.Request.Seed, fingerprint, response.Status))})
	})
	return result, err
}

// Rerun executes the exact canonical input and seed recorded by a prior run.
// It is intentionally separate from current-session snapshotting: #237 owns
// determining whether live feasibility inputs have changed.
func (s *Service) Rerun(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, runID ids.XID) (data.SolveRun, error) {
	source, err := s.Get(ctx, organizationID, schoolYearID, programID, sessionID, runID)
	if err != nil {
		return data.SolveRun{}, err
	}
	var request solvercontract.Request
	if err := json.Unmarshal(source.RequestDocument, &request); err != nil {
		return data.SolveRun{}, fmt.Errorf("rerun solve run: decode recorded request: %w", err)
	}
	seed := source.Seed
	return s.Start(ctx, organizationID, actor, StartInput{SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
		RerunOfSolveRunID: &source.ID, Request: request, Seed: &seed})
}

func (s *Service) Get(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, runID ids.XID) (data.SolveRun, error) {
	if s == nil || s.database == nil {
		return data.SolveRun{}, errors.New("get solve run: service is not configured")
	}
	var result data.SolveRun
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetSolveRun(ctx, schoolYearID, programID, sessionID, runID)
		return err
	})
	return result, err
}

func generatedSeed() (int64, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return 0, fmt.Errorf("generate solve seed: %w", err)
	}
	return int64(binary.BigEndian.Uint64(bytes[:]) & ((1 << 63) - 1)), nil
}
