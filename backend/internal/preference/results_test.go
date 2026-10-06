package preference

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestInterestResultsUseInstrumentScaleAndSubmitterDenominator(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	survey := data.InterestProfileSurvey{ID: "survey", State: data.InterestProfileSurveyOpen, ClosesAt: &expired, ScaleVersion: "custom"}
	students := []data.ResponseTrackingStudentRow{{ID: "one", Responded: true}, {ID: "two", Responded: true}, {ID: "three"}}
	questions := []data.InterestProfileSurveyQuestion{{InterestAreaID: "area", Label: "Area", Ordinal: 1}, {InterestAreaID: "empty", Label: "Empty", Ordinal: 2}}
	options := []data.InterestProfileSurveyScaleOption{{Value: "not_interested", Label: "No", Ordinal: 1}, {Value: "interested", Label: "Yes", Ordinal: 2}}
	answers := []data.SurveyResultAnswer{{StudentID: "one", InterestAreaID: "area", Rating: data.InterestProfileNotInterested}, {StudentID: "two", InterestAreaID: "area", Rating: data.InterestProfileUnrated}, {StudentID: "outside", InterestAreaID: "area", Rating: data.InterestProfileInterested}}
	result := buildInterestProfileResults(survey, students, questions, options, answers)
	require.Equal(t, "closed", result.State)
	require.Equal(t, "custom", result.ScaleVersion)
	require.Equal(t, 3, result.TotalStudents)
	require.Equal(t, 2, result.RespondedStudents)
	require.InDelta(t, 66.666666, result.CompletionPercentage, 0.000001)
	require.Equal(t, 1, result.Items[0].ExplicitAnswers)
	require.Equal(t, 1, result.Items[0].Unanswered)
	require.Equal(t, []RatingCount{{Value: "not_interested", Label: "No", Ordinal: 1, Count: 1}, {Value: "interested", Label: "Yes", Ordinal: 2}}, result.Items[0].RatingCounts)
	require.Zero(t, result.Items[1].ExplicitAnswers)
	require.Equal(t, 2, result.Items[1].Unanswered)
	empty := buildInterestProfileResults(survey, nil, questions, options, answers)
	require.Zero(t, empty.CompletionPercentage)
	require.Len(t, empty.Items, 2)
	require.Zero(t, empty.Items[0].Unanswered)
	require.Zero(t, empty.Items[0].ExplicitAnswers)
}

func TestInterestResultsExcludeUnratedScaleAndRestoreMissingCanonicalRatings(t *testing.T) {
	for _, missing := range []data.InterestProfileRating{data.InterestProfileVeryInterested, data.InterestProfileInterested, data.InterestProfileNotInterested} {
		t.Run(string(missing), func(t *testing.T) {
			students := []data.ResponseTrackingStudentRow{{ID: "explicit", Responded: true}, {ID: "unrated", Responded: true}, {ID: "nonresponder"}}
			questions := []data.InterestProfileSurveyQuestion{{InterestAreaID: "area", Label: "Area", Ordinal: 1}, {InterestAreaID: "empty", Label: "Empty", Ordinal: 2}}
			options := []data.InterestProfileSurveyScaleOption{{Value: "unrated", Label: "Not an explicit answer", Ordinal: 1}}
			for _, option := range defaultInterestProfileSurveyScale {
				if option.Value != string(missing) {
					options = append(options, data.InterestProfileSurveyScaleOption{Value: option.Value, Label: "Custom " + option.Label, Ordinal: len(options) + 1})
				}
			}
			answers := []data.SurveyResultAnswer{
				{StudentID: "explicit", InterestAreaID: "area", Rating: missing},
				{StudentID: "unrated", InterestAreaID: "area", Rating: data.InterestProfileUnrated},
			}
			result := buildInterestProfileResults(data.InterestProfileSurvey{ScaleVersion: "custom"}, students, questions, options, answers)
			require.Equal(t, 3, result.TotalStudents)
			require.Equal(t, 2, result.RespondedStudents)
			require.Equal(t, "custom", result.ScaleVersion)
			require.Len(t, result.ScaleOptions, 3)
			for i, option := range result.ScaleOptions {
				require.NotEqual(t, "unrated", option.Value)
				require.Equal(t, option.Value, result.Items[0].RatingCounts[i].Value)
				require.Equal(t, option.Label, result.Items[0].RatingCounts[i].Label)
				require.Equal(t, option.Ordinal, result.Items[0].RatingCounts[i].Ordinal)
				if option.Value == string(missing) {
					require.Equal(t, 4, option.Ordinal, "fallback is appended after configured canonical options")
					require.Equal(t, 1, result.Items[0].RatingCounts[i].Count)
				} else {
					require.Contains(t, option.Label, "Custom ")
					require.Zero(t, result.Items[0].RatingCounts[i].Count)
				}
			}
			require.Equal(t, 1, result.Items[0].ExplicitAnswers)
			require.Equal(t, 1, result.Items[0].Unanswered)
			require.Zero(t, result.Items[1].ExplicitAnswers)
			require.Equal(t, 2, result.Items[1].Unanswered)
			for _, count := range result.Items[1].RatingCounts {
				require.Zero(t, count.Count)
			}
		})
	}
}

func TestRankedResultsIncludeAllRanksAndNegativeAnswers(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	session := data.Session{State: data.SessionVotingOpen, RankedChoice: &data.RankedChoiceConfiguration{RankDepth: 3, Deadline: &expired}}
	students := []data.ResponseTrackingStudentRow{{ID: "one", Responded: true}, {ID: "two", Responded: true}, {ID: "three", Responded: true}, {ID: "nonresponder"}}
	offerings := []data.Offering{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "zero", Name: "Zero"}}
	rank := 2
	answers := []data.SessionResultAnswer{
		{StudentID: "one", OfferingID: "a", Answer: data.RankedChoiceRanked, Rank: &rank},
		{StudentID: "two", OfferingID: "a", Answer: data.RankedChoiceInterested},
		{StudentID: "three", OfferingID: "a", Answer: data.RankedChoiceNotInterested},
		{StudentID: "one", OfferingID: "b", Answer: data.RankedChoiceNoResponse},
		{StudentID: "outside", OfferingID: "zero", Answer: data.RankedChoiceInterested},
	}
	result := buildRankedChoiceResults(session, students, offerings, answers)
	require.Equal(t, "voting_closed", result.State)
	require.Equal(t, 4, result.TotalStudents)
	require.Equal(t, 3, result.RespondedStudents)
	require.Equal(t, 75.0, result.CompletionPercentage)
	require.Equal(t, []RankCount{{Rank: 1}, {Rank: 2, Count: 1}, {Rank: 3}}, result.Items[0].RankCounts)
	require.Equal(t, 3, result.Items[0].ExplicitAnswers)
	require.Zero(t, result.Items[0].Unanswered)
	require.Equal(t, 1, result.Items[0].Interested)
	require.Equal(t, 1, result.Items[0].NotInterested)
	require.Equal(t, 3, result.Items[1].Unanswered)
	require.Zero(t, result.Items[2].ExplicitAnswers)
	require.Equal(t, 3, result.Items[2].Unanswered)
}

func TestReportFiltersApplyToEveryCompletionProjection(t *testing.T) {
	students := []data.ResponseTrackingStudentRow{
		{ID: "one", GradeLevelID: xidPtr("g1"), HomeroomID: "h1", Responded: true},
		{ID: "two", GradeLevelID: xidPtr("g2"), HomeroomID: "h1"},
		{ID: "three", GradeLevelID: xidPtr("g1"), HomeroomID: "h2"},
		{ID: "four", HomeroomID: "h2"},
	}
	for _, test := range []struct {
		name             string
		filter           data.PreferenceResultsFilter
		total, responded int
	}{
		{"all", data.PreferenceResultsFilter{}, 4, 1},
		{"grade OR", data.PreferenceResultsFilter{GradeLevelIDs: []string{"g1", "g2"}}, 3, 1},
		{"homeroom OR", data.PreferenceResultsFilter{HomeroomIDs: []string{"h1", "h2"}}, 4, 1},
		{"intersection", data.PreferenceResultsFilter{GradeLevelIDs: []string{"g1"}, HomeroomIDs: []string{"h1"}}, 1, 1},
		{"no match", data.PreferenceResultsFilter{GradeLevelIDs: []string{"foreign"}}, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			relationships := []data.GuardianRelationship{{StudentID: "two", AdultID: "adult"}, {StudentID: "three", AdultID: "adult"}, {StudentID: "four", AdultID: "adult"}}
			report := buildResponseTracking(ResponseTrackingInterestProfile, "survey", "Survey", "year", "program", test.filter.Students(students), relationships, []data.Adult{{ID: "adult"}})
			require.Equal(t, test.total, report.TotalStudents)
			require.Equal(t, test.responded, report.RespondedStudents)
			require.Len(t, report.NonResponders, test.total-test.responded)
			require.Len(t, report.GuardianFollowUp, test.total-test.responded)
			for _, breakdown := range [][]ResponseTrackingBreakdown{report.GradeBreakdown, report.HomeroomBreakdown} {
				total, responded := 0, 0
				for _, row := range breakdown {
					total += row.TotalStudents
					responded += row.RespondedStudents
				}
				require.Equal(t, test.total, total)
				require.Equal(t, test.responded, responded)
			}
		})
	}
}
