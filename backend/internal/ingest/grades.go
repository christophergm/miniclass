package ingest

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest/roster"
)

// matchGrades implements the update-only grades_csv source. The CSV has no
// opaque identifier, so its complete name cell is matched against the two
// names the student may use in this application.
func matchGrades(_ context.Context, parsed any, state CurrentState) (Preview, error) {
	rows, ok := parsed.([]roster.GradeRecord)
	if !ok {
		return Preview{}, errors.New("grades_csv matcher received an unexpected parsed document")
	}

	preview := Preview{
		Rows:                         make([]SourceRowPreview, 0, len(rows)),
		GuardianRelationshipRemovals: []GuardianRelationshipRemoval{},
		Exclusions:                   []ExclusionPreview{},
		Warnings:                     []PreviewNotice{},
	}
	duplicateGrades := make(map[string]string, len(rows))
	contradictoryDuplicates := make(map[string]struct{})
	for _, source := range rows {
		nameKey := normalizeImportName(source.StudentName)
		gradeKey := normalizeImportText(source.Grade)
		if previous, exists := duplicateGrades[nameKey]; exists && previous != gradeKey {
			// A source that assigns two grades to the same normalized name cannot
			// be made safe by choosing one. Both rows remain visible conflicts.
			contradictoryDuplicates[nameKey] = struct{}{}
			continue
		}
		duplicateGrades[nameKey] = gradeKey
	}

	for index, source := range rows {
		record := classifyGrade(source, state.Students, state.GradeLevels)
		nameKey := normalizeImportName(source.StudentName)
		if _, exists := contradictoryDuplicates[nameKey]; exists {
			record.Outcome = OutcomeConflict
			record.Changes = nil
			record.Detail = fmt.Sprintf("normalized student name is assigned more than one grade in the source (including %q)", source.Grade)
		}
		preview.Rows = append(preview.Rows, SourceRowPreview{
			Number:                   index + 2,
			SourceExternalIdentifier: source.StudentName,
			Outcome:                  record.Outcome,
			Records:                  []RecordPreview{record},
		})
		addCount(&preview.Counts, record.Outcome)
	}
	return preview, nil
}

func classifyGrade(source roster.GradeRecord, students []data.Student, levels []data.GradeLevel) RecordPreview {
	record := RecordPreview{RecordType: "student", SourceExternalIdentifier: source.StudentName}
	if normalizeImportName(source.StudentName) == "" {
		record.Outcome, record.Detail = OutcomeError, "grade row has no student name"
		return record
	}

	candidates := matchingGradeStudents(source.StudentName, students)
	if len(candidates) == 0 {
		// This kind is deliberately update-only. An unmatched row is reviewable
		// but never becomes a new student with a missing required homeroom.
		record.Outcome, record.Detail = OutcomeConflict, "no existing student matches this normalized whole name; the row will be skipped"
		return record
	}
	if len(candidates) > 1 {
		record.Outcome, record.Detail = OutcomeConflict, "normalized whole name matches more than one student; the importer will not choose"
		return record
	}
	student := candidates[0]
	record.ExistingID = string(student.ID)
	if student.DeletedAt != nil {
		record.Outcome, record.Detail = OutcomeConflict, fmt.Sprintf("student matches a soft-deleted record; restore it before importing (deleted at %s)", student.DeletedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
		record.DeletedAt = student.DeletedAt
		return record
	}

	level, detail := resolveGradeLevel(source.Grade, levels)
	if detail != "" {
		record.Outcome, record.Detail = OutcomeError, detail
		return record
	}
	if student.GradeLevelID != nil && *student.GradeLevelID == level.ID {
		record.Outcome = OutcomeUnchanged
		return record
	}

	var before any
	if student.GradeLevelID != nil {
		before = string(*student.GradeLevelID)
	}
	record.Outcome = OutcomeUpdate
	record.Changes = []FieldChange{{Field: "grade_level_id", Before: before, After: string(level.ID)}}
	record.Detail = fmt.Sprintf("grade resolves to %q (%s)", level.Label, level.Code)
	return record
}

func matchingGradeStudents(sourceName string, students []data.Student) []data.Student {
	key := normalizeImportName(sourceName)
	result := make([]data.Student, 0)
	seen := make(map[ids.XID]struct{})
	for _, student := range students {
		legal := normalizeImportName(student.LegalGivenName + " " + student.LegalFamilyName)
		preferred := ""
		if student.PreferredGivenName != nil {
			preferred = normalizeImportName(*student.PreferredGivenName + " " + student.LegalFamilyName)
		}
		if key != legal && (preferred == "" || key != preferred) {
			continue
		}
		if _, exists := seen[student.ID]; exists {
			continue
		}
		seen[student.ID] = struct{}{}
		result = append(result, student)
	}
	return result
}

func resolveGradeLevel(source string, levels []data.GradeLevel) (data.GradeLevel, string) {
	key := normalizeImportText(source)
	if key == "" {
		return data.GradeLevel{}, "grade label is empty or unrecognized"
	}
	var matches []data.GradeLevel
	seen := make(map[ids.XID]struct{})
	for _, level := range levels {
		if key != normalizeImportText(level.Code) && key != normalizeImportText(level.Label) {
			continue
		}
		if _, exists := seen[level.ID]; exists {
			continue
		}
		seen[level.ID] = struct{}{}
		matches = append(matches, level)
	}
	if len(matches) == 0 {
		return data.GradeLevel{}, fmt.Sprintf("grade label %q is not recognized by the grade-level vocabulary", source)
	}
	if len(matches) > 1 {
		return data.GradeLevel{}, fmt.Sprintf("grade label %q matches more than one grade-level vocabulary entry", source)
	}
	return matches[0], ""
}

// normalizeImportName intentionally receives the whole name cell. Do not
// split it into tokens: Appendix A.5 includes two-word surnames as a defect
// case, and only the canonical given/family combinations may be compared.
func normalizeImportName(value string) string { return normalizeImportText(value) }

func normalizeImportText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
