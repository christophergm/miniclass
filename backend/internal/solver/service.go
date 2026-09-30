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

var ErrInputFingerprintMismatch = errors.New("recorded solve run input fingerprint does not match the supplied current snapshot")

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
	if err := validateSuccessfulResponse(input.Request, response); err != nil {
		return data.SolveRun{}, fmt.Errorf("start solve run: %w", err)
	}
	effectiveWeightsDocument, err := json.Marshal(map[string]int{"high_rank_max": input.Request.QualityConfig.HighRankMax})
	if err != nil {
		return data.SolveRun{}, fmt.Errorf("start solve run: encode effective weights: %w", err)
	}
	metricsDocument, err := metricsDocument(response)
	if err != nil {
		return data.SolveRun{}, fmt.Errorf("start solve run: encode metrics: %w", err)
	}
	var result data.SolveRun
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		result, err = tx.CreateSolveRun(ctx, data.CreateSolveRunInput{SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID,
			RerunOfSolveRunID: input.RerunOfSolveRunID, ContractVersion: input.Request.Version, Seed: input.Request.Seed, InputFingerprint: fingerprint,
			RequestDocument: json.RawMessage(requestDocument), ResponseDocument: json.RawMessage(responseDocument), EffectiveWeightsDocument: effectiveWeightsDocument, MetricsDocument: metricsDocument,
			SolverStatus: response.Status, DeterministicDuration: 0})
		if err != nil {
			return err
		}
		if successfulStatus(response.Status) {
			assignments := assignmentInputs(input, result.ID, response)
			if _, err = tx.ReplaceDraftAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, assignments); err != nil {
				return err
			}
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSolveRun, ObjectType: "solve_run", ObjectID: &result.ID, SchoolYearID: &input.SchoolYearID,
			ChangeSummary: json.RawMessage(fmt.Sprintf(`{"contract_version":%q,"seed":%d,"input_fingerprint":%q,"solver_status":%q}`, input.Request.Version, input.Request.Seed, fingerprint, response.Status))})
	})
	return result, err
}

func successfulStatus(status string) bool { return status == "optimal" || status == "feasible" }

func validateSuccessfulResponse(request solvercontract.Request, response solvercontract.Response) error {
	if !successfulStatus(response.Status) {
		return nil
	}
	participants := make(map[string]struct{}, len(request.Participants))
	for _, participant := range request.Participants {
		participants[participant.ID] = struct{}{}
	}
	offerings := make(map[string]struct{}, len(request.Offerings))
	for _, offering := range request.Offerings {
		offerings[offering.ID] = struct{}{}
	}
	if len(response.Assignments) != len(participants) {
		return errors.New("successful solver response must assign every participant exactly once")
	}
	assigned := make(map[string]string, len(response.Assignments))
	for _, assignment := range response.Assignments {
		if _, found := participants[assignment.ParticipantID]; !found {
			return fmt.Errorf("successful solver response assigns unknown participant %q", assignment.ParticipantID)
		}
		if _, found := offerings[assignment.OfferingID]; !found {
			return fmt.Errorf("successful solver response uses unknown offering %q", assignment.OfferingID)
		}
		assigned[assignment.ParticipantID] = assignment.OfferingID
	}
	for _, pin := range request.Pins {
		if assigned[pin.ParticipantID] != pin.OfferingID {
			return fmt.Errorf("successful solver response did not retain pin for participant %q", pin.ParticipantID)
		}
	}
	return nil
}

func assignmentInputs(input StartInput, runID ids.XID, response solvercontract.Response) []data.CreateAssignmentInput {
	pins := make(map[string]string, len(input.Request.Pins))
	for _, pin := range input.Request.Pins {
		pins[pin.ParticipantID] = pin.OfferingID
	}
	assignments := make([]data.CreateAssignmentInput, 0, len(response.Assignments))
	for _, assignment := range response.Assignments {
		assignments = append(assignments, data.CreateAssignmentInput{SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID,
			StudentID: ids.XID(assignment.ParticipantID), OfferingID: ids.XID(assignment.OfferingID), SolveRunID: runID, Origin: "solver",
			Pinned: pins[assignment.ParticipantID] == assignment.OfferingID, RealizedQuality: assignment.RealizedQuality})
	}
	return assignments
}

func metricsDocument(response solvercontract.Response) ([]byte, error) {
	distribution := map[string]int{
		solvercontract.QualityTop: 0, solvercontract.QualityHigh: 0, solvercontract.QualityAcceptable: 0,
		solvercontract.QualityNeutral: 0, solvercontract.QualityUnwanted: 0,
	}
	for _, assignment := range response.Assignments {
		distribution[assignment.RealizedQuality]++
	}
	return json.Marshal(map[string]any{"quality_distribution": distribution})
}

// Rerun executes a current canonical snapshot with the seed recorded by a
// prior run. It refuses a moved snapshot rather than misrepresenting a new
// answer as a reproduction (SPEC §20.2).
func (s *Service) Rerun(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, runID ids.XID, request solvercontract.Request) (data.SolveRun, error) {
	source, err := s.Get(ctx, organizationID, schoolYearID, programID, sessionID, runID)
	if err != nil {
		return data.SolveRun{}, err
	}
	request.Seed = source.Seed
	fingerprint, err := solvercontract.Fingerprint(request)
	if err != nil {
		return data.SolveRun{}, fmt.Errorf("rerun solve run: canonicalize supplied request: %w", err)
	}
	if fingerprint != source.InputFingerprint {
		return data.SolveRun{}, ErrInputFingerprintMismatch
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
