package program

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestArtifactPrivacyFilteringAndMissingData(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	secret := "SECRET old preferred name"
	r := artifactRoster{
		session: data.Session{Name: "Synthetic session", DraftAssignmentsStale: true},
		students: []data.Student{
			{ID: "active", LegalGivenName: "Alex", LegalFamilyName: "Synthetic", PreferredGivenName: &secret, DeletedAt: &now},
			{ID: "placeholder", LegalGivenName: "Child", LegalFamilyName: "A", IsPlaceholder: true, PreferredGivenName: &secret},
			{ID: "unplaced", LegalGivenName: "Unplaced", LegalFamilyName: "Synthetic"},
			{ID: "excluded", LegalGivenName: "SECRET", LegalFamilyName: "Excluded"},
			{ID: "nonmember", LegalGivenName: "SECRET", LegalFamilyName: "Nonmember"},
			{ID: "deleted-unplaced", LegalGivenName: "SECRET", DeletedAt: &now},
		},
		memberships:       []data.ProgramMembership{{StudentID: "active", LegalGivenName: "SECRET historical"}, {StudentID: "placeholder"}, {StudentID: "unplaced"}, {StudentID: "excluded"}, {StudentID: "deleted-unplaced"}},
		nonParticipations: []data.SessionNonParticipation{{StudentID: "excluded", Reason: "SECRET reason"}},
		offerings:         []data.Offering{{ID: "o", Name: "Synthetic offering"}, {ID: "empty", Name: "Empty offering"}},
		assignments:       []data.Assignment{{StudentID: "active", OfferingID: "o"}, {StudentID: "placeholder", OfferingID: "o"}, {StudentID: "excluded", OfferingID: "o"}, {StudentID: "nonmember", OfferingID: "o"}},
	}
	for _, kind := range []ArtifactKind{ArtifactClassList, ArtifactHomeroomDismissal} {
		doc := renderArtifact(r, kind, now)
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		require.NotContains(t, string(raw), "SECRET")
		require.Equal(t, now, doc.GeneratedAt)
		expectedWarnings := []ArtifactWarning{
			{Code: "unplaced_students", Message: "Participating students without a current placement are not included in the document.", StudentNames: []string{"Unplaced Synthetic"}},
			{Code: "stale_assignments", Message: "Current assignments may be stale after roster or catalog changes.", StudentNames: []string{}},
			{Code: "missing_grade", Message: "Placed students have no grade set.", StudentNames: []string{"Child A", "Deleted student"}},
			{Code: "missing_homeroom", Message: "Placed students have no homeroom set.", StudentNames: []string{"Child A", "Deleted student"}},
		}
		if kind == ArtifactClassList {
			expectedWarnings = append(expectedWarnings, ArtifactWarning{Code: "missing_destination", Message: `Offering "Empty offering" has no meeting destination set.`, StudentNames: []string{}})
		}
		expectedWarnings = append(expectedWarnings, ArtifactWarning{Code: "missing_destination", Message: `Offering "Synthetic offering" has no meeting destination set.`, StudentNames: []string{"Child A", "Deleted student"}})
		require.Equal(t, expectedWarnings, doc.Warnings)
		sectionsJSON, err := json.Marshal(doc.Sections)
		require.NoError(t, err)
		require.NotContains(t, string(sectionsJSON), "has no meeting destination")
		require.NotContains(t, string(sectionsJSON), "Placed students have no")
		require.Contains(t, string(raw), "Deleted student")
		require.Contains(t, string(raw), "Child A")
		require.Contains(t, string(raw), "Grade not set")
		require.Contains(t, string(raw), "Homeroom not set")
		require.Contains(t, string(raw), "Meeting location not set")
		if kind == ArtifactClassList {
			require.Len(t, doc.Sections, 2)
			require.Equal(t, "empty", doc.Sections[0].ID)
			require.Contains(t, doc.Sections[0].Blocks, ArtifactBlock{Kind: "paragraph", Label: "Enrolled count", Text: "0"})
			require.Contains(t, doc.Sections[1].Blocks, ArtifactBlock{Kind: "paragraph", Label: "Enrolled count", Text: "2"})
		} else {
			require.Len(t, doc.Sections, 1)
			for _, b := range doc.Sections[0].Blocks {
				require.Contains(t, []string{"paragraph", "bullet"}, b.Kind)
				require.NotEqual(t, "Enrolled count", b.Label)
			}
		}
	}
}

func TestArtifactOrderingAndIdentifierGrouping(t *testing.T) {
	g1, g2 := ids.XID("g1"), ids.XID("g2")
	r := artifactRoster{session: data.Session{MeetingDates: []time.Time{time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)}},
		grades:      []data.GradeLevel{{ID: g1, Label: "1", Ordinal: 1}, {ID: g2, Label: "2", Ordinal: 2}},
		homerooms:   []data.Homeroom{{ID: "h2", Name: "Room"}, {ID: "h1", Name: "room"}},
		offerings:   []data.Offering{{ID: "o2", Name: "class", Location: "Gym", MeetingPoint: "Gate", Description: "Description", MeetingInstructions: "Instructions", MinGradeLevelID: g1, MaxGradeLevelID: g2}, {ID: "o1", Name: "Class", Location: "Gym", MeetingPoint: "Gym"}},
		students:    []data.Student{{ID: "s1", LegalGivenName: "Zed", GradeLevelID: &g1, HomeroomID: "h1"}, {ID: "s2", LegalGivenName: "Amy", GradeLevelID: &g2, HomeroomID: "h1"}, {ID: "s3", LegalGivenName: "Amy", HomeroomID: "h2"}, {ID: "s4", LegalGivenName: "Missing"}},
		memberships: []data.ProgramMembership{{StudentID: "s1"}, {StudentID: "s2"}, {StudentID: "s3"}, {StudentID: "s4"}},
		assignments: []data.Assignment{{StudentID: "s2", OfferingID: "o2"}, {StudentID: "s1", OfferingID: "o2"}, {StudentID: "s3", OfferingID: "o1"}, {StudentID: "s4", OfferingID: "o1"}},
	}
	doc := renderArtifact(r, ArtifactClassList, time.Now())
	require.Equal(t, []string{"o1", "o2"}, []string{doc.Sections[0].ID, doc.Sections[1].ID})
	numbered := []string{}
	for _, b := range doc.Sections[1].Blocks {
		if b.Kind == "numbered" {
			numbered = append(numbered, b.Label)
		}
	}
	require.Equal(t, []string{"Zed", "Amy"}, numbered)
	require.Contains(t, doc.Sections[1].Blocks, ArtifactBlock{Kind: "paragraph", Label: "Grades", Text: "1–2"})
	require.Contains(t, doc.Sections[1].Blocks, ArtifactBlock{Kind: "paragraph", Label: "Meeting point", Text: "Gate"})
	for _, b := range doc.Sections[0].Blocks {
		require.NotEqual(t, "Meeting point", b.Label)
	}
	dismissal := renderArtifact(r, ArtifactHomeroomDismissal, time.Now())
	require.Equal(t, []string{"homeroom-not-set", "h1", "h2"}, []string{dismissal.Sections[0].ID, dismissal.Sections[1].ID, dismissal.Sections[2].ID})
	require.Equal(t, []ArtifactBlock{{Kind: "paragraph", Label: "class", Text: "meet at Gate"}, {Kind: "bullet", Text: "Amy (Grade 2)"}, {Kind: "bullet", Text: "Zed (Grade 1)"}}, dismissal.Sections[1].Blocks)
	require.Equal(t, []ArtifactWarning{
		{Code: "missing_grade", Message: "Placed students have no grade set.", StudentNames: []string{"Amy", "Missing"}},
		{Code: "missing_homeroom", Message: "Placed students have no homeroom set.", StudentNames: []string{"Missing"}},
	}, dismissal.Warnings)
}

func TestArtifactMissingDataWarningsRespectRenderedFieldsAndIdentifiers(t *testing.T) {
	gradeID := ids.XID("grade")
	r := artifactRoster{
		grades:    []data.GradeLevel{{ID: gradeID, Label: "1", Ordinal: 1}, {ID: "blank-grade", Label: "  "}},
		homerooms: []data.Homeroom{{ID: "room", Name: "Synthetic room"}, {ID: "blank-room", Name: "  "}},
		offerings: []data.Offering{
			{ID: "a", Name: "Duplicate", Location: "  ", MeetingPoint: "\t"},
			{ID: "b", Name: "Duplicate"},
			{ID: "c", Name: "Gate only", MeetingPoint: "Gate"},
			{ID: "d", Name: "Gym only", Location: "Gym"},
		},
		students: []data.Student{
			{ID: "a", LegalGivenName: "Zed", GradeLevelID: &gradeID, HomeroomID: "room"},
			{ID: "b", LegalGivenName: "Amy", GradeLevelID: &gradeID, HomeroomID: "room"},
			{ID: "c", LegalGivenName: "Missing", GradeLevelID: func() *ids.XID { id := ids.XID("blank-grade"); return &id }(), HomeroomID: "blank-room"},
		},
		memberships: []data.ProgramMembership{{StudentID: "a"}, {StudentID: "b"}, {StudentID: "c"}},
		assignments: []data.Assignment{{StudentID: "a", OfferingID: "a"}, {StudentID: "b", OfferingID: "b"}, {StudentID: "c", OfferingID: "c"}},
	}
	for _, kind := range []ArtifactKind{ArtifactClassList, ArtifactHomeroomDismissal} {
		doc := renderArtifact(r, kind, time.Now())
		require.Equal(t, []ArtifactWarning{
			{Code: "missing_grade", Message: "Placed students have no grade set.", StudentNames: []string{"Missing"}},
			{Code: "missing_homeroom", Message: "Placed students have no homeroom set.", StudentNames: []string{"Missing"}},
			{Code: "missing_destination", Message: `Offering "Duplicate" has no meeting destination set.`, StudentNames: []string{"Zed"}},
			{Code: "missing_destination", Message: `Offering "Duplicate" has no meeting destination set.`, StudentNames: []string{"Amy"}},
		}, doc.Warnings)
		if kind == ArtifactHomeroomDismissal {
			require.Equal(t, "blank-room", doc.Sections[0].ID)
			require.Equal(t, "Homeroom not set", doc.Sections[0].Title)
			require.Equal(t, []ArtifactBlock{{Kind: "paragraph", Label: "Gate only", Text: "meet at Gate"}, {Kind: "bullet", Text: "Missing (Grade not set)"}}, doc.Sections[0].Blocks)
			require.Len(t, doc.Sections[1].Blocks, 4)
		}
	}
}

func TestGenerateArtifactRejectsUnknownKind(t *testing.T) {
	_, err := New(nil).GenerateArtifact(t.Context(), "org", "year", "program", "session", "unknown")
	require.ErrorIs(t, err, ErrArtifactKind)
}
