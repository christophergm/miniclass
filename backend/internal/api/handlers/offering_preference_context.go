package handlers

import (
	"context"

	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
)

type OfferingPreferenceInterestAreaResponse struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type OfferingPreferenceOfferingResponse struct {
	ID           string                                  `json:"id"`
	Name         string                                  `json:"name"`
	Description  string                                  `json:"description"`
	InterestArea *OfferingPreferenceInterestAreaResponse `json:"interest_area"`
	MinGrade     string                                  `json:"min_grade"`
	MaxGrade     string                                  `json:"max_grade"`
	Enrolled     int                                     `json:"enrolled"`
	Capacity     int                                     `json:"capacity"`
	Warnings     []DraftWarningResponse                  `json:"warnings"`
}

type OfferingPreferencePlacementResponse struct {
	Status          string  `json:"status" enum:"here,unplaced,elsewhere"`
	OfferingID      *string `json:"offering_id" nullable:"true"`
	OfferingName    string  `json:"offering_name,omitempty"`
	RealizedQuality string  `json:"realized_quality,omitempty"`
}

type OfferingPreferenceStudentResponse struct {
	StudentID         string                              `json:"student_id"`
	DisplayName       string                              `json:"display_name"`
	GradeLabel        string                              `json:"grade_label"`
	HomeroomName      string                              `json:"homeroom_name"`
	Category          string                              `json:"category"`
	Rank              *int                                `json:"rank,omitempty" nullable:"true"`
	PotentialQuality  string                              `json:"potential_quality"`
	TopicRatingHint   string                              `json:"topic_rating_hint,omitempty"`
	OutsideGradeRange bool                                `json:"outside_grade_range"`
	Placement         OfferingPreferencePlacementResponse `json:"placement"`
	HasNotes          bool                                `json:"has_notes"`
}

type OfferingPreferenceGroupResponse struct {
	Category string                              `json:"category"`
	Students []OfferingPreferenceStudentResponse `json:"students"`
}

type OfferingPreferenceContextResponse struct {
	Offering      OfferingPreferenceOfferingResponse `json:"offering"`
	DraftRevision int64                              `json:"draft_revision"`
	Comments      []PlacementCommentResponse         `json:"comments"`
	Groups        []OfferingPreferenceGroupResponse  `json:"groups" doc:"Non-empty groups only, in canonical preference display order."`
}

type OfferingPreferenceContextOutput struct {
	Body OfferingPreferenceContextResponse
}

// GetOfferingPreferenceContext returns the administrator assignment sidebar
// projection. Sensitive comments are filtered here at the response boundary.
func (h *ProgramHandler) GetOfferingPreferenceContext(ctx context.Context, input *OfferingPathInput) (*OfferingPreferenceContextOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, offeringNotFound()
	}
	value, err := h.service.GetOfferingPreferenceContext(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.OfferingID))
	if err != nil {
		return nil, offeringProblem(err)
	}
	return &OfferingPreferenceContextOutput{Body: offeringPreferenceContextResponse(value, canViewSensitivePlacementComments(account))}, nil
}

func offeringPreferenceContextResponse(value programservice.OfferingPreferenceContext, includeSensitive bool) OfferingPreferenceContextResponse {
	var interestArea *OfferingPreferenceInterestAreaResponse
	if value.InterestArea != nil {
		interestArea = &OfferingPreferenceInterestAreaResponse{ID: string(value.InterestArea.ID), Label: value.InterestArea.Label}
	}
	visibleComments := visiblePlacementComments(value.Comments, includeSensitive)
	visibleAssignmentComments := visiblePlacementComments(value.AssignmentComments, includeSensitive)
	commentedAssignments := make(map[ids.XID]bool)
	for _, comment := range visibleAssignmentComments {
		if comment.HostType == data.CommentHostAssignment {
			commentedAssignments[comment.HostID] = true
		}
	}
	response := OfferingPreferenceContextResponse{
		Offering:      OfferingPreferenceOfferingResponse{ID: string(value.Offering.ID), Name: value.Offering.Name, Description: value.Offering.Description, InterestArea: interestArea, MinGrade: value.MinGradeLabel, MaxGrade: value.MaxGradeLabel, Enrolled: value.Enrolled, Capacity: value.Offering.Capacity, Warnings: warningResponses(value.Warnings)},
		DraftRevision: value.DraftRevision,
		Comments:      make([]PlacementCommentResponse, 0, len(visibleComments)),
		Groups:        make([]OfferingPreferenceGroupResponse, 0, len(value.Groups)),
	}
	for _, comment := range visibleComments {
		response.Comments = append(response.Comments, placementCommentResponse(comment))
	}
	for _, group := range value.Groups {
		item := OfferingPreferenceGroupResponse{Category: string(group.Category), Students: make([]OfferingPreferenceStudentResponse, 0, len(group.Students))}
		for _, student := range group.Students {
			placement := OfferingPreferencePlacementResponse{Status: student.Placement.Status, OfferingName: student.Placement.OfferingName, RealizedQuality: student.Placement.RealizedQuality}
			if student.Placement.OfferingID != nil {
				id := string(*student.Placement.OfferingID)
				placement.OfferingID = &id
			}
			var rank *int
			if student.Preference.Rank > 0 {
				value := student.Preference.Rank
				rank = &value
			}
			item.Students = append(item.Students, OfferingPreferenceStudentResponse{
				StudentID: string(student.Participant.StudentID), DisplayName: student.Participant.DisplayName, GradeLabel: student.Participant.GradeLabel, HomeroomName: student.Participant.HomeroomName,
				Category: string(student.Preference.Category), Rank: rank, PotentialQuality: string(student.Preference.PotentialQuality), TopicRatingHint: string(student.Preference.TopicRatingHint),
				OutsideGradeRange: student.OutsideGradeRange, Placement: placement, HasNotes: student.Placement.AssignmentID != nil && commentedAssignments[*student.Placement.AssignmentID],
			})
		}
		response.Groups = append(response.Groups, item)
	}
	return response
}

func canViewSensitivePlacementComments(account auth.AccountPrincipal) bool {
	return account.Role == auth.RoleOwner || account.Role == auth.RoleAdministrator
}

func visiblePlacementComments(comments []data.PlacementComment, includeSensitive bool) []data.PlacementComment {
	result := make([]data.PlacementComment, 0, len(comments))
	for _, comment := range comments {
		if comment.Sensitivity == "sensitive" && !includeSensitive {
			continue
		}
		result = append(result, comment)
	}
	return result
}
