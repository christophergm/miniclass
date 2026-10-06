package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestEvaluateAssignmentQualityReportsNamedWarningsAndHistoricalQuality(t *testing.T) {
	minimum := 3
	result := EvaluateAssignmentQuality(AssignmentQualitySnapshot{
		Session:      data.Session{ID: "session", DraftAssignmentsStale: true},
		Grades:       []data.GradeLevel{{ID: "grade-1", Ordinal: 1}, {ID: "grade-2", Ordinal: 2}},
		Participants: []data.ProgramMembership{{StudentID: "ada", LegalGivenName: "Ada", LegalFamilyName: "One", GradeLevelID: xid("grade-1")}, {StudentID: "bea", LegalGivenName: "Bea", LegalFamilyName: "Two", GradeLevelID: xid("grade-2")}, {StudentID: "cy", LegalGivenName: "Cy", LegalFamilyName: "Three", GradeLevelID: xid("grade-1")}},
		Offerings:    []data.Offering{{ID: "offering", Capacity: 3, MinimumViableEnrollment: &minimum, MinGradeLevelID: "grade-1", MaxGradeLevelID: "grade-1"}},
		Assignments:  []data.Assignment{{ID: "assignment-ada", StudentID: "ada", OfferingID: "offering", RealizedQuality: "top"}, {ID: "assignment-bea", StudentID: "bea", OfferingID: "offering", RealizedQuality: "neutral"}},
		Exclusions:   []data.AssignmentExclusion{{StudentID: "ada", OfferingID: "offering"}},
		Overrides:    []data.AssignmentOverride{{AssignmentID: "assignment-bea"}},
		Answers:      []data.SessionResultAnswer{{StudentID: "ada", OfferingID: "offering", Answer: data.RankedChoiceNotInterested}},
	})
	require.Equal(t, []string{"Cy Three"}, placementNames(result.Unplaced))
	require.Equal(t, []string{"Ada One"}, placementNames(result.Unwanted))
	require.Equal(t, []string{"Bea Two"}, placementNames(result.NoSignal))
	require.Equal(t, []string{"Bea Two"}, placementNames(result.Overridden))
	require.Equal(t, 1, result.QualityDistribution["top"])
	require.Equal(t, 1, result.QualityDistribution["neutral"])
	require.Equal(t, []string{"below-minimum-enrollment"}, draftWarningIDs(result.Offerings[0].Warnings))
	require.ElementsMatch(t, []string{"below-minimum-enrollment", "catalog-grade-gap", "exclusion-overridden", "grade-out-of-range", "no-preference-signal", "non-preferred-placement", "stale-draft"}, draftWarningIDs(result.Warnings))
	// Stored quality is a historical fact even when current preferences differ.
	require.Equal(t, "top", result.Placements[0].Assignment.RealizedQuality)
	require.Equal(t, data.RankedChoiceNotInterested, result.Placements[0].CurrentPreference)

	capacity := EvaluateAssignmentQuality(AssignmentQualitySnapshot{Participants: []data.ProgramMembership{{StudentID: "a"}, {StudentID: "b"}}, Offerings: []data.Offering{{ID: "offering", Capacity: 1}}, Assignments: []data.Assignment{{ID: "one", StudentID: "a", OfferingID: "offering"}, {ID: "two", StudentID: "b", OfferingID: "offering"}}})
	require.Equal(t, []string{"capacity-exceeded"}, draftWarningIDs(capacity.Offerings[0].Warnings))
	require.Equal(t, 3, countDraftWarnings(capacity.Warnings, "capacity-exceeded")) // offering plus one warning on each placement
}

func xid(value string) *ids.XID { id := ids.XID(value); return &id }
func placementNames(values []DraftPlacement) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.StudentName)
	}
	return result
}
func draftWarningIDs(values []DraftWarning) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ID)
	}
	return result
}
func countDraftWarnings(values []DraftWarning, id string) int {
	count := 0
	for _, value := range values {
		if value.ID == id {
			count++
		}
	}
	return count
}
