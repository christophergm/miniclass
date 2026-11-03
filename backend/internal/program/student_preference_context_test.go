package program

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestBuildStudentPreferenceContextMapsCurrentRankingsAndProfile(t *testing.T) {
	gradeID, areaID := ids.XID("grade-synthetic"), ids.XID("area-synthetic")
	preferred := "Preferred"
	rank := 1
	submittedAt := time.Date(2026, time.November, 2, 12, 0, 0, 0, time.UTC)
	snapshot := studentPreferenceContextSnapshot{
		Session:   data.Session{ID: "session", DraftRevision: 4, RankedChoice: &data.RankedChoiceConfiguration{RankDepth: 3}},
		Student:   data.Student{ID: "student", LegalGivenName: "Legal", LegalFamilyName: "Student", PreferredGivenName: &preferred, GradeLevelID: &gradeID, HomeroomID: "room"},
		Grades:    []data.GradeLevel{{ID: gradeID, Label: "Synthetic Grade", Ordinal: 2}},
		Homerooms: []data.Homeroom{{ID: "room", Name: "Synthetic Room"}},
		Offerings: []data.Offering{
			{ID: "ranked", Name: "Ranked", MinGradeLevelID: gradeID, MaxGradeLevelID: gradeID, InterestAreaID: &areaID},
			{ID: "interested", Name: "Interested", MinGradeLevelID: gradeID, MaxGradeLevelID: gradeID},
			{ID: "blank", Name: "Blank", MinGradeLevelID: gradeID, MaxGradeLevelID: gradeID},
			{ID: "not-interested", Name: "Not Interested", MinGradeLevelID: gradeID, MaxGradeLevelID: gradeID},
		},
		Areas:        []data.InterestArea{{ID: areaID, Label: "Synthetic Area"}, {ID: "unrated", Label: "Synthetic Unrated"}},
		ProfileAreas: []data.InterestArea{{ID: areaID, Label: "Synthetic Area"}, {ID: "unrated", Label: "Synthetic Unrated"}},
		Assignments:  []data.Assignment{{ID: "assignment", StudentID: "student", OfferingID: "ranked", RealizedQuality: "top"}},
		Comments:     []data.PlacementComment{{ID: "comment", HostType: data.CommentHostAssignment, HostID: "assignment", Body: "Synthetic note"}},
		Answers: []data.SessionResultAnswer{
			{StudentID: "student", OfferingID: "ranked", Answer: data.RankedChoiceRanked, Rank: &rank},
			{StudentID: "student", OfferingID: "interested", Answer: data.RankedChoiceInterested},
			{StudentID: "student", OfferingID: "blank", Answer: data.RankedChoiceNoResponse},
			{StudentID: "student", OfferingID: "not-interested", Answer: data.RankedChoiceNotInterested},
		},
		RankedSubmissions:  []data.RankedChoiceSubmission{{ID: "ranking"}},
		ProfileSubmissions: []data.InterestProfileSubmission{{ID: "profile", SurveyID: preferenceContextXIDPointer("survey")}},
		EffectiveProfile:   []data.EffectiveInterestProfileValue{{InterestAreaID: areaID, Rating: data.InterestProfileVeryInterested, SubmissionID: "profile", SubmittedAt: submittedAt}},
		Surveys:            []data.InterestProfileSurvey{{ID: "survey", Name: "Synthetic survey"}},
		Relationships:      []data.GuardianRelationship{{AdultID: "adult", RelationshipType: data.GuardianRelationshipParent}},
		Adults:             []data.Adult{{ID: "adult", LegalGivenName: "Guardian", LegalFamilyName: "Synthetic"}},
		HighRankMax:        3,
	}

	context := buildStudentPreferenceContext(snapshot)
	require.Equal(t, int64(4), context.DraftRevision)
	require.Equal(t, "Preferred Student", context.Student.DisplayName)
	require.Equal(t, "Legal Student", context.Student.LegalName)
	require.Equal(t, "Synthetic Grade", context.Student.GradeLabel)
	require.Equal(t, "Synthetic Room", context.Student.HomeroomName)
	require.Equal(t, "ranked", string(*context.Student.Placement.OfferingID))
	require.Len(t, context.CurrentComments, 1)
	require.Equal(t, []StudentLinkedAdult{{DisplayName: "Guardian Synthetic", Relationship: "parent"}}, context.Student.LinkedAdults)
	require.True(t, context.Rankings.RankedChoiceEnabled)
	require.True(t, context.Rankings.Submitted)
	require.Equal(t, []string{"ranked", "interested", "blank", "not-interested"}, rankingOfferingIDs(context.Rankings.Rows))
	require.Equal(t, PotentialQualityTop, context.Rankings.Rows[0].PotentialQuality)
	require.True(t, context.Rankings.Rows[0].IsCurrentPlacement)
	require.Equal(t, PotentialQualityNeutral, context.Rankings.Rows[2].PotentialQuality)
	require.True(t, context.InterestProfile.Responded)
	require.Equal(t, []StudentInterestArea{{ID: string(areaID), Label: "Synthetic Area", SurveyName: "Synthetic survey", SubmittedAt: &submittedAt, Offerings: []StudentInterestOffering{{ID: "ranked", Name: "Ranked"}}}}, context.InterestProfile.VeryInterested)
	require.Equal(t, []StudentInterestArea{{ID: "unrated", Label: "Synthetic Unrated", Offerings: []StudentInterestOffering{}}}, context.InterestProfile.NotRated)
}

func TestStudentPriorPlacementUsesHistoricalRankedAnswerOrStoredQuality(t *testing.T) {
	for _, test := range []struct {
		name, quality, want string
		response            *data.RankedChoiceResponse
	}{
		{name: "rank", response: &data.RankedChoiceResponse{Answer: data.RankedChoiceRanked, Rank: intPointer(1)}, want: "1st choice"},
		{name: "interested", response: &data.RankedChoiceResponse{Answer: data.RankedChoiceInterested}, want: "Interested"},
		{name: "not interested", response: &data.RankedChoiceResponse{Answer: data.RankedChoiceNotInterested}, want: "Not interested"},
		{name: "high", quality: "high", want: "Topic match: high"},
		{name: "acceptable", quality: "acceptable", want: "Topic match: acceptable"},
		{name: "unwanted", quality: "unwanted", want: "Topic match: unwanted"},
		{name: "neutral", quality: "neutral", want: "No preference signal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, historicalPreferenceLabel(test.quality, test.response))
		})
	}
}

func rankingOfferingIDs(rows []StudentRanking) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.OfferingID)
	}
	return result
}

func preferenceContextXIDPointer(value ids.XID) *ids.XID { return &value }
func intPointer(value int) *int                          { return &value }
