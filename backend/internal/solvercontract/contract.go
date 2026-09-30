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

type Request struct {
	Version              string        `json:"version"`
	Seed                 int64         `json:"seed"`
	MaxDeterministicTime float64       `json:"max_deterministic_time"`
	Participants         []Participant `json:"participants"`
	Offerings            []Offering    `json:"offerings"`
}

type Participant struct {
	ID           string `json:"id"`
	GradeOrdinal int    `json:"grade_ordinal"`
}

type Offering struct {
	ID              string `json:"id"`
	Capacity        int    `json:"capacity"`
	MinGradeOrdinal int    `json:"min_grade_ordinal"`
	MaxGradeOrdinal int    `json:"max_grade_ordinal"`
}

type Response struct {
	Version             string               `json:"version"`
	Seed                int64                `json:"seed"`
	Status              string               `json:"status"`
	Assignments         []Assignment         `json:"assignments"`
	ConflictDiagnostics []ConflictDiagnostic `json:"conflict_diagnostics"`
}

type Assignment struct {
	ParticipantID string `json:"participant_id"`
	OfferingID    string `json:"offering_id"`
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
	}
	sort.Slice(r.Offerings, func(i, j int) bool { return r.Offerings[i].ID < r.Offerings[j].ID })
	sort.Slice(r.Participants, func(i, j int) bool { return r.Participants[i].ID < r.Participants[j].ID })
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
		if assignment.ParticipantID == "" || assignment.OfferingID == "" {
			return errors.New("solver assignments require participant and offering ids")
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
