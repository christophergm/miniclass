package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (h *ProgramHandler) GetProgramObjectiveWeights(ctx context.Context, input *GetProgramObjectiveWeightsInput) (*ProgramObjectiveWeightsOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	row, err := h.service.GetProgramObjectiveWeights(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID))
	if err != nil {
		return nil, objectiveWeightsProblem(err)
	}
	return &ProgramObjectiveWeightsOutput{Body: programObjectiveWeightsResponse(row)}, nil
}

func (h *ProgramHandler) UpdateProgramObjectiveWeights(ctx context.Context, input *UpdateProgramObjectiveWeightsInput) (*ProgramObjectiveWeightsOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	row, err := h.service.UpdateProgramObjectiveWeights(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), objectiveWeights(input.Body))
	if err != nil {
		return nil, objectiveWeightsProblem(err)
	}
	return &ProgramObjectiveWeightsOutput{Body: programObjectiveWeightsResponse(row)}, nil
}

func (h *ProgramHandler) GetSessionObjectiveWeights(ctx context.Context, input *GetSessionObjectiveWeightsInput) (*SessionObjectiveWeightsOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	row, err := h.service.GetSessionObjectiveWeights(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, objectiveWeightsProblem(err)
	}
	return &SessionObjectiveWeightsOutput{Body: sessionObjectiveWeightsResponse(row)}, nil
}

func (h *ProgramHandler) UpdateSessionObjectiveWeights(ctx context.Context, input *UpdateSessionObjectiveWeightsInput) (*SessionObjectiveWeightsOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	row, err := h.service.UpdateSessionObjectiveWeights(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), objectiveWeightOverrides(input.Body.Overrides), input.Body.Reason)
	if err != nil {
		return nil, objectiveWeightsProblem(err)
	}
	return &SessionObjectiveWeightsOutput{Body: sessionObjectiveWeightsResponse(row)}, nil
}

func (h *ProgramHandler) DeleteSessionObjectiveWeights(ctx context.Context, input *DeleteSessionObjectiveWeightsInput) (*SessionObjectiveWeightsOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	row, err := h.service.ClearSessionObjectiveWeights(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, objectiveWeightsProblem(err)
	}
	return &SessionObjectiveWeightsOutput{Body: sessionObjectiveWeightsResponse(row)}, nil
}

func objectiveWeights(value ObjectiveWeightsResponse) data.ObjectiveWeights {
	return data.ObjectiveWeights{RankHighMax: value.RankHighMax, DeficitUnwantedIncrement: value.DeficitUnwantedIncrement, DeficitNeutralIncrement: value.DeficitNeutralIncrement, DeficitAcceptableIncrement: value.DeficitAcceptableIncrement, DeficitInfluence: value.DeficitInfluence, RepeatOfferingPenalty: value.RepeatOfferingPenalty, RepeatInterestAreaPenalty: value.RepeatInterestAreaPenalty, TagPrefersWeight: value.TagPrefersWeight, TagDiscouragesWeight: value.TagDiscouragesWeight, PairingPrefersWeight: value.PairingPrefersWeight, PairingDiscouragesWeight: value.PairingDiscouragesWeight, BelowMinimumEnrollmentPenalty: value.BelowMinimumEnrollmentPenalty, TagBalancePenalty: value.TagBalancePenalty}
}
func objectiveWeightOverrides(value ObjectiveWeightOverridesResponse) data.ObjectiveWeightOverrides {
	return data.ObjectiveWeightOverrides{RankHighMax: value.RankHighMax, DeficitUnwantedIncrement: value.DeficitUnwantedIncrement, DeficitNeutralIncrement: value.DeficitNeutralIncrement, DeficitAcceptableIncrement: value.DeficitAcceptableIncrement, DeficitInfluence: value.DeficitInfluence, RepeatOfferingPenalty: value.RepeatOfferingPenalty, RepeatInterestAreaPenalty: value.RepeatInterestAreaPenalty, TagPrefersWeight: value.TagPrefersWeight, TagDiscouragesWeight: value.TagDiscouragesWeight, PairingPrefersWeight: value.PairingPrefersWeight, PairingDiscouragesWeight: value.PairingDiscouragesWeight, BelowMinimumEnrollmentPenalty: value.BelowMinimumEnrollmentPenalty, TagBalancePenalty: value.TagBalancePenalty}
}
func objectiveWeightsResponse(value data.ObjectiveWeights) ObjectiveWeightsResponse {
	return ObjectiveWeightsResponse{RankHighMax: value.RankHighMax, DeficitUnwantedIncrement: value.DeficitUnwantedIncrement, DeficitNeutralIncrement: value.DeficitNeutralIncrement, DeficitAcceptableIncrement: value.DeficitAcceptableIncrement, DeficitInfluence: value.DeficitInfluence, RepeatOfferingPenalty: value.RepeatOfferingPenalty, RepeatInterestAreaPenalty: value.RepeatInterestAreaPenalty, TagPrefersWeight: value.TagPrefersWeight, TagDiscouragesWeight: value.TagDiscouragesWeight, PairingPrefersWeight: value.PairingPrefersWeight, PairingDiscouragesWeight: value.PairingDiscouragesWeight, BelowMinimumEnrollmentPenalty: value.BelowMinimumEnrollmentPenalty, TagBalancePenalty: value.TagBalancePenalty}
}
func objectiveWeightOverridesResponse(value data.ObjectiveWeightOverrides) ObjectiveWeightOverridesResponse {
	return ObjectiveWeightOverridesResponse{RankHighMax: value.RankHighMax, DeficitUnwantedIncrement: value.DeficitUnwantedIncrement, DeficitNeutralIncrement: value.DeficitNeutralIncrement, DeficitAcceptableIncrement: value.DeficitAcceptableIncrement, DeficitInfluence: value.DeficitInfluence, RepeatOfferingPenalty: value.RepeatOfferingPenalty, RepeatInterestAreaPenalty: value.RepeatInterestAreaPenalty, TagPrefersWeight: value.TagPrefersWeight, TagDiscouragesWeight: value.TagDiscouragesWeight, PairingPrefersWeight: value.PairingPrefersWeight, PairingDiscouragesWeight: value.PairingDiscouragesWeight, BelowMinimumEnrollmentPenalty: value.BelowMinimumEnrollmentPenalty, TagBalancePenalty: value.TagBalancePenalty}
}
func programObjectiveWeightsResponse(value data.ObjectiveWeightsView) ProgramObjectiveWeightsResponse {
	return ProgramObjectiveWeightsResponse{ProgramID: string(value.ProgramID), Defaults: objectiveWeightsResponse(value.Defaults), Effective: objectiveWeightsResponse(value.Effective)}
}
func sessionObjectiveWeightsResponse(value data.ObjectiveWeightsView) SessionObjectiveWeightsResponse {
	return SessionObjectiveWeightsResponse{ProgramID: string(value.ProgramID), SessionID: string(value.SessionID), Defaults: objectiveWeightsResponse(value.Defaults), Overrides: objectiveWeightOverridesResponse(value.Overrides), Effective: objectiveWeightsResponse(value.Effective)}
}

func objectiveWeightsProblem(err error) error {
	if data.IsSchoolYearClosed(err) {
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	}
	if strings.Contains(err.Error(), "session not found") {
		return sessionNotFound()
	}
	if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "program not found") {
		return programNotFound()
	}
	var pgErr *pgconn.PgError
	if errors.Is(err, programservice.ErrObjectiveWeightsInvalid) || (errors.As(err, &pgErr) && pgErr.Code == "23514") {
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	}
	return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change objective weights")
}
