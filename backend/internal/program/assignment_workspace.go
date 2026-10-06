package program

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

// AssignmentWorkspace is the complete persisted draft projection used by
// administrator placement clients (SPEC §§8.6, 16.2, 17.13, 18.1).
// It intentionally represents an empty or incomplete draft as empty slices.
type AssignmentWorkspace struct {
	Session             data.Session
	Participants        []data.ProgramMembership
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
		result = AssignmentWorkspace{Session: session, Participants: participants, Offerings: offerings, Assignments: assignments, Exclusions: exclusions, Overrides: overrides, Comments: comments, RankedChoiceAnswers: answers}
		return nil
	})
	if err != nil {
		return AssignmentWorkspace{}, fmt.Errorf("get assignment workspace: %w", err)
	}
	return result, nil
}
