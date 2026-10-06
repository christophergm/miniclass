package solver

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/solvercontract"
	"github.com/jackc/pgx/v5"
)

// Snapshot is the authoritative, consistently-read state used to start or
// reproduce a solve. DraftRevision identifies the exact editing draft used as
// the stability baseline (SPEC §§17.8–17.9, 20.2).
type Snapshot struct {
	DraftRevision int64
	Request       solvercontract.Request
}

const defaultMaxDeterministicTime = 10000

// CompileSnapshot constructs the sidecar document exclusively from persisted,
// tenant-scoped session state. It deliberately retains invalid pins and stale
// placement references so the solver can report them rather than hiding them.
func (s *Service) CompileSnapshot(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (Snapshot, error) {
	if s == nil || s.database == nil {
		return Snapshot{}, errors.New("compile solver snapshot: service is not configured")
	}
	var result Snapshot
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		grades, err := tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		gradeOrdinals := make(map[ids.XID]int, len(grades))
		for _, grade := range grades {
			gradeOrdinals[grade.ID] = grade.Ordinal
		}
		memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		nonParticipations, err := tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		nonParticipating := make(map[ids.XID]struct{}, len(nonParticipations))
		for _, row := range nonParticipations {
			nonParticipating[row.StudentID] = struct{}{}
		}
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		choices := make(map[ids.XID][]solvercontract.RankedChoice)
		for _, answer := range answers {
			choice := solvercontract.RankedChoice{OfferingID: string(answer.OfferingID), Response: string(answer.Answer)}
			if answer.Rank != nil {
				choice.Rank = *answer.Rank
			}
			choices[answer.StudentID] = append(choices[answer.StudentID], choice)
		}
		participants := make([]solvercontract.Participant, 0, len(memberships))
		for _, member := range memberships {
			if _, skipped := nonParticipating[member.StudentID]; skipped {
				continue
			}
			if member.GradeLevelID == nil {
				return fmt.Errorf("participant %q has no grade level", member.StudentID)
			}
			ordinal, found := gradeOrdinals[*member.GradeLevelID]
			if !found {
				return fmt.Errorf("participant %q references an unknown grade level", member.StudentID)
			}
			profile, err := tx.EffectiveInterestProfile(ctx, schoolYearID, programID, member.StudentID)
			if err != nil {
				return err
			}
			interest := make([]solvercontract.InterestRating, 0, len(profile))
			for _, value := range profile {
				if value.Rating == data.InterestProfileUnrated {
					continue
				}
				interest = append(interest, solvercontract.InterestRating{InterestAreaID: string(value.InterestAreaID), Rating: string(value.Rating)})
			}
			participant := solvercontract.Participant{ID: string(member.StudentID), GradeOrdinal: ordinal, InterestProfile: interest}
			if submitted, found := choices[member.StudentID]; found {
				participant.RankedChoices = &solvercontract.RankedChoices{Choices: submitted}
			}
			participants = append(participants, participant)
		}
		offeringRows, err := tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		offerings := make([]solvercontract.Offering, 0, len(offeringRows))
		for _, row := range offeringRows {
			min, minFound := gradeOrdinals[row.MinGradeLevelID]
			max, maxFound := gradeOrdinals[row.MaxGradeLevelID]
			if !minFound || !maxFound {
				return fmt.Errorf("offering %q references an unknown grade level", row.ID)
			}
			var area *string
			if row.InterestAreaID != nil {
				value := string(*row.InterestAreaID)
				area = &value
			}
			offerings = append(offerings, solvercontract.Offering{ID: string(row.ID), Capacity: row.Capacity, MinGradeOrdinal: min, MaxGradeOrdinal: max, InterestAreaID: area})
		}
		assignments, err := tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		overrides, err := tx.ListAssignmentOverrides(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		overridesByAssignment := make(map[ids.XID][]data.AssignmentOverride, len(overrides))
		for _, row := range overrides {
			overridesByAssignment[row.AssignmentID] = append(overridesByAssignment[row.AssignmentID], row)
		}
		pins := make([]solvercontract.PinnedPlacement, 0, len(assignments))
		prior := make([]solvercontract.Placement, 0, len(assignments))
		exceptions := make([]solvercontract.AuthorizedPinnedException, 0, len(overrides))
		for _, row := range assignments {
			placement := solvercontract.Placement{ParticipantID: string(row.StudentID), OfferingID: string(row.OfferingID)}
			prior = append(prior, placement)
			if !row.Pinned {
				continue
			}
			pins = append(pins, solvercontract.PinnedPlacement(placement))
			for _, override := range overridesByAssignment[row.ID] {
				exceptions = append(exceptions, solvercontract.AuthorizedPinnedException{ParticipantID: placement.ParticipantID, OfferingID: placement.OfferingID, Rule: override.Rule})
			}
		}
		exclusionRows, err := tx.ListAssignmentExclusions(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		exclusions := make([]solvercontract.Placement, 0, len(exclusionRows))
		for _, row := range exclusionRows {
			exclusions = append(exclusions, solvercontract.Placement{ParticipantID: string(row.StudentID), OfferingID: string(row.OfferingID)})
		}
		weights, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		override, err := tx.GetSessionObjectiveWeightOverrides(ctx, schoolYearID, programID, sessionID)
		if err == nil {
			weights.Weights = weights.Weights.With(override.Overrides)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		result = Snapshot{DraftRevision: session.DraftRevision, Request: solvercontract.Request{Version: solvercontract.Version, MaxDeterministicTime: defaultMaxDeterministicTime, QualityConfig: solvercontract.QualityConfig{HighRankMax: weights.Weights.RankHighMax}, Participants: participants, Offerings: offerings, Pins: pins, Exclusions: exclusions, AuthorizedExceptions: exceptions, PriorPlacements: prior}}
		return nil
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("compile solver snapshot: %w", err)
	}
	if _, err := solvercontract.CanonicalJSON(result.Request); err != nil {
		return Snapshot{}, fmt.Errorf("compile solver snapshot: %w", err)
	}
	return result, nil
}
