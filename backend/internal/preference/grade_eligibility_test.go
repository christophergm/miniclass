package preference

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestGradeEligibleOfferingsUsesInclusiveVocabularyOrdinals(t *testing.T) {
	grades := []data.GradeLevel{
		{ID: "grade-z", Label: "Reception", Ordinal: 1},
		{ID: "grade-a", Label: "Y1", Ordinal: 2},
		{ID: "grade-m", Label: "Y2", Ordinal: 3},
	}
	offerings := []data.Offering{
		{ID: "below", MinGradeLevelID: "grade-z", MaxGradeLevelID: "grade-z"},
		{ID: "at-min", MinGradeLevelID: "grade-a", MaxGradeLevelID: "grade-m"},
		{ID: "at-max", MinGradeLevelID: "grade-z", MaxGradeLevelID: "grade-a"},
		{ID: "above", MinGradeLevelID: "grade-m", MaxGradeLevelID: "grade-m"},
		{ID: "unknown-bound", MinGradeLevelID: "missing", MaxGradeLevelID: "grade-m"},
	}
	studentGrade := ids.XID("grade-a")
	require.Equal(t, offerings[1:3], gradeEligibleOfferings(&studentGrade, grades, offerings))
	require.Empty(t, gradeEligibleOfferings(nil, grades, offerings))
	unknown := ids.XID("missing")
	require.Empty(t, gradeEligibleOfferings(&unknown, grades, offerings))
}

func TestCompleteHiddenResponsesPreservesCompleteCatalogValidation(t *testing.T) {
	eligible := []data.Offering{{ID: "eligible"}}
	offerings := append(append([]data.Offering{}, eligible...), data.Offering{ID: "hidden"})
	responses := []data.RankedChoiceResponseInput{{OfferingID: "eligible", Answer: data.RankedChoiceInterested}}
	complete := completeHiddenRankedChoiceResponses(responses, offerings, eligible)
	require.Equal(t, []data.RankedChoiceResponseInput{
		{OfferingID: "eligible", Answer: data.RankedChoiceInterested},
		{OfferingID: "hidden", Answer: data.RankedChoiceNoResponse},
	}, complete)
	require.NoError(t, ValidateRankedChoiceResponseSetWithDepth(complete, offerings, 1))
	require.Len(t, responses, 1)

	missing := completeHiddenRankedChoiceResponses(nil, offerings, eligible)
	require.ErrorIs(t, ValidateRankedChoiceResponseSetWithDepth(missing, offerings, 1), ErrRankedChoiceNotComplete)

	// Existing explicit responses, including organiser-entered choices, are not rewritten.
	explicit := []data.RankedChoiceResponseInput{
		{OfferingID: "eligible", Answer: data.RankedChoiceInterested},
		{OfferingID: "hidden", Answer: data.RankedChoiceInterested},
	}
	require.Equal(t, explicit, completeHiddenRankedChoiceResponses(explicit, offerings, eligible))
}
