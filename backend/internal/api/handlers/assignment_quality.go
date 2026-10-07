package handlers

import (
	"context"

	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
)

type DraftWarningResponse struct {
	ID            string                   `json:"id"`
	Severity      string                   `json:"severity" enum:"warning,info"`
	HostType      string                   `json:"host_type" enum:"assignment,offering,session"`
	HostID        string                   `json:"host_id"`
	AssignmentID  *string                  `json:"assignment_id,omitempty"`
	StudentID     *string                  `json:"student_id,omitempty"`
	OfferingID    *string                  `json:"offering_id,omitempty"`
	Message       string                   `json:"message,omitempty" doc:"Readable explanation of this warning occurrence when available."`
	AffectedAreas []CatalogAreaGapResponse `json:"affected_areas,omitempty" doc:"Missing interest areas for catalog-area-gap; high_rating_count counts participating students rating the area very interested."`
}
type DraftPlacementResponse struct {
	Assignment        AssignmentResponse     `json:"assignment"`
	StudentName       string                 `json:"student_name"`
	CurrentPreference string                 `json:"current_preference,omitempty"`
	Warnings          []DraftWarningResponse `json:"warnings"`
}
type OfferingOccupancyResponse struct {
	OfferingID              string                 `json:"offering_id"`
	Enrolled                int                    `json:"enrolled"`
	Capacity                int                    `json:"capacity"`
	MinimumViableEnrollment *int                   `json:"minimum_viable_enrollment,omitempty"`
	Warnings                []DraftWarningResponse `json:"warnings"`
}
type AssignmentQualityResponse struct {
	Unplaced            []DraftPlacementResponse    `json:"unplaced"`
	Unwanted            []DraftPlacementResponse    `json:"unwanted"`
	NoSignal            []DraftPlacementResponse    `json:"no_signal"`
	Overridden          []DraftPlacementResponse    `json:"overridden"`
	Placements          []DraftPlacementResponse    `json:"placements"`
	Offerings           []OfferingOccupancyResponse `json:"offerings"`
	QualityDistribution map[string]int              `json:"quality_distribution"`
	Warnings            []DraftWarningResponse      `json:"warnings"`
}
type AssignmentQualityOutput struct{ Body AssignmentQualityResponse }

// GetAssignmentQuality is the non-blocking administrator draft review view.
func (h *ProgramHandler) GetAssignmentQuality(ctx context.Context, input *SessionPathInput) (*AssignmentQualityOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	value, err := h.service.GetAssignmentQuality(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID))
	if err != nil {
		return nil, sessionProblem(err)
	}
	return &AssignmentQualityOutput{Body: assignmentQualityResponse(value)}, nil
}

func assignmentQualityResponse(value programservice.AssignmentQuality) AssignmentQualityResponse {
	return AssignmentQualityResponse{Unplaced: placementResponses(value.Unplaced), Unwanted: placementResponses(value.Unwanted), NoSignal: placementResponses(value.NoSignal), Overridden: placementResponses(value.Overridden), Placements: placementResponses(value.Placements), Offerings: occupancyResponses(value.Offerings), QualityDistribution: value.QualityDistribution, Warnings: warningResponses(value.Warnings)}
}
func placementResponses(values []programservice.DraftPlacement) []DraftPlacementResponse {
	r := make([]DraftPlacementResponse, 0, len(values))
	for _, v := range values {
		r = append(r, DraftPlacementResponse{Assignment: AssignmentResponse{ID: string(v.Assignment.ID), StudentID: string(v.Assignment.StudentID), OfferingID: string(v.Assignment.OfferingID), Origin: v.Assignment.Origin, Pinned: v.Assignment.Pinned, RealizedQuality: v.Assignment.RealizedQuality}, StudentName: v.StudentName, CurrentPreference: string(v.CurrentPreference), Warnings: warningResponses(v.Warnings)})
	}
	return r
}
func occupancyResponses(values []programservice.OfferingOccupancy) []OfferingOccupancyResponse {
	r := make([]OfferingOccupancyResponse, 0, len(values))
	for _, v := range values {
		r = append(r, OfferingOccupancyResponse{OfferingID: string(v.OfferingID), Enrolled: v.Enrolled, Capacity: v.Capacity, MinimumViableEnrollment: v.MinimumViableEnrollment, Warnings: warningResponses(v.Warnings)})
	}
	return r
}
func warningResponses(values []programservice.DraftWarning) []DraftWarningResponse {
	r := make([]DraftWarningResponse, 0, len(values))
	for _, v := range values {
		item := DraftWarningResponse{ID: v.ID, Severity: v.Severity, HostType: v.HostType, HostID: string(v.HostID), Message: v.Message}
		for _, area := range v.AffectedAreas {
			item.AffectedAreas = append(item.AffectedAreas, CatalogAreaGapResponse{ID: string(area.ID), Label: area.Label, HighRatingCount: area.HighRatingCount})
		}
		if v.AssignmentID != nil {
			x := string(*v.AssignmentID)
			item.AssignmentID = &x
		}
		if v.StudentID != nil {
			x := string(*v.StudentID)
			item.StudentID = &x
		}
		if v.OfferingID != nil {
			x := string(*v.OfferingID)
			item.OfferingID = &x
		}
		r = append(r, item)
	}
	return r
}
