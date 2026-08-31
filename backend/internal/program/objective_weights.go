package program

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var ErrObjectiveWeightsInvalid = errors.New("objective weights are invalid")

func (s *Service) GetProgramObjectiveWeights(ctx context.Context, organizationID string, schoolYearID, programID ids.XID) (data.ObjectiveWeightsView, error) {
	if s == nil || s.database == nil {
		return data.ObjectiveWeightsView{}, errors.New("get program objective weights: data service is nil")
	}
	var result data.ObjectiveWeightsView
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		row, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result = data.ObjectiveWeightsView{ProgramID: programID, Defaults: row.Weights, Effective: row.Weights}
		return nil
	})
	if err != nil {
		return data.ObjectiveWeightsView{}, fmt.Errorf("get program objective weights: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateProgramObjectiveWeights(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID ids.XID, weights data.ObjectiveWeights) (data.ObjectiveWeightsView, error) {
	if s == nil || s.database == nil {
		return data.ObjectiveWeightsView{}, errors.New("update program objective weights: data service is nil")
	}
	if err := validateObjectiveWeights(weights); err != nil {
		return data.ObjectiveWeightsView{}, err
	}
	var result data.ObjectiveWeightsView
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		current, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		updated, err := tx.UpdateProgramObjectiveWeights(ctx, schoolYearID, programID, weights)
		if err != nil {
			return err
		}
		result = data.ObjectiveWeightsView{ProgramID: programID, Defaults: updated.Weights, Effective: updated.Weights}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionObjectiveWeightsChange, ObjectType: "program_objective_weights", ObjectID: &updated.ID, SchoolYearID: &schoolYearID, ChangeSummary: objectiveWeightsSummary(current.Weights, updated.Weights)})
	})
	if err != nil {
		return data.ObjectiveWeightsView{}, fmt.Errorf("update program objective weights: %w", err)
	}
	return result, nil
}

func (s *Service) GetSessionObjectiveWeights(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (data.ObjectiveWeightsView, error) {
	if s == nil || s.database == nil {
		return data.ObjectiveWeightsView{}, errors.New("get session objective weights: data service is nil")
	}
	var result data.ObjectiveWeightsView
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		defaults, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		overrides, err := tx.GetSessionObjectiveWeightOverrides(ctx, schoolYearID, programID, sessionID)
		if errors.Is(err, pgx.ErrNoRows) {
			result = data.ObjectiveWeightsView{ProgramID: programID, SessionID: sessionID, Defaults: defaults.Weights, Effective: defaults.Weights}
			return nil
		}
		if err != nil {
			return err
		}
		result = data.ObjectiveWeightsView{ProgramID: programID, SessionID: sessionID, Defaults: defaults.Weights, Overrides: overrides.Overrides, Effective: defaults.Weights.With(overrides.Overrides)}
		return nil
	})
	if err != nil {
		return data.ObjectiveWeightsView{}, fmt.Errorf("get session objective weights: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateSessionObjectiveWeights(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, overrides data.ObjectiveWeightOverrides, reason string) (data.ObjectiveWeightsView, error) {
	if s == nil || s.database == nil {
		return data.ObjectiveWeightsView{}, errors.New("update session objective weights: data service is nil")
	}
	if err := validateObjectiveWeightOverrides(overrides); err != nil {
		return data.ObjectiveWeightsView{}, err
	}
	if strings.TrimSpace(reason) == "" {
		return data.ObjectiveWeightsView{}, fmt.Errorf("%w: reason is required for a session override", ErrObjectiveWeightsInvalid)
	}
	var result data.ObjectiveWeightsView
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		defaults, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		previous, err := tx.GetSessionObjectiveWeightOverrides(ctx, schoolYearID, programID, sessionID)
		hadPrevious := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		updated, err := tx.UpsertSessionObjectiveWeightOverrides(ctx, schoolYearID, programID, sessionID, overrides)
		if err != nil {
			return err
		}
		result = data.ObjectiveWeightsView{ProgramID: programID, SessionID: sessionID, Defaults: defaults.Weights, Overrides: updated.Overrides, Effective: defaults.Weights.With(updated.Overrides)}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionObjectiveWeightsChange, ObjectType: "session_objective_weight_overrides", ObjectID: &updated.ID, SchoolYearID: &schoolYearID, Reason: strings.TrimSpace(reason), ChangeSummary: objectiveOverrideSummary(previous.Overrides, updated.Overrides, !hadPrevious)})
	})
	if err != nil {
		return data.ObjectiveWeightsView{}, fmt.Errorf("update session objective weights: %w", err)
	}
	return result, nil
}

func (s *Service) ClearSessionObjectiveWeights(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID) (data.ObjectiveWeightsView, error) {
	if s == nil || s.database == nil {
		return data.ObjectiveWeightsView{}, errors.New("clear session objective weights: data service is nil")
	}
	var result data.ObjectiveWeightsView
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		defaults, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		deleted, err := tx.DeleteSessionObjectiveWeightOverrides(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		result = data.ObjectiveWeightsView{ProgramID: programID, SessionID: sessionID, Defaults: defaults.Weights, Effective: defaults.Weights}
		if !deleted {
			tx.NoAuditRequired("session objective-weight overrides were already clear")
			return nil
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionObjectiveWeightsChange, ObjectType: "session_objective_weight_overrides", SchoolYearID: &schoolYearID, Reason: "organizer cleared session objective-weight overrides", ChangeSummary: mustJSON(map[string]any{"cleared": true})})
	})
	if err != nil {
		return data.ObjectiveWeightsView{}, fmt.Errorf("clear session objective weights: %w", err)
	}
	return result, nil
}

func validateObjectiveWeights(weights data.ObjectiveWeights) error {
	if weights.RankHighMax < 2 {
		return fmt.Errorf("%w: rank_high_max must be at least 2", ErrObjectiveWeightsInvalid)
	}
	values := []float64{weights.DeficitUnwantedIncrement, weights.DeficitNeutralIncrement, weights.DeficitAcceptableIncrement, weights.DeficitInfluence, weights.RepeatOfferingPenalty, weights.RepeatInterestAreaPenalty, weights.TagPrefersWeight, weights.TagDiscouragesWeight, weights.PairingPrefersWeight, weights.PairingDiscouragesWeight, weights.BelowMinimumEnrollmentPenalty, weights.TagBalancePenalty}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("%w: values must be finite and non-negative", ErrObjectiveWeightsInvalid)
		}
	}
	return nil
}

func validateObjectiveWeightOverrides(overrides data.ObjectiveWeightOverrides) error {
	if overrides.RankHighMax != nil && *overrides.RankHighMax < 2 {
		return fmt.Errorf("%w: rank_high_max must be at least 2", ErrObjectiveWeightsInvalid)
	}
	values := []*float64{overrides.DeficitUnwantedIncrement, overrides.DeficitNeutralIncrement, overrides.DeficitAcceptableIncrement, overrides.DeficitInfluence, overrides.RepeatOfferingPenalty, overrides.RepeatInterestAreaPenalty, overrides.TagPrefersWeight, overrides.TagDiscouragesWeight, overrides.PairingPrefersWeight, overrides.PairingDiscouragesWeight, overrides.BelowMinimumEnrollmentPenalty, overrides.TagBalancePenalty}
	for _, value := range values {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0) {
			return fmt.Errorf("%w: values must be finite and non-negative", ErrObjectiveWeightsInvalid)
		}
	}
	return nil
}

func objectiveWeightsSummary(before, after data.ObjectiveWeights) json.RawMessage {
	return mustJSON(map[string]any{"before": before, "after": after})
}
func objectiveOverrideSummary(before, after data.ObjectiveWeightOverrides, created bool) json.RawMessage {
	return mustJSON(map[string]any{"created": created, "before": before, "after": after})
}
