package handlers

import (
	"context"
	"errors"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
)

type StudentPreferenceContextInput struct {
	SessionPathInput
	StudentID string `path:"studentID" minLength:"1" doc:"Opaque student identifier."`
}

type StudentPreferenceContextOutput struct {
	Body StudentPreferenceContextResponse
}

type StudentPreferenceContextResponse struct {
	Student         StudentPreferenceStudentResponse `json:"student"`
	DraftRevision   int64                            `json:"draft_revision"`
	Comments        []PlacementCommentResponse       `json:"comments"`
	Rankings        StudentRankingsResponse          `json:"rankings"`
	PriorPlacements []StudentPriorPlacementResponse  `json:"prior_placements"`
	InterestProfile StudentInterestProfileResponse   `json:"interest_profile"`
}

type StudentPreferenceStudentResponse struct {
	ID           string                          `json:"id"`
	DisplayName  string                          `json:"display_name"`
	LegalName    string                          `json:"legal_name"`
	GradeLabel   string                          `json:"grade_label"`
	HomeroomName string                          `json:"homeroom_name"`
	Placement    StudentCurrentPlacementResponse `json:"placement"`
	LinkedAdults []StudentLinkedAdultResponse    `json:"linked_adults"`
}

type StudentCurrentPlacementResponse struct {
	OfferingID      *string `json:"offering_id" nullable:"true"`
	OfferingName    string  `json:"offering_name,omitempty"`
	RealizedQuality string  `json:"realized_quality,omitempty"`
}

type StudentLinkedAdultResponse struct {
	DisplayName  string `json:"display_name"`
	Relationship string `json:"relationship"`
}

type StudentRankingsResponse struct {
	RankedChoiceEnabled bool                     `json:"ranked_choice_enabled"`
	Submitted           bool                     `json:"submitted"`
	Rows                []StudentRankingResponse `json:"rows"`
}

type StudentRankingResponse struct {
	OfferingID         string `json:"offering_id"`
	OfferingName       string `json:"offering_name"`
	InterestAreaLabel  string `json:"interest_area_label"`
	Response           string `json:"response" enum:"ranked,interested,not_interested,no_response"`
	Rank               *int   `json:"rank,omitempty" nullable:"true"`
	PotentialQuality   string `json:"potential_quality" enum:"top,high,acceptable,neutral,unwanted"`
	IsCurrentPlacement bool   `json:"is_current_placement"`
}

type StudentPriorPlacementResponse struct {
	SessionID         string                     `json:"session_id"`
	SessionName       string                     `json:"session_name"`
	OfferingID        string                     `json:"offering_id"`
	OfferingName      string                     `json:"offering_name"`
	InterestAreaLabel string                     `json:"interest_area_label"`
	RealizedQuality   string                     `json:"realized_quality"`
	PreferenceLabel   string                     `json:"preference_label"`
	Comments          []PlacementCommentResponse `json:"comments"`
}

type StudentInterestProfileResponse struct {
	Responded      bool                          `json:"responded"`
	VeryInterested []StudentInterestAreaResponse `json:"very_interested"`
	Interested     []StudentInterestAreaResponse `json:"interested"`
	NotInterested  []StudentInterestAreaResponse `json:"not_interested"`
	NotRated       []StudentInterestAreaResponse `json:"not_rated"`
}

type StudentInterestAreaResponse struct {
	ID          string                            `json:"id"`
	Label       string                            `json:"label"`
	SurveyName  string                            `json:"survey_name,omitempty"`
	SubmittedAt *string                           `json:"submitted_at,omitempty" format:"date-time" nullable:"true"`
	Offerings   []StudentInterestOfferingResponse `json:"offerings"`
}

type StudentInterestOfferingResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetStudentPreferenceContext returns the assignment-sidebar projection for a
// single student. Sensitive comment bodies are filtered at this response edge.
func (h *ProgramHandler) GetStudentPreferenceContext(ctx context.Context, input *StudentPreferenceContextInput) (*StudentPreferenceContextOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	value, err := h.service.GetStudentPreferenceContext(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), ids.XID(input.StudentID))
	if err != nil {
		return nil, studentPreferenceContextProblem(err)
	}
	return &StudentPreferenceContextOutput{Body: studentPreferenceContextResponse(value, canViewSensitivePlacementComments(account))}, nil
}

func studentPreferenceContextProblem(err error) error {
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, programservice.ErrStudentNotProgramMember) {
		return sessionNotFound()
	}
	return sessionProblem(err)
}

func studentPreferenceContextResponse(value programservice.StudentPreferenceContext, includeSensitive bool) StudentPreferenceContextResponse {
	student := StudentPreferenceStudentResponse{ID: value.Student.ID, DisplayName: value.Student.DisplayName, LegalName: value.Student.LegalName, GradeLabel: value.Student.GradeLabel, HomeroomName: value.Student.HomeroomName, LinkedAdults: make([]StudentLinkedAdultResponse, 0, len(value.Student.LinkedAdults))}
	if value.Student.Placement.OfferingID != nil {
		id := string(*value.Student.Placement.OfferingID)
		student.Placement.OfferingID = &id
	}
	student.Placement.OfferingName, student.Placement.RealizedQuality = value.Student.Placement.OfferingName, value.Student.Placement.RealizedQuality
	for _, adult := range value.Student.LinkedAdults {
		student.LinkedAdults = append(student.LinkedAdults, StudentLinkedAdultResponse{DisplayName: adult.DisplayName, Relationship: adult.Relationship})
	}
	result := StudentPreferenceContextResponse{Student: student, DraftRevision: value.DraftRevision, Comments: placementCommentResponses(visiblePlacementComments(value.CurrentComments, includeSensitive)), Rankings: studentRankingsResponse(value.Rankings), PriorPlacements: make([]StudentPriorPlacementResponse, 0, len(value.PriorPlacements)), InterestProfile: studentInterestProfileResponse(value.InterestProfile)}
	for _, placement := range value.PriorPlacements {
		result.PriorPlacements = append(result.PriorPlacements, StudentPriorPlacementResponse{SessionID: placement.SessionID, SessionName: placement.SessionName, OfferingID: placement.OfferingID, OfferingName: placement.OfferingName, InterestAreaLabel: placement.InterestAreaLabel, RealizedQuality: placement.RealizedQuality, PreferenceLabel: placement.PreferenceLabel, Comments: placementCommentResponses(visiblePlacementComments(placement.Comments, includeSensitive))})
	}
	return result
}

func studentRankingsResponse(value programservice.StudentRankings) StudentRankingsResponse {
	result := StudentRankingsResponse{RankedChoiceEnabled: value.RankedChoiceEnabled, Submitted: value.Submitted, Rows: make([]StudentRankingResponse, 0, len(value.Rows))}
	for _, row := range value.Rows {
		result.Rows = append(result.Rows, StudentRankingResponse{OfferingID: row.OfferingID, OfferingName: row.OfferingName, InterestAreaLabel: row.InterestAreaLabel, Response: row.Response, Rank: row.Rank, PotentialQuality: string(row.PotentialQuality), IsCurrentPlacement: row.IsCurrentPlacement})
	}
	return result
}

func studentInterestProfileResponse(value programservice.StudentInterestProfile) StudentInterestProfileResponse {
	return StudentInterestProfileResponse{Responded: value.Responded, VeryInterested: studentInterestAreaResponses(value.VeryInterested), Interested: studentInterestAreaResponses(value.Interested), NotInterested: studentInterestAreaResponses(value.NotInterested), NotRated: studentInterestAreaResponses(value.NotRated)}
}

func studentInterestAreaResponses(values []programservice.StudentInterestArea) []StudentInterestAreaResponse {
	result := make([]StudentInterestAreaResponse, 0, len(values))
	for _, value := range values {
		item := StudentInterestAreaResponse{ID: value.ID, Label: value.Label, SurveyName: value.SurveyName, Offerings: make([]StudentInterestOfferingResponse, 0, len(value.Offerings))}
		if value.SubmittedAt != nil {
			formatted := value.SubmittedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			item.SubmittedAt = &formatted
		}
		for _, offering := range value.Offerings {
			item.Offerings = append(item.Offerings, StudentInterestOfferingResponse{ID: offering.ID, Name: offering.Name})
		}
		result = append(result, item)
	}
	return result
}

func placementCommentResponses(values []data.PlacementComment) []PlacementCommentResponse {
	result := make([]PlacementCommentResponse, 0, len(values))
	for _, value := range values {
		result = append(result, placementCommentResponse(value))
	}
	return result
}
