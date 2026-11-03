package program

import "github.com/chrismott/miniclass/internal/data"

// OfferingPreferenceCategory identifies the source and meaning of a student's
// preference for one offering. The values and their display order implement
// SPEC §§13.3–13.5 and §17.4.1.
type OfferingPreferenceCategory string

const (
	OfferingPreferenceRank                OfferingPreferenceCategory = "rank"
	OfferingPreferenceTopicVeryInterested OfferingPreferenceCategory = "topic_very_interested"
	OfferingPreferenceRankedInterested    OfferingPreferenceCategory = "ranked_interested"
	OfferingPreferenceTopicInterested     OfferingPreferenceCategory = "topic_interested"
	OfferingPreferenceTopicNotInterested  OfferingPreferenceCategory = "topic_not_interested"
	OfferingPreferenceRankedNotInterested OfferingPreferenceCategory = "ranked_not_interested"
	OfferingPreferenceUnknown             OfferingPreferenceCategory = "unknown"
)

// PotentialQuality is the quality a student would realize if manually placed
// in an offering under the current preference context. It intentionally does
// not replace the historical realized quality stored on an assignment.
type PotentialQuality string

const (
	PotentialQualityTop        PotentialQuality = "top"
	PotentialQualityHigh       PotentialQuality = "high"
	PotentialQualityAcceptable PotentialQuality = "acceptable"
	PotentialQualityNeutral    PotentialQuality = "neutral"
	PotentialQualityUnwanted   PotentialQuality = "unwanted"
)

// OfferingPreferenceSortKey gives consumers one canonical display order. Sort
// by Category, then Rank; Rank is set only for the rank category.
type OfferingPreferenceSortKey struct {
	Category int
	Rank     int
}

// Less reports whether this key appears before other in the canonical display
// order.
func (key OfferingPreferenceSortKey) Less(other OfferingPreferenceSortKey) bool {
	if key.Category != other.Category {
		return key.Category < other.Category
	}
	return key.Rank < other.Rank
}

const (
	offeringPreferenceOrderRank = iota
	offeringPreferenceOrderTopicVeryInterested
	offeringPreferenceOrderRankedInterested
	offeringPreferenceOrderTopicInterested
	offeringPreferenceOrderTopicNotInterested
	offeringPreferenceOrderRankedNotInterested
	offeringPreferenceOrderUnknown
)

// OfferingPreference is the current, per-offering preference context. A
// TopicRatingHint is populated only when a ranked-choice submitter left this
// offering blank; it is display context and never changes the category or
// potential quality.
type OfferingPreference struct {
	Category         OfferingPreferenceCategory
	PotentialQuality PotentialQuality
	Rank             int
	SortKey          OfferingPreferenceSortKey
	TopicRatingHint  data.InterestProfileRating
}

// DeriveOfferingPreference maps a student's preference data for one offering
// onto a display category and potential quality. This intentionally mirrors
// solver/tests/test_contract.py::test_preference_models_map_to_realized_quality
// so drift between manual-placement context and solver quality is visible.
func DeriveOfferingPreference(hasRankedChoiceSubmission bool, rankedAnswer *data.SessionResultAnswer, interestRating *data.InterestProfileRating, offeringHasInterestArea bool, highRankMax int) OfferingPreference {
	if hasRankedChoiceSubmission {
		if rankedAnswer == nil || rankedAnswer.Answer == data.RankedChoiceNoResponse {
			result := unknownOfferingPreference()
			if offeringHasInterestArea && isInterestRating(interestRating) {
				result.TopicRatingHint = *interestRating
			}
			return result
		}

		switch rankedAnswer.Answer {
		case data.RankedChoiceRanked:
			if rankedAnswer.Rank == nil || *rankedAnswer.Rank < 1 {
				return unknownOfferingPreference()
			}
			rank := *rankedAnswer.Rank
			quality := PotentialQualityAcceptable
			if rank == 1 {
				quality = PotentialQualityTop
			} else if rank <= highRankMax {
				quality = PotentialQualityHigh
			}
			return OfferingPreference{Category: OfferingPreferenceRank, PotentialQuality: quality, Rank: rank, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRank, Rank: rank}}
		case data.RankedChoiceInterested:
			return OfferingPreference{Category: OfferingPreferenceRankedInterested, PotentialQuality: PotentialQualityAcceptable, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRankedInterested}}
		case data.RankedChoiceNotInterested:
			return OfferingPreference{Category: OfferingPreferenceRankedNotInterested, PotentialQuality: PotentialQualityUnwanted, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderRankedNotInterested}}
		default:
			return unknownOfferingPreference()
		}
	}

	if !offeringHasInterestArea || !isInterestRating(interestRating) {
		return unknownOfferingPreference()
	}
	switch *interestRating {
	case data.InterestProfileVeryInterested:
		return OfferingPreference{Category: OfferingPreferenceTopicVeryInterested, PotentialQuality: PotentialQualityHigh, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicVeryInterested}}
	case data.InterestProfileInterested:
		return OfferingPreference{Category: OfferingPreferenceTopicInterested, PotentialQuality: PotentialQualityAcceptable, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicInterested}}
	case data.InterestProfileNotInterested:
		return OfferingPreference{Category: OfferingPreferenceTopicNotInterested, PotentialQuality: PotentialQualityUnwanted, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderTopicNotInterested}}
	default:
		return unknownOfferingPreference()
	}
}

func unknownOfferingPreference() OfferingPreference {
	return OfferingPreference{Category: OfferingPreferenceUnknown, PotentialQuality: PotentialQualityNeutral, SortKey: OfferingPreferenceSortKey{Category: offeringPreferenceOrderUnknown}}
}

func isInterestRating(rating *data.InterestProfileRating) bool {
	if rating == nil {
		return false
	}
	switch *rating {
	case data.InterestProfileVeryInterested, data.InterestProfileInterested, data.InterestProfileNotInterested:
		return true
	default:
		return false
	}
}
