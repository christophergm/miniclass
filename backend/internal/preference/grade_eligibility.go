package preference

import (
	"context"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

func rankedChoiceEligibleOfferings(ctx context.Context, tx *data.Tx, schoolYearID ids.XID, student data.Student, offerings []data.Offering) ([]data.Offering, error) {
	grades, err := tx.ListGradeLevels(ctx, schoolYearID, true)
	if err != nil {
		return nil, err
	}
	return gradeEligibleOfferings(student.GradeLevelID, grades, offerings), nil
}

// Grade windows are inclusive and use the school year's vocabulary ordering
// (SPEC §§8.4, 10.1), never grade labels or opaque identifier ordering.
func gradeEligibleOfferings(studentGradeID *ids.XID, grades []data.GradeLevel, offerings []data.Offering) []data.Offering {
	eligible := make([]data.Offering, 0, len(offerings))
	if studentGradeID == nil {
		return eligible
	}
	ordinals := make(map[ids.XID]int, len(grades))
	for _, grade := range grades {
		ordinals[grade.ID] = grade.Ordinal
	}
	studentOrdinal, known := ordinals[*studentGradeID]
	if !known {
		return eligible
	}
	for _, offering := range offerings {
		minimum, minKnown := ordinals[offering.MinGradeLevelID]
		maximum, maxKnown := ordinals[offering.MaxGradeLevelID]
		if minKnown && maxKnown && studentOrdinal >= minimum && studentOrdinal <= maximum {
			eligible = append(eligible, offering)
		}
	}
	return eligible
}

// Keep the complete-catalog response contract (SPEC §13.3) while allowing
// forms to omit hidden classes. Explicit responses remain unchanged.
func completeHiddenRankedChoiceResponses(responses []data.RankedChoiceResponseInput, offerings, eligible []data.Offering) []data.RankedChoiceResponseInput {
	visible := make(map[ids.XID]bool, len(eligible))
	for _, offering := range eligible {
		visible[offering.ID] = true
	}
	answered := make(map[ids.XID]bool, len(responses))
	for _, response := range responses {
		answered[response.OfferingID] = true
	}
	complete := append([]data.RankedChoiceResponseInput(nil), responses...)
	for _, offering := range offerings {
		if !visible[offering.ID] && !answered[offering.ID] {
			complete = append(complete, data.RankedChoiceResponseInput{OfferingID: offering.ID, Answer: data.RankedChoiceNoResponse})
		}
	}
	return complete
}
