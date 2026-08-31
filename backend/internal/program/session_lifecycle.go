package program

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

var (
	ErrSessionTransitionInvalid        = errors.New("session transition is not permitted")
	ErrSessionTransitionGate           = errors.New("session transition gate failed")
	ErrSessionTransitionReasonRequired = errors.New("a reason is required to confirm a backward session transition")
	ErrSessionReadOnly                 = errors.New("the session is complete and read-only")
)

// SessionTransitionInput is the organizer's requested lifecycle change. A
// backward transition is previewed when Confirm is false, so the caller can
// show the invalidation summary before applying the change.
type SessionTransitionInput struct {
	NextState data.SessionState
	Reason    string
	Confirm   bool
}

// SessionTransitionWarning describes a non-blocking consequence of a
// transition. Warnings are returned during the confirmation preview and again
// after a backward transition is applied, so clients do not need to rebuild
// the explanation from state alone.
type SessionTransitionWarning struct {
	Code                string
	Message             string
	InvalidationSummary []string
}

// SessionTransitionResult is returned by both the preview and apply paths.
// Applied is false only for a backward transition awaiting confirmation.
type SessionTransitionResult struct {
	Session              data.Session
	FromState            data.SessionState
	ToState              data.SessionState
	Applied              bool
	RequiresConfirmation bool
	Warnings             []SessionTransitionWarning
}

// SessionTransitionPlan is the pure state-machine decision. It is deliberately
// independent of HTTP and persistence so every edge can be tested as a table.
type SessionTransitionPlan struct {
	FromState                 data.SessionState
	ToState                   data.SessionState
	Backward                  bool
	MarkDraftAssignmentsStale bool
	Warnings                  []SessionTransitionWarning
}

// LegalSessionTransitions returns the only states reachable from state. The
// ordering follows the state diagram in SPEC §14.6.
func LegalSessionTransitions(state data.SessionState) []data.SessionState {
	var result []data.SessionState
	switch state {
	case data.SessionPlanning:
		result = []data.SessionState{data.SessionCatalogPublished}
	case data.SessionCatalogPublished:
		result = []data.SessionState{data.SessionVotingOpen, data.SessionAssigning}
	case data.SessionVotingOpen:
		result = []data.SessionState{data.SessionVotingClosed}
	case data.SessionVotingClosed:
		result = []data.SessionState{data.SessionAssigning, data.SessionVotingOpen}
	case data.SessionAssigning:
		result = []data.SessionState{data.SessionPublished, data.SessionVotingClosed}
	case data.SessionPublished:
		result = []data.SessionState{data.SessionComplete, data.SessionAssigning}
	case data.SessionComplete:
		result = nil
	}
	return slices.Clone(result)
}

func IsLegalSessionTransition(from, to data.SessionState) bool {
	return slices.Contains(LegalSessionTransitions(from), to)
}

func IsBackwardSessionTransition(from, to data.SessionState) bool {
	return (from == data.SessionVotingClosed && to == data.SessionVotingOpen) ||
		(from == data.SessionAssigning && to == data.SessionVotingClosed) ||
		(from == data.SessionPublished && to == data.SessionAssigning)
}

// PlanSessionTransition applies the state gates currently available to this
// phase. The catalog count is enough to enforce the explicit prohibition on
// opening voting with no catalog; assignment and ranked-choice facts will be
// added by their owning phases without changing this state graph.
func PlanSessionTransition(from, to data.SessionState, offeringCount int, draftAssignmentsStale bool) (SessionTransitionPlan, error) {
	plan := SessionTransitionPlan{FromState: from, ToState: to}
	if !IsLegalSessionTransition(from, to) {
		allowed := LegalSessionTransitions(from)
		return plan, fmt.Errorf("%w: %s cannot move to %s (allowed next states: %s)", ErrSessionTransitionInvalid, sessionStateLabel(from), sessionStateLabel(to), sessionStateList(allowed))
	}

	if from == data.SessionCatalogPublished && to == data.SessionVotingOpen && offeringCount == 0 {
		return plan, fmt.Errorf("%w: voting cannot open until the catalog has at least one offering", ErrSessionTransitionGate)
	}
	if from == data.SessionAssigning && to == data.SessionPublished && draftAssignmentsStale {
		return plan, fmt.Errorf("%w: stale draft assignments must be regenerated before the session can be published", ErrSessionTransitionGate)
	}

	plan.Backward = IsBackwardSessionTransition(from, to)
	if !plan.Backward {
		return plan, nil
	}

	plan.Warnings = append(plan.Warnings, SessionTransitionWarning{
		Code:                "backward-transition",
		Message:             fmt.Sprintf("Moving from %s back to %s is allowed, but downstream work may need to be redone.", sessionStateLabel(from), sessionStateLabel(to)),
		InvalidationSummary: []string{"Downstream lifecycle work is no longer current."},
	})
	// A backward transition can expose draft assignment data that was computed
	// from an earlier view of the session. Preserve it and mark it explicitly;
	// never delete it as a side effect of reopening an earlier stage.
	if from == data.SessionAssigning || from == data.SessionPublished || draftAssignmentsStale {
		plan.MarkDraftAssignmentsStale = true
		plan.Warnings = append(plan.Warnings, SessionTransitionWarning{
			Code:                "stale-draft",
			Message:             "Draft assignments are retained and marked stale; they were computed from superseded session inputs and must be regenerated before publication.",
			InvalidationSummary: []string{"Draft assignments are marked stale, not deleted."},
		})
	}
	if from == data.SessionPublished {
		plan.Warnings = append(plan.Warnings, SessionTransitionWarning{
			Code:                "published-links-invalidated",
			Message:             "Published external links will stop resolving while this session is no longer Published; publish the revised session to restore visibility.",
			InvalidationSummary: []string{"Published links stop resolving until the session is published again."},
		})
	}
	return plan, nil
}

func ensureSessionMutable(session data.Session) error {
	if session.State == data.SessionComplete {
		return ErrSessionReadOnly
	}
	return nil
}

// TransitionSession changes lifecycle state atomically with its audit record.
// A backward request without confirmation is a read-only preview, not a
// rejected mutation, which keeps the warning flow compatible with §5.2.
func (s *Service) TransitionSession(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID ids.XID, input SessionTransitionInput) (SessionTransitionResult, error) {
	if s == nil || s.database == nil {
		return SessionTransitionResult{}, errors.New("transition session: data service is nil")
	}

	if !input.Confirm {
		var result SessionTransitionResult
		applyForward := false
		err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
			current, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
			if err != nil {
				return err
			}
			plan, err := sessionTransitionPlan(ctx, tx, current, input.NextState)
			if err != nil {
				return err
			}
			if !plan.Backward {
				applyForward = true
				return nil
			}
			if err := loadSessionTransitionDates(ctx, tx, &current); err != nil {
				return err
			}
			result = SessionTransitionResult{Session: current, FromState: plan.FromState, ToState: plan.ToState, RequiresConfirmation: true, Warnings: plan.Warnings}
			return nil
		})
		if err != nil {
			return SessionTransitionResult{}, fmt.Errorf("preview session transition: %w", err)
		}
		if !applyForward {
			return result, nil
		}
	}

	var result SessionTransitionResult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSessionForUpdate(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		plan, err := sessionTransitionPlan(ctx, tx, current, input.NextState)
		if err != nil {
			return err
		}
		if plan.Backward && strings.TrimSpace(input.Reason) == "" {
			return ErrSessionTransitionReasonRequired
		}
		updated, err := tx.UpdateSessionLifecycle(ctx, schoolYearID, programID, sessionID, input.NextState, current.DraftAssignmentsStale || plan.MarkDraftAssignmentsStale)
		if err != nil {
			return err
		}
		if err := loadSessionTransitionDates(ctx, tx, &updated); err != nil {
			return err
		}
		result = SessionTransitionResult{Session: updated, FromState: plan.FromState, ToState: plan.ToState, Applied: true, Warnings: plan.Warnings}
		id, year := updated.ID, updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{
			Action:        audit.ActionSessionStateTransition,
			ObjectType:    "session",
			ObjectID:      &id,
			SchoolYearID:  &year,
			Reason:        input.Reason,
			ChangeSummary: sessionTransitionSummary(plan, updated),
		})
	})
	if err != nil {
		return SessionTransitionResult{}, fmt.Errorf("transition session: %w", err)
	}
	return result, nil
}

func sessionTransitionPlan(ctx context.Context, tx *data.Tx, current data.Session, next data.SessionState) (SessionTransitionPlan, error) {
	offeringCount, err := tx.CountOfferings(ctx, current.SchoolYearID, current.ProgramID, current.ID)
	if err != nil {
		return SessionTransitionPlan{}, err
	}
	return PlanSessionTransition(current.State, next, offeringCount, current.DraftAssignmentsStale)
}

func loadSessionTransitionDates(ctx context.Context, tx *data.Tx, session *data.Session) error {
	dates, err := tx.ListMeetingDates(ctx, session.SchoolYearID, session.ProgramID, session.ID)
	if err != nil {
		return err
	}
	// Keeping date loading here avoids a second session read transaction for the
	// transition response.
	session.MeetingDates = make([]time.Time, 0, len(dates))
	for _, date := range dates {
		session.MeetingDates = append(session.MeetingDates, date.Date)
	}
	return nil
}

func sessionTransitionSummary(plan SessionTransitionPlan, session data.Session) json.RawMessage {
	warnings := make([]map[string]any, 0, len(plan.Warnings))
	for _, warning := range plan.Warnings {
		warnings = append(warnings, map[string]any{"code": warning.Code, "message": warning.Message, "invalidation_summary": warning.InvalidationSummary})
	}
	value := map[string]any{
		"from_state":              plan.FromState,
		"to_state":                plan.ToState,
		"backward":                plan.Backward,
		"draft_assignments_stale": session.DraftAssignmentsStale,
		"warnings":                warnings,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return encoded
}

func sessionStateLabel(state data.SessionState) string {
	if state == "" {
		return "<empty>"
	}
	words := strings.Split(string(state), "_")
	for index, word := range words {
		if word != "" {
			words[index] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, "")
}

func sessionStateList(states []data.SessionState) string {
	if len(states) == 0 {
		return "none"
	}
	values := make([]string, 0, len(states))
	for _, state := range states {
		values = append(values, sessionStateLabel(state))
	}
	return strings.Join(values, ", ")
}
