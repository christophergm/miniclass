package program

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

// These identifiers are stable API vocabulary from SPEC §16.5 and the
// catalog-authoring checks in §14.2. A warning is advice for the organiser;
// it is never a validation error.
const (
	CatalogCapacityShortWarning     = "catalog-capacity-short"
	CatalogGradeGapWarning          = "catalog-grade-gap"
	CatalogMinimumViabilityWarning  = "catalog-minimum-viability"
	CatalogAreaGapWarning           = "catalog-area-gap"
	CatalogUnmatchedOfferingWarning = "catalog-unmatched-offering"
	CatalogWarningSeverity          = "warning"
	CatalogInfoSeverity             = "info"
)

// CatalogParticipant is the small, immutable input needed for catalog grade
// coverage. It deliberately carries the grade ordinal instead of looking up a
// display name while evaluating, so the result remains a pure snapshot.
type CatalogParticipant struct {
	StudentID    ids.XID
	GradeLevelID ids.XID
	GradeLabel   string
	GradeOrdinal int
	KnownGrade   bool
}

// CatalogAreaDemand is an aggregate student signal. Preference submissions
// are not persisted until a later phase, so the live service currently passes
// no signals. Keeping the input explicit lets the evaluator implement §14.2
// now and makes the future preference reader a snapshot adapter rather than a
// rewrite of warning logic.
type CatalogAreaDemand struct {
	InterestAreaID  ids.XID
	HighRatingCount int
}

type CatalogFeasibilitySnapshot struct {
	Participants     []CatalogParticipant
	Grades           []data.GradeLevel
	Offerings        []data.Offering
	InterestAreas    []data.InterestArea
	AreaDemand       []CatalogAreaDemand
	HasRankedChoices bool
}

type CatalogGradeGap struct {
	ID               ids.XID
	Label            string
	ParticipantCount int
}

type CatalogAreaGap struct {
	ID              ids.XID
	Label           string
	HighRatingCount int
}

type CatalogFeasibilityWarning struct {
	ID                           string
	Severity                     string
	Message                      string
	ParticipantCount             int
	TotalCapacity                int
	TotalMinimumViableEnrollment int
	Shortfall                    int
	AffectedGrades               []CatalogGradeGap
	AffectedAreas                []CatalogAreaGap
	OfferingIDs                  []ids.XID
}

type CatalogFeasibility struct {
	ParticipantCount int
	Warnings         []CatalogFeasibilityWarning
}

// EvaluateCatalogFeasibility evaluates the five §14.2 checks in a fixed
// order. Inputs are treated as a snapshot: the function performs no I/O and
// does not mutate any caller-owned slice. That makes repeated evaluations
// byte-for-byte stable for the same snapshot and keeps warning computation
// separate from action validation (§5.2).
func EvaluateCatalogFeasibility(snapshot CatalogFeasibilitySnapshot) CatalogFeasibility {
	participantCount := len(snapshot.Participants)
	result := CatalogFeasibility{ParticipantCount: participantCount, Warnings: make([]CatalogFeasibilityWarning, 0, 5)}

	totalCapacity := 0
	totalMinimum := 0
	for _, offering := range snapshot.Offerings {
		totalCapacity += offering.Capacity
		if offering.MinimumViableEnrollment != nil {
			totalMinimum += *offering.MinimumViableEnrollment
		}
	}
	if totalCapacity < participantCount {
		result.Warnings = append(result.Warnings, CatalogFeasibilityWarning{
			ID:               CatalogCapacityShortWarning,
			Severity:         CatalogWarningSeverity,
			Message:          fmt.Sprintf("Catalog capacity is short by %d students (%d seats for %d participating students).", participantCount-totalCapacity, totalCapacity, participantCount),
			ParticipantCount: participantCount,
			TotalCapacity:    totalCapacity,
			Shortfall:        participantCount - totalCapacity,
			AffectedGrades:   []CatalogGradeGap{},
			AffectedAreas:    []CatalogAreaGap{},
			OfferingIDs:      allOfferingIDs(snapshot.Offerings),
		})
	}

	gradeGaps := uncoveredGrades(snapshot)
	if len(gradeGaps) > 0 {
		result.Warnings = append(result.Warnings, CatalogFeasibilityWarning{
			ID:               CatalogGradeGapWarning,
			Severity:         CatalogWarningSeverity,
			Message:          gradeGapMessage(gradeGaps),
			ParticipantCount: participantCount,
			AffectedGrades:   gradeGaps,
			AffectedAreas:    []CatalogAreaGap{},
			OfferingIDs:      []ids.XID{},
		})
	}

	if totalMinimum > participantCount {
		result.Warnings = append(result.Warnings, CatalogFeasibilityWarning{
			ID:                           CatalogMinimumViabilityWarning,
			Severity:                     CatalogWarningSeverity,
			Message:                      fmt.Sprintf("Minimum viable enrollment exceeds participating students by %d (%d minimum places for %d participating students).", totalMinimum-participantCount, totalMinimum, participantCount),
			ParticipantCount:             participantCount,
			TotalMinimumViableEnrollment: totalMinimum,
			Shortfall:                    totalMinimum - participantCount,
			AffectedGrades:               []CatalogGradeGap{},
			AffectedAreas:                []CatalogAreaGap{},
			OfferingIDs:                  allOfferingIDs(snapshot.Offerings),
		})
	}

	areaGaps := uncoveredAreas(snapshot)
	if len(areaGaps) > 0 {
		result.Warnings = append(result.Warnings, CatalogFeasibilityWarning{
			ID:               CatalogAreaGapWarning,
			Severity:         CatalogInfoSeverity,
			Message:          areaGapMessage(areaGaps),
			ParticipantCount: participantCount,
			AffectedGrades:   []CatalogGradeGap{},
			AffectedAreas:    areaGaps,
			OfferingIDs:      []ids.XID{},
		})
	}

	if !snapshot.HasRankedChoices {
		unmatched := unmatchedOfferingIDs(snapshot.Offerings)
		if len(unmatched) > 0 {
			result.Warnings = append(result.Warnings, CatalogFeasibilityWarning{
				ID:               CatalogUnmatchedOfferingWarning,
				Severity:         CatalogWarningSeverity,
				Message:          fmt.Sprintf("%d offering%s have no interest area and no ranked choices; manual placement is required.", len(unmatched), pluralSuffix(len(unmatched))),
				ParticipantCount: participantCount,
				AffectedGrades:   []CatalogGradeGap{},
				AffectedAreas:    []CatalogAreaGap{},
				OfferingIDs:      unmatched,
			})
		}
	}

	return result
}

// GetCatalogFeasibility computes a read-only snapshot for one session. All
// inputs are read inside one tenant transaction so warnings cannot mix rows
// from different organisations or observe a partially changed catalog.
func (s *Service) GetCatalogFeasibility(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (CatalogFeasibility, error) {
	if s == nil || s.database == nil {
		return CatalogFeasibility{}, fmt.Errorf("get catalog feasibility: data service is nil")
	}
	var result CatalogFeasibility
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSession(ctx, schoolYearID, programID, sessionID); err != nil {
			return err
		}
		memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		nonParticipations, err := tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		offerings, err := tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		grades, err := tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		areas, err := tx.ListInterestAreas(ctx, schoolYearID, programID, false)
		if err != nil {
			return err
		}

		excluded := make(map[ids.XID]struct{}, len(nonParticipations))
		for _, row := range nonParticipations {
			excluded[row.StudentID] = struct{}{}
		}
		gradeByID := make(map[ids.XID]data.GradeLevel, len(grades))
		for _, grade := range grades {
			gradeByID[grade.ID] = grade
		}
		participants := make([]CatalogParticipant, 0, len(memberships))
		for _, membership := range memberships {
			if _, ok := excluded[membership.StudentID]; ok {
				continue
			}
			if membership.GradeLevelID == nil {
				participants = append(participants, CatalogParticipant{StudentID: membership.StudentID})
				continue
			}
			grade, ok := gradeByID[*membership.GradeLevelID]
			if !ok {
				participants = append(participants, CatalogParticipant{StudentID: membership.StudentID})
				continue
			}
			participants = append(participants, CatalogParticipant{StudentID: membership.StudentID, GradeLevelID: grade.ID, GradeLabel: grade.Label, GradeOrdinal: grade.Ordinal, KnownGrade: true})
		}

		// Ranked choices and preference signals are not persisted until their
		// later phase. The explicit false/empty values are intentional: an
		// untagged offering is currently reachable only through manual placement,
		// while area coverage waits for actual student signals.
		result = EvaluateCatalogFeasibility(CatalogFeasibilitySnapshot{
			Participants:     participants,
			Grades:           grades,
			Offerings:        offerings,
			InterestAreas:    areas,
			AreaDemand:       nil,
			HasRankedChoices: false,
		})
		return nil
	})
	if err != nil {
		return CatalogFeasibility{}, fmt.Errorf("get catalog feasibility: %w", err)
	}
	return result, nil
}

func uncoveredGrades(snapshot CatalogFeasibilitySnapshot) []CatalogGradeGap {
	counts := make(map[ids.XID]int)
	grades := make(map[ids.XID]CatalogGradeGap)
	for _, participant := range snapshot.Participants {
		if !participant.KnownGrade {
			continue
		}
		counts[participant.GradeLevelID]++
		grades[participant.GradeLevelID] = CatalogGradeGap{ID: participant.GradeLevelID, Label: participant.GradeLabel}
	}
	for id, gap := range grades {
		gap.ParticipantCount = counts[id]
		grades[id] = gap
		admitted := false
		for _, offering := range snapshot.Offerings {
			minOrdinal, maxOrdinal, ok := offeringGradeOrdinals(snapshot, offering)
			if !ok {
				continue
			}
			for _, participant := range snapshot.Participants {
				if participant.GradeLevelID == id && participant.GradeOrdinal >= minOrdinal && participant.GradeOrdinal <= maxOrdinal {
					admitted = true
					break
				}
			}
			if admitted {
				break
			}
		}
		if admitted {
			delete(grades, id)
		}
	}
	result := make([]CatalogGradeGap, 0, len(grades))
	for _, gap := range grades {
		result = append(result, gap)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func offeringGradeOrdinals(snapshot CatalogFeasibilitySnapshot, offering data.Offering) (int, int, bool) {
	var minOrdinal, maxOrdinal int
	var minFound, maxFound bool
	for _, grade := range snapshot.Grades {
		if grade.ID == offering.MinGradeLevelID {
			minOrdinal, minFound = grade.Ordinal, true
		}
		if grade.ID == offering.MaxGradeLevelID {
			maxOrdinal, maxFound = grade.Ordinal, true
		}
	}
	// Pure callers may provide only participating grades. The live service
	// supplies the complete school-year vocabulary, but this fallback keeps the
	// evaluator useful for smaller snapshots and remains deterministic.
	if !minFound || !maxFound {
		for _, participant := range snapshot.Participants {
			if !participant.KnownGrade {
				continue
			}
			if participant.GradeLevelID == offering.MinGradeLevelID {
				minOrdinal, minFound = participant.GradeOrdinal, true
			}
			if participant.GradeLevelID == offering.MaxGradeLevelID {
				maxOrdinal, maxFound = participant.GradeOrdinal, true
			}
		}
	}
	return minOrdinal, maxOrdinal, minFound && maxFound
}

func uncoveredAreas(snapshot CatalogFeasibilitySnapshot) []CatalogAreaGap {
	areaLabels := make(map[ids.XID]string, len(snapshot.InterestAreas))
	offeringAreas := make(map[ids.XID]struct{}, len(snapshot.Offerings))
	for _, area := range snapshot.InterestAreas {
		areaLabels[area.ID] = area.Label
	}
	for _, offering := range snapshot.Offerings {
		if offering.InterestAreaID != nil {
			offeringAreas[*offering.InterestAreaID] = struct{}{}
		}
	}
	counts := make(map[ids.XID]int)
	for _, demand := range snapshot.AreaDemand {
		if demand.HighRatingCount > 0 {
			if _, current := areaLabels[demand.InterestAreaID]; current {
				counts[demand.InterestAreaID] += demand.HighRatingCount
			}
		}
	}
	result := make([]CatalogAreaGap, 0)
	for id, count := range counts {
		if _, covered := offeringAreas[id]; covered {
			continue
		}
		result = append(result, CatalogAreaGap{ID: id, Label: areaLabels[id], HighRatingCount: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func unmatchedOfferingIDs(offerings []data.Offering) []ids.XID {
	result := make([]ids.XID, 0)
	for _, offering := range offerings {
		if offering.InterestAreaID == nil {
			result = append(result, offering.ID)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func allOfferingIDs(offerings []data.Offering) []ids.XID {
	result := make([]ids.XID, 0, len(offerings))
	for _, offering := range offerings {
		result = append(result, offering.ID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func gradeGapMessage(gaps []CatalogGradeGap) string {
	labels := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		labels = append(labels, fmt.Sprintf("%s (%d participating)", gap.Label, gap.ParticipantCount))
	}
	return fmt.Sprintf("No offering admits participating grade%s: %s.", pluralSuffix(len(gaps)), joinLabels(labels))
}

func areaGapMessage(gaps []CatalogAreaGap) string {
	labels := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		labels = append(labels, fmt.Sprintf("%s (%d highly-rated)", gap.Label, gap.HighRatingCount))
	}
	return fmt.Sprintf("No offering covers highly-rated area%s: %s.", pluralSuffix(len(gaps)), joinLabels(labels))
}

func joinLabels(labels []string) string {
	return strings.Join(labels, ", ")
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
