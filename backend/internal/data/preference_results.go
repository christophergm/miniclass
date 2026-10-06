package data

import (
	"context"
	"fmt"
	"slices"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

// PreferenceResultsFilter selects current roster attributes: OR within a group,
// AND between groups. Empty groups do not restrict the instrument's audience.
type PreferenceResultsFilter struct {
	GradeLevelIDs []string
	HomeroomIDs   []string
}

func (f PreferenceResultsFilter) Students(rows []ResponseTrackingStudentRow) []ResponseTrackingStudentRow {
	result := make([]ResponseTrackingStudentRow, 0, len(rows))
	for _, row := range rows {
		if len(f.GradeLevelIDs) > 0 && (row.GradeLevelID == nil || !slices.Contains(f.GradeLevelIDs, string(*row.GradeLevelID))) {
			continue
		}
		if len(f.HomeroomIDs) > 0 && !slices.Contains(f.HomeroomIDs, string(row.HomeroomID)) {
			continue
		}
		result = append(result, row)
	}
	return result
}

func reportFilter(filters []PreferenceResultsFilter) PreferenceResultsFilter {
	if len(filters) > 0 {
		return filters[0]
	}
	return PreferenceResultsFilter{}
}

type SurveyResultAnswer struct {
	StudentID      ids.XID
	InterestAreaID ids.XID
	Rating         InterestProfileRating
}

type SessionResultAnswer struct {
	StudentID  ids.XID
	OfferingID ids.XID
	Answer     RankedChoiceAnswer
	Rank       *int
}

func (tx *Tx) ListCurrentSurveyResultAnswers(ctx context.Context, year, program, survey ids.XID) ([]SurveyResultAnswer, error) {
	rows, err := tx.queries.ListCurrentSurveyResultAnswers(ctx, db.ListCurrentSurveyResultAnswersParams{OrganizationID: tx.organizationID, SchoolYearID: year, ProgramID: program, SurveyID: &survey})
	if err != nil {
		return nil, fmt.Errorf("list current survey result answers: %w", err)
	}
	result := make([]SurveyResultAnswer, 0, len(rows))
	for _, row := range rows {
		result = append(result, SurveyResultAnswer{StudentID: row.StudentID, InterestAreaID: row.InterestAreaID, Rating: InterestProfileRating(row.Response)})
	}
	return result, nil
}

func (tx *Tx) ListCurrentSessionResultAnswers(ctx context.Context, year, program, session ids.XID) ([]SessionResultAnswer, error) {
	rows, err := tx.queries.ListCurrentSessionResultAnswers(ctx, db.ListCurrentSessionResultAnswersParams{OrganizationID: tx.organizationID, SchoolYearID: year, ProgramID: program, SessionID: session})
	if err != nil {
		return nil, fmt.Errorf("list current session result answers: %w", err)
	}
	result := make([]SessionResultAnswer, 0, len(rows))
	for _, row := range rows {
		var rank *int
		if row.Rank.Valid {
			value := int(row.Rank.Int32)
			rank = &value
		}
		result = append(result, SessionResultAnswer{StudentID: row.StudentID, OfferingID: row.OfferingID, Answer: RankedChoiceAnswer(row.Response), Rank: rank})
	}
	return result, nil
}
