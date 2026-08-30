package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest/roster"
	"github.com/stretchr/testify/require"
)

func TestRegistryHasPhaseTwoKindsAndRequiredHooks(t *testing.T) {
	registry := NewRegistry()
	require.Equal(t, []string{KindRosterJSON, KindGradesCSV}, registry.Kinds())
	for _, name := range registry.Kinds() {
		kind, ok := registry.Lookup(name)
		require.True(t, ok)
		require.NotNil(t, kind.Parser)
		require.NotNil(t, kind.Matcher)
		require.NotNil(t, kind.Writer)
	}
	require.Error(t, registry.Register(Kind{Name: KindRosterJSON, Parser: func([]byte) (any, error) { return nil, nil }, Matcher: unsupportedMatcher, Writer: unavailableWriter}))
	require.Error(t, NewEmptyRegistry().Register(Kind{Name: "incomplete"}))
}

func TestContentHashCoversExactSubmittedBytes(t *testing.T) {
	document := []byte("[{}]\n")
	digest := sha256.Sum256(document)
	require.Equal(t, hex.EncodeToString(digest[:]), ContentHash(document))
	require.NotEqual(t, ContentHash(document), ContentHash([]byte("[{}]")))
}

func TestRosterPreviewIsARecordTreeAndCountsDistinctRecords(t *testing.T) {
	document := parsePreviewDocument(t, `[
      {"id":"adult-1","given_name":"Alex","family_name":"Adult","email":"alex@example.test","relationships":[
        {"relationship":"MOM","child":{"id":"child-1","given_name":"Casey","family_name":"One","groups":[{"id":"room-1","class":"classroom","name":"Room 1","band":"display"}]}},
        {"relationship":"DAD","child":{"id":"child-2","given_name":"Riley","family_name":"Two","groups":[{"id":"room-2","class":"classroom","name":"Room 2","band":"display"}]}}
      ]}
    ]`)
	state := CurrentState{SchoolYear: data.SchoolYear{ID: "year-1", State: data.SchoolYearSetup}, Homerooms: []data.Homeroom{
		{ID: "homeroom-1", ExternalIdentifier: stringPointer("room-1")},
		{ID: "homeroom-2", ExternalIdentifier: stringPointer("room-2")},
	}}

	preview := classifyRoster(document, state)
	require.Equal(t, OutcomeCounts{Create: 5}, preview.Counts)
	require.Len(t, preview.Rows, 1)
	require.Equal(t, OutcomeCreate, preview.Rows[0].Outcome)
	require.Len(t, preview.Rows[0].Records, 5, "one wide row contains the adult, two students, and two guardian edges")
	require.Equal(t, "adult-1", preview.Rows[0].Records[0].SourceExternalIdentifier)
	require.Equal(t, "guardian_relationship", preview.Rows[0].Records[4].RecordType)

	state.Adults = []data.Adult{{ID: "adult-db", ExternalIdentifier: stringPointer("adult-1"), LegalGivenName: "Alex", LegalFamilyName: "Adult", Email: stringPointer("alex@example.test"), ParticipationIntent: func() *data.AdultParticipationIntent { value := data.AdultParticipationHelp; return &value }()}}
	state.Students = []data.Student{
		{ID: "student-db-1", ExternalIdentifier: stringPointer("child-1"), LegalGivenName: "Casey", LegalFamilyName: "One", HomeroomID: "homeroom-1", GradeLevelID: stringID("grade-1"), PreferredGivenName: stringPointer("Cass")},
		{ID: "student-db-2", ExternalIdentifier: stringPointer("child-2"), LegalGivenName: "Riley", LegalFamilyName: "Two", HomeroomID: "homeroom-2"},
	}
	state.Relationships = []data.GuardianRelationship{
		{ID: "relationship-db-1", AdultID: "adult-db", StudentID: "student-db-1", RelationshipType: data.GuardianRelationshipParent},
		{ID: "relationship-db-2", AdultID: "adult-db", StudentID: "student-db-2", RelationshipType: data.GuardianRelationshipParent},
	}

	unchanged := classifyRoster(document, state)
	require.Equal(t, OutcomeCounts{Unchanged: 5}, unchanged.Counts)
	require.Empty(t, unchanged.GuardianRelationshipRemovals)

	*state.Adults[0].Email = "edited@example.test"
	edited := classifyRoster(document, state)
	require.Equal(t, OutcomeCounts{Update: 1, Unchanged: 4}, edited.Counts)
	adultRecord := edited.Rows[0].Records[0]
	require.Equal(t, OutcomeUpdate, adultRecord.Outcome)
	require.Equal(t, []FieldChange{{Field: "email", Before: "edited@example.test", After: "alex@example.test"}}, adultRecord.Changes)
	studentRecord := edited.Rows[0].Records[1]
	require.Equal(t, OutcomeUnchanged, studentRecord.Outcome)
	require.Empty(t, studentRecord.Changes, "omitted preferred name, grade, phone, and participation fields are never changed")
}

func TestRosterPreviewGoldenCorpusIsIdempotentAndFieldSpecific(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("roster", "testdata", "synthetic_roster.json"))
	require.NoError(t, err)
	document, err := roster.ParseDocument(encoded)
	require.NoError(t, err)

	fresh := CurrentState{SchoolYear: data.SchoolYear{ID: "year-golden", State: data.SchoolYearSetup}, Homerooms: goldenHomerooms(document)}
	preview := classifyRoster(document, fresh)
	require.Equal(t, OutcomeCounts{Create: 714}, preview.Counts)
	require.Len(t, preview.Rows, 226)
	require.Len(t, preview.Exclusions, 160, "62 excluded children and 98 excluded adults are reported outside the five outcomes")

	current := goldenCurrentState(document)
	repeated := classifyRoster(document, current)
	require.Equal(t, OutcomeCounts{Unchanged: 714}, repeated.Counts)
	require.Empty(t, repeated.GuardianRelationshipRemovals)

	current.Adults[0].Email = stringPointer("hand-edited@example.test")
	edited := classifyRoster(document, current)
	require.Equal(t, OutcomeCounts{Update: 1, Unchanged: 713}, edited.Counts)
	require.Equal(t, []FieldChange{{Field: "email", Before: "hand-edited@example.test", After: "synthetic-001@example.test"}}, edited.Rows[0].Records[0].Changes)
}

func TestRosterPreviewReportsSoftDeleteUnresolvedClassroomAndEdgeRemoval(t *testing.T) {
	document := parsePreviewDocument(t, `[
      {"id":"adult-1","given_name":"Alex","family_name":"Adult","relationships":[
        {"relationship":"MOM","child":{"id":"child-1","given_name":"Casey","family_name":"One","groups":[{"id":"missing-room","class":"classroom","name":"Missing Room","band":"7"}]}}
      ]}
    ]`)
	deletedAt := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	state := CurrentState{
		SchoolYear: data.SchoolYear{ID: "year-1", State: data.SchoolYearSetup},
		Adults:     []data.Adult{{ID: "adult-db", ExternalIdentifier: stringPointer("adult-1"), LegalGivenName: "Alex", LegalFamilyName: "Adult", DeletedAt: &deletedAt}},
		Homerooms:  []data.Homeroom{},
	}
	preview := classifyRoster(document, state)
	require.Equal(t, OutcomeConflict, preview.CountsToRecord("adult-1", "adult"))
	require.Contains(t, preview.Rows[0].Records[0].Detail, "deleted at 2026-08-30T10:00:00Z")
	require.Equal(t, OutcomeError, preview.Rows[0].Records[1].Outcome)
	require.Contains(t, preview.Rows[0].Records[1].Detail, `source id "missing-room"`)

	// A represented adult row owns only its listed edge. An active edge to a
	// different child is therefore disclosed for removal, not hidden by the
	// person-level outcome.
	removalDocument := parsePreviewDocument(t, `[
      {"id":"adult-2","given_name":"Sam","family_name":"Adult","relationships":[
        {"relationship":"MOM","child":{"id":"child-1","given_name":"Casey","family_name":"One","groups":[{"id":"room-1","class":"classroom","name":"Room 1","band":"display"}]}}
      ]}
    ]`)
	state = CurrentState{
		SchoolYear: data.SchoolYear{ID: "year-1", State: data.SchoolYearSetup},
		Adults:     []data.Adult{{ID: "adult-db", ExternalIdentifier: stringPointer("adult-2"), LegalGivenName: "Sam", LegalFamilyName: "Adult"}},
		Students: []data.Student{
			{ID: "student-db-1", ExternalIdentifier: stringPointer("child-1"), LegalGivenName: "Casey", LegalFamilyName: "One", HomeroomID: "homeroom-1"},
			{ID: "student-db-2", ExternalIdentifier: stringPointer("child-2"), LegalGivenName: "Riley", LegalFamilyName: "Two", HomeroomID: "homeroom-1"},
		},
		Relationships: []data.GuardianRelationship{
			{ID: "relationship-db-1", AdultID: "adult-db", StudentID: "student-db-1", RelationshipType: data.GuardianRelationshipParent},
			{ID: "relationship-db-2", AdultID: "adult-db", StudentID: "student-db-2", RelationshipType: data.GuardianRelationshipGuardian},
		},
		Homerooms: []data.Homeroom{{ID: "homeroom-1", ExternalIdentifier: stringPointer("room-1")}},
	}
	preview = classifyRoster(removalDocument, state)
	require.Len(t, preview.GuardianRelationshipRemovals, 1)
	require.Equal(t, "child-2", preview.GuardianRelationshipRemovals[0].StudentExternalIdentifier)
	require.Equal(t, "relationship-db-2", preview.GuardianRelationshipRemovals[0].ExistingID)
}

func TestRosterPreviewTreatsAnEmptyAdultRowAsAuthoritative(t *testing.T) {
	document := roster.Document{
		Result: roster.Result{
			Adults:   []roster.Adult{{SourceExternalIdentifier: "adult-1", GivenName: "Alex", FamilyName: "Adult"}},
			Students: []roster.Student{{SourceExternalIdentifier: "child-1", GivenName: "Casey", FamilyName: "One", ClassroomExternalIdentifier: "room-1"}},
		},
		Rows: []roster.SourceRow{{Number: 1, Adult: roster.Adult{SourceExternalIdentifier: "adult-1", GivenName: "Alex", FamilyName: "Adult"}}},
	}
	state := CurrentState{
		SchoolYear:    data.SchoolYear{ID: "year-1", State: data.SchoolYearSetup},
		Adults:        []data.Adult{{ID: "adult-db", ExternalIdentifier: stringPointer("adult-1"), LegalGivenName: "Alex", LegalFamilyName: "Adult"}},
		Students:      []data.Student{{ID: "student-db", ExternalIdentifier: stringPointer("child-1"), LegalGivenName: "Casey", LegalFamilyName: "One"}},
		Relationships: []data.GuardianRelationship{{ID: "relationship-db", AdultID: "adult-db", StudentID: "student-db", RelationshipType: data.GuardianRelationshipParent}},
	}

	preview := classifyRoster(document, state)
	require.Len(t, preview.GuardianRelationshipRemovals, 1)
	require.Equal(t, "relationship-db", preview.GuardianRelationshipRemovals[0].ExistingID)
}

func parsePreviewDocument(t *testing.T, source string) roster.Document {
	t.Helper()
	document, err := roster.ParseDocument([]byte(strings.TrimSpace(source)))
	require.NoError(t, err)
	return document
}

func stringPointer(value string) *string { return &value }

func stringID(value string) *ids.XID {
	result := ids.XID(value)
	return &result
}

func (p Preview) CountsToRecord(sourceID, recordType string) Outcome {
	for _, row := range p.Rows {
		for _, record := range row.Records {
			if record.SourceExternalIdentifier == sourceID && record.RecordType == recordType {
				return record.Outcome
			}
		}
	}
	return ""
}

func goldenHomerooms(document roster.Document) []data.Homeroom {
	seen := map[string]bool{}
	result := make([]data.Homeroom, 0, roster.SyntheticClassroomCnt)
	for _, source := range document.Result.Students {
		id := source.ClassroomExternalIdentifier
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, data.Homeroom{ID: ids.XID("db-" + id), ExternalIdentifier: stringPointer(id)})
	}
	return result
}

func goldenCurrentState(document roster.Document) CurrentState {
	state := CurrentState{SchoolYear: data.SchoolYear{ID: "year-golden", State: data.SchoolYearSetup}, Homerooms: goldenHomerooms(document)}
	adultIDs := make(map[string]ids.XID, len(document.Result.Adults))
	studentIDs := make(map[string]ids.XID, len(document.Result.Students))
	for _, source := range document.Result.Adults {
		id := ids.XID("db-" + source.SourceExternalIdentifier)
		adultIDs[source.SourceExternalIdentifier] = id
		state.Adults = append(state.Adults, data.Adult{
			ID: id, LegalGivenName: source.GivenName, LegalFamilyName: source.FamilyName,
			Email: optionalString(source.Email), ExternalIdentifier: stringPointer(source.SourceExternalIdentifier),
		})
	}
	for _, source := range document.Result.Students {
		id := ids.XID("db-" + source.SourceExternalIdentifier)
		studentIDs[source.SourceExternalIdentifier] = id
		state.Students = append(state.Students, data.Student{
			ID: id, LegalGivenName: source.GivenName, LegalFamilyName: source.FamilyName,
			HomeroomID: ids.XID("db-" + source.ClassroomExternalIdentifier), ExternalIdentifier: stringPointer(source.SourceExternalIdentifier),
		})
	}
	for _, source := range document.Result.GuardianRelationships {
		state.Relationships = append(state.Relationships, data.GuardianRelationship{
			ID:      ids.XID("db-" + relationshipKey(source.AdultExternalIdentifier, source.StudentExternalIdentifier)),
			AdultID: adultIDs[source.AdultExternalIdentifier], StudentID: studentIDs[source.StudentExternalIdentifier],
			RelationshipType: data.GuardianRelationshipType(source.RelationshipType),
		})
	}
	return state
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return stringPointer(value)
}
