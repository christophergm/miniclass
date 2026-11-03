package program

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

// OfferingPreferencePlacement describes a student's current placement in the
// session. The offering fields are set only for an elsewhere placement.
type OfferingPreferencePlacement struct {
	Status          string
	OfferingID      *ids.XID
	OfferingName    string
	RealizedQuality string
	AssignmentID    *ids.XID
}

// OfferingPreferenceStudent is one visible member of an offering's placement
// sidebar. OutsideGradeRange is retained for a current placement even when it
// falls outside the offering's configured grade window (SPEC §16.5).
type OfferingPreferenceStudent struct {
	Participant       AssignmentParticipant
	Preference        OfferingPreference
	Placement         OfferingPreferencePlacement
	OutsideGradeRange bool
}

// OfferingPreferenceGroup is a non-empty group in canonical preference display
// order. Empty groups are omitted so clients do not need to infer that policy.
type OfferingPreferenceGroup struct {
	Category OfferingPreferenceCategory
	Students []OfferingPreferenceStudent
}

// OfferingPreferenceContext is the read model for an offering assignment
// sidebar. It implements SPEC §§13.3–13.5, §16.5, and §19.1.
type OfferingPreferenceContext struct {
	Offering           data.Offering
	InterestArea       *data.InterestArea
	MinGradeLabel      string
	MaxGradeLabel      string
	Enrolled           int
	Warnings           []DraftWarning
	DraftRevision      int64
	Groups             []OfferingPreferenceGroup
	Comments           []data.PlacementComment
	AssignmentComments []data.PlacementComment
}

// GetOfferingPreferenceContext reads the entire sidebar projection in one
// tenant-scoped transaction. It never uses names as identifiers.
func (s *Service) GetOfferingPreferenceContext(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, offeringID ids.XID) (OfferingPreferenceContext, error) {
	if s == nil || s.database == nil {
		return OfferingPreferenceContext{}, errors.New("get offering preference context: data service is nil")
	}
	var result OfferingPreferenceContext
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		offering, err := tx.GetOffering(ctx, schoolYearID, programID, sessionID, offeringID)
		if err != nil {
			return err
		}
		grades, err := tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		homerooms, err := tx.ListHomerooms(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		areas, err := tx.ListInterestAreas(ctx, schoolYearID, programID, true)
		if err != nil {
			return err
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
		participants := make([]data.ProgramMembership, 0, len(memberships))
		for _, membership := range memberships {
			if _, excluded := nonParticipating[membership.StudentID]; !excluded {
				participants = append(participants, membership)
			}
		}
		students, err := tx.ListStudents(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		assignments, err := tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		participants, assignments = activeAssignmentRoster(participants, assignments, students)
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		comments, err := tx.ListPlacementComments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		weights, err := effectiveSessionWeights(ctx, tx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		offerings, err := tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		submissions := make(map[ids.XID]bool, len(participants))
		profiles := make(map[ids.XID][]data.EffectiveInterestProfileValue, len(participants))
		for _, participant := range participants {
			rows, err := tx.ListRankedChoiceSubmissions(ctx, schoolYearID, programID, sessionID, participant.StudentID)
			if err != nil {
				return err
			}
			submissions[participant.StudentID] = len(rows) > 0
			profiles[participant.StudentID], err = tx.EffectiveInterestProfile(ctx, schoolYearID, programID, participant.StudentID)
			if err != nil {
				return err
			}
		}

		result = evaluateOfferingPreferenceContext(offeringPreferenceContextSnapshot{
			Session: session, Offering: offering, Grades: grades, Homerooms: homerooms, InterestAreas: areas,
			Participants: participants, Students: students, Offerings: offerings, Assignments: assignments, Answers: answers,
			Comments: comments, Submissions: submissions, Profiles: profiles, HighRankMax: weights.RankHighMax,
		})
		return nil
	})
	if err != nil {
		return OfferingPreferenceContext{}, fmt.Errorf("get offering preference context: %w", err)
	}
	return result, nil
}

func effectiveSessionWeights(ctx context.Context, tx *data.Tx, schoolYearID, programID, sessionID ids.XID) (data.ObjectiveWeights, error) {
	defaults, err := tx.GetProgramObjectiveWeights(ctx, schoolYearID, programID)
	if err != nil {
		return data.ObjectiveWeights{}, err
	}
	overrides, err := tx.GetSessionObjectiveWeightOverrides(ctx, schoolYearID, programID, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaults.Weights, nil
	}
	if err != nil {
		return data.ObjectiveWeights{}, err
	}
	return defaults.Weights.With(overrides.Overrides), nil
}

type offeringPreferenceContextSnapshot struct {
	Session       data.Session
	Offering      data.Offering
	Grades        []data.GradeLevel
	Homerooms     []data.Homeroom
	InterestAreas []data.InterestArea
	Participants  []data.ProgramMembership
	Students      []data.Student
	Offerings     []data.Offering
	Assignments   []data.Assignment
	Answers       []data.SessionResultAnswer
	Comments      []data.PlacementComment
	Submissions   map[ids.XID]bool
	Profiles      map[ids.XID][]data.EffectiveInterestProfileValue
	HighRankMax   int
}

func evaluateOfferingPreferenceContext(snapshot offeringPreferenceContextSnapshot) OfferingPreferenceContext {
	gradeByID := make(map[ids.XID]data.GradeLevel, len(snapshot.Grades))
	for _, grade := range snapshot.Grades {
		gradeByID[grade.ID] = grade
	}
	var area *data.InterestArea
	if snapshot.Offering.InterestAreaID != nil {
		for _, candidate := range snapshot.InterestAreas {
			if candidate.ID == *snapshot.Offering.InterestAreaID {
				value := candidate
				area = &value
				break
			}
		}
	}
	assignments := make(map[ids.XID]data.Assignment, len(snapshot.Assignments))
	offerings := make(map[ids.XID]string, len(snapshot.Offerings))
	for _, offering := range snapshot.Offerings {
		offerings[offering.ID] = offering.Name
	}
	enrolled := 0
	for _, assignment := range snapshot.Assignments {
		assignments[assignment.StudentID] = assignment
		if assignment.OfferingID == snapshot.Offering.ID {
			enrolled++
		}
	}
	context := OfferingPreferenceContext{
		Offering: snapshot.Offering, InterestArea: area, Enrolled: enrolled, DraftRevision: snapshot.Session.DraftRevision,
		Warnings: []DraftWarning{}, Groups: []OfferingPreferenceGroup{}, Comments: []data.PlacementComment{}, AssignmentComments: []data.PlacementComment{},
	}
	context.MinGradeLabel = gradeByID[snapshot.Offering.MinGradeLevelID].Label
	context.MaxGradeLabel = gradeByID[snapshot.Offering.MaxGradeLevelID].Label
	if enrolled > snapshot.Offering.Capacity {
		context.Warnings = append(context.Warnings, offeringWarning("capacity-exceeded", snapshot.Offering.ID, "warning"))
	}
	if snapshot.Offering.MinimumViableEnrollment != nil && enrolled < *snapshot.Offering.MinimumViableEnrollment {
		context.Warnings = append(context.Warnings, offeringWarning("below-minimum-enrollment", snapshot.Offering.ID, "warning"))
	}
	for _, comment := range snapshot.Comments {
		if comment.HostType == data.CommentHostOffering && comment.HostID == snapshot.Offering.ID {
			context.Comments = append(context.Comments, comment)
		}
		if comment.HostType == data.CommentHostAssignment {
			context.AssignmentComments = append(context.AssignmentComments, comment)
		}
	}
	participants := assignmentParticipants(snapshot.Participants, snapshot.Students, snapshot.Grades, snapshot.Homerooms)
	groups := map[OfferingPreferenceCategory][]OfferingPreferenceStudent{}
	for _, participant := range participants {
		assignment, assigned := assignments[participant.StudentID]
		gradeInRange := participant.GradeOrdinal != nil && *participant.GradeOrdinal >= gradeByID[snapshot.Offering.MinGradeLevelID].Ordinal && *participant.GradeOrdinal <= gradeByID[snapshot.Offering.MaxGradeLevelID].Ordinal
		if !gradeInRange && (!assigned || assignment.OfferingID != snapshot.Offering.ID) {
			continue
		}
		placement := OfferingPreferencePlacement{Status: "unplaced"}
		if assigned {
			assignmentID := assignment.ID
			placement.AssignmentID = &assignmentID
			placement.RealizedQuality = assignment.RealizedQuality
			if assignment.OfferingID == snapshot.Offering.ID {
				placement.Status = "here"
			} else {
				placement.Status = "elsewhere"
				id := assignment.OfferingID
				placement.OfferingID = &id
				placement.OfferingName = offerings[assignment.OfferingID]
			}
		}
		preference := deriveOfferingStudentPreference(snapshot.Submissions[participant.StudentID], participant.StudentID, snapshot.Offering, snapshot.Answers, snapshot.Profiles[participant.StudentID], snapshot.HighRankMax)
		student := OfferingPreferenceStudent{Participant: participant, Preference: preference, Placement: placement, OutsideGradeRange: !gradeInRange}
		groups[preference.Category] = append(groups[preference.Category], student)
	}
	for _, category := range offeringPreferenceCategories() {
		if students := groups[category]; len(students) > 0 {
			context.Groups = append(context.Groups, OfferingPreferenceGroup{Category: category, Students: students})
		}
	}
	sortOfferingPreferenceGroups(context.Groups)
	return context
}

func deriveOfferingStudentPreference(hasSubmission bool, studentID ids.XID, offering data.Offering, answers []data.SessionResultAnswer, profile []data.EffectiveInterestProfileValue, highRankMax int) OfferingPreference {
	var answer *data.SessionResultAnswer
	for _, candidate := range answers {
		if candidate.StudentID == studentID && candidate.OfferingID == offering.ID {
			value := candidate
			answer = &value
			break
		}
	}
	var rating *data.InterestProfileRating
	if offering.InterestAreaID != nil {
		for _, candidate := range profile {
			if candidate.InterestAreaID == *offering.InterestAreaID {
				value := candidate.Rating
				rating = &value
				break
			}
		}
	}
	return DeriveOfferingPreference(hasSubmission, answer, rating, offering.InterestAreaID != nil, highRankMax)
}

func offeringPreferenceCategories() []OfferingPreferenceCategory {
	return []OfferingPreferenceCategory{OfferingPreferenceRank, OfferingPreferenceTopicVeryInterested, OfferingPreferenceRankedInterested, OfferingPreferenceTopicInterested, OfferingPreferenceTopicNotInterested, OfferingPreferenceRankedNotInterested, OfferingPreferenceUnknown}
}

func sortOfferingPreferenceGroups(groups []OfferingPreferenceGroup) {
	for groupIndex := range groups {
		students := groups[groupIndex].Students
		sort.SliceStable(students, func(i, j int) bool {
			if placementOrder(students[i].Placement.Status) != placementOrder(students[j].Placement.Status) {
				return placementOrder(students[i].Placement.Status) < placementOrder(students[j].Placement.Status)
			}
			return students[i].Participant.DisplayName < students[j].Participant.DisplayName
		})
		groups[groupIndex].Students = students
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return groupPreferenceSortKey(groups[i].Category).Less(groupPreferenceSortKey(groups[j].Category))
	})
}

func groupPreferenceSortKey(category OfferingPreferenceCategory) OfferingPreferenceSortKey {
	for _, candidate := range offeringPreferenceCategories() {
		if category == candidate {
			switch candidate {
			case OfferingPreferenceRank:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderRank}
			case OfferingPreferenceTopicVeryInterested:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicVeryInterested}
			case OfferingPreferenceRankedInterested:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderRankedInterested}
			case OfferingPreferenceTopicInterested:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicInterested}
			case OfferingPreferenceTopicNotInterested:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicNotInterested}
			case OfferingPreferenceRankedNotInterested:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderRankedNotInterested}
			default:
				return OfferingPreferenceSortKey{Category: offeringPreferenceOrderUnknown}
			}
		}
	}
	return OfferingPreferenceSortKey{Category: offeringPreferenceOrderUnknown}
}

func placementOrder(status string) int {
	switch status {
	case "here":
		return 0
	case "unplaced":
		return 1
	default:
		return 2
	}
}
