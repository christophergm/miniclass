package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest/roster"
)

// Commit applies the exact document reviewed by a caller's preview. The
// preview is recomputed inside the write transaction so the same classification
// drives both the safety check and the mutations; no import batch is persisted.
func (s *Service) Commit(ctx context.Context, organizationID string, schoolYearID ids.XID, kindName string, document []byte, contentHash string, actor audit.Actor) (Preview, error) {
	if s == nil || s.database == nil {
		return Preview{}, errors.New("commit import: data service is nil")
	}
	if strings.TrimSpace(organizationID) == "" {
		return Preview{}, errors.New("commit import: organization id is empty")
	}
	if strings.TrimSpace(string(schoolYearID)) == "" {
		return Preview{}, errors.New("commit import: school year id is empty")
	}
	if !strings.EqualFold(strings.TrimSpace(contentHash), ContentHash(document)) {
		return Preview{}, ErrContentHashMismatch
	}
	kind, ok := s.registry.Lookup(kindName)
	if !ok {
		return Preview{}, fmt.Errorf("%w: %q", ErrUnknownKind, kindName)
	}

	var result Preview
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		year, err := tx.GetSchoolYearByID(ctx, schoolYearID)
		if err != nil {
			return err
		}
		if year.State == data.SchoolYearClosed {
			return ErrSchoolYearClosed
		}

		parsed, err := kind.Parser(document)
		if err != nil {
			return fmt.Errorf("parse %s: %w: %v", kind.Name, ErrInvalidSource, err)
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
		result.ContentHash = ContentHash(document)
		if result.Counts.Error > 0 {
			return ErrCommitHasErrors
		}

		request := CommitRequest{
			Kind: kind.Name, SchoolYearID: string(schoolYearID), ContentHash: result.ContentHash,
			Document: append([]byte(nil), document...), Parsed: parsed, Preview: result, Tx: tx, State: state,
		}
		if err := kind.Writer(ctx, request); err != nil {
			return err
		}
		yearID := year.ID
		return tx.Record(ctx, audit.Entry{
			Action: audit.ActionImportCommit, ObjectType: "import", SchoolYearID: &yearID,
			ChangeSummary: importCommitSummary(request),
		})
	})
	if err != nil {
		return Preview{}, fmt.Errorf("commit import: %w", err)
	}
	return result, nil
}

// State is supplied to a kind writer by Service.Commit. Keeping it on the
// request lets a writer use the same snapshot that produced the preview.
// It is not serialized or retained after the transaction ends.

func commitRoster(ctx context.Context, request CommitRequest) error {
	document, ok := request.Parsed.(roster.Document)
	if !ok {
		return errors.New("commit roster: unexpected parsed document")
	}
	if request.Tx == nil {
		return errors.New("commit roster: transaction is nil")
	}

	records := previewRecordIndex(request.Preview)
	adultMatches := indexAdults(request.State.Adults)
	studentMatches := indexStudents(request.State.Students)
	adultBySource := make(map[string]data.Adult, len(document.Result.Adults))
	studentBySource := make(map[string]data.Student, len(document.Result.Students))

	for _, source := range document.Result.Adults {
		record, found := records[personRecordKey("adult", source.SourceExternalIdentifier)]
		if !found {
			return fmt.Errorf("commit roster: adult %q is absent from the preview", source.SourceExternalIdentifier)
		}
		if record.Outcome == OutcomeConflict {
			continue
		}
		matches := adultMatches[source.SourceExternalIdentifier]
		switch record.Outcome {
		case OutcomeUnchanged:
			if len(matches) != 1 {
				return fmt.Errorf("commit roster: unchanged adult %q has no unique match", source.SourceExternalIdentifier)
			}
			adultBySource[source.SourceExternalIdentifier] = matches[0]
		case OutcomeCreate:
			externalIdentifier := strings.TrimSpace(source.SourceExternalIdentifier)
			var email *string
			if strings.TrimSpace(source.Email) != "" {
				value := strings.TrimSpace(source.Email)
				email = &value
			}
			created, err := request.Tx.CreateAdult(ctx, ids.XID(request.SchoolYearID), source.GivenName, source.FamilyName, nil, email, nil, &externalIdentifier, nil)
			if err != nil {
				return fmt.Errorf("create adult %q: %w", source.SourceExternalIdentifier, err)
			}
			adultBySource[source.SourceExternalIdentifier] = created
		case OutcomeUpdate:
			if len(matches) != 1 {
				return fmt.Errorf("commit roster: updated adult %q has no unique match", source.SourceExternalIdentifier)
			}
			current := matches[0]
			givenName, familyName := current.LegalGivenName, current.LegalFamilyName
			if strings.TrimSpace(source.GivenName) != "" {
				givenName = source.GivenName
			}
			if strings.TrimSpace(source.FamilyName) != "" {
				familyName = source.FamilyName
			}
			email := current.Email
			if strings.TrimSpace(source.Email) != "" {
				value := strings.TrimSpace(source.Email)
				email = &value
			}
			updated, err := request.Tx.UpdateAdult(ctx, ids.XID(request.SchoolYearID), current.ID, givenName, familyName, current.PreferredGivenName, email, current.Phone, current.ExternalIdentifier, current.ParticipationIntent)
			if err != nil {
				return fmt.Errorf("update adult %q: %w", source.SourceExternalIdentifier, err)
			}
			adultBySource[source.SourceExternalIdentifier] = updated
		}
	}

	for _, source := range document.Result.Students {
		record, found := records[personRecordKey("student", source.SourceExternalIdentifier)]
		if !found {
			return fmt.Errorf("commit roster: student %q is absent from the preview", source.SourceExternalIdentifier)
		}
		if record.Outcome == OutcomeConflict {
			continue
		}
		matches := studentMatches[source.SourceExternalIdentifier]
		homeroomID, detail := resolveHomeroom(source, indexHomerooms(request.State.Homerooms))
		if detail != "" {
			return fmt.Errorf("commit roster: %s", detail)
		}
		switch record.Outcome {
		case OutcomeUnchanged:
			if len(matches) != 1 {
				return fmt.Errorf("commit roster: unchanged student %q has no unique match", source.SourceExternalIdentifier)
			}
			studentBySource[source.SourceExternalIdentifier] = matches[0]
		case OutcomeCreate:
			externalIdentifier := strings.TrimSpace(source.SourceExternalIdentifier)
			created, err := request.Tx.CreateStudent(ctx, ids.XID(request.SchoolYearID), nil, homeroomID, source.GivenName, source.FamilyName, nil, &externalIdentifier, nil)
			if err != nil {
				return fmt.Errorf("create student %q: %w", source.SourceExternalIdentifier, err)
			}
			studentBySource[source.SourceExternalIdentifier] = created
		case OutcomeUpdate:
			if len(matches) != 1 {
				return fmt.Errorf("commit roster: updated student %q has no unique match", source.SourceExternalIdentifier)
			}
			current := matches[0]
			givenName, familyName := current.LegalGivenName, current.LegalFamilyName
			if strings.TrimSpace(source.GivenName) != "" {
				givenName = source.GivenName
			}
			if strings.TrimSpace(source.FamilyName) != "" {
				familyName = source.FamilyName
			}
			updated, err := request.Tx.UpdateStudent(ctx, ids.XID(request.SchoolYearID), current.ID, givenName, familyName, current.PreferredGivenName, current.GradeLevelID, homeroomID, current.ExternalIdentifier, current.PriorYearStudentID)
			if err != nil {
				return fmt.Errorf("update student %q: %w", source.SourceExternalIdentifier, err)
			}
			studentBySource[source.SourceExternalIdentifier] = updated
		}
	}

	currentRelationships := indexRelationships(request.State.Relationships)
	for _, source := range document.Result.GuardianRelationships {
		key := relationshipKey(source.AdultExternalIdentifier, source.StudentExternalIdentifier)
		record, found := records[key]
		if !found {
			return fmt.Errorf("commit roster: relationship %q is absent from the preview", key)
		}
		if record.Outcome == OutcomeConflict {
			continue
		}
		adult, adultOK := adultBySource[source.AdultExternalIdentifier]
		student, studentOK := studentBySource[source.StudentExternalIdentifier]
		if !adultOK || !studentOK {
			return fmt.Errorf("commit roster: relationship %q references a skipped person", key)
		}
		relationshipType := data.GuardianRelationshipType(source.RelationshipType)
		switch record.Outcome {
		case OutcomeUnchanged:
			continue
		case OutcomeCreate:
			if _, err := request.Tx.CreateGuardianRelationship(ctx, ids.XID(request.SchoolYearID), adult.ID, student.ID, relationshipType); err != nil {
				return fmt.Errorf("create relationship %q: %w", key, err)
			}
		case OutcomeUpdate:
			current, exists := currentRelationships[relationshipKey(string(adult.ID), string(student.ID))]
			if !exists {
				return fmt.Errorf("commit roster: updated relationship %q has no current match", key)
			}
			if _, err := request.Tx.UpdateGuardianRelationship(ctx, ids.XID(request.SchoolYearID), current.ID, relationshipType); err != nil {
				return fmt.Errorf("update relationship %q: %w", key, err)
			}
		}
	}

	for _, removal := range request.Preview.GuardianRelationshipRemovals {
		if _, represented := adultBySource[removal.AdultExternalIdentifier]; !represented {
			continue
		}
		deleted, err := request.Tx.DeleteGuardianRelationship(ctx, ids.XID(request.SchoolYearID), ids.XID(removal.ExistingID))
		if err != nil {
			return fmt.Errorf("remove relationship %q: %w", removal.ExistingID, err)
		}
		if !deleted {
			return fmt.Errorf("remove relationship %q: row was not found", removal.ExistingID)
		}
		relationshipID := ids.XID(removal.ExistingID)
		yearID := ids.XID(request.SchoolYearID)
		if err := request.Tx.Record(ctx, audit.Entry{
			Action: audit.ActionHardDelete, ObjectType: "guardian_relationship", ObjectID: &relationshipID, SchoolYearID: &yearID,
			ChangeSummary: guardianRemovalSummary(removal),
		}); err != nil {
			return err
		}
	}
	return nil
}

func commitGrades(ctx context.Context, request CommitRequest) error {
	rows, ok := request.Parsed.([]roster.GradeRecord)
	if !ok {
		return errors.New("commit grades: unexpected parsed document")
	}
	if request.Tx == nil {
		return errors.New("commit grades: transaction is nil")
	}

	updated := make(map[ids.XID]struct{})
	for index, source := range rows {
		if index >= len(request.Preview.Rows) || len(request.Preview.Rows[index].Records) != 1 {
			return fmt.Errorf("commit grades: row %d is absent from the preview", index+2)
		}
		record := request.Preview.Rows[index].Records[0]
		if record.Outcome != OutcomeUpdate {
			continue
		}
		candidates := matchingGradeStudents(source.StudentName, request.State.Students)
		if len(candidates) != 1 {
			return fmt.Errorf("commit grades: row %d no longer has a unique student match", index+2)
		}
		student := candidates[0]
		if _, exists := updated[student.ID]; exists {
			continue
		}
		level, detail := resolveGradeLevel(source.Grade, request.State.GradeLevels)
		if detail != "" {
			return fmt.Errorf("commit grades: row %d: %s", index+2, detail)
		}
		if _, err := request.Tx.UpdateStudent(ctx, ids.XID(request.SchoolYearID), student.ID,
			student.LegalGivenName, student.LegalFamilyName, student.PreferredGivenName, &level.ID,
			student.HomeroomID, student.ExternalIdentifier, student.PriorYearStudentID); err != nil {
			return fmt.Errorf("update grade for %q: %w", source.StudentName, err)
		}
		updated[student.ID] = struct{}{}
	}
	return nil
}

func previewRecordIndex(preview Preview) map[string]RecordPreview {
	result := make(map[string]RecordPreview)
	for _, row := range preview.Rows {
		for _, record := range row.Records {
			key := record.SourceExternalIdentifier
			if record.RecordType != "guardian_relationship" {
				key = personRecordKey(record.RecordType, record.SourceExternalIdentifier)
			}
			result[key] = record
		}
	}
	return result
}

func personRecordKey(recordType, sourceIdentifier string) string {
	return recordType + "\x00" + strings.TrimSpace(sourceIdentifier)
}

func importCommitSummary(request CommitRequest) json.RawMessage {
	value := map[string]any{
		"kind": request.Kind, "content_hash": request.ContentHash, "counts": request.Preview.Counts,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode import commit summary"}`)
	}
	return encoded
}

func guardianRemovalSummary(removal GuardianRelationshipRemoval) json.RawMessage {
	value := map[string]any{
		"before": map[string]any{
			"adult_external_identifier":   removal.AdultExternalIdentifier,
			"student_external_identifier": removal.StudentExternalIdentifier,
			"relationship_type":           removal.RelationshipType,
		},
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode guardian relationship removal summary"}`)
	}
	return encoded
}
