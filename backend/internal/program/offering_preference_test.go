package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestDeriveOfferingPreference(t *testing.T) {
	t.Parallel()

	rank := func(value int) *int { return &value }
	rating := func(value data.InterestProfileRating) *data.InterestProfileRating { return &value }
	answer := func(value data.RankedChoiceAnswer, rank *int) *data.SessionResultAnswer {
		return &data.SessionResultAnswer{Answer: value, Rank: rank}
	}

	for _, test := range []struct {
		name        string
		submitted   bool
		answer      *data.SessionResultAnswer
		rating      *data.InterestProfileRating
		hasArea     bool
		highRankMax int
		want        OfferingPreference
	}{
		{
			name:        "first ranked choice is top",
			submitted:   true,
			answer:      answer(data.RankedChoiceRanked, rank(1)),
			highRankMax: 3,
			want:        OfferingPreference{Category: OfferingPreferenceRank, PotentialQuality: PotentialQualityTop, Rank: 1, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRank, Rank: 1}},
		},
		{
			name:        "high rank boundary is high",
			submitted:   true,
			answer:      answer(data.RankedChoiceRanked, rank(3)),
			highRankMax: 3,
			want:        OfferingPreference{Category: OfferingPreferenceRank, PotentialQuality: PotentialQualityHigh, Rank: 3, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRank, Rank: 3}},
		},
		{
			name:        "rank below high boundary is acceptable",
			submitted:   true,
			answer:      answer(data.RankedChoiceRanked, rank(4)),
			highRankMax: 3,
			want:        OfferingPreference{Category: OfferingPreferenceRank, PotentialQuality: PotentialQualityAcceptable, Rank: 4, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRank, Rank: 4}},
		},
		{
			name:      "ranked interested is acceptable",
			submitted: true,
			answer:    answer(data.RankedChoiceInterested, nil),
			want:      OfferingPreference{Category: OfferingPreferenceRankedInterested, PotentialQuality: PotentialQualityAcceptable, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRankedInterested}},
		},
		{
			name:      "ranked not interested is unwanted",
			submitted: true,
			answer:    answer(data.RankedChoiceNotInterested, nil),
			want:      OfferingPreference{Category: OfferingPreferenceRankedNotInterested, PotentialQuality: PotentialQualityUnwanted, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRankedNotInterested}},
		},
		{
			name:      "all no response submission remains unknown",
			submitted: true,
			answer:    answer(data.RankedChoiceNoResponse, nil),
			rating:    rating(data.InterestProfileVeryInterested),
			hasArea:   true,
			want:      OfferingPreference{Category: OfferingPreferenceUnknown, PotentialQuality: PotentialQualityNeutral, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderUnknown}, TopicRatingHint: data.InterestProfileVeryInterested},
		},
		{
			name:    "very interested topic is high without ranked submission",
			rating:  rating(data.InterestProfileVeryInterested),
			hasArea: true,
			want:    OfferingPreference{Category: OfferingPreferenceTopicVeryInterested, PotentialQuality: PotentialQualityHigh, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicVeryInterested}},
		},
		{
			name:    "interested topic is acceptable without ranked submission",
			rating:  rating(data.InterestProfileInterested),
			hasArea: true,
			want:    OfferingPreference{Category: OfferingPreferenceTopicInterested, PotentialQuality: PotentialQualityAcceptable, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicInterested}},
		},
		{
			name:    "not interested topic is unwanted without ranked submission",
			rating:  rating(data.InterestProfileNotInterested),
			hasArea: true,
			want:    OfferingPreference{Category: OfferingPreferenceTopicNotInterested, PotentialQuality: PotentialQualityUnwanted, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicNotInterested}},
		},
		{
			name:    "no profile is unknown",
			hasArea: true,
			want:    unknownOfferingPreference(),
		},
		{
			name:    "unrated topic is unknown",
			rating:  rating(data.InterestProfileUnrated),
			hasArea: true,
			want:    unknownOfferingPreference(),
		},
		{
			name:   "offering without area is unknown",
			rating: rating(data.InterestProfileVeryInterested),
			want:   unknownOfferingPreference(),
		},
		{
			name:      "blank ranked offering exposes only topic hint",
			submitted: true,
			rating:    rating(data.InterestProfileInterested),
			hasArea:   true,
			want:      OfferingPreference{Category: OfferingPreferenceUnknown, PotentialQuality: PotentialQualityNeutral, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderUnknown}, TopicRatingHint: data.InterestProfileInterested},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := DeriveOfferingPreference(test.submitted, test.answer, test.rating, test.hasArea, test.highRankMax)
			require.Equal(t, test.want, got)
		})
	}
}

func TestOfferingPreferenceSortKeyOrdersCategoriesAndRanks(t *testing.T) {
	t.Parallel()

	keys := []OfferingPreferenceSortKey{
		{Category: offeringPreferenceOrderRank, Rank: 1},
		{Category: offeringPreferenceOrderRank, Rank: 2},
		{Category: offeringPreferenceOrderTopicVeryInterested},
		{Category: offeringPreferenceOrderRankedInterested},
		{Category: offeringPreferenceOrderTopicInterested},
		{Category: offeringPreferenceOrderTopicNotInterested},
		{Category: offeringPreferenceOrderRankedNotInterested},
		{Category: offeringPreferenceOrderUnknown},
	}
	for index := range keys[:len(keys)-1] {
		require.True(t, keys[index].Less(keys[index+1]))
	}
}
