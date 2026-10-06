package program

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

// AssignmentOperationInput is the common optimistic-concurrency envelope for
// manual draft changes (SPEC §§16.2, 16.7, 17.12).
type AssignmentOperationInput struct {
	SchoolYearID, ProgramID, SessionID ids.XID
	ExpectedRevision                   int64
	ConfirmViolations                  bool
	Reason                             string
}

type AssignmentOperationResult struct {
	Assignments   []data.Assignment
	DraftRevision int64
}

// HardRuleViolation tells a client to present the final-state warnings and
// resubmit deliberately. It is not a refusal to make the placement.
type HardRuleViolation struct{ Rules []string }

func (e *HardRuleViolation) Error() string {
	return "assignment operation requires confirmation: " + strings.Join(e.Rules, ", ")
}

var ErrDraftRevisionConflict = errors.New("assignment draft revision is stale")

func (s *Service) MoveAssignment(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, studentID, offeringID ids.XID) (AssignmentOperationResult, error) {
	return s.changeAssignments(ctx, organizationID, actor, input, "move", map[ids.XID]ids.XID{studentID: offeringID}, nil)
}

func (s *Service) SwapAssignments(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, firstStudentID, secondStudentID ids.XID) (AssignmentOperationResult, error) {
	if firstStudentID == secondStudentID {
		return AssignmentOperationResult{}, errors.New("swap assignments: students must differ")
	}
	return s.changeAssignments(ctx, organizationID, actor, input, "swap", nil, []ids.XID{firstStudentID, secondStudentID})
}

func (s *Service) SetAssignmentPin(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, studentID ids.XID, pinned bool) (AssignmentOperationResult, error) {
	if s == nil || s.database == nil {
		return AssignmentOperationResult{}, errors.New("set assignment pin: data service is nil")
	}
	var result AssignmentOperationResult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.AdvanceDraftRevision(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, input.ExpectedRevision); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrDraftRevisionConflict
			}
			return err
		}
		assignments, err := tx.ListAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if assignment.StudentID == studentID {
				updated, err := tx.UpdateAssignmentPin(ctx, assignment, pinned)
				if err != nil {
					return err
				}
				if !pinned {
					if err := tx.DeleteAssignmentOverrides(ctx, assignment); err != nil {
						return err
					}
				}
				result.Assignments = assignments
				for i := range result.Assignments {
					if result.Assignments[i].ID == updated.ID {
						result.Assignments[i] = updated
					}
				}
				result.DraftRevision = input.ExpectedRevision + 1
				return tx.Record(ctx, audit.Entry{Action: audit.ActionManualOperation, ObjectType: "assignment", ObjectID: &updated.ID, SchoolYearID: &input.SchoolYearID, Reason: strings.TrimSpace(input.Reason), ChangeSummary: mustJSON(map[string]any{"operation": map[bool]string{true: "pin", false: "unpin"}[pinned], "student_id": studentID, "pinned": pinned})})
			}
		}
		return fmt.Errorf("set assignment pin: assignment for student %q not found", studentID)
	})
	if err != nil {
		return AssignmentOperationResult{}, fmt.Errorf("set assignment pin: %w", err)
	}
	return result, nil
}

func (s *Service) changeAssignments(ctx context.Context, organizationID string, actor audit.Actor, input AssignmentOperationInput, operation string, moves map[ids.XID]ids.XID, swap []ids.XID) (AssignmentOperationResult, error) {
	if s == nil || s.database == nil {
		return AssignmentOperationResult{}, errors.New("change assignments: data service is nil")
	}
	var result AssignmentOperationResult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.AdvanceDraftRevision(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, input.ExpectedRevision); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrDraftRevisionConflict
			}
			return err
		}
		assignments, err := tx.ListAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		if len(swap) != 0 {
			byStudent := assignmentMap(assignments)
			first, firstOK := byStudent[swap[0]]
			second, secondOK := byStudent[swap[1]]
			if !firstOK || !secondOK {
				return errors.New("swap assignments: both students must be assigned")
			}
			moves = map[ids.XID]ids.XID{swap[0]: second.OfferingID, swap[1]: first.OfferingID}
		}
		final := assignmentMap(assignments)
		for student, offering := range moves {
			current, exists := final[student]
			if exists {
				current.OfferingID, current.Pinned, current.Origin, current.SolveRunID, current.RealizedQuality = offering, true, "manual", nil, ""
				final[student] = current
			} else {
				final[student] = data.Assignment{SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID, StudentID: student, OfferingID: offering, Origin: "manual", Pinned: true}
			}
		}
		violations, err := evaluateManualAssignments(ctx, tx, input, final, moves)
		if err != nil {
			return err
		}
		if len(violations) > 0 && !input.ConfirmViolations {
			return &HardRuleViolation{Rules: violations}
		}
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		inputs := assignmentInputs(final, answers)
		updated, err := tx.ReplaceDraftAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, inputs)
		if err != nil {
			return err
		}
		for _, assignment := range updated {
			if _, changed := moves[assignment.StudentID]; changed {
				for _, rule := range violationsForStudent(violations, assignment.StudentID) {
					if _, err := tx.CreateAssignmentOverride(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, assignment.ID, rule, input.Reason, actor.Label); err != nil {
						return err
					}
				}
			}
		}
		result = AssignmentOperationResult{Assignments: updated, DraftRevision: input.ExpectedRevision + 1}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionManualOperation, ObjectType: "assignment", SchoolYearID: &input.SchoolYearID, Reason: strings.TrimSpace(input.Reason), ChangeSummary: mustJSON(map[string]any{"operation": operation, "placements": moves, "violations": violations})})
	})
	if err != nil {
		return AssignmentOperationResult{}, fmt.Errorf("%s assignments: %w", operation, err)
	}
	return result, nil
}

func assignmentMap(rows []data.Assignment) map[ids.XID]data.Assignment {
	result := make(map[ids.XID]data.Assignment, len(rows))
	for _, row := range rows {
		result[row.StudentID] = row
	}
	return result
}

func assignmentInputs(rows map[ids.XID]data.Assignment, answers []data.SessionResultAnswer) []data.CreateAssignmentInput {
	quality := make(map[ids.XID]map[ids.XID]string)
	for _, answer := range answers {
		if answer.Rank != nil {
			if quality[answer.StudentID] == nil {
				quality[answer.StudentID] = map[ids.XID]string{}
			}
			if *answer.Rank == 1 {
				quality[answer.StudentID][answer.OfferingID] = "top"
			} else {
				quality[answer.StudentID][answer.OfferingID] = "high"
			}
		}
	}
	students := make([]ids.XID, 0, len(rows))
	for student := range rows {
		students = append(students, student)
	}
	sort.Slice(students, func(i, j int) bool { return students[i] < students[j] })
	result := make([]data.CreateAssignmentInput, 0, len(rows))
	for _, student := range students {
		row := rows[student]
		realized := row.RealizedQuality
		if realized == "" {
			realized = "neutral"
		}
		if row.RealizedQuality == "" && quality[student] != nil && quality[student][row.OfferingID] != "" {
			realized = quality[student][row.OfferingID]
		}
		result = append(result, data.CreateAssignmentInput{SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, StudentID: row.StudentID, OfferingID: row.OfferingID, SolveRunID: row.SolveRunID, Origin: row.Origin, Pinned: row.Pinned, RealizedQuality: realized})
	}
	return result
}

func evaluateManualAssignments(ctx context.Context, tx *data.Tx, input AssignmentOperationInput, final map[ids.XID]data.Assignment, changed map[ids.XID]ids.XID) ([]string, error) {
	members, err := tx.ListProgramMemberships(ctx, input.SchoolYearID, input.ProgramID)
	if err != nil {
		return nil, err
	}
	member := map[ids.XID]data.ProgramMembership{}
	for _, row := range members {
		member[row.StudentID] = row
	}
	nonParticipants, err := tx.ListSessionNonParticipations(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
	if err != nil {
		return nil, err
	}
	absent := map[ids.XID]bool{}
	for _, row := range nonParticipants {
		absent[row.StudentID] = true
	}
	offerings, err := tx.ListOfferings(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
	if err != nil {
		return nil, err
	}
	offering := map[ids.XID]data.Offering{}
	for _, row := range offerings {
		offering[row.ID] = row
	}
	grades, err := tx.ListGradeLevels(ctx, input.SchoolYearID, true)
	if err != nil {
		return nil, err
	}
	grade := map[ids.XID]int{}
	for _, row := range grades {
		grade[row.ID] = row.Ordinal
	}
	exclusions, err := tx.ListAssignmentExclusions(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
	if err != nil {
		return nil, err
	}
	excluded := map[[2]ids.XID]bool{}
	for _, row := range exclusions {
		excluded[[2]ids.XID{row.StudentID, row.OfferingID}] = true
	}
	counts := map[ids.XID]int{}
	for _, row := range final {
		counts[row.OfferingID]++
	}
	violations := []string{}
	for student, target := range changed {
		membership, ok := member[student]
		if !ok || absent[student] {
			return nil, fmt.Errorf("student %q is not participating", student)
		}
		currentOffering, ok := offering[target]
		if !ok {
			return nil, fmt.Errorf("offering %q not found", target)
		}
		if counts[target] > currentOffering.Capacity {
			violations = append(violations, string(student)+":capacity")
		}
		if membership.GradeLevelID == nil || grade[*membership.GradeLevelID] < grade[currentOffering.MinGradeLevelID] || grade[*membership.GradeLevelID] > grade[currentOffering.MaxGradeLevelID] {
			violations = append(violations, string(student)+":grade-window")
		}
		if excluded[[2]ids.XID{student, target}] {
			violations = append(violations, string(student)+":exclusion")
		}
	}
	return violations, nil
}

func violationsForStudent(violations []string, student ids.XID) []string {
	prefix := string(student) + ":"
	result := []string{}
	for _, violation := range violations {
		if strings.HasPrefix(violation, prefix) {
			result = append(result, strings.TrimPrefix(violation, prefix))
		}
	}
	return result
}
