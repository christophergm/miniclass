package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestEvaluateCatalogFeasibilityReportsAllApplicableChecks(t *testing.T) {
	g1 := data.GradeLevel{ID: "grade-1", Label: "Grade 1", Ordinal: 1}
	g2 := data.GradeLevel{ID: "grade-2", Label: "Grade 2", Ordinal: 2}
	areaCovered := data.InterestArea{ID: "area-covered", Label: "Art"}
	areaMissing := data.InterestArea{ID: "area-missing", Label: "Nature"}
	participants := []CatalogParticipant{
		{StudentID: "student-1", GradeLevelID: g1.ID, GradeLabel: g1.Label, GradeOrdinal: g1.Ordinal, KnownGrade: true},
		{StudentID: "student-2", GradeLevelID: g1.ID, GradeLabel: g1.Label, GradeOrdinal: g1.Ordinal, KnownGrade: true},
		{StudentID: "student-3", GradeLevelID: g2.ID, GradeLabel: g2.Label, GradeOrdinal: g2.Ordinal, KnownGrade: true},
	}
	offerings := []data.Offering{
		{ID: "offering-untagged", Capacity: 2, MinGradeLevelID: g1.ID, MaxGradeLevelID: g1.ID},
		{ID: "offering-art", Capacity: 1, MinGradeLevelID: g1.ID, MaxGradeLevelID: g1.ID, InterestAreaID: xidPointer(areaCovered.ID)},
	}
	result := EvaluateCatalogFeasibility(CatalogFeasibilitySnapshot{
		Participants:  participants,
		Grades:        []data.GradeLevel{g1, g2},
		Offerings:     offerings,
		InterestAreas: []data.InterestArea{areaCovered, areaMissing},
		AreaDemand:    []CatalogAreaDemand{{InterestAreaID: areaMissing.ID, HighRatingCount: 3}},
	})

	require.Equal(t, 3, result.ParticipantCount)
	require.Equal(t, []string{CatalogGradeGapWarning, CatalogAreaGapWarning, CatalogUnmatchedOfferingWarning}, warningIDs(result))
	require.Equal(t, []ids.XID{"offering-untagged"}, result.Warnings[2].OfferingIDs)
	require.Equal(t, []CatalogGradeGap{{ID: g2.ID, Label: g2.Label, ParticipantCount: 1}}, result.Warnings[0].AffectedGrades)
	require.Equal(t, []CatalogAreaGap{{ID: areaMissing.ID, Label: areaMissing.Label, HighRatingCount: 3}}, result.Warnings[1].AffectedAreas)
	require.Equal(t, CatalogInfoSeverity, result.Warnings[1].Severity)
}

func TestEvaluateCatalogFeasibilityReportsCapacityAndMinimumSeparately(t *testing.T) {
	participants := []CatalogParticipant{{StudentID: "student-1"}, {StudentID: "student-2"}, {StudentID: "student-3"}}
	capacityResult := EvaluateCatalogFeasibility(CatalogFeasibilitySnapshot{
		Participants: participants,
		Offerings:    []data.Offering{{ID: "small", Capacity: 2}},
	})
	require.Equal(t, []string{CatalogCapacityShortWarning, CatalogUnmatchedOfferingWarning}, warningIDs(capacityResult))
	require.Equal(t, 2, capacityResult.Warnings[0].TotalCapacity)
	require.Equal(t, 1, capacityResult.Warnings[0].Shortfall)

	minimum := 2
	minimumResult := EvaluateCatalogFeasibility(CatalogFeasibilitySnapshot{
		Participants: participants,
		Offerings:    []data.Offering{{ID: "large", Capacity: 4, MinimumViableEnrollment: &minimum, InterestAreaID: xidPointer("area")}},
	})
	require.Empty(t, minimumResult.Warnings)

	minimum = 4
	minimumResult = EvaluateCatalogFeasibility(CatalogFeasibilitySnapshot{
		Participants: participants,
		Offerings:    []data.Offering{{ID: "large", Capacity: 4, MinimumViableEnrollment: &minimum, InterestAreaID: xidPointer("area")}},
	})
	require.Equal(t, []string{CatalogMinimumViabilityWarning}, warningIDs(minimumResult))
	require.Equal(t, 1, minimumResult.Warnings[0].Shortfall)
}

func TestEvaluateCatalogFeasibilityDoesNotWarnForCoveredAreasOrRankedChoices(t *testing.T) {
	area := data.InterestArea{ID: "area", Label: "Art"}
	result := EvaluateCatalogFeasibility(CatalogFeasibilitySnapshot{
		Participants:     []CatalogParticipant{{StudentID: "student"}},
		Grades:           []data.GradeLevel{{ID: "grade", Ordinal: 1}},
		Offerings:        []data.Offering{{ID: "offering", Capacity: 1, MinGradeLevelID: "grade", MaxGradeLevelID: "grade", InterestAreaID: xidPointer(area.ID)}},
		InterestAreas:    []data.InterestArea{area},
		AreaDemand:       []CatalogAreaDemand{{InterestAreaID: area.ID, HighRatingCount: 4}},
		HasRankedChoices: true,
	})
	require.Empty(t, result.Warnings)
}

func TestEvaluateCatalogFeasibilityIsDeterministic(t *testing.T) {
	g1 := data.GradeLevel{ID: "grade-1", Label: "Grade 1", Ordinal: 1}
	g2 := data.GradeLevel{ID: "grade-2", Label: "Grade 2", Ordinal: 2}
	snapshot := CatalogFeasibilitySnapshot{
		Participants: []CatalogParticipant{
			{StudentID: "student-2", GradeLevelID: g2.ID, GradeLabel: g2.Label, GradeOrdinal: g2.Ordinal, KnownGrade: true},
			{StudentID: "student-1", GradeLevelID: g1.ID, GradeLabel: g1.Label, GradeOrdinal: g1.Ordinal, KnownGrade: true},
		},
		Grades: []data.GradeLevel{g2, g1},
		Offerings: []data.Offering{
			{ID: "offering-b", Capacity: 1, MinGradeLevelID: g1.ID, MaxGradeLevelID: g2.ID},
			{ID: "offering-a", Capacity: 1, MinGradeLevelID: g1.ID, MaxGradeLevelID: g2.ID},
		},
	}
	first := EvaluateCatalogFeasibility(snapshot)
	second := EvaluateCatalogFeasibility(snapshot)
	require.Equal(t, first, second)
	require.Equal(t, []ids.XID{"offering-a", "offering-b"}, first.Warnings[0].OfferingIDs)
}

func warningIDs(value CatalogFeasibility) []string {
	result := make([]string, 0, len(value.Warnings))
	for _, warning := range value.Warnings {
		result = append(result, warning.ID)
	}
	return result
}

func xidPointer(value ids.XID) *ids.XID { return &value }
