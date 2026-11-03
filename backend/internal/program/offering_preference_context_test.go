package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestEvaluateOfferingPreferenceContextGroupsAndOrdersParticipants(t *testing.T) {
	gradeLower := data.GradeLevel{ID: "grade-lower", Label: "Synthetic Lower", Ordinal: 1}
	gradeTarget := data.GradeLevel{ID: "grade-target", Label: "Synthetic Target", Ordinal: 2}
	areaID := ids.XID("area-synthetic")
	two := 2
	snapshot := offeringPreferenceContextSnapshot{
		Session:       data.Session{ID: "session-synthetic", DraftRevision: 7},
		Offering:      data.Offering{ID: "offering-target", Name: "Synthetic Target Offering", Capacity: 2, MinGradeLevelID: gradeTarget.ID, MaxGradeLevelID: gradeTarget.ID, InterestAreaID: &areaID},
		Offerings:     []data.Offering{{ID: "offering-target", Name: "Synthetic Target Offering"}, {ID: "offering-elsewhere", Name: "Synthetic Elsewhere Offering"}},
		Grades:        []data.GradeLevel{gradeLower, gradeTarget},
		Homerooms:     []data.Homeroom{{ID: "room-synthetic", Name: "Synthetic Room"}},
		InterestAreas: []data.InterestArea{{ID: areaID, Label: "Synthetic Interest"}},
		Participants: []data.ProgramMembership{
			{StudentID: "student-here", LegalGivenName: "Here", LegalFamilyName: "Synthetic", GradeLevelID: &gradeLower.ID},
			{StudentID: "student-topic", LegalGivenName: "Topic", LegalFamilyName: "Synthetic", GradeLevelID: &gradeTarget.ID},
			{StudentID: "student-elsewhere", LegalGivenName: "Elsewhere", LegalFamilyName: "Synthetic", GradeLevelID: &gradeTarget.ID},
			{StudentID: "student-blank", LegalGivenName: "Blank", LegalFamilyName: "Synthetic", GradeLevelID: &gradeTarget.ID},
			{StudentID: "student-excluded", LegalGivenName: "Excluded", LegalFamilyName: "Synthetic", GradeLevelID: &gradeLower.ID},
		},
		Students: []data.Student{
			{ID: "student-here", HomeroomID: "room-synthetic"}, {ID: "student-topic", HomeroomID: "room-synthetic"}, {ID: "student-elsewhere", HomeroomID: "room-synthetic"}, {ID: "student-blank", HomeroomID: "room-synthetic"}, {ID: "student-excluded", HomeroomID: "room-synthetic"},
		},
		Assignments: []data.Assignment{
			{ID: "assignment-here", StudentID: "student-here", OfferingID: "offering-target", RealizedQuality: "high"},
			{ID: "assignment-elsewhere", StudentID: "student-elsewhere", OfferingID: "offering-elsewhere", RealizedQuality: "neutral"},
		},
		Answers: []data.SessionResultAnswer{
			{StudentID: "student-here", OfferingID: "offering-target", Answer: data.RankedChoiceRanked, Rank: &two},
			{StudentID: "student-elsewhere", OfferingID: "offering-target", Answer: data.RankedChoiceNotInterested},
			{StudentID: "student-blank", OfferingID: "offering-target", Answer: data.RankedChoiceNoResponse},
		},
		Submissions: map[ids.XID]bool{"student-here": true, "student-elsewhere": true, "student-blank": true},
		Profiles: map[ids.XID][]data.EffectiveInterestProfileValue{
			"student-topic": {{InterestAreaID: areaID, Rating: data.InterestProfileInterested}},
			"student-blank": {{InterestAreaID: areaID, Rating: data.InterestProfileVeryInterested}},
		},
		HighRankMax: 3,
	}

	context := evaluateOfferingPreferenceContext(snapshot)
	require.Equal(t, int64(7), context.DraftRevision)
	require.Equal(t, "Synthetic Target", context.MinGradeLabel)
	require.Equal(t, "Synthetic Target", context.MaxGradeLabel)
	require.Equal(t, []OfferingPreferenceCategory{OfferingPreferenceRank, OfferingPreferenceTopicInterested, OfferingPreferenceRankedNotInterested, OfferingPreferenceUnknown}, groupCategories(context.Groups))

	ranked := context.Groups[0].Students[0]
	require.Equal(t, ids.XID("student-here"), ranked.Participant.StudentID)
	require.Equal(t, "here", ranked.Placement.Status)
	require.True(t, ranked.OutsideGradeRange)
	require.Equal(t, PotentialQualityHigh, ranked.Preference.PotentialQuality)

	elsewhere := context.Groups[2].Students[0]
	require.Equal(t, "elsewhere", elsewhere.Placement.Status)
	require.NotNil(t, elsewhere.Placement.OfferingID)
	require.Equal(t, ids.XID("offering-elsewhere"), *elsewhere.Placement.OfferingID)
	require.Equal(t, "Synthetic Elsewhere Offering", elsewhere.Placement.OfferingName)

	blank := context.Groups[3].Students[0]
	require.Equal(t, OfferingPreferenceUnknown, blank.Preference.Category)
	require.Equal(t, data.InterestProfileVeryInterested, blank.Preference.TopicRatingHint)
	require.Equal(t, "unplaced", blank.Placement.Status)
	for _, group := range context.Groups {
		for _, student := range group.Students {
			require.NotEqual(t, ids.XID("student-excluded"), student.Participant.StudentID)
		}
	}
}

func groupCategories(groups []OfferingPreferenceGroup) []OfferingPreferenceCategory {
	result := make([]OfferingPreferenceCategory, 0, len(groups))
	for _, group := range groups {
		result = append(result, group.Category)
	}
	return result
}
