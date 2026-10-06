package preference

import (
	"context"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

type RatingCount struct {
	Value, Label   string
	Ordinal, Count int
}
type RankCount struct{ Rank, Count int }
type InterestProfileResultItem struct {
	ID                                   ids.XID
	Label                                string
	Ordinal, ExplicitAnswers, Unanswered int
	RatingCounts                         []RatingCount
}
type RankedChoiceResultItem struct {
	ID                          ids.XID
	Label                       string
	ExplicitAnswers, Unanswered int
	RankCounts                  []RankCount
	Interested, NotInterested   int
}
type InterestProfileResults struct {
	ResponseTrackingSummary
	ScaleVersion string
	ScaleOptions []PreferenceFormScaleOption
	Items        []InterestProfileResultItem
}
type RankedChoiceResults struct {
	ResponseTrackingSummary
	RankDepth int
	Items     []RankedChoiceResultItem
}

func (s *Service) GetInterestProfileResults(ctx context.Context, organizationID string, year, program, surveyID ids.XID, filter data.PreferenceResultsFilter) (InterestProfileResults, error) {
	if s == nil || s.database == nil {
		return InterestProfileResults{}, ErrPreferenceServiceNil
	}
	var result InterestProfileResults
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, year, program, surveyID)
		if err != nil {
			return err
		}
		students, err := tx.ListInterestProfileResponseTrackingStudents(ctx, year, program, surveyID)
		if err != nil {
			return err
		}
		questions, err := tx.ListInterestProfileSurveyQuestions(ctx, year, program, surveyID)
		if err != nil {
			return err
		}
		// SPEC §13.6.1: the question snapshot fixes identity and order, not
		// display wording. Retired areas still label historical questions.
		areas, err := tx.ListInterestAreas(ctx, year, program, true)
		if err != nil {
			return err
		}
		labels := make(map[ids.XID]string, len(areas))
		for _, area := range areas {
			labels[area.ID] = area.Label
		}
		for i := range questions {
			questions[i].Label = labels[questions[i].InterestAreaID]
		}
		options, err := tx.ListInterestProfileSurveyScaleOptions(ctx, year, program, surveyID)
		if err != nil {
			return err
		}
		answers, err := tx.ListCurrentSurveyResultAnswers(ctx, year, program, surveyID)
		if err != nil {
			return err
		}
		result = buildInterestProfileResults(survey, filter.Students(students), questions, options, answers)
		return nil
	})
	return result, err
}

func (s *Service) GetRankedChoiceResults(ctx context.Context, organizationID string, year, program, sessionID ids.XID, filter data.PreferenceResultsFilter) (RankedChoiceResults, error) {
	if s == nil || s.database == nil {
		return RankedChoiceResults{}, ErrPreferenceServiceNil
	}
	var result RankedChoiceResults
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, year, program, sessionID)
		if err != nil {
			return err
		}
		if session.RankedChoice == nil {
			return ErrRankedChoiceNotConfigured
		}
		students, err := tx.ListRankedChoiceResponseTrackingStudents(ctx, year, program, sessionID)
		if err != nil {
			return err
		}
		offerings, err := tx.ListOfferings(ctx, year, program, sessionID)
		if err != nil {
			return err
		}
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, year, program, sessionID)
		if err != nil {
			return err
		}
		result = buildRankedChoiceResults(session, filter.Students(students), offerings, answers)
		return nil
	})
	return result, err
}

func resultsSummary(kind ResponseTrackingInstrumentType, id ids.XID, name, state string, year, program ids.XID, students []data.ResponseTrackingStudentRow) (ResponseTrackingSummary, map[ids.XID]bool) {
	summary := ResponseTrackingSummary{InstrumentType: kind, InstrumentID: id, InstrumentName: name, State: state, SchoolYearID: year, ProgramID: program, TotalStudents: len(students)}
	submitters := make(map[ids.XID]bool)
	for _, student := range students {
		if student.Responded {
			summary.RespondedStudents++
			submitters[student.ID] = true
		}
	}
	summary.CompletionPercentage = completionPercentage(summary.RespondedStudents, summary.TotalStudents)
	return summary, submitters
}

func buildInterestProfileResults(survey data.InterestProfileSurvey, students []data.ResponseTrackingStudentRow, questions []data.InterestProfileSurveyQuestion, options []data.InterestProfileSurveyScaleOption, answers []data.SurveyResultAnswer) InterestProfileResults {
	summary, submitters := resultsSummary(ResponseTrackingInterestProfile, survey.ID, survey.Name, string(effectiveSurveyState(survey, time.Now().UTC())), survey.SchoolYearID, survey.ProgramID, students)
	result := InterestProfileResults{ResponseTrackingSummary: summary, ScaleVersion: survey.ScaleVersion, ScaleOptions: []PreferenceFormScaleOption{}, Items: []InterestProfileResultItem{}}
	seenOptions := make(map[string]bool)
	maxOrdinal := 0
	for _, option := range options {
		if !isExplicitInterestRating(data.InterestProfileRating(option.Value)) {
			continue
		}
		result.ScaleOptions = append(result.ScaleOptions, PreferenceFormScaleOption{Value: option.Value, Label: option.Label, Ordinal: option.Ordinal})
		seenOptions[option.Value] = true
		maxOrdinal = max(maxOrdinal, option.Ordinal)
	}
	includedAreas := make(map[ids.XID]bool, len(questions))
	for _, question := range questions {
		includedAreas[question.InterestAreaID] = true
	}
	counts := make(map[ids.XID]map[string]int)
	usedRatings := make(map[string]bool)
	for _, answer := range answers {
		// SPEC §13.5: unrated is absence, even if the configured scale lists it.
		if !submitters[answer.StudentID] || !includedAreas[answer.InterestAreaID] || !isExplicitInterestRating(answer.Rating) {
			continue
		}
		if counts[answer.InterestAreaID] == nil {
			counts[answer.InterestAreaID] = make(map[string]int)
		}
		counts[answer.InterestAreaID][string(answer.Rating)]++
		usedRatings[string(answer.Rating)] = true
	}
	// Stored canonical answers remain explicit when scale metadata is incomplete.
	// Keep instrument labels/order and append only the missing values in use.
	for _, option := range defaultInterestProfileSurveyScale {
		if usedRatings[option.Value] && !seenOptions[option.Value] {
			maxOrdinal++
			result.ScaleOptions = append(result.ScaleOptions, PreferenceFormScaleOption{Value: option.Value, Label: option.Label, Ordinal: maxOrdinal})
		}
	}
	for _, question := range questions {
		item := InterestProfileResultItem{ID: question.InterestAreaID, Label: question.Label, Ordinal: question.Ordinal, RatingCounts: []RatingCount{}}
		for _, option := range result.ScaleOptions {
			count := counts[question.InterestAreaID][option.Value]
			item.RatingCounts = append(item.RatingCounts, RatingCount{Value: option.Value, Label: option.Label, Ordinal: option.Ordinal, Count: count})
			item.ExplicitAnswers += count
		}
		item.Unanswered = summary.RespondedStudents - item.ExplicitAnswers
		result.Items = append(result.Items, item)
	}
	return result
}

func isExplicitInterestRating(rating data.InterestProfileRating) bool {
	switch rating {
	case data.InterestProfileVeryInterested, data.InterestProfileInterested, data.InterestProfileNotInterested:
		return true
	default:
		return false
	}
}

func buildRankedChoiceResults(session data.Session, students []data.ResponseTrackingStudentRow, offerings []data.Offering, answers []data.SessionResultAnswer) RankedChoiceResults {
	state := session.State
	if state == data.SessionVotingOpen && session.RankedChoice.Deadline != nil && !time.Now().UTC().Before(*session.RankedChoice.Deadline) {
		state = data.SessionVotingClosed
	}
	summary, submitters := resultsSummary(ResponseTrackingRankedChoice, session.ID, session.Name, string(state), session.SchoolYearID, session.ProgramID, students)
	result := RankedChoiceResults{ResponseTrackingSummary: summary, RankDepth: session.RankedChoice.RankDepth, Items: []RankedChoiceResultItem{}}
	byOffering := make(map[ids.XID]*RankedChoiceResultItem)
	for _, offering := range offerings {
		item := RankedChoiceResultItem{ID: offering.ID, Label: offering.Name, RankCounts: []RankCount{}}
		for rank := 1; rank <= result.RankDepth; rank++ {
			item.RankCounts = append(item.RankCounts, RankCount{Rank: rank})
		}
		result.Items = append(result.Items, item)
	}
	for i := range result.Items {
		byOffering[result.Items[i].ID] = &result.Items[i]
	}
	for _, answer := range answers {
		item := byOffering[answer.OfferingID]
		if item == nil || !submitters[answer.StudentID] {
			continue
		}
		switch answer.Answer {
		case data.RankedChoiceRanked:
			if answer.Rank == nil || *answer.Rank < 1 || *answer.Rank > result.RankDepth {
				continue
			}
			item.RankCounts[*answer.Rank-1].Count++
		case data.RankedChoiceInterested:
			item.Interested++
		case data.RankedChoiceNotInterested:
			item.NotInterested++
		default:
			continue
		}
		item.ExplicitAnswers++
	}
	for i := range result.Items {
		result.Items[i].Unanswered = summary.RespondedStudents - result.Items[i].ExplicitAnswers
	}
	return result
}
