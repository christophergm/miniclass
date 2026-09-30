// Package solvercontract owns the versioned, self-contained sidecar wire format.
package solvercontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

const Version = "v1"

const (
	QualityTop        = "top"
	QualityHigh       = "high"
	QualityAcceptable = "acceptable"
	QualityNeutral    = "neutral"
	QualityUnwanted   = "unwanted"

	RankedResponse        = "ranked"
	InterestedResponse    = "interested"
	NotInterestedResponse = "not_interested"
	VeryInterestedRating  = "very_interested"
)

type Request struct {
	Version              string        `json:"version"`
	Seed                 int64         `json:"seed"`
	MaxDeterministicTime float64       `json:"max_deterministic_time"`
	QualityConfig        QualityConfig `json:"quality_config"`
	Participants         []Participant `json:"participants"`
	Offerings            []Offering    `json:"offerings"`
}

// QualityConfig holds the program's v0 ranked-choice boundary from SPEC
// §17.4.1. The remaining quality mappings are the specified v0 defaults.
type QualityConfig struct {
	HighRankMax int `json:"high_rank_max"`
}

type Participant struct {
	ID              string           `json:"id"`
	GradeOrdinal    int              `json:"grade_ordinal"`
	RankedChoices   *RankedChoices   `json:"ranked_choices,omitempty"`
	InterestProfile []InterestRating `json:"interest_profile"`
}

// RankedChoices is present only when a student submitted session choices. Its
// presence takes precedence over the standing interest profile (SPEC §13.4).
type RankedChoices struct {
	Choices []RankedChoice `json:"choices"`
}

type RankedChoice struct {
	OfferingID string `json:"offering_id"`
	Response   string `json:"response"`
	Rank       int    `json:"rank,omitempty"`
}

type InterestRating struct {
	InterestAreaID string `json:"interest_area_id"`
	Rating         string `json:"rating"`
}

type Offering struct {
	ID              string  `json:"id"`
	Capacity        int     `json:"capacity"`
	MinGradeOrdinal int     `json:"min_grade_ordinal"`
	MaxGradeOrdinal int     `json:"max_grade_ordinal"`
	InterestAreaID  *string `json:"interest_area_id,omitempty"`
}

type Response struct {
	Version             string               `json:"version"`
	Seed                int64                `json:"seed"`
	Status              string               `json:"status"`
	Assignments         []Assignment         `json:"assignments"`
	ConflictDiagnostics []ConflictDiagnostic `json:"conflict_diagnostics"`
}

type Assignment struct {
	ParticipantID   string `json:"participant_id"`
	OfferingID      string `json:"offering_id"`
	RealizedQuality string `json:"realized_quality"`
}

// ConflictDiagnostic reserves the v1 boundary for a future minimal or
// near-minimal infeasibility explanation (SPEC §17.10). The v0 model returns
// an empty list rather than attempting general conflict-set extraction.
type ConflictDiagnostic struct {
	Code           string   `json:"code"`
	ParticipantIDs []string `json:"participant_ids"`
	OfferingIDs    []string `json:"offering_ids"`
}

// CanonicalJSON validates and stably orders every contract collection before encoding.
// Its bytes are the persistence and fingerprint representation required by SPEC §17.8.
func CanonicalJSON(request Request) ([]byte, error) {
	if err := request.Canonicalize(); err != nil {
		return nil, err
	}
	return json.Marshal(request)
}

func Fingerprint(request Request) (string, error) {
	canonical, err := CanonicalJSON(request)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func (r *Request) Canonicalize() error {
	if r == nil {
		return errors.New("solver request is nil")
	}
	if r.Version != Version {
		return fmt.Errorf("solver request version must be %q", Version)
	}
	if r.MaxDeterministicTime <= 0 {
		return errors.New("solver request max_deterministic_time must be positive")
	}
	if r.QualityConfig.HighRankMax < 2 {
		return errors.New("solver request quality_config.high_rank_max must be at least two")
	}
	if len(r.Offerings) == 0 {
		return errors.New("solver request requires at least one offering")
	}
	offeringIDs := make(map[string]struct{}, len(r.Offerings))
	for index := range r.Offerings {
		offering := &r.Offerings[index]
		if offering.ID == "" || offering.Capacity < 0 || offering.MinGradeOrdinal <= 0 || offering.MaxGradeOrdinal < offering.MinGradeOrdinal {
			return errors.New("solver offerings require an id, non-negative capacity, and ordered positive grade window")
		}
		if _, exists := offeringIDs[offering.ID]; exists {
			return errors.New("solver offering ids must be unique")
		}
		if offering.InterestAreaID != nil && *offering.InterestAreaID == "" {
			return errors.New("solver offering interest_area_id must be non-empty when present")
		}
		offeringIDs[offering.ID] = struct{}{}
	}
	participantIDs := make(map[string]struct{}, len(r.Participants))
	for index := range r.Participants {
		participant := &r.Participants[index]
		if participant.ID == "" {
			return errors.New("solver participants require an id")
		}
		if _, exists := participantIDs[participant.ID]; exists {
			return errors.New("solver participant ids must be unique")
		}
		participantIDs[participant.ID] = struct{}{}
		if participant.GradeOrdinal <= 0 {
			return fmt.Errorf("participant %q requires a positive grade ordinal", participant.ID)
		}
		if err := participant.canonicalizePreferences(offeringIDs); err != nil {
			return fmt.Errorf("participant %q: %w", participant.ID, err)
		}
	}
	sort.Slice(r.Offerings, func(i, j int) bool { return r.Offerings[i].ID < r.Offerings[j].ID })
	sort.Slice(r.Participants, func(i, j int) bool { return r.Participants[i].ID < r.Participants[j].ID })
	return nil
}

func (p *Participant) canonicalizePreferences(offeringIDs map[string]struct{}) error {
	if p.InterestProfile == nil {
		p.InterestProfile = []InterestRating{}
	}
	interestAreaIDs := make(map[string]struct{}, len(p.InterestProfile))
	for index := range p.InterestProfile {
		rating := &p.InterestProfile[index]
		if rating.InterestAreaID == "" {
			return errors.New("interest profile ratings require an interest_area_id")
		}
		if _, exists := interestAreaIDs[rating.InterestAreaID]; exists {
			return errors.New("interest profile ratings must be unique per interest area")
		}
		interestAreaIDs[rating.InterestAreaID] = struct{}{}
		if rating.Rating != VeryInterestedRating && rating.Rating != InterestedResponse && rating.Rating != NotInterestedResponse {
			return errors.New("interest profile ratings must be very_interested, interested, or not_interested")
		}
	}
	sort.Slice(p.InterestProfile, func(i, j int) bool { return p.InterestProfile[i].InterestAreaID < p.InterestProfile[j].InterestAreaID })
	if p.RankedChoices == nil {
		return nil
	}
	if p.RankedChoices.Choices == nil {
		p.RankedChoices.Choices = []RankedChoice{}
	}
	offeringChoiceIDs := make(map[string]struct{}, len(p.RankedChoices.Choices))
	ranks := make(map[int]struct{}, len(p.RankedChoices.Choices))
	for index := range p.RankedChoices.Choices {
		choice := &p.RankedChoices.Choices[index]
		if choice.OfferingID == "" {
			return errors.New("ranked choices require an offering_id")
		}
		if _, exists := offeringIDs[choice.OfferingID]; !exists {
			return errors.New("ranked choices must reference request offerings")
		}
		if _, exists := offeringChoiceIDs[choice.OfferingID]; exists {
			return errors.New("ranked choices must contain at most one response per offering")
		}
		offeringChoiceIDs[choice.OfferingID] = struct{}{}
		switch choice.Response {
		case RankedResponse:
			if choice.Rank <= 0 {
				return errors.New("ranked choices require a positive rank for ranked responses")
			}
			if _, exists := ranks[choice.Rank]; exists {
				return errors.New("ranked choice ranks must be unique per participant")
			}
			ranks[choice.Rank] = struct{}{}
		case InterestedResponse, NotInterestedResponse:
			if choice.Rank != 0 {
				return errors.New("only ranked responses may include rank")
			}
		default:
			return errors.New("ranked choice response must be ranked, interested, or not_interested")
		}
	}
	sort.Slice(p.RankedChoices.Choices, func(i, j int) bool {
		return p.RankedChoices.Choices[i].OfferingID < p.RankedChoices.Choices[j].OfferingID
	})
	return nil
}

// CanonicalResponseJSON validates and stably orders a sidecar response before
// persistence or comparison.
func CanonicalResponseJSON(response Response) ([]byte, error) {
	if err := response.Canonicalize(); err != nil {
		return nil, err
	}
	return json.Marshal(response)
}

func (r *Response) Canonicalize() error {
	if r == nil || r.Version != Version || r.Status == "" {
		return errors.New("solver response has an invalid version or status")
	}
	seen := make(map[string]struct{}, len(r.Assignments))
	for _, assignment := range r.Assignments {
		if assignment.ParticipantID == "" || assignment.OfferingID == "" || !validQuality(assignment.RealizedQuality) {
			return errors.New("solver assignments require participant, offering, and realized quality")
		}
		if _, exists := seen[assignment.ParticipantID]; exists {
			return errors.New("solver response assigns a participant more than once")
		}
		seen[assignment.ParticipantID] = struct{}{}
	}
	for index := range r.ConflictDiagnostics {
		diagnostic := &r.ConflictDiagnostics[index]
		if diagnostic.Code == "" {
			return errors.New("solver conflict diagnostics require a code")
		}
		sort.Strings(diagnostic.ParticipantIDs)
		sort.Strings(diagnostic.OfferingIDs)
	}
	sort.Slice(r.Assignments, func(i, j int) bool { return r.Assignments[i].ParticipantID < r.Assignments[j].ParticipantID })
	sort.Slice(r.ConflictDiagnostics, func(i, j int) bool { return r.ConflictDiagnostics[i].Code < r.ConflictDiagnostics[j].Code })
	if r.ConflictDiagnostics == nil {
		r.ConflictDiagnostics = []ConflictDiagnostic{}
	}
	return nil
}

func validQuality(quality string) bool {
	switch quality {
	case QualityTop, QualityHigh, QualityAcceptable, QualityNeutral, QualityUnwanted:
		return true
	default:
		return false
	}
}
