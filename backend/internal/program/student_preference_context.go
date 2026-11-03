package program

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
)

// StudentPreferenceContext is the assignment-sidebar projection for one
// program member. It implements SPEC §§13.2–13.5 and §17.4.1.
type StudentPreferenceContext struct {
	Student         StudentPreferenceStudent
	DraftRevision   int64
	CurrentComments []data.PlacementComment
	Rankings        StudentRankings
	PriorPlacements []StudentPriorPlacement
	InterestProfile StudentInterestProfile
}

type StudentPreferenceStudent struct {
	ID, DisplayName, LegalName, GradeLabel, HomeroomName string
	Placement                                            StudentCurrentPlacement
	LinkedAdults                                         []StudentLinkedAdult
}

type StudentCurrentPlacement struct {
	OfferingID      *ids.XID
	OfferingName    string
	RealizedQuality string
}

type StudentLinkedAdult struct {
	DisplayName, Relationship string
}

type StudentRankings struct {
	RankedChoiceEnabled bool
	Submitted           bool
	Rows                []StudentRanking
}

type StudentRanking struct {
	OfferingID, OfferingName, InterestAreaLabel, Response string
	Rank                                                  *int
	PotentialQuality                                      PotentialQuality
	IsCurrentPlacement                                    bool
}

type StudentPriorPlacement struct {
	SessionID, SessionName, OfferingID, OfferingName, InterestAreaLabel string
	RealizedQuality, PreferenceLabel                                    string
	Comments                                                            []data.PlacementComment
	firstMeeting                                                        *time.Time
}

type StudentInterestProfile struct {
	Responded                                           bool
	VeryInterested, Interested, NotInterested, NotRated []StudentInterestArea
}

type StudentInterestArea struct {
	ID, Label, SurveyName string
	SubmittedAt           *time.Time
	Offerings             []StudentInterestOffering
}

type StudentInterestOffering struct{ ID, Name string }

// GetStudentPreferenceContext loads the complete projection in one
// tenant-scoped read transaction. Joins use only opaque identifiers.
func (s *Service) GetStudentPreferenceContext(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, studentID ids.XID) (StudentPreferenceContext, error) {
	if s == nil || s.database == nil {
		return StudentPreferenceContext{}, errors.New("get student preference context: data service is nil")
	}
	var result StudentPreferenceContext
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
		if err != nil {
			return err
		}
		memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		if !containsProgramMember(memberships, studentID) {
			return ErrStudentNotProgramMember
		}
		grades, err := tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		homerooms, err := tx.ListHomerooms(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		offerings, err := tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		profileAreas, err := tx.ListInterestAreas(ctx, schoolYearID, programID, false)
		if err != nil {
			return err
		}
		areas, err := tx.ListInterestAreas(ctx, schoolYearID, programID, true)
		if err != nil {
			return err
		}
		assignments, err := tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		comments, err := tx.ListPlacementComments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		rankedSubmissions, err := tx.ListRankedChoiceSubmissions(ctx, schoolYearID, programID, sessionID, studentID)
		if err != nil {
			return err
		}
		profileSubmissions, err := tx.ListInterestProfileSubmissions(ctx, schoolYearID, programID, studentID)
		if err != nil {
			return err
		}
		effectiveProfile, err := tx.EffectiveInterestProfile(ctx, schoolYearID, programID, studentID)
		if err != nil {
			return err
		}
		surveys, err := tx.ListInterestProfileSurveys(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		relationships, err := tx.ListGuardianRelationships(ctx, schoolYearID, data.GuardianRelationshipFilter{StudentID: studentID})
		if err != nil {
			return err
		}
		adults, err := tx.ListAdults(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		weights, err := effectiveSessionWeights(ctx, tx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		result = buildStudentPreferenceContext(studentPreferenceContextSnapshot{
			Session: session, Student: student, Memberships: memberships, Grades: grades, Homerooms: homerooms,
			Offerings: offerings, Areas: areas, ProfileAreas: profileAreas, Assignments: assignments, Comments: comments, Answers: answers,
			RankedSubmissions: rankedSubmissions, ProfileSubmissions: profileSubmissions, EffectiveProfile: effectiveProfile,
			Surveys: surveys, Relationships: relationships, Adults: adults, HighRankMax: weights.RankHighMax,
		})

		sessions, err := tx.ListSessions(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		for _, priorSession := range sessions {
			if priorSession.ID == sessionID || (priorSession.State != data.SessionPublished && priorSession.State != data.SessionComplete) {
				continue
			}
			priorAssignments, err := tx.ListAssignments(ctx, schoolYearID, programID, priorSession.ID)
			if err != nil {
				return err
			}
			priorAssignment, found := assignmentForStudent(priorAssignments, studentID)
			if !found {
				continue
			}
			priorOfferings, err := tx.ListOfferings(ctx, schoolYearID, programID, priorSession.ID)
			if err != nil {
				return err
			}
			priorOffering, found := offeringByID(priorOfferings, priorAssignment.OfferingID)
			if !found {
				continue
			}
			priorComments, err := tx.ListPlacementComments(ctx, schoolYearID, programID, priorSession.ID)
			if err != nil {
				return err
			}
			priorSubmissions, err := tx.ListRankedChoiceSubmissions(ctx, schoolYearID, programID, priorSession.ID, studentID)
			if err != nil {
				return err
			}
			var priorResponses []data.RankedChoiceResponse
			if len(priorSubmissions) > 0 {
				priorResponses, err = tx.ListRankedChoiceResponses(ctx, schoolYearID, programID, priorSession.ID, priorSubmissions[len(priorSubmissions)-1].ID)
				if err != nil {
					return err
				}
			}
			meetingDates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, priorSession.ID)
			if err != nil {
				return err
			}
			result.PriorPlacements = append(result.PriorPlacements, buildStudentPriorPlacement(priorSession, priorAssignment, priorOffering, areas, priorResponses, priorComments, meetingDates))
		}
		sortStudentPriorPlacements(result.PriorPlacements)
		return nil
	})
	if err != nil {
		return StudentPreferenceContext{}, fmt.Errorf("get student preference context: %w", err)
	}
	return result, nil
}

type studentPreferenceContextSnapshot struct {
	Session            data.Session
	Student            data.Student
	Memberships        []data.ProgramMembership
	Grades             []data.GradeLevel
	Homerooms          []data.Homeroom
	Offerings          []data.Offering
	Areas              []data.InterestArea
	ProfileAreas       []data.InterestArea
	Assignments        []data.Assignment
	Comments           []data.PlacementComment
	Answers            []data.SessionResultAnswer
	RankedSubmissions  []data.RankedChoiceSubmission
	ProfileSubmissions []data.InterestProfileSubmission
	EffectiveProfile   []data.EffectiveInterestProfileValue
	Surveys            []data.InterestProfileSurvey
	Relationships      []data.GuardianRelationship
	Adults             []data.Adult
	HighRankMax        int
}

func buildStudentPreferenceContext(s studentPreferenceContextSnapshot) StudentPreferenceContext {
	gradeLabels := make(map[ids.XID]string, len(s.Grades))
	gradeOrdinals := make(map[ids.XID]int, len(s.Grades))
	for _, grade := range s.Grades {
		gradeLabels[grade.ID], gradeOrdinals[grade.ID] = grade.Label, grade.Ordinal
	}
	homeroomNames := make(map[ids.XID]string, len(s.Homerooms))
	for _, room := range s.Homerooms {
		homeroomNames[room.ID] = room.Name
	}
	assignment, assigned := assignmentForStudent(s.Assignments, s.Student.ID)
	current := StudentCurrentPlacement{}
	if assigned {
		if offering, found := offeringByID(s.Offerings, assignment.OfferingID); found {
			id := offering.ID
			current.OfferingID, current.OfferingName, current.RealizedQuality = &id, offering.Name, assignment.RealizedQuality
		}
	}
	result := StudentPreferenceContext{
		Student:       StudentPreferenceStudent{ID: string(s.Student.ID), DisplayName: people.DisplayName(s.Student.PreferredGivenName, &s.Student.LegalGivenName, &s.Student.LegalFamilyName), LegalName: s.Student.LegalGivenName + " " + s.Student.LegalFamilyName, GradeLabel: gradeLabels[valueID(s.Student.GradeLevelID)], HomeroomName: homeroomNames[s.Student.HomeroomID], Placement: current, LinkedAdults: linkedAdults(s.Relationships, s.Adults)},
		DraftRevision: s.Session.DraftRevision, CurrentComments: assignmentComments(s.Comments, assignment), PriorPlacements: []StudentPriorPlacement{},
		Rankings:        StudentRankings{RankedChoiceEnabled: s.Session.RankedChoice != nil, Rows: []StudentRanking{}},
		InterestProfile: buildStudentInterestProfile(s.ProfileSubmissions, s.EffectiveProfile, s.Surveys, s.ProfileAreas, s.Offerings),
	}
	if s.Session.RankedChoice == nil {
		return result
	}
	result.Rankings.Submitted = len(s.RankedSubmissions) > 0
	if !result.Rankings.Submitted {
		return result
	}
	areaLabels := interestAreaLabels(s.Areas)
	for _, offering := range s.Offerings {
		if !studentEligibleForOffering(s.Student, offering, gradeOrdinals) {
			continue
		}
		answer := studentSessionAnswer(s.Answers, s.Student.ID, offering.ID)
		preference := DeriveOfferingPreference(true, answer, nil, offering.InterestAreaID != nil, s.HighRankMax)
		response := string(data.RankedChoiceNoResponse)
		var rank *int
		if answer != nil {
			response, rank = string(answer.Answer), answer.Rank
		}
		result.Rankings.Rows = append(result.Rankings.Rows, StudentRanking{OfferingID: string(offering.ID), OfferingName: offering.Name, InterestAreaLabel: areaLabels[valueID(offering.InterestAreaID)], Response: response, Rank: rank, PotentialQuality: preference.PotentialQuality, IsCurrentPlacement: assigned && assignment.OfferingID == offering.ID})
	}
	sort.SliceStable(result.Rankings.Rows, func(i, j int) bool { return studentRankingLess(result.Rankings.Rows[i], result.Rankings.Rows[j]) })
	return result
}

func buildStudentInterestProfile(submissions []data.InterestProfileSubmission, effective []data.EffectiveInterestProfileValue, surveys []data.InterestProfileSurvey, areas []data.InterestArea, offerings []data.Offering) StudentInterestProfile {
	result := StudentInterestProfile{Responded: len(submissions) > 0, VeryInterested: []StudentInterestArea{}, Interested: []StudentInterestArea{}, NotInterested: []StudentInterestArea{}, NotRated: []StudentInterestArea{}}
	submissionByID := make(map[ids.XID]data.InterestProfileSubmission, len(submissions))
	for _, submission := range submissions {
		submissionByID[submission.ID] = submission
	}
	surveyNames := make(map[ids.XID]string, len(surveys))
	for _, survey := range surveys {
		surveyNames[survey.ID] = survey.Name
	}
	effectiveByArea := make(map[ids.XID]data.EffectiveInterestProfileValue, len(effective))
	for _, value := range effective {
		effectiveByArea[value.InterestAreaID] = value
	}
	for _, area := range areas {
		item := StudentInterestArea{ID: string(area.ID), Label: area.Label, Offerings: []StudentInterestOffering{}}
		for _, offering := range offerings {
			if offering.InterestAreaID != nil && *offering.InterestAreaID == area.ID {
				item.Offerings = append(item.Offerings, StudentInterestOffering{ID: string(offering.ID), Name: offering.Name})
			}
		}
		value, rated := effectiveByArea[area.ID]
		if rated && value.Rating != data.InterestProfileUnrated {
			submission := submissionByID[value.SubmissionID]
			item.SurveyName, item.SubmittedAt = surveyNames[valueID(submission.SurveyID)], &value.SubmittedAt
		}
		switch value.Rating {
		case data.InterestProfileVeryInterested:
			result.VeryInterested = append(result.VeryInterested, item)
		case data.InterestProfileInterested:
			result.Interested = append(result.Interested, item)
		case data.InterestProfileNotInterested:
			result.NotInterested = append(result.NotInterested, item)
		default:
			result.NotRated = append(result.NotRated, item)
		}
	}
	return result
}

func buildStudentPriorPlacement(session data.Session, assignment data.Assignment, offering data.Offering, areas []data.InterestArea, responses []data.RankedChoiceResponse, comments []data.PlacementComment, dates []data.MeetingDate) StudentPriorPlacement {
	item := StudentPriorPlacement{SessionID: string(session.ID), SessionName: session.Name, OfferingID: string(offering.ID), OfferingName: offering.Name, InterestAreaLabel: interestAreaLabels(areas)[valueID(offering.InterestAreaID)], RealizedQuality: assignment.RealizedQuality, PreferenceLabel: historicalPreferenceLabel(assignment.RealizedQuality, rankedResponseForOffering(responses, offering.ID)), Comments: assignmentComments(comments, assignment)}
	if len(dates) > 0 {
		value := dates[0].Date
		item.firstMeeting = &value
	}
	return item
}

func linkedAdults(relationships []data.GuardianRelationship, adults []data.Adult) []StudentLinkedAdult {
	byID := make(map[ids.XID]data.Adult, len(adults))
	for _, adult := range adults {
		byID[adult.ID] = adult
	}
	result := make([]StudentLinkedAdult, 0, len(relationships))
	for _, relationship := range relationships {
		if adult, ok := byID[relationship.AdultID]; ok {
			result = append(result, StudentLinkedAdult{DisplayName: people.DisplayName(adult.PreferredGivenName, &adult.LegalGivenName, &adult.LegalFamilyName), Relationship: string(relationship.RelationshipType)})
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].DisplayName < result[j].DisplayName })
	return result
}

func assignmentComments(comments []data.PlacementComment, assignment data.Assignment) []data.PlacementComment {
	result := []data.PlacementComment{}
	if assignment.ID == "" {
		return result
	}
	for _, comment := range comments {
		if comment.HostType == data.CommentHostAssignment && comment.HostID == assignment.ID {
			result = append(result, comment)
		}
	}
	return result
}

func historicalPreferenceLabel(quality string, response *data.RankedChoiceResponse) string {
	if response != nil {
		switch response.Answer {
		case data.RankedChoiceRanked:
			if response.Rank != nil {
				return ordinal(*response.Rank) + " choice"
			}
		case data.RankedChoiceInterested:
			return "Interested"
		case data.RankedChoiceNotInterested:
			return "Not interested"
		case data.RankedChoiceNoResponse:
			return "No response"
		}
	}
	switch quality {
	case string(PotentialQualityHigh):
		return "Topic match: high"
	case string(PotentialQualityAcceptable):
		return "Topic match: acceptable"
	case string(PotentialQualityUnwanted):
		return "Topic match: unwanted"
	default:
		return "No preference signal"
	}
}

func ordinal(value int) string {
	suffix := "th"
	if value%100 < 11 || value%100 > 13 {
		switch value % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", value, suffix)
}

func studentRankingLess(a, b StudentRanking) bool {
	order := map[string]int{string(data.RankedChoiceRanked): 0, string(data.RankedChoiceInterested): 1, string(data.RankedChoiceNoResponse): 2, string(data.RankedChoiceNotInterested): 3}
	if order[a.Response] != order[b.Response] {
		return order[a.Response] < order[b.Response]
	}
	if a.Response == string(data.RankedChoiceRanked) && a.Rank != nil && b.Rank != nil && *a.Rank != *b.Rank {
		return *a.Rank < *b.Rank
	}
	return a.OfferingName < b.OfferingName
}

func sortStudentPriorPlacements(rows []StudentPriorPlacement) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].firstMeeting == nil || rows[j].firstMeeting == nil {
			if rows[i].firstMeeting == nil && rows[j].firstMeeting != nil {
				return false
			}
			if rows[i].firstMeeting != nil && rows[j].firstMeeting == nil {
				return true
			}
			return rows[i].SessionName < rows[j].SessionName
		}
		return rows[i].firstMeeting.After(*rows[j].firstMeeting)
	})
}

func containsProgramMember(rows []data.ProgramMembership, studentID ids.XID) bool {
	for _, row := range rows {
		if row.StudentID == studentID {
			return true
		}
	}
	return false
}
func assignmentForStudent(rows []data.Assignment, studentID ids.XID) (data.Assignment, bool) {
	for _, row := range rows {
		if row.StudentID == studentID {
			return row, true
		}
	}
	return data.Assignment{}, false
}
func offeringByID(rows []data.Offering, id ids.XID) (data.Offering, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return data.Offering{}, false
}
func rankedResponseForOffering(rows []data.RankedChoiceResponse, id ids.XID) *data.RankedChoiceResponse {
	for _, row := range rows {
		if row.OfferingID == id {
			value := row
			return &value
		}
	}
	return nil
}
func studentSessionAnswer(rows []data.SessionResultAnswer, studentID, offeringID ids.XID) *data.SessionResultAnswer {
	for _, row := range rows {
		if row.StudentID == studentID && row.OfferingID == offeringID {
			value := row
			return &value
		}
	}
	return nil
}
func interestAreaLabels(rows []data.InterestArea) map[ids.XID]string {
	result := make(map[ids.XID]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.Label
	}
	return result
}
func valueID(value *ids.XID) ids.XID {
	if value == nil {
		return ""
	}
	return *value
}
func studentEligibleForOffering(student data.Student, offering data.Offering, ordinals map[ids.XID]int) bool {
	if student.GradeLevelID == nil {
		return false
	}
	grade, ok := ordinals[*student.GradeLevelID]
	return ok && grade >= ordinals[offering.MinGradeLevelID] && grade <= ordinals[offering.MaxGradeLevelID]
}
