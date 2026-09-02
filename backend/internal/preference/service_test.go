package preference

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestEffectiveProfileOverlaysOnlySubmittedAreas(t *testing.T) {
	when := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	submissions := []data.InterestProfileSubmission{
		{ID: "submission-old", SubmittedAt: when},
		{ID: "submission-new", SubmittedAt: when.Add(time.Hour)},
	}
	responses := []data.InterestProfileResponse{
		{SubmissionID: "submission-old", InterestAreaID: "area-art", Rating: data.InterestProfileVeryInterested},
		{SubmissionID: "submission-old", InterestAreaID: "area-music", Rating: data.InterestProfileUnrated},
		{SubmissionID: "submission-new", InterestAreaID: "area-art", Rating: data.InterestProfileNotInterested},
	}

	values := EffectiveProfile(submissions, responses)
	require.Equal(t, []ProfileValue{
		{InterestAreaID: "area-art", State: ProfileRated, Rating: data.InterestProfileNotInterested, SubmissionID: "submission-new"},
		{InterestAreaID: "area-music", State: ProfileUnrated, Rating: data.InterestProfileUnrated, SubmissionID: "submission-old"},
	}, values)
	require.Equal(t, ProfileValue{InterestAreaID: "area-new", State: ProfileNoResponse}, ProfileValueForArea("area-new", submissions, responses))
}

func TestEffectiveProfileUsesSubmissionIDAsDeterministicTieBreaker(t *testing.T) {
	when := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	submissions := []data.InterestProfileSubmission{
		{ID: "submission-b", SubmittedAt: when},
		{ID: "submission-a", SubmittedAt: when},
	}
	responses := []data.InterestProfileResponse{
		{SubmissionID: "submission-b", InterestAreaID: "area", Rating: data.InterestProfileNotInterested},
		{SubmissionID: "submission-a", InterestAreaID: "area", Rating: data.InterestProfileInterested},
	}

	value := ProfileValueForArea("area", submissions, responses)
	require.Equal(t, ProfileRated, value.State)
	require.Equal(t, data.InterestProfileNotInterested, value.Rating)
	require.Equal(t, ids.XID("submission-b"), value.SubmissionID)
}

func TestValidateRankedChoiceResponseSetRequiresCompleteUniqueResponses(t *testing.T) {
	offerings := []data.Offering{{ID: "offering-a"}, {ID: "offering-b"}}
	rankOne, rankTwo := 1, 2
	valid := []data.RankedChoiceResponseInput{
		{OfferingID: "offering-a", Answer: data.RankedChoiceRanked, Rank: &rankOne},
		{OfferingID: "offering-b", Answer: data.RankedChoiceInterested},
	}
	require.NoError(t, ValidateRankedChoiceResponseSet(valid, offerings))

	duplicateRank := []data.RankedChoiceResponseInput{
		{OfferingID: "offering-a", Answer: data.RankedChoiceRanked, Rank: &rankOne},
		{OfferingID: "offering-b", Answer: data.RankedChoiceRanked, Rank: &rankOne},
	}
	require.ErrorIs(t, ValidateRankedChoiceResponseSet(duplicateRank, offerings), ErrRankedChoiceInvalid)

	missing := []data.RankedChoiceResponseInput{{OfferingID: "offering-a", Answer: data.RankedChoiceRanked, Rank: &rankTwo}}
	require.ErrorIs(t, ValidateRankedChoiceResponseSet(missing, offerings), ErrRankedChoiceNotComplete)

	foreign := []data.RankedChoiceResponseInput{
		{OfferingID: "offering-a", Answer: data.RankedChoiceNoResponse},
		{OfferingID: "offering-foreign", Answer: data.RankedChoiceInterested},
	}
	require.ErrorIs(t, ValidateRankedChoiceResponseSet(foreign, offerings), ErrRankedChoiceInvalid)
}

func TestValidateRankedChoiceResponseSetHonorsSessionRankDepth(t *testing.T) {
	offerings := []data.Offering{{ID: "offering-a"}, {ID: "offering-b"}}
	rankTwo := 2
	responses := []data.RankedChoiceResponseInput{
		{OfferingID: "offering-a", Answer: data.RankedChoiceRanked, Rank: &rankTwo},
		{OfferingID: "offering-b", Answer: data.RankedChoiceInterested},
	}

	require.ErrorIs(t, ValidateRankedChoiceResponseSetWithDepth(responses, offerings, 1), ErrRankedChoiceInvalid)
	require.NoError(t, ValidateRankedChoiceResponseSetWithDepth(responses, offerings, 2))
}
