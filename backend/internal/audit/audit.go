// Package audit contains the application-owned audit vocabulary and entry
// shape. Persistence remains in internal/data so callers cannot bypass the
// tenant transaction boundary.
package audit

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/chrismott/miniclass/internal/ids"
)

// ActorType is the closed set of principals that can produce an audit entry.
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeLink   ActorType = "link"
	ActorTypeSystem ActorType = "system"
)

// Action is intentionally Go-owned. The vocabulary grows with application
// features and does not need a migration for each new action.
type Action string

const (
	ActionCreate                    Action = "create"
	ActionEdit                      Action = "edit"
	ActionSoftDelete                Action = "soft_delete"
	ActionHardDelete                Action = "hard_delete"
	ActionImportCommit              Action = "import_commit"
	ActionSchoolYearCreate          Action = "school_year_create"
	ActionSchoolYearStateTransition Action = "school_year_state_transition"
	ActionProgramCreate             Action = "program_create"
	ActionMembershipChange          Action = "membership_change"
	ActionSessionNonParticipation   Action = "session_non_participation"
	ActionSessionStateTransition    Action = "session_state_transition"
	ActionOfferingEdit              Action = "offering_edit_after_publish"
	ActionTagDefinitionChange       Action = "tag_definition_change"
	ActionTagAssignmentChange       Action = "tag_assignment_change"
	ActionPairingChange             Action = "pairing_change"
	ActionExclusionChange           Action = "exclusion_change"
	ActionVocabularyChange          Action = "vocabulary_change"
	ActionSolveRun                  Action = "solve_run"
	ActionManualOperation           Action = "manual_operation"
	ActionOverride                  Action = "override"
	ActionPublish                   Action = "publish"
	ActionRepublish                 Action = "republish"
	ActionLinkGenerate              Action = "link_generate"
	ActionLinkRegenerate            Action = "link_regenerate"
	ActionLinkRevoke                Action = "link_revoke"
	ActionPermissionChange          Action = "permission_change"
	ActionAdministratorAdd          Action = "administrator_add"
	ActionAdministratorRemove       Action = "administrator_remove"
)

// Actor identifies who performed a unit of work. ActorLabel is copied into
// each entry so later redaction can preserve the fact that an action occurred.
type Actor struct {
	Type   ActorType
	UserID *ids.XID
	Label  string
}

// Entry is the event-specific part of an audit record. The transaction actor
// is supplied to data.InTenant and is attached when Record is called.
type Entry struct {
	Action        Action
	ObjectType    string
	ObjectID      *ids.XID
	ChangeSummary json.RawMessage
	Reason        string
	SchoolYearID  *ids.XID
	RequestID     string
}

// Validate checks fields that must be present for a useful immutable record.
func (e Entry) Validate() error {
	if strings.TrimSpace(string(e.Action)) == "" {
		return errors.New("audit entry: action is empty")
	}
	if strings.TrimSpace(e.ObjectType) == "" {
		return errors.New("audit entry: object type is empty")
	}
	if len(e.ChangeSummary) > 0 && !json.Valid(e.ChangeSummary) {
		return errors.New("audit entry: change summary is not valid JSON")
	}
	return nil
}

// Validate checks that the transaction actor can be persisted.
func (a Actor) Validate() error {
	switch a.Type {
	case ActorTypeUser, ActorTypeLink, ActorTypeSystem:
	default:
		return errors.New("audit actor: type is invalid")
	}
	if strings.TrimSpace(a.Label) == "" {
		return errors.New("audit actor: label is empty")
	}
	if a.Type == ActorTypeUser && a.UserID == nil {
		return errors.New("audit actor: user actor requires a user id")
	}
	if a.Type != ActorTypeUser && a.UserID != nil {
		return errors.New("audit actor: non-user actor cannot have a user id")
	}
	return nil
}
