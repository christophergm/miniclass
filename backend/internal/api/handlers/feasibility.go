package handlers

import (
	"context"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
)

type CatalogGradeGapResponse struct {
	ID               string `json:"id" doc:"Opaque grade-level identifier."`
	Label            string `json:"label"`
	ParticipantCount int    `json:"participant_count"`
}

type CatalogAreaGapResponse struct {
	ID              string `json:"id" doc:"Opaque interest-area identifier."`
	Label           string `json:"label"`
	HighRatingCount int    `json:"high_rating_count"`
}

type CatalogFeasibilityWarningResponse struct {
	ID                           string                    `json:"id" doc:"Stable warning identifier."`
	Severity                     string                    `json:"severity" enum:"warning,info"`
	Message                      string                    `json:"message"`
	ParticipantCount             int                       `json:"participant_count"`
	TotalCapacity                int                       `json:"total_capacity"`
	TotalMinimumViableEnrollment int                       `json:"total_minimum_viable_enrollment"`
	Shortfall                    int                       `json:"shortfall"`
	AffectedGrades               []CatalogGradeGapResponse `json:"affected_grades"`
	AffectedAreas                []CatalogAreaGapResponse  `json:"affected_areas"`
	OfferingIDs                  []string                  `json:"offering_ids" doc:"Opaque offering identifiers affected by this warning."`
}

type CatalogFeasibilityResponse struct {
	ParticipantCount int                                 `json:"participant_count"`
	Warnings         []CatalogFeasibilityWarningResponse `json:"warnings"`
}

type CatalogFeasibilityOutput struct{ Body CatalogFeasibilityResponse }

// GetCatalogFeasibility is a read-only advisory operation. Its warnings do
// not participate in any session or offering validation path, per SPEC §5.2.
func (h *ProgramHandler) GetCatalogFeasibility(ctx context.Context, input *SessionPathInput) (*CatalogFeasibilityOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	result, err := h.service.GetCatalogFeasibility(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &CatalogFeasibilityOutput{Body: catalogFeasibilityResponse(result)}, nil
}

func (h *ProgramHandler) sessionResponseWithFeasibility(ctx context.Context, organizationID string, row data.Session) (SessionResponse, error) {
	result := sessionResponse(row)
	feasibility, err := h.service.GetCatalogFeasibility(ctx, organizationID, row.SchoolYearID, row.ProgramID, row.ID)
	if err != nil {
		return SessionResponse{}, err
	}
	result.FeasibilityWarnings = catalogWarningResponses(feasibility.Warnings)
	return result, nil
}

func catalogFeasibilityResponse(value programservice.CatalogFeasibility) CatalogFeasibilityResponse {
	return CatalogFeasibilityResponse{ParticipantCount: value.ParticipantCount, Warnings: catalogWarningResponses(value.Warnings)}
}

func catalogWarningResponses(values []programservice.CatalogFeasibilityWarning) []CatalogFeasibilityWarningResponse {
	result := make([]CatalogFeasibilityWarningResponse, 0, len(values))
	for _, value := range values {
		grades := make([]CatalogGradeGapResponse, 0, len(value.AffectedGrades))
		for _, grade := range value.AffectedGrades {
			grades = append(grades, CatalogGradeGapResponse{ID: string(grade.ID), Label: grade.Label, ParticipantCount: grade.ParticipantCount})
		}
		areas := make([]CatalogAreaGapResponse, 0, len(value.AffectedAreas))
		for _, area := range value.AffectedAreas {
			areas = append(areas, CatalogAreaGapResponse{ID: string(area.ID), Label: area.Label, HighRatingCount: area.HighRatingCount})
		}
		offeringIDs := make([]string, 0, len(value.OfferingIDs))
		for _, id := range value.OfferingIDs {
			offeringIDs = append(offeringIDs, string(id))
		}
		result = append(result, CatalogFeasibilityWarningResponse{
			ID: value.ID, Severity: value.Severity, Message: value.Message,
			ParticipantCount: value.ParticipantCount, TotalCapacity: value.TotalCapacity,
			TotalMinimumViableEnrollment: value.TotalMinimumViableEnrollment, Shortfall: value.Shortfall,
			AffectedGrades: grades, AffectedAreas: areas, OfferingIDs: offeringIDs,
		})
	}
	return result
}
