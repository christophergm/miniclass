package integration

import (
	"context"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

// SPEC §§13.5 and 13.6.1: absence is not a rating; labels follow the vocabulary.
func TestInterestResultsCanonicalRatingsAndCurrentVocabularyLabels(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	org := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, org)
	factory := factories.New(harness.Database, string(org), actor)
	fixture := createPreferenceFixture(t, harness, factory, "results-regressions")
	service := preference.New(harness.Database)
	secondStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Unrated Results", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, secondStudent.ID)
	require.NoError(t, err)
	survey, err := service.CreateInterestProfileSurvey(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name: "Synthetic Incomplete Results Scale", ScaleVersion: "custom-incomplete",
		Audience:     preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers},
		Questions:    []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.secondArea.ID}, {InterestAreaID: fixture.area.ID}},
		ScaleOptions: []preference.InterestProfileSurveyScaleOptionInput{{Value: "unrated", Label: "Skip", Ordinal: 1}, {Value: "interested", Label: "Custom positive", Ordinal: 2}},
	})
	require.NoError(t, err)
	closing := time.Now().UTC().Add(time.Hour)
	_, err = service.TransitionInterestProfileSurvey(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closing})
	require.NoError(t, err)
	for _, input := range []preference.InterestProfileSurveySubmissionInput{
		{StudentID: fixture.student.ID, Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested}, {InterestAreaID: fixture.secondArea.ID, Rating: data.InterestProfileNotInterested}}},
		{StudentID: secondStudent.ID, Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileUnrated}, {InterestAreaID: fixture.secondArea.ID, Rating: data.InterestProfileInterested}}},
	} {
		input.SchoolYearID, input.ProgramID, input.SurveyID, input.Channel = fixture.year.ID, fixture.program.ID, survey.Survey.ID, data.PreferenceChannelAdministratorOnBehalf
		_, err = service.SubmitInterestProfileSurvey(ctx, string(org), actor, input)
		require.NoError(t, err)
	}
	before, err := service.GetInterestProfileResults(ctx, string(org), fixture.year.ID, fixture.program.ID, survey.Survey.ID, data.PreferenceResultsFilter{})
	require.NoError(t, err)
	vocabulary := program.New(harness.Database)
	renamedLabel, retiredLabel, retired := "Synthetic Renamed Topic", "Synthetic Renamed Retired Topic", true
	_, err = vocabulary.UpdateInterestArea(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, fixture.area.ID, program.InterestAreaUpdate{Label: &renamedLabel})
	require.NoError(t, err)
	_, err = vocabulary.UpdateInterestArea(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, fixture.secondArea.ID, program.InterestAreaUpdate{Label: &retiredLabel, Retired: &retired})
	require.NoError(t, err)
	after, err := service.GetInterestProfileResults(ctx, string(org), fixture.year.ID, fixture.program.ID, survey.Survey.ID, data.PreferenceResultsFilter{})
	require.NoError(t, err)
	require.Len(t, after.Items, 2)
	require.Equal(t, retiredLabel, after.Items[0].Label, "retired questions still resolve the current vocabulary label")
	require.Equal(t, renamedLabel, after.Items[1].Label, "an open survey must not display the snapshotted label")
	for i := range before.Items {
		require.Equal(t, before.Items[i].ID, after.Items[i].ID)
		require.Equal(t, before.Items[i].Ordinal, after.Items[i].Ordinal)
		require.Equal(t, before.Items[i].RatingCounts, after.Items[i].RatingCounts)
	}
	require.Equal(t, fixture.secondArea.ID, after.Items[0].ID)
	require.Equal(t, fixture.area.ID, after.Items[1].ID)
	require.Equal(t, 2, after.Items[0].ExplicitAnswers)
	require.Zero(t, after.Items[0].Unanswered)
	require.Equal(t, 1, after.Items[1].ExplicitAnswers)
	require.Equal(t, 1, after.Items[1].Unanswered)
	require.Equal(t, []preference.PreferenceFormScaleOption{{Value: "interested", Label: "Custom positive", Ordinal: 2}, {Value: "very_interested", Label: "Very interested", Ordinal: 3}, {Value: "not_interested", Label: "Not interested", Ordinal: 4}}, after.ScaleOptions)
	require.Equal(t, []preference.RatingCount{{Value: "interested", Label: "Custom positive", Ordinal: 2, Count: 1}, {Value: "very_interested", Label: "Very interested", Ordinal: 3}, {Value: "not_interested", Label: "Not interested", Ordinal: 4, Count: 1}}, after.Items[0].RatingCounts)
	require.Equal(t, []preference.RatingCount{{Value: "interested", Label: "Custom positive", Ordinal: 2}, {Value: "very_interested", Label: "Very interested", Ordinal: 3, Count: 1}, {Value: "not_interested", Label: "Not interested", Ordinal: 4}}, after.Items[1].RatingCounts)
}

// SPEC §§13.3, 13.5–13.7, 19.4–19.5: retained history is not additional demand.
func TestPreferenceResultsCurrentAnswersFiltersAndIsolation(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	org := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, org)
	factory := factories.New(harness.Database, string(org), actor)
	fixture := createPreferenceFixture(t, harness, factory, "results")
	service := preference.New(harness.Database)
	secondGrade, err := factory.CreateGradeLevel(ctx, fixture.year.ID, "results-second", "Synthetic Second Grade")
	require.NoError(t, err)
	secondRoom, err := factory.CreateHomeroom(ctx, fixture.year.ID, "Synthetic Second Room")
	require.NoError(t, err)
	students := []data.Student{fixture.student}
	for i, input := range []people.StudentCreateInput{
		{LegalGivenName: "Synthetic", LegalFamilyName: "Results Two", GradeLevelID: &secondGrade.ID, HomeroomID: fixture.student.HomeroomID},
		{LegalGivenName: "Synthetic", LegalFamilyName: "Results Three", GradeLevelID: &fixture.grade.ID, HomeroomID: secondRoom.ID},
		{LegalGivenName: "Synthetic", LegalFamilyName: "Results Four", GradeLevelID: &secondGrade.ID, HomeroomID: secondRoom.ID},
	} {
		student, err := factory.CreateStudent(ctx, fixture.year.ID, input)
		require.NoError(t, err, i)
		_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, student.ID)
		require.NoError(t, err)
		students = append(students, student)
	}
	guardian, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Results Guardian"})
	require.NoError(t, err)
	for _, student := range []data.Student{students[0], students[2], students[3]} {
		_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: guardian.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent})
		require.NoError(t, err)
	}
	thirdArea, err := factory.CreateInterestArea(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Zero Answer Area")
	require.NoError(t, err)
	survey, err := service.CreateInterestProfileSurvey(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name: "Synthetic Results Survey", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers}, ScaleVersion: "results-custom",
		Questions:    []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}, {InterestAreaID: fixture.secondArea.ID}, {InterestAreaID: thirdArea.ID}},
		ScaleOptions: []preference.InterestProfileSurveyScaleOptionInput{{Value: "not_interested", Label: "No", Ordinal: 1}, {Value: "interested", Label: "Yes", Ordinal: 2}, {Value: "very_interested", Label: "Very", Ordinal: 3}},
	})
	require.NoError(t, err)
	closing := time.Now().UTC().Add(time.Hour)
	_, err = service.TransitionInterestProfileSurvey(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closing})
	require.NoError(t, err)
	submitInterest := func(surveyID, studentID ids.XID, answers []data.InterestProfileAnswer) {
		t.Helper()
		_, err := service.SubmitInterestProfileSurvey(ctx, string(org), actor, preference.InterestProfileSurveySubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: surveyID, StudentID: studentID, Channel: data.PreferenceChannelAdministratorOnBehalf, Answers: answers})
		require.NoError(t, err)
	}
	submitInterest(survey.Survey.ID, students[0].ID, []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested}, {InterestAreaID: fixture.secondArea.ID, Rating: data.InterestProfileInterested}})
	submitInterest(survey.Survey.ID, students[0].ID, []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileNotInterested}})
	submitInterest(survey.Survey.ID, students[1].ID, []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileUnrated}})
	otherSurvey := openPreferenceSurvey(t, harness, org, actor, fixture)
	submitInterest(otherSurvey, students[0].ID, []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested}})
	submitInterest(otherSurvey, students[2].ID, []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested}})

	all, err := service.GetInterestProfileResults(ctx, string(org), fixture.year.ID, fixture.program.ID, survey.Survey.ID, data.PreferenceResultsFilter{})
	require.NoError(t, err)
	require.Equal(t, "open", all.State)
	require.Equal(t, "results-custom", all.ScaleVersion)
	require.Equal(t, 4, all.TotalStudents)
	require.Equal(t, 2, all.RespondedStudents)
	require.Equal(t, 50.0, all.CompletionPercentage)
	require.Len(t, all.Items, 3)
	require.Equal(t, fixture.area.ID, all.Items[0].ID)
	require.Equal(t, 1, all.Items[0].ExplicitAnswers)
	require.Equal(t, 1, all.Items[0].Unanswered)
	require.Equal(t, []preference.RatingCount{{Value: "not_interested", Label: "No", Ordinal: 1, Count: 1}, {Value: "interested", Label: "Yes", Ordinal: 2}, {Value: "very_interested", Label: "Very", Ordinal: 3}}, all.Items[0].RatingCounts)
	require.Equal(t, 1, all.Items[1].ExplicitAnswers, "partial resubmission retains this survey's prior per-area rating")
	require.Equal(t, 1, all.Items[1].Unanswered)
	require.Zero(t, all.Items[2].ExplicitAnswers)
	require.Equal(t, 2, all.Items[2].Unanswered, "non-submitters do not contribute unanswered votes")
	other, err := service.GetInterestProfileResults(ctx, string(org), fixture.year.ID, fixture.program.ID, otherSurvey, data.PreferenceResultsFilter{})
	require.NoError(t, err)
	require.Zero(t, other.Items[1].ExplicitAnswers, "never fill an item from another survey")
	require.Equal(t, 2, other.Items[1].Unanswered)

	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Results Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offerings := []data.Offering{}
	for _, name := range []string{"Synthetic A", "Synthetic B", "Synthetic C"} {
		offering, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, name, "Synthetic description", nil, 10, fixture.grade.ID, secondGrade.ID, "", "", "", nil)
		require.NoError(t, err)
		offerings = append(offerings, offering)
	}
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 2, closing)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	submitRanked := func(studentID ids.XID, answers []data.RankedChoiceResponseInput) {
		t.Helper()
		_, err := service.SubmitRankedChoices(ctx, string(org), actor, preference.RankedChoiceSubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: studentID, Channel: data.PreferenceChannelAdministratorOnBehalf, Responses: answers})
		require.NoError(t, err)
	}
	rankOne, rankTwo := 1, 2
	submitRanked(students[0].ID, []data.RankedChoiceResponseInput{{OfferingID: offerings[0].ID, Answer: data.RankedChoiceRanked, Rank: &rankOne}, {OfferingID: offerings[1].ID, Answer: data.RankedChoiceInterested}, {OfferingID: offerings[2].ID, Answer: data.RankedChoiceInterested}})
	submitRanked(students[0].ID, []data.RankedChoiceResponseInput{{OfferingID: offerings[0].ID, Answer: data.RankedChoiceNotInterested}, {OfferingID: offerings[1].ID, Answer: data.RankedChoiceRanked, Rank: &rankTwo}, {OfferingID: offerings[2].ID, Answer: data.RankedChoiceNoResponse}})
	submitRanked(students[1].ID, []data.RankedChoiceResponseInput{{OfferingID: offerings[0].ID, Answer: data.RankedChoiceInterested}, {OfferingID: offerings[1].ID, Answer: data.RankedChoiceNoResponse}, {OfferingID: offerings[2].ID, Answer: data.RankedChoiceNoResponse}})
	_, err = service.SubmitRankedChoices(ctx, string(org), actor, preference.RankedChoiceSubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: students[0].ID, Channel: data.PreferenceChannelAdministratorOnBehalf, Responses: []data.RankedChoiceResponseInput{{OfferingID: offerings[0].ID, Answer: data.RankedChoiceRanked, Rank: &rankOne}, {OfferingID: offerings[1].ID, Answer: data.RankedChoiceRanked, Rank: &rankOne}, {OfferingID: offerings[2].ID, Answer: data.RankedChoiceNoResponse}}})
	require.ErrorIs(t, err, preference.ErrRankedChoiceInvalid)
	ranked, err := service.GetRankedChoiceResults(ctx, string(org), fixture.year.ID, fixture.program.ID, session.ID, data.PreferenceResultsFilter{})
	require.NoError(t, err)
	require.Equal(t, "voting_open", ranked.State)
	require.Equal(t, 2, ranked.RankDepth)
	require.Equal(t, 4, ranked.TotalStudents)
	require.Equal(t, 2, ranked.RespondedStudents)
	require.Equal(t, 2, ranked.Items[0].ExplicitAnswers)
	require.Equal(t, 1, ranked.Items[0].Interested)
	require.Equal(t, 1, ranked.Items[0].NotInterested)
	require.Equal(t, []preference.RankCount{{Rank: 1}, {Rank: 2}}, ranked.Items[0].RankCounts)
	require.Equal(t, []preference.RankCount{{Rank: 1}, {Rank: 2, Count: 1}}, ranked.Items[1].RankCounts)
	require.Equal(t, 1, ranked.Items[1].Unanswered)
	require.Zero(t, ranked.Items[2].ExplicitAnswers, "ranked response replacement clears earlier interest")
	require.Equal(t, 2, ranked.Items[2].Unanswered)

	for _, test := range []struct {
		name             string
		filter           data.PreferenceResultsFilter
		total, responded int
	}{
		{"empty is all", data.PreferenceResultsFilter{}, 4, 2},
		{"OR within grades", data.PreferenceResultsFilter{GradeLevelIDs: []string{string(fixture.grade.ID), string(secondGrade.ID)}, HomeroomIDs: []string{string(fixture.student.HomeroomID)}}, 2, 2},
		{"OR within homerooms", data.PreferenceResultsFilter{GradeLevelIDs: []string{string(fixture.grade.ID)}, HomeroomIDs: []string{string(fixture.student.HomeroomID), string(secondRoom.ID)}}, 2, 1},
		{"AND between groups", data.PreferenceResultsFilter{GradeLevelIDs: []string{string(fixture.grade.ID)}, HomeroomIDs: []string{string(fixture.student.HomeroomID)}}, 1, 1},
		{"nonresponder only", data.PreferenceResultsFilter{GradeLevelIDs: []string{string(fixture.grade.ID)}, HomeroomIDs: []string{string(secondRoom.ID)}}, 1, 0},
		{"no match", data.PreferenceResultsFilter{GradeLevelIDs: []string{"not-a-grade"}}, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			interest, err := service.GetInterestProfileResults(ctx, string(org), fixture.year.ID, fixture.program.ID, survey.Survey.ID, test.filter)
			require.NoError(t, err)
			ranked, err := service.GetRankedChoiceResults(ctx, string(org), fixture.year.ID, fixture.program.ID, session.ID, test.filter)
			require.NoError(t, err)
			for _, summary := range []preference.ResponseTrackingSummary{interest.ResponseTrackingSummary, ranked.ResponseTrackingSummary} {
				require.Equal(t, test.total, summary.TotalStudents)
				require.Equal(t, test.responded, summary.RespondedStudents)
			}
			interestTracking, err := service.GetInterestProfileResponseTracking(ctx, string(org), fixture.year.ID, fixture.program.ID, survey.Survey.ID, test.filter)
			require.NoError(t, err)
			rankedTracking, err := service.GetRankedChoiceResponseTracking(ctx, string(org), fixture.year.ID, fixture.program.ID, session.ID, test.filter)
			require.NoError(t, err)
			for _, report := range []preference.ResponseTracking{interestTracking, rankedTracking} {
				require.Equal(t, test.total, report.TotalStudents)
				require.Equal(t, test.responded, report.RespondedStudents)
				require.Len(t, report.NonResponders, test.total-test.responded)
				require.Len(t, report.GuardianFollowUp, test.total-test.responded)
				for _, breakdown := range [][]preference.ResponseTrackingBreakdown{report.GradeBreakdown, report.HomeroomBreakdown} {
					total, responded := 0, 0
					for _, row := range breakdown {
						total += row.TotalStudents
						responded += row.RespondedStudents
					}
					require.Equal(t, test.total, total)
					require.Equal(t, test.responded, responded)
				}
			}
			summaries, err := service.ListResponseTrackingSummaries(ctx, string(org), fixture.year.ID, fixture.program.ID, test.filter)
			require.NoError(t, err)
			require.Len(t, summaries, 3)
			for _, summary := range summaries {
				if summary.InstrumentID == otherSurvey {
					continue
				}
				require.Equal(t, test.total, summary.TotalStudents)
				require.Equal(t, test.responded, summary.RespondedStudents)
			}
			if test.responded == 0 {
				for _, item := range interest.Items {
					require.Zero(t, item.ExplicitAnswers)
					require.Zero(t, item.Unanswered)
				}
				for _, item := range ranked.Items {
					require.Zero(t, item.ExplicitAnswers)
					require.Zero(t, item.Unanswered)
				}
			}
		})
	}
	_, err = service.TransitionInterestProfileSurvey(ctx, string(org), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyClosed, Reason: "synthetic results close"})
	require.NoError(t, err)
	closed, err := service.GetInterestProfileResults(ctx, string(org), fixture.year.ID, fixture.program.ID, survey.Survey.ID, data.PreferenceResultsFilter{})
	require.NoError(t, err)
	require.Equal(t, "closed", closed.State)
	require.Equal(t, all.Items, closed.Items)

	foreign := harness.MintOrganization(t)
	_, err = service.GetInterestProfileResults(ctx, string(foreign), fixture.year.ID, fixture.program.ID, survey.Survey.ID, data.PreferenceResultsFilter{})
	require.Error(t, err)
	_, err = service.GetRankedChoiceResults(ctx, string(foreign), fixture.year.ID, fixture.program.ID, session.ID, data.PreferenceResultsFilter{})
	require.Error(t, err)
	for _, readOrg := range []ids.XID{org, foreign} {
		require.NoError(t, harness.Database.InTenantRead(ctx, string(readOrg), func(ctx context.Context, tx *data.Tx) error {
			interestAnswers, err := tx.ListCurrentSurveyResultAnswers(ctx, fixture.year.ID, fixture.program.ID, survey.Survey.ID)
			require.NoError(t, err)
			rankedAnswers, err := tx.ListCurrentSessionResultAnswers(ctx, fixture.year.ID, fixture.program.ID, session.ID)
			require.NoError(t, err)
			if readOrg == foreign {
				require.Empty(t, interestAnswers, "foreign instrument is invisible")
				require.Empty(t, rankedAnswers, "foreign instrument is invisible")
			} else {
				require.Len(t, interestAnswers, 3)
				require.Len(t, rankedAnswers, 6, "one latest complete response per student, including stored no-response rows")
			}
			return nil
		}))
	}
}
