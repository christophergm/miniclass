package ingest

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest/roster"
)

// ErrSchoolYearClosed identifies the read-only lifecycle state that refuses
// preview, even though closed records remain readable history.
var ErrSchoolYearClosed = errors.New("school year is closed")

// ErrInvalidSource identifies a document that cannot be parsed by its
// registered source kind.
var ErrInvalidSource = errors.New("import source is invalid")

// InvalidSourceError carries the parser's own reason for refusing a document.
// A rejected upload is the organiser's to fix, so the reason has to survive as
// far as the response: "the submitted import document is invalid" alone leaves
// them with a file, a refusal, and nothing to change.
type InvalidSourceError struct {
	Kind   string
	Reason string
}

func (e *InvalidSourceError) Error() string {
	return fmt.Sprintf("parse %s: %s: %s", e.Kind, ErrInvalidSource, e.Reason)
}

// Unwrap keeps errors.Is(err, ErrInvalidSource) true for every caller that
// only classifies the failure.
func (e *InvalidSourceError) Unwrap() error { return ErrInvalidSource }

func invalidSource(kindName string, err error) error {
	return &InvalidSourceError{Kind: kindName, Reason: err.Error()}
}

// CurrentState is the tenant- and year-scoped snapshot consumed by a matcher.
// It is deliberately an application-data shape, not generated SQL.
type CurrentState struct {
	SchoolYear    data.SchoolYear
	Students      []data.Student
	Adults        []data.Adult
	Relationships []data.GuardianRelationship
	GradeLevels   []data.GradeLevel
	Homerooms     []data.Homeroom
}

// Service loads the current tenant state and executes a registered matcher in
// one read-only transaction. It has no commit method by design: P2-4 is the
// preview half of the protocol.
type Service struct {
	database *data.DB
	registry *Registry
}

// NewPreviewService constructs the database-backed preview service. A nil
// database remains useful for OpenAPI construction, but cannot serve a
// preview request.
func NewPreviewService(database *data.DB) *Service {
	return &Service{database: database, registry: NewRegistry()}
}

// NewPreviewServiceWithRegistry allows tests and future applications to
// supply a deliberately scoped source registry.
func NewPreviewServiceWithRegistry(database *data.DB, registry *Registry) *Service {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Service{database: database, registry: registry}
}

// Preview parses and classifies a submitted source document. The content hash
// covers the exact submitted bytes, including whitespace, so a later commit
// can reject a document that was changed after the human review.
func (s *Service) Preview(ctx context.Context, organizationID string, schoolYearID ids.XID, kindName string, document []byte) (Preview, error) {
	if s == nil || s.database == nil {
		return Preview{}, errors.New("preview import: data service is nil")
	}
	if strings.TrimSpace(organizationID) == "" {
		return Preview{}, errors.New("preview import: organization id is empty")
	}
	if strings.TrimSpace(string(schoolYearID)) == "" {
		return Preview{}, errors.New("preview import: school year id is empty")
	}
	kind, ok := s.registry.Lookup(kindName)
	if !ok {
		return Preview{}, fmt.Errorf("%w: %q", ErrUnknownKind, kindName)
	}

	contentHash := ContentHash(document)
	var result Preview
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		year, err := tx.GetSchoolYearByID(ctx, schoolYearID)
		if err != nil {
			return err
		}
		if year.State == data.SchoolYearClosed {
			return ErrSchoolYearClosed
		}

		parsed, err := kind.Parser(document)
		if err != nil {
			return invalidSource(kind.Name, err)
		}
		state, err := loadCurrentState(ctx, tx, year)
		if err != nil {
			return err
		}

		result, err = kind.Matcher(ctx, parsed, state)
		if err != nil {
			return err
		}
		result.Kind = kind.Name
		result.SchoolYearID = string(schoolYearID)
		result.ContentHash = contentHash
		return nil
	})
	if err != nil {
		return Preview{}, fmt.Errorf("preview import: %w", err)
	}
	return result, nil
}

func loadCurrentState(ctx context.Context, tx *data.Tx, year data.SchoolYear) (CurrentState, error) {
	state := CurrentState{SchoolYear: year}
	var err error
	state.Students, err = tx.ListStudents(ctx, year.ID, true)
	if err != nil {
		return CurrentState{}, err
	}
	state.Adults, err = tx.ListAdults(ctx, year.ID, true)
	if err != nil {
		return CurrentState{}, err
	}
	state.Relationships, err = tx.ListGuardianRelationships(ctx, year.ID, data.GuardianRelationshipFilter{})
	if err != nil {
		return CurrentState{}, err
	}
	state.GradeLevels, err = tx.ListGradeLevels(ctx, false)
	if err != nil {
		return CurrentState{}, err
	}
	state.Homerooms, err = tx.ListHomerooms(ctx, false)
	if err != nil {
		return CurrentState{}, err
	}
	return state, nil
}

func matchRoster(_ context.Context, parsed any, state CurrentState) (Preview, error) {
	document, ok := parsed.(roster.Document)
	if !ok {
		return Preview{}, errors.New("roster_json matcher received an unexpected parsed document")
	}
	return classifyRoster(document, state), nil
}

func classifyRoster(document roster.Document, state CurrentState) Preview {
	studentMatches := indexStudents(state.Students)
	adultMatches := indexAdults(state.Adults)
	homerooms := indexHomerooms(state.Homerooms)
	relationships := indexRelationships(state.Relationships)

	studentPreviews := make(map[string]RecordPreview, len(document.Result.Students))
	for _, source := range document.Result.Students {
		studentPreviews[source.SourceExternalIdentifier] = classifyStudent(source, studentMatches[source.SourceExternalIdentifier], homerooms)
	}
	adultPreviews := make(map[string]RecordPreview, len(document.Result.Adults))
	for _, source := range document.Result.Adults {
		adultPreviews[source.SourceExternalIdentifier] = classifyAdult(source, adultMatches[source.SourceExternalIdentifier])
	}
	relationshipPreviews := make(map[string]RecordPreview, len(document.Result.GuardianRelationships))
	for _, source := range document.Result.GuardianRelationships {
		key := relationshipKey(source.AdultExternalIdentifier, source.StudentExternalIdentifier)
		relationshipPreviews[key] = classifyRelationship(source, adultPreviews, studentPreviews, adultMatches, studentMatches, relationships)
	}

	result := Preview{
		Rows:                         make([]SourceRowPreview, 0, len(document.Rows)),
		GuardianRelationshipRemovals: make([]GuardianRelationshipRemoval, 0),
		Exclusions:                   make([]ExclusionPreview, 0, len(document.Result.ExcludedChildren)+len(document.Result.ExcludedAdults)),
		Warnings:                     make([]PreviewNotice, 0, len(document.Result.Warnings)),
	}
	for _, exclusion := range document.Result.ExcludedChildren {
		result.Exclusions = append(result.Exclusions, ExclusionPreview{
			RecordType: "student", SourceExternalIdentifier: exclusion.SourceExternalIdentifier,
			GivenName: exclusion.GivenName, FamilyName: exclusion.FamilyName, Reason: exclusion.Reason,
		})
	}
	for _, exclusion := range document.Result.ExcludedAdults {
		result.Exclusions = append(result.Exclusions, ExclusionPreview{
			RecordType: "adult", SourceExternalIdentifier: exclusion.SourceExternalIdentifier,
			GivenName: exclusion.GivenName, FamilyName: exclusion.FamilyName, Reason: exclusion.Reason,
		})
	}
	for _, warning := range document.Result.Warnings {
		result.Warnings = append(result.Warnings, PreviewNotice{
			Code: warning.Code, Detail: warning.Message, RecordType: "guardian_relationship",
			SourceValue:              warning.SourceValue,
			SourceExternalIdentifier: relationshipKey(warning.AdultExternalIdentifier, warning.StudentExternalIdentifier),
			AdultExternalIdentifier:  warning.AdultExternalIdentifier, StudentExternalIdentifier: warning.StudentExternalIdentifier,
		})
	}

	sourceEdgesByAdult := make(map[string]map[string]struct{})
	for _, sourceRow := range document.Rows {
		adultID := sourceRow.Adult.SourceExternalIdentifier
		if _, retained := adultPreviews[adultID]; !retained {
			continue
		}
		row := SourceRowPreview{
			Number: sourceRow.Number, SourceExternalIdentifier: adultID,
			Records: make([]RecordPreview, 0, 1+len(sourceRow.Students)+len(sourceRow.GuardianRelationships)),
		}
		// The adult row is authoritative even when it contains no relationships:
		// an empty edge set means all existing edges for this adult are omitted.
		sourceEdgesByAdult[adultID] = make(map[string]struct{})
		row.Records = append(row.Records, adultPreviews[adultID])
		seenStudents := make(map[string]struct{})
		for _, source := range sourceRow.Students {
			if _, seen := seenStudents[source.SourceExternalIdentifier]; seen {
				continue
			}
			seenStudents[source.SourceExternalIdentifier] = struct{}{}
			if record, ok := studentPreviews[source.SourceExternalIdentifier]; ok {
				row.Records = append(row.Records, record)
			}
		}
		seenRelationships := make(map[string]struct{})
		for _, source := range sourceRow.GuardianRelationships {
			key := relationshipKey(source.AdultExternalIdentifier, source.StudentExternalIdentifier)
			if _, seen := seenRelationships[key]; seen {
				continue
			}
			seenRelationships[key] = struct{}{}
			if record, ok := relationshipPreviews[key]; ok {
				row.Records = append(row.Records, record)
			}
			sourceEdgesByAdult[adultID][key] = struct{}{}
		}
		row.Outcome = rollup(row.Records)
		result.Rows = append(result.Rows, row)
	}

	for _, current := range state.Relationships {
		adultID := externalIdentifier(currentAdult(current, state.Adults))
		studentID := externalIdentifier(currentStudent(current, state.Students))
		if adultID == "" || studentID == "" {
			continue
		}
		sourceEdges, represented := sourceEdgesByAdult[adultID]
		if !represented {
			continue
		}
		key := relationshipKey(adultID, studentID)
		if _, present := sourceEdges[key]; present {
			continue
		}
		result.GuardianRelationshipRemovals = append(result.GuardianRelationshipRemovals, GuardianRelationshipRemoval{
			ExistingID: string(current.ID), AdultExternalIdentifier: adultID, StudentExternalIdentifier: studentID,
			RelationshipType: string(current.RelationshipType), Detail: "the represented source row omits this guardian relationship",
		})
	}
	sort.Slice(result.GuardianRelationshipRemovals, func(i, j int) bool {
		left, right := result.GuardianRelationshipRemovals[i], result.GuardianRelationshipRemovals[j]
		if left.AdultExternalIdentifier != right.AdultExternalIdentifier {
			return left.AdultExternalIdentifier < right.AdultExternalIdentifier
		}
		return left.StudentExternalIdentifier < right.StudentExternalIdentifier
	})

	for _, record := range adultPreviews {
		addCount(&result.Counts, record.Outcome)
	}
	for _, record := range studentPreviews {
		addCount(&result.Counts, record.Outcome)
	}
	for _, record := range relationshipPreviews {
		addCount(&result.Counts, record.Outcome)
	}
	return result
}

func classifyStudent(source roster.Student, matches []data.Student, homerooms map[string][]data.Homeroom) RecordPreview {
	record := RecordPreview{RecordType: "student", SourceExternalIdentifier: source.SourceExternalIdentifier}
	if strings.TrimSpace(source.SourceExternalIdentifier) == "" {
		record.Outcome, record.Detail = OutcomeError, "student has no external identifier"
		return record
	}
	if len(matches) > 1 {
		record.Outcome, record.Detail = OutcomeConflict, "external identifier matches more than one student in this school year"
		for _, match := range matches {
			if match.DeletedAt != nil {
				record.DeletedAt = match.DeletedAt
				record.Detail = fmt.Sprintf("external identifier matches more than one student, including a soft-deleted record deleted at %s", match.DeletedAt.UTC().Format(time.RFC3339))
				break
			}
		}
		return record
	}
	if len(matches) == 1 && matches[0].DeletedAt != nil {
		record.Outcome, record.Detail, record.ExistingID, record.DeletedAt = OutcomeConflict,
			fmt.Sprintf("student matches a soft-deleted record; restore it through the student restore surface before importing (deleted at %s)", matches[0].DeletedAt.UTC().Format("2006-01-02T15:04:05Z07:00")), string(matches[0].ID), matches[0].DeletedAt
		return record
	}
	homeroomID, detail := resolveHomeroom(source, homerooms)
	if detail != "" {
		record.Outcome, record.Detail = OutcomeError, detail
		return record
	}
	if len(matches) == 0 {
		if strings.TrimSpace(source.GivenName) == "" || strings.TrimSpace(source.FamilyName) == "" {
			record.Outcome, record.Detail = OutcomeError, fmt.Sprintf("student source id %q has no legal given and family name", source.SourceExternalIdentifier)
			return record
		}
		record.Outcome = OutcomeCreate
		record.Changes = []FieldChange{
			{Field: "legal_given_name", After: source.GivenName}, {Field: "legal_family_name", After: source.FamilyName},
			{Field: "homeroom_id", After: string(homeroomID)},
		}
		return record
	}
	current := matches[0]
	record.ExistingID = string(current.ID)
	if source.GivenName != "" && source.GivenName != current.LegalGivenName {
		record.Changes = append(record.Changes, FieldChange{Field: "legal_given_name", Before: current.LegalGivenName, After: source.GivenName})
	}
	if source.FamilyName != "" && source.FamilyName != current.LegalFamilyName {
		record.Changes = append(record.Changes, FieldChange{Field: "legal_family_name", Before: current.LegalFamilyName, After: source.FamilyName})
	}
	if current.HomeroomID != homeroomID {
		record.Changes = append(record.Changes, FieldChange{Field: "homeroom_id", Before: string(current.HomeroomID), After: string(homeroomID)})
	}
	record.Outcome = OutcomeUnchanged
	if len(record.Changes) > 0 {
		record.Outcome = OutcomeUpdate
	}
	return record
}

func classifyAdult(source roster.Adult, matches []data.Adult) RecordPreview {
	record := RecordPreview{RecordType: "adult", SourceExternalIdentifier: source.SourceExternalIdentifier}
	if strings.TrimSpace(source.SourceExternalIdentifier) == "" {
		record.Outcome, record.Detail = OutcomeError, "adult has no external identifier"
		return record
	}
	if len(matches) > 1 {
		record.Outcome, record.Detail = OutcomeConflict, "external identifier matches more than one adult in this school year"
		for _, match := range matches {
			if match.DeletedAt != nil {
				record.DeletedAt = match.DeletedAt
				record.Detail = fmt.Sprintf("external identifier matches more than one adult, including a soft-deleted record deleted at %s", match.DeletedAt.UTC().Format(time.RFC3339))
				break
			}
		}
		return record
	}
	if len(matches) == 1 && matches[0].DeletedAt != nil {
		record.Outcome, record.Detail, record.ExistingID, record.DeletedAt = OutcomeConflict,
			fmt.Sprintf("adult matches a soft-deleted record; restore it through the adult restore surface before importing (deleted at %s)", matches[0].DeletedAt.UTC().Format("2006-01-02T15:04:05Z07:00")), string(matches[0].ID), matches[0].DeletedAt
		return record
	}
	if len(matches) == 0 {
		if strings.TrimSpace(source.GivenName) == "" || strings.TrimSpace(source.FamilyName) == "" {
			record.Outcome, record.Detail = OutcomeError, fmt.Sprintf("adult source id %q has no legal given and family name", source.SourceExternalIdentifier)
			return record
		}
		record.Outcome = OutcomeCreate
		record.Changes = []FieldChange{{Field: "legal_given_name", After: source.GivenName}, {Field: "legal_family_name", After: source.FamilyName}}
		if source.Email != "" {
			record.Changes = append(record.Changes, FieldChange{Field: "email", After: source.Email})
		}
		return record
	}
	current := matches[0]
	record.ExistingID = string(current.ID)
	if source.GivenName != "" && source.GivenName != current.LegalGivenName {
		record.Changes = append(record.Changes, FieldChange{Field: "legal_given_name", Before: current.LegalGivenName, After: source.GivenName})
	}
	if source.FamilyName != "" && source.FamilyName != current.LegalFamilyName {
		record.Changes = append(record.Changes, FieldChange{Field: "legal_family_name", Before: current.LegalFamilyName, After: source.FamilyName})
	}
	if source.Email != "" && (current.Email == nil || source.Email != *current.Email) {
		var before any
		if current.Email != nil {
			before = *current.Email
		}
		record.Changes = append(record.Changes, FieldChange{Field: "email", Before: before, After: source.Email})
	}
	record.Outcome = OutcomeUnchanged
	if len(record.Changes) > 0 {
		record.Outcome = OutcomeUpdate
	}
	return record
}

func classifyRelationship(source roster.GuardianRelationship, adultPreviews, studentPreviews map[string]RecordPreview, adults map[string][]data.Adult, students map[string][]data.Student, current map[string]data.GuardianRelationship) RecordPreview {
	key := relationshipKey(source.AdultExternalIdentifier, source.StudentExternalIdentifier)
	record := RecordPreview{
		RecordType: "guardian_relationship", SourceExternalIdentifier: key,
		AdultExternalIdentifier: source.AdultExternalIdentifier, StudentExternalIdentifier: source.StudentExternalIdentifier,
	}
	adultRecord, adultOK := adultPreviews[source.AdultExternalIdentifier]
	studentRecord, studentOK := studentPreviews[source.StudentExternalIdentifier]
	if !adultOK || !studentOK {
		record.Outcome, record.Detail = OutcomeError, "guardian relationship references a source record that is not present in the parsed canonical set"
		return record
	}
	if adultRecord.Outcome == OutcomeError || studentRecord.Outcome == OutcomeError {
		record.Outcome, record.Detail = OutcomeError, "guardian relationship references a source record that cannot be imported"
		return record
	}
	if adultRecord.Outcome == OutcomeConflict || studentRecord.Outcome == OutcomeConflict {
		record.Outcome, record.Detail = OutcomeConflict, "guardian relationship references a conflicting person record"
		return record
	}
	if len(adults[source.AdultExternalIdentifier]) > 1 || len(students[source.StudentExternalIdentifier]) > 1 {
		record.Outcome, record.Detail = OutcomeConflict, "guardian relationship references an ambiguous external identifier"
		return record
	}
	if len(adults[source.AdultExternalIdentifier]) == 1 && adults[source.AdultExternalIdentifier][0].DeletedAt != nil {
		record.Outcome, record.Detail = OutcomeConflict, "guardian relationship references a soft-deleted adult"
		return record
	}
	if len(students[source.StudentExternalIdentifier]) == 1 && students[source.StudentExternalIdentifier][0].DeletedAt != nil {
		record.Outcome, record.Detail = OutcomeConflict, "guardian relationship references a soft-deleted student"
		return record
	}
	if len(adults[source.AdultExternalIdentifier]) == 1 && len(students[source.StudentExternalIdentifier]) == 1 {
		currentRelationship, found := current[relationshipKey(string(adults[source.AdultExternalIdentifier][0].ID), string(students[source.StudentExternalIdentifier][0].ID))]
		if found {
			record.ExistingID = string(currentRelationship.ID)
			if string(currentRelationship.RelationshipType) == source.RelationshipType {
				record.Outcome = OutcomeUnchanged
				return record
			}
			record.Outcome = OutcomeUpdate
			record.Changes = []FieldChange{{Field: "relationship_type", Before: string(currentRelationship.RelationshipType), After: source.RelationshipType}}
			return record
		}
	}
	record.Outcome = OutcomeCreate
	record.Changes = []FieldChange{{Field: "relationship_type", After: source.RelationshipType}}
	return record
}

func resolveHomeroom(source roster.Student, homerooms map[string][]data.Homeroom) (ids.XID, string) {
	externalID := strings.TrimSpace(source.ClassroomExternalIdentifier)
	if externalID == "" {
		externalID = strings.TrimSpace(source.ClassroomID)
	}
	if externalID == "" {
		return "", fmt.Sprintf("student source id %q has an unresolved classroom: source id is empty, label %q, band %q", source.SourceExternalIdentifier, source.ClassroomLabel, source.ClassroomBand)
	}
	matches := homerooms[externalID]
	if len(matches) == 0 {
		return "", fmt.Sprintf("student source id %q has an unresolved classroom: source id %q, label %q, band %q; create it in vocabulary settings before previewing again", source.SourceExternalIdentifier, externalID, source.ClassroomLabel, source.ClassroomBand)
	}
	if len(matches) > 1 {
		return "", fmt.Sprintf("student source id %q has an ambiguous classroom external identifier %q", source.SourceExternalIdentifier, externalID)
	}
	return matches[0].ID, ""
}

func indexStudents(rows []data.Student) map[string][]data.Student {
	result := make(map[string][]data.Student)
	for _, row := range rows {
		if id := externalIdentifier(row); id != "" {
			result[id] = append(result[id], row)
		}
	}
	return result
}

func indexAdults(rows []data.Adult) map[string][]data.Adult {
	result := make(map[string][]data.Adult)
	for _, row := range rows {
		if id := externalIdentifier(row); id != "" {
			result[id] = append(result[id], row)
		}
	}
	return result
}

func indexHomerooms(rows []data.Homeroom) map[string][]data.Homeroom {
	result := make(map[string][]data.Homeroom)
	for _, row := range rows {
		if row.ExternalIdentifier != nil && strings.TrimSpace(*row.ExternalIdentifier) != "" {
			result[strings.TrimSpace(*row.ExternalIdentifier)] = append(result[strings.TrimSpace(*row.ExternalIdentifier)], row)
		}
	}
	return result
}

func indexRelationships(rows []data.GuardianRelationship) map[string]data.GuardianRelationship {
	result := make(map[string]data.GuardianRelationship, len(rows))
	for _, row := range rows {
		result[relationshipKey(string(row.AdultID), string(row.StudentID))] = row
	}
	return result
}

func currentAdult(relationship data.GuardianRelationship, adults []data.Adult) data.Adult {
	for _, adult := range adults {
		if adult.ID == relationship.AdultID {
			return adult
		}
	}
	return data.Adult{}
}

func currentStudent(relationship data.GuardianRelationship, students []data.Student) data.Student {
	for _, student := range students {
		if student.ID == relationship.StudentID {
			return student
		}
	}
	return data.Student{}
}

func externalIdentifier(value any) string {
	switch row := value.(type) {
	case data.Adult:
		if row.ExternalIdentifier != nil {
			return strings.TrimSpace(*row.ExternalIdentifier)
		}
	case data.Student:
		if row.ExternalIdentifier != nil {
			return strings.TrimSpace(*row.ExternalIdentifier)
		}
	}
	return ""
}

func relationshipKey(adultID, studentID string) string {
	return strings.TrimSpace(adultID) + "\x00" + strings.TrimSpace(studentID)
}

func rollup(records []RecordPreview) Outcome {
	priority := []Outcome{OutcomeError, OutcomeConflict, OutcomeUpdate, OutcomeCreate, OutcomeUnchanged}
	for _, want := range priority {
		for _, record := range records {
			if record.Outcome == want {
				return want
			}
		}
	}
	return OutcomeUnchanged
}

func addCount(counts *OutcomeCounts, outcome Outcome) {
	switch outcome {
	case OutcomeCreate:
		counts.Create++
	case OutcomeUpdate:
		counts.Update++
	case OutcomeUnchanged:
		counts.Unchanged++
	case OutcomeConflict:
		counts.Conflict++
	case OutcomeError:
		counts.Error++
	}
}
