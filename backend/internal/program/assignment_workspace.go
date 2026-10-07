package program

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
)

// AssignmentParticipant adds persisted roster context to a workspace membership.
type AssignmentParticipant struct {
	data.ProgramMembership
	DisplayName  string
	GradeLabel   string
	GradeOrdinal *int
	HomeroomName string
}

// AssignmentWorkspace is the complete persisted draft projection used by
// administrator placement clients (SPEC §§8.6, 16.2, 17.13, 18.1).
// It intentionally represents an empty or incomplete draft as empty slices.
type AssignmentWorkspace struct {
	Session             data.Session
	Participants        []AssignmentParticipant
	Offerings           []data.Offering
	Assignments         []data.Assignment
	Exclusions          []data.AssignmentExclusion
	Overrides           []data.AssignmentOverride
	Comments            []data.PlacementComment
	RankedChoiceAnswers []data.SessionResultAnswer
}

func (s *Service) GetAssignmentWorkspace(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (AssignmentWorkspace, error) {
	if s == nil || s.database == nil {
		return AssignmentWorkspace{}, errors.New("get assignment workspace: data service is nil")
	}
	var result AssignmentWorkspace
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		meetingDates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		session.MeetingDates = meetingDateTimes(meetingDates)
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
		for _, row := range memberships {
			if _, excluded := nonParticipating[row.StudentID]; !excluded {
				participants = append(participants, row)
			}
		}
		students, err := tx.ListStudents(ctx, schoolYearID, true)
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
		offerings, err := tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		assignments, err := tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		exclusions, err := tx.ListAssignmentExclusions(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		overrides, err := tx.ListAssignmentOverrides(ctx, schoolYearID, programID, sessionID)
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
		result = AssignmentWorkspace{Session: session, Participants: assignmentParticipants(participants, students, grades, homerooms), Offerings: offerings, Assignments: assignments, Exclusions: exclusions, Overrides: overrides, Comments: comments, RankedChoiceAnswers: answers}
		return nil
	})
	if err != nil {
		return AssignmentWorkspace{}, fmt.Errorf("get assignment workspace: %w", err)
	}
	return result, nil
}

func assignmentParticipants(memberships []data.ProgramMembership, students []data.Student, grades []data.GradeLevel, homerooms []data.Homeroom) []AssignmentParticipant {
	studentByID := make(map[ids.XID]data.Student, len(students))
	for _, student := range students {
		studentByID[student.ID] = student
	}
	gradeByID := make(map[ids.XID]data.GradeLevel, len(grades))
	for _, grade := range grades {
		gradeByID[grade.ID] = grade
	}
	homeroomByID := make(map[ids.XID]data.Homeroom, len(homerooms))
	for _, homeroom := range homerooms {
		homeroomByID[homeroom.ID] = homeroom
	}
	result := make([]AssignmentParticipant, 0, len(memberships))
	for _, membership := range memberships {
		student := studentByID[membership.StudentID]
		participant := AssignmentParticipant{
			ProgramMembership: membership,
			DisplayName:       people.DisplayName(student.PreferredGivenName, &membership.LegalGivenName, &membership.LegalFamilyName),
			HomeroomName:      homeroomByID[student.HomeroomID].Name,
		}
		if membership.GradeLevelID != nil {
			if grade, ok := gradeByID[*membership.GradeLevelID]; ok {
				participant.GradeLabel = grade.Label
				participant.GradeOrdinal = &grade.Ordinal
			}
		}
		result = append(result, participant)
	}
	return result
}
