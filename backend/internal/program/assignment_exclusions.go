package program

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

// AssignmentExclusionResult reports an edit together with any placement that
// now conflicts. Authoring a rule never moves a student (SPEC §17.12).
type AssignmentExclusionResult struct {
	Exclusions             []data.AssignmentExclusion
	ConflictingAssignments []data.Assignment
	DraftRevision          int64
}

func (s *Service) AddAssignmentExclusion(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, studentID, offeringID ids.XID) (AssignmentExclusionResult, error) {
	return s.changeAssignmentExclusion(ctx, organizationID, actor, input, studentID, offeringID, "", true)
}

func (s *Service) RemoveAssignmentExclusion(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, exclusionID ids.XID) (AssignmentExclusionResult, error) {
	return s.changeAssignmentExclusion(ctx, organizationID, actor, input, "", "", exclusionID, false)
}

func (s *Service) changeAssignmentExclusion(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, studentID, offeringID, exclusionID ids.XID, add bool) (AssignmentExclusionResult, error) {
	if s == nil || s.database == nil {
		return AssignmentExclusionResult{}, errors.New("change assignment exclusion: data service is nil")
	}
	var result AssignmentExclusionResult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		exclusions, err := tx.ListAssignmentExclusions(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		var target *data.AssignmentExclusion
		for i := range exclusions {
			if (add && exclusions[i].StudentID == studentID && exclusions[i].OfferingID == offeringID) || (!add && exclusions[i].ID == exclusionID) {
				target = &exclusions[i]
				break
			}
		}
		if target != nil && add {
			tx.NoAuditRequired("assignment exclusion was already present")
			return assignmentExclusionResult(ctx, tx, input, exclusions, session.DraftRevision, studentID, offeringID, &result)
		}
		if target == nil && !add {
			tx.NoAuditRequired("assignment exclusion was already absent")
			return assignmentExclusionResult(ctx, tx, input, exclusions, session.DraftRevision, "", "", &result)
		}
		if _, err := tx.AdvanceDraftRevision(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, input.ExpectedRevision); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrDraftRevisionConflict
			}
			return err
		}
		if add {
			if err := validateAssignmentExclusionScope(ctx, tx, input, studentID, offeringID); err != nil {
				return err
			}
			created, err := tx.CreateAssignmentExclusion(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, studentID, offeringID)
			if err != nil {
				return err
			}
			exclusions = append(exclusions, created)
			if err := assignmentExclusionResult(ctx, tx, input, exclusions, input.ExpectedRevision+1, studentID, offeringID, &result); err != nil {
				return err
			}
			return tx.Record(ctx, audit.Entry{Action: audit.ActionExclusionChange, ObjectType: "assignment_exclusion", ObjectID: &created.ID, SchoolYearID: &input.SchoolYearID, Reason: strings.TrimSpace(input.Reason), ChangeSummary: mustJSON(map[string]any{"operation": "add", "student_id": studentID, "offering_id": offeringID, "conflicting_assignment_ids": assignmentIDs(result.ConflictingAssignments)})})
		}
		if _, err := tx.DeleteAssignmentExclusion(ctx, *target); err != nil {
			return err
		}
		assignments, err := tx.ListAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if assignment.StudentID == target.StudentID && assignment.OfferingID == target.OfferingID {
				if err := tx.DeleteAssignmentOverrideRule(ctx, assignment, "exclusion"); err != nil {
					return err
				}
			}
		}
		kept := make([]data.AssignmentExclusion, 0, len(exclusions)-1)
		for _, exclusion := range exclusions {
			if exclusion.ID != target.ID {
				kept = append(kept, exclusion)
			}
		}
		result = AssignmentExclusionResult{Exclusions: kept, ConflictingAssignments: []data.Assignment{}, DraftRevision: input.ExpectedRevision + 1}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionExclusionChange, ObjectType: "assignment_exclusion", ObjectID: &target.ID, SchoolYearID: &input.SchoolYearID, Reason: strings.TrimSpace(input.Reason), ChangeSummary: mustJSON(map[string]any{"operation": "remove", "student_id": target.StudentID, "offering_id": target.OfferingID})})
	})
	if err != nil {
		return AssignmentExclusionResult{}, fmt.Errorf("change assignment exclusion: %w", err)
	}
	return result, nil
}

func assignmentExclusionResult(ctx context.Context, tx *data.Tx, input AssignmentOperationInput, exclusions []data.AssignmentExclusion, revision int64, studentID, offeringID ids.XID, result *AssignmentExclusionResult) error {
	assignments, err := tx.ListAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
	if err != nil {
		return err
	}
	conflicts := make([]data.Assignment, 0, 1)
	for _, assignment := range assignments {
		if assignment.StudentID == studentID && assignment.OfferingID == offeringID {
			conflicts = append(conflicts, assignment)
		}
	}
	*result = AssignmentExclusionResult{Exclusions: exclusions, ConflictingAssignments: conflicts, DraftRevision: revision}
	return nil
}

func validateAssignmentExclusionScope(ctx context.Context, tx *data.Tx, input AssignmentOperationInput, studentID, offeringID ids.XID) error {
	members, err := tx.ListProgramMemberships(ctx, input.SchoolYearID, input.ProgramID)
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.StudentID == studentID {
			goto offering
		}
	}
	return fmt.Errorf("student %q is not a member of this program", studentID)

offering:
	offerings, err := tx.ListOfferings(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
	if err != nil {
		return err
	}
	for _, offering := range offerings {
		if offering.ID == offeringID {
			return nil
		}
	}
	return fmt.Errorf("offering %q not found", offeringID)
}

func assignmentIDs(assignments []data.Assignment) []ids.XID {
	result := make([]ids.XID, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, assignment.ID)
	}
	return result
}
