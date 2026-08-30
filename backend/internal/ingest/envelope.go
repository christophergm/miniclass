// Package ingest owns the source-kind registry and the read-only half of the
// two-phase import protocol.
package ingest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/chrismott/miniclass/internal/ingest/roster"
)

const (
	KindRosterJSON = "roster_json"
	KindGradesCSV  = "grades_csv"
)

// Outcome is the complete set of classifications from SPEC §11.5.
type Outcome string

const (
	OutcomeCreate    Outcome = "Create"
	OutcomeUpdate    Outcome = "Update"
	OutcomeUnchanged Outcome = "Unchanged"
	OutcomeConflict  Outcome = "Conflict"
	OutcomeError     Outcome = "Error"
)

var (
	ErrUnknownKind          = errors.New("import kind is not registered")
	ErrUnsupportedKind      = errors.New("import kind is not supported by this phase")
	ErrCommitNotImplemented = errors.New("import commit is not implemented in the preview phase")
)

// Parser translates one source document into a kind-owned canonical document.
type Parser func([]byte) (any, error)

// Matcher classifies a parsed document against a read-only database snapshot.
type Matcher func(context.Context, any, CurrentState) (Preview, error)

// Writer is the phase-two seam. The preview phase registers the writer
// contract but deliberately does not mutate domain data.
type Writer func(context.Context, CommitRequest) error

// Kind is one pluggable import source. The record types remain owned by the
// kind while the envelope owns the lifecycle and preview shape.
type Kind struct {
	Name    string
	Parser  Parser
	Matcher Matcher
	Writer  Writer
}

// Registry is an immutable-after-construction lookup of import kinds. A
// registry is independent per caller so tests cannot leak registrations.
type Registry struct {
	kinds map[string]Kind
	order []string
}

// NewRegistry returns the Phase 2 registry containing both declared source
// kinds. grades_csv is registered now so adding its kind does not require a
// new envelope; its matching and writing behavior arrives in P2-6.
func NewRegistry() *Registry {
	registry := &Registry{kinds: make(map[string]Kind)}
	registry.MustRegister(Kind{
		Name:    KindRosterJSON,
		Parser:  func(document []byte) (any, error) { return roster.ParseDocument(document) },
		Matcher: matchRoster,
		Writer:  unavailableWriter,
	})
	registry.MustRegister(Kind{
		Name:    KindGradesCSV,
		Parser:  func(document []byte) (any, error) { return roster.ParseGradesCSV(bytes.NewReader(document)) },
		Matcher: unsupportedMatcher,
		Writer:  unavailableWriter,
	})
	return registry
}

// NewEmptyRegistry returns a registry for tests and future applications that
// need to opt into a smaller set of source kinds.
func NewEmptyRegistry() *Registry {
	return &Registry{kinds: make(map[string]Kind)}
}

// Register adds a source kind and rejects empty or duplicate names.
func (r *Registry) Register(kind Kind) error {
	if r == nil {
		return errors.New("register import kind: registry is nil")
	}
	if kind.Name == "" {
		return errors.New("register import kind: name is empty")
	}
	if kind.Parser == nil || kind.Matcher == nil || kind.Writer == nil {
		return errors.New("register import kind: parser, matcher, and writer are required")
	}
	if r.kinds == nil {
		r.kinds = make(map[string]Kind)
	}
	if _, exists := r.kinds[kind.Name]; exists {
		return errors.New("register import kind: duplicate name")
	}
	r.kinds[kind.Name] = kind
	r.order = append(r.order, kind.Name)
	return nil
}

// MustRegister registers a kind during construction and panics only for a
// programmer error in the static registry definition.
func (r *Registry) MustRegister(kind Kind) {
	if err := r.Register(kind); err != nil {
		panic(err)
	}
}

// Lookup returns a registered kind by its URL-safe name.
func (r *Registry) Lookup(name string) (Kind, bool) {
	if r == nil {
		return Kind{}, false
	}
	kind, ok := r.kinds[name]
	return kind, ok
}

// Kinds returns names in registration order for diagnostics and contract
// tests.
func (r *Registry) Kinds() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.order...)
}

// CommitRequest carries the stateless hash guard into the future commit phase.
// It intentionally contains the preview rather than a server-side batch ID.
type CommitRequest struct {
	Kind         string
	SchoolYearID string
	ContentHash  string
	Preview      Preview
}

// FieldChange is one asserted source field that differs from the stored value.
// Before and After intentionally retain null as a meaningful value for future
// source kinds that can clear an asserted field.
type FieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// RecordPreview is the per-record node in a source-row preview tree.
type RecordPreview struct {
	RecordType                string        `json:"record_type"`
	SourceExternalIdentifier  string        `json:"source_external_identifier"`
	AdultExternalIdentifier   string        `json:"adult_external_identifier,omitempty"`
	StudentExternalIdentifier string        `json:"student_external_identifier,omitempty"`
	ExistingID                string        `json:"existing_id,omitempty"`
	Outcome                   Outcome       `json:"outcome"`
	Changes                   []FieldChange `json:"changes,omitempty"`
	Detail                    string        `json:"detail,omitempty"`
	DeletedAt                 *time.Time    `json:"deleted_at,omitempty"`
}

// SourceRowPreview is a source row with its child record nodes. Outcome is a
// deterministic roll-up of the contained records.
type SourceRowPreview struct {
	Number                   int             `json:"number"`
	SourceExternalIdentifier string          `json:"source_external_identifier"`
	Outcome                  Outcome         `json:"outcome"`
	Records                  []RecordPreview `json:"records"`
}

// GuardianRelationshipRemoval calls out the destructive edge changes
// separately from additions and updates, as required by SPEC §11.4–§11.5.
type GuardianRelationshipRemoval struct {
	ExistingID                string `json:"existing_id"`
	AdultExternalIdentifier   string `json:"adult_external_identifier"`
	StudentExternalIdentifier string `json:"student_external_identifier"`
	RelationshipType          string `json:"relationship_type"`
	Detail                    string `json:"detail"`
}

// PreviewNotice is an informational or warning item retained in the response
// without changing the five import outcomes.
type PreviewNotice struct {
	Code                      string `json:"code"`
	Detail                    string `json:"detail"`
	SourceValue               string `json:"source_value,omitempty"`
	RecordType                string `json:"record_type,omitempty"`
	SourceExternalIdentifier  string `json:"source_external_identifier,omitempty"`
	AdultExternalIdentifier   string `json:"adult_external_identifier,omitempty"`
	StudentExternalIdentifier string `json:"student_external_identifier,omitempty"`
}

// ExclusionPreview reports source filters without inventing a sixth outcome.
type ExclusionPreview struct {
	RecordType               string `json:"record_type"`
	SourceExternalIdentifier string `json:"source_external_identifier"`
	GivenName                string `json:"given_name,omitempty"`
	FamilyName               string `json:"family_name,omitempty"`
	Reason                   string `json:"reason"`
}

// OutcomeCounts is the distribution over distinct canonical source records.
type OutcomeCounts struct {
	Create    int `json:"create"`
	Update    int `json:"update"`
	Unchanged int `json:"unchanged"`
	Conflict  int `json:"conflict"`
	Error     int `json:"error"`
}

// Preview is the durable response shape for the stateless first phase.
type Preview struct {
	Kind                         string                        `json:"kind"`
	SchoolYearID                 string                        `json:"school_year_id"`
	ContentHash                  string                        `json:"content_hash"`
	Rows                         []SourceRowPreview            `json:"rows"`
	GuardianRelationshipRemovals []GuardianRelationshipRemoval `json:"guardian_relationship_removals"`
	Exclusions                   []ExclusionPreview            `json:"exclusions"`
	Warnings                     []PreviewNotice               `json:"warnings"`
	Counts                       OutcomeCounts                 `json:"counts"`
}

// ContentHash returns the hex-encoded SHA-256 digest used to bind a preview to
// the exact document submitted for a future commit.
func ContentHash(document []byte) string {
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:])
}

func unavailableWriter(context.Context, CommitRequest) error { return ErrCommitNotImplemented }

func unsupportedMatcher(context.Context, any, CurrentState) (Preview, error) {
	return Preview{}, ErrUnsupportedKind
}
