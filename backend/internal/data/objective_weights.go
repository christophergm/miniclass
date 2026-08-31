package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ObjectiveWeights is the complete, serializable objective configuration used
// by the assignment engine. It intentionally mirrors the self-contained solve
// document described by ADR 0003 and can later be stored unchanged on a run.
type ObjectiveWeights struct {
	RankHighMax                   int     `json:"rank_high_max"`
	DeficitUnwantedIncrement      float64 `json:"deficit_unwanted_increment"`
	DeficitNeutralIncrement       float64 `json:"deficit_neutral_increment"`
	DeficitAcceptableIncrement    float64 `json:"deficit_acceptable_increment"`
	DeficitInfluence              float64 `json:"deficit_influence"`
	RepeatOfferingPenalty         float64 `json:"repeat_offering_penalty"`
	RepeatInterestAreaPenalty     float64 `json:"repeat_interest_area_penalty"`
	TagPrefersWeight              float64 `json:"tag_prefers_weight"`
	TagDiscouragesWeight          float64 `json:"tag_discourages_weight"`
	PairingPrefersWeight          float64 `json:"pairing_prefers_weight"`
	PairingDiscouragesWeight      float64 `json:"pairing_discourages_weight"`
	BelowMinimumEnrollmentPenalty float64 `json:"below_minimum_enrollment_penalty"`
	TagBalancePenalty             float64 `json:"tag_balance_penalty"`
}

// ObjectiveWeightOverrides has the same shape as ObjectiveWeights, but nil
// fields mean that the programme default remains effective for that field.
type ObjectiveWeightOverrides struct {
	RankHighMax                   *int     `json:"rank_high_max,omitempty"`
	DeficitUnwantedIncrement      *float64 `json:"deficit_unwanted_increment,omitempty"`
	DeficitNeutralIncrement       *float64 `json:"deficit_neutral_increment,omitempty"`
	DeficitAcceptableIncrement    *float64 `json:"deficit_acceptable_increment,omitempty"`
	DeficitInfluence              *float64 `json:"deficit_influence,omitempty"`
	RepeatOfferingPenalty         *float64 `json:"repeat_offering_penalty,omitempty"`
	RepeatInterestAreaPenalty     *float64 `json:"repeat_interest_area_penalty,omitempty"`
	TagPrefersWeight              *float64 `json:"tag_prefers_weight,omitempty"`
	TagDiscouragesWeight          *float64 `json:"tag_discourages_weight,omitempty"`
	PairingPrefersWeight          *float64 `json:"pairing_prefers_weight,omitempty"`
	PairingDiscouragesWeight      *float64 `json:"pairing_discourages_weight,omitempty"`
	BelowMinimumEnrollmentPenalty *float64 `json:"below_minimum_enrollment_penalty,omitempty"`
	TagBalancePenalty             *float64 `json:"tag_balance_penalty,omitempty"`
}

type ProgramObjectiveWeights struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	Weights        ObjectiveWeights
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SessionObjectiveWeightOverrides struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SessionID      ids.XID
	Overrides      ObjectiveWeightOverrides
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ObjectiveWeightsView struct {
	ProgramID ids.XID
	SessionID ids.XID
	Defaults  ObjectiveWeights
	Overrides ObjectiveWeightOverrides
	Effective ObjectiveWeights
}

func (tx *Tx) CreateProgramObjectiveWeights(ctx context.Context, schoolYearID, programID ids.XID) (ProgramObjectiveWeights, error) {
	row, err := tx.queries.CreateProgramObjectiveWeights(ctx, db.CreateProgramObjectiveWeightsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return ProgramObjectiveWeights{}, fmt.Errorf("create program objective weights: %w", err)
	}
	return programObjectiveWeights(row)
}

func (tx *Tx) GetProgramObjectiveWeights(ctx context.Context, schoolYearID, programID ids.XID) (ProgramObjectiveWeights, error) {
	row, err := tx.queries.GetProgramObjectiveWeights(ctx, db.GetProgramObjectiveWeightsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return ProgramObjectiveWeights{}, fmt.Errorf("get program objective weights: %w", err)
	}
	return programObjectiveWeights(row)
}

func (tx *Tx) UpdateProgramObjectiveWeights(ctx context.Context, schoolYearID, programID ids.XID, weights ObjectiveWeights) (ProgramObjectiveWeights, error) {
	row, err := tx.queries.UpdateProgramObjectiveWeights(ctx, db.UpdateProgramObjectiveWeightsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
		RankHighMax: int32(weights.RankHighMax), DeficitUnwantedIncrement: weights.DeficitUnwantedIncrement,
		DeficitNeutralIncrement: weights.DeficitNeutralIncrement, DeficitAcceptableIncrement: weights.DeficitAcceptableIncrement,
		DeficitInfluence: weights.DeficitInfluence, RepeatOfferingPenalty: weights.RepeatOfferingPenalty,
		RepeatInterestAreaPenalty: weights.RepeatInterestAreaPenalty, TagPrefersWeight: weights.TagPrefersWeight,
		TagDiscouragesWeight: weights.TagDiscouragesWeight, PairingPrefersWeight: weights.PairingPrefersWeight,
		PairingDiscouragesWeight: weights.PairingDiscouragesWeight, BelowMinimumEnrollmentPenalty: weights.BelowMinimumEnrollmentPenalty,
		TagBalancePenalty: weights.TagBalancePenalty,
	})
	if err != nil {
		return ProgramObjectiveWeights{}, wrapProgramMutationError("update program objective weights", err)
	}
	return programObjectiveWeights(row)
}

func (tx *Tx) GetSessionObjectiveWeightOverrides(ctx context.Context, schoolYearID, programID, sessionID ids.XID) (SessionObjectiveWeightOverrides, error) {
	row, err := tx.queries.GetSessionObjectiveWeightOverrides(ctx, db.GetSessionObjectiveWeightOverridesParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return SessionObjectiveWeightOverrides{}, fmt.Errorf("get session objective weight overrides: %w", err)
	}
	return sessionObjectiveWeightOverrides(row)
}

func (tx *Tx) UpsertSessionObjectiveWeightOverrides(ctx context.Context, schoolYearID, programID, sessionID ids.XID, overrides ObjectiveWeightOverrides) (SessionObjectiveWeightOverrides, error) {
	row, err := tx.queries.UpsertSessionObjectiveWeightOverrides(ctx, db.UpsertSessionObjectiveWeightOverridesParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
		RankHighMax: int4Param(overrides.RankHighMax), DeficitUnwantedIncrement: float8Param(overrides.DeficitUnwantedIncrement),
		DeficitNeutralIncrement: float8Param(overrides.DeficitNeutralIncrement), DeficitAcceptableIncrement: float8Param(overrides.DeficitAcceptableIncrement),
		DeficitInfluence: float8Param(overrides.DeficitInfluence), RepeatOfferingPenalty: float8Param(overrides.RepeatOfferingPenalty),
		RepeatInterestAreaPenalty: float8Param(overrides.RepeatInterestAreaPenalty), TagPrefersWeight: float8Param(overrides.TagPrefersWeight),
		TagDiscouragesWeight: float8Param(overrides.TagDiscouragesWeight), PairingPrefersWeight: float8Param(overrides.PairingPrefersWeight),
		PairingDiscouragesWeight: float8Param(overrides.PairingDiscouragesWeight), BelowMinimumEnrollmentPenalty: float8Param(overrides.BelowMinimumEnrollmentPenalty),
		TagBalancePenalty: float8Param(overrides.TagBalancePenalty),
	})
	if err != nil {
		return SessionObjectiveWeightOverrides{}, wrapProgramMutationError("upsert session objective weight overrides", err)
	}
	return sessionObjectiveWeightOverrides(row)
}

func (tx *Tx) DeleteSessionObjectiveWeightOverrides(ctx context.Context, schoolYearID, programID, sessionID ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteSessionObjectiveWeightOverrides(ctx, db.DeleteSessionObjectiveWeightOverridesParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return false, wrapProgramMutationError("delete session objective weight overrides", err)
	}
	return rows == 1, nil
}

func programObjectiveWeights(row db.ProgramObjectiveWeight) (ProgramObjectiveWeights, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return ProgramObjectiveWeights{}, err
	}
	updated, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return ProgramObjectiveWeights{}, err
	}
	return ProgramObjectiveWeights{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, Weights: ObjectiveWeights{
		RankHighMax: int(row.RankHighMax), DeficitUnwantedIncrement: row.DeficitUnwantedIncrement, DeficitNeutralIncrement: row.DeficitNeutralIncrement,
		DeficitAcceptableIncrement: row.DeficitAcceptableIncrement, DeficitInfluence: row.DeficitInfluence, RepeatOfferingPenalty: row.RepeatOfferingPenalty,
		RepeatInterestAreaPenalty: row.RepeatInterestAreaPenalty, TagPrefersWeight: row.TagPrefersWeight, TagDiscouragesWeight: row.TagDiscouragesWeight,
		PairingPrefersWeight: row.PairingPrefersWeight, PairingDiscouragesWeight: row.PairingDiscouragesWeight,
		BelowMinimumEnrollmentPenalty: row.BelowMinimumEnrollmentPenalty, TagBalancePenalty: row.TagBalancePenalty,
	}, CreatedAt: created, UpdatedAt: updated}, nil
}

func sessionObjectiveWeightOverrides(row db.SessionObjectiveWeightOverride) (SessionObjectiveWeightOverrides, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return SessionObjectiveWeightOverrides{}, err
	}
	updated, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return SessionObjectiveWeightOverrides{}, err
	}
	return SessionObjectiveWeightOverrides{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, Overrides: ObjectiveWeightOverrides{
		RankHighMax: int4Value(row.RankHighMax), DeficitUnwantedIncrement: float8Value(row.DeficitUnwantedIncrement), DeficitNeutralIncrement: float8Value(row.DeficitNeutralIncrement),
		DeficitAcceptableIncrement: float8Value(row.DeficitAcceptableIncrement), DeficitInfluence: float8Value(row.DeficitInfluence), RepeatOfferingPenalty: float8Value(row.RepeatOfferingPenalty),
		RepeatInterestAreaPenalty: float8Value(row.RepeatInterestAreaPenalty), TagPrefersWeight: float8Value(row.TagPrefersWeight), TagDiscouragesWeight: float8Value(row.TagDiscouragesWeight),
		PairingPrefersWeight: float8Value(row.PairingPrefersWeight), PairingDiscouragesWeight: float8Value(row.PairingDiscouragesWeight),
		BelowMinimumEnrollmentPenalty: float8Value(row.BelowMinimumEnrollmentPenalty), TagBalancePenalty: float8Value(row.TagBalancePenalty),
	}, CreatedAt: created, UpdatedAt: updated}, nil
}

func (w ObjectiveWeights) With(overrides ObjectiveWeightOverrides) ObjectiveWeights {
	if overrides.RankHighMax != nil {
		w.RankHighMax = *overrides.RankHighMax
	}
	if overrides.DeficitUnwantedIncrement != nil {
		w.DeficitUnwantedIncrement = *overrides.DeficitUnwantedIncrement
	}
	if overrides.DeficitNeutralIncrement != nil {
		w.DeficitNeutralIncrement = *overrides.DeficitNeutralIncrement
	}
	if overrides.DeficitAcceptableIncrement != nil {
		w.DeficitAcceptableIncrement = *overrides.DeficitAcceptableIncrement
	}
	if overrides.DeficitInfluence != nil {
		w.DeficitInfluence = *overrides.DeficitInfluence
	}
	if overrides.RepeatOfferingPenalty != nil {
		w.RepeatOfferingPenalty = *overrides.RepeatOfferingPenalty
	}
	if overrides.RepeatInterestAreaPenalty != nil {
		w.RepeatInterestAreaPenalty = *overrides.RepeatInterestAreaPenalty
	}
	if overrides.TagPrefersWeight != nil {
		w.TagPrefersWeight = *overrides.TagPrefersWeight
	}
	if overrides.TagDiscouragesWeight != nil {
		w.TagDiscouragesWeight = *overrides.TagDiscouragesWeight
	}
	if overrides.PairingPrefersWeight != nil {
		w.PairingPrefersWeight = *overrides.PairingPrefersWeight
	}
	if overrides.PairingDiscouragesWeight != nil {
		w.PairingDiscouragesWeight = *overrides.PairingDiscouragesWeight
	}
	if overrides.BelowMinimumEnrollmentPenalty != nil {
		w.BelowMinimumEnrollmentPenalty = *overrides.BelowMinimumEnrollmentPenalty
	}
	if overrides.TagBalancePenalty != nil {
		w.TagBalancePenalty = *overrides.TagBalancePenalty
	}
	return w
}

func int4Param(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}
func float8Param(value *float64) pgtype.Float8 {
	if value == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *value, Valid: true}
}
func int4Value(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int32)
	return &result
}
func float8Value(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

// Registry-only methods keep generated queries behind internal/data while
// allowing the generic Layer 2 harness to prove both configuration tables.
func (tx *Tx) ListAllProgramObjectiveWeightsForRegistry(ctx context.Context) ([]ProgramObjectiveWeights, error) {
	rows, err := tx.queries.ListAllProgramObjectiveWeightsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]ProgramObjectiveWeights, 0, len(rows))
	for _, row := range rows {
		value, err := programObjectiveWeights(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func (tx *Tx) FindProgramObjectiveWeightsForRegistry(ctx context.Context, id ids.XID) (ProgramObjectiveWeights, error) {
	row, err := tx.queries.FindProgramObjectiveWeightsForRegistry(ctx, db.FindProgramObjectiveWeightsForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramObjectiveWeights{}, nil
	}
	if err != nil {
		return ProgramObjectiveWeights{}, err
	}
	return programObjectiveWeights(row)
}
func (tx *Tx) UpdateProgramObjectiveWeightsForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.UpdateProgramObjectiveWeightsForRegistry(ctx, db.UpdateProgramObjectiveWeightsForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}
func (tx *Tx) DeleteProgramObjectiveWeightsForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteProgramObjectiveWeightsForRegistry(ctx, db.DeleteProgramObjectiveWeightsForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}
func (tx *Tx) ListAllSessionObjectiveWeightOverridesForRegistry(ctx context.Context) ([]SessionObjectiveWeightOverrides, error) {
	rows, err := tx.queries.ListAllSessionObjectiveWeightOverridesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]SessionObjectiveWeightOverrides, 0, len(rows))
	for _, row := range rows {
		value, err := sessionObjectiveWeightOverrides(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func (tx *Tx) FindSessionObjectiveWeightOverridesForRegistry(ctx context.Context, id ids.XID) (SessionObjectiveWeightOverrides, error) {
	row, err := tx.queries.FindSessionObjectiveWeightOverridesForRegistry(ctx, db.FindSessionObjectiveWeightOverridesForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionObjectiveWeightOverrides{}, nil
	}
	if err != nil {
		return SessionObjectiveWeightOverrides{}, err
	}
	return sessionObjectiveWeightOverrides(row)
}
func (tx *Tx) UpdateSessionObjectiveWeightOverridesForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.UpdateSessionObjectiveWeightOverridesForRegistry(ctx, db.UpdateSessionObjectiveWeightOverridesForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}
func (tx *Tx) DeleteSessionObjectiveWeightOverridesForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteSessionObjectiveWeightOverridesForRegistry(ctx, db.DeleteSessionObjectiveWeightOverridesForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}
