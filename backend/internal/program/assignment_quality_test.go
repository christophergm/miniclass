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
	ada := placementByStudent(t, result.Placements, "ada")
	require.Equal(t, "top", ada.Assignment.RealizedQuality)
	require.Equal(t, data.RankedChoiceNotInterested, ada.CurrentPreference)

	capacity := EvaluateAssignmentQuality(AssignmentQualitySnapshot{Participants: []data.ProgramMembership{{StudentID: "a"}, {StudentID: "b"}}, Offerings: []data.Offering{{ID: "offering", Capacity: 1}}, Assignments: []data.Assignment{{ID: "one", StudentID: "a", OfferingID: "offering"}, {ID: "two", StudentID: "b", OfferingID: "offering"}}})
	require.Equal(t, []string{"capacity-exceeded"}, draftWarningIDs(capacity.Offerings[0].Warnings))
	require.Equal(t, 3, countDraftWarnings(capacity.Warnings, "capacity-exceeded")) // offering plus one warning on each placement
}

// SPEC §§16.5–16.6: each missing highly-rated area remains a visible occurrence.
func TestEvaluateAssignmentQualityCatalogAreaGapContext(t *testing.T) {
	result := EvaluateAssignmentQuality(AssignmentQualitySnapshot{
		Session: data.Session{ID: "session"},
		InterestAreas: []data.InterestArea{
			{ID: "missing-one", Label: "Synthetic Arts"},
			{ID: "missing-two", Label: "Synthetic Arts"},
			{ID: "covered", Label: "Synthetic Science"},
			{ID: "interested-only", Label: "Synthetic Music"},
			{ID: "unrated", Label: "Synthetic Games"},
			{ID: "no-demand", Label: "Synthetic Drama"},
		},
		Offerings: []data.Offering{
			{ID: "science-offering", InterestAreaID: xid("covered")},
			{ID: "untagged-offering"},
		},
		Profiles: map[ids.XID]map[ids.XID]data.InterestProfileRating{
			"student-one": {
				"missing-one":     data.InterestProfileVeryInterested,
				"missing-two":     data.InterestProfileVeryInterested,
				"covered":         data.InterestProfileVeryInterested,
				"interested-only": data.InterestProfileInterested,
				"unrated":         data.InterestProfileUnrated,
			},
			"student-two": {"missing-one": data.InterestProfileVeryInterested},
		},
	})

	require.Equal(t, []string{"catalog-area-gap", "catalog-area-gap"}, draftWarningIDs(result.Warnings))
	require.Equal(t, []DraftWarning{
		{
			ID: "catalog-area-gap", Severity: "info", HostType: "session", HostID: "session",
			Message:       `No offering covers interest area "Synthetic Arts", despite 2 participating students rating it very interested.`,
			AffectedAreas: []CatalogAreaGap{{ID: "missing-one", Label: "Synthetic Arts", HighRatingCount: 2}},
		},
		{
			ID: "catalog-area-gap", Severity: "info", HostType: "session", HostID: "session",
			Message:       `No offering covers interest area "Synthetic Arts", despite 1 participating student rating it very interested.`,
			AffectedAreas: []CatalogAreaGap{{ID: "missing-two", Label: "Synthetic Arts", HighRatingCount: 1}},
		},
	}, result.Warnings)
}

func xid(value string) *ids.XID { id := ids.XID(value); return &id }
func placementNames(values []DraftPlacement) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.StudentName)
	}
	return result
}

func placementByStudent(t *testing.T, values []DraftPlacement, studentID ids.XID) DraftPlacement {
	t.Helper()
	for _, value := range values {
		if value.Assignment.StudentID == studentID {
			return value
		}
	}
	require.FailNowf(t, "placement not found", "student ID %q", studentID)
	return DraftPlacement{}
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
