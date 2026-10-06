package program

import (
	"context"
	"fmt"
	"sort"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

// DraftWarning is a non-blocking, stable warning from SPEC §16.5.
type DraftWarning struct {
	ID, Severity, HostType string
	HostID                 ids.XID
	AssignmentID           *ids.XID
	StudentID              *ids.XID
	OfferingID             *ids.XID
}

type DraftPlacement struct {
	Assignment        data.Assignment
	StudentName       string
	CurrentPreference data.RankedChoiceAnswer
	Warnings          []DraftWarning
}

type OfferingOccupancy struct {
	OfferingID              ids.XID
	Enrolled, Capacity      int
	MinimumViableEnrollment *int
	Warnings                []DraftWarning
}

// AssignmentQuality is the administrator review projection required by SPEC
// §§16.5–16.6 and 19.1–19.2. Named lists precede aggregate metrics.
type AssignmentQuality struct {
	Unplaced, Unwanted, NoSignal, Overridden []DraftPlacement
	Placements                               []DraftPlacement
	Offerings                                []OfferingOccupancy
	QualityDistribution                      map[string]int
	Warnings                                 []DraftWarning
}

type AssignmentQualitySnapshot struct {
	Session       data.Session
	Participants  []data.ProgramMembership
	Grades        []data.GradeLevel
	Offerings     []data.Offering
	Assignments   []data.Assignment
	Exclusions    []data.AssignmentExclusion
	Overrides     []data.AssignmentOverride
	Answers       []data.SessionResultAnswer
	Profiles      map[ids.XID]map[ids.XID]data.InterestProfileRating
	InterestAreas []data.InterestArea
}

func EvaluateAssignmentQuality(s AssignmentQualitySnapshot) AssignmentQuality {
	r := AssignmentQuality{Unplaced: []DraftPlacement{}, Unwanted: []DraftPlacement{}, NoSignal: []DraftPlacement{}, Overridden: []DraftPlacement{}, Placements: []DraftPlacement{}, Offerings: []OfferingOccupancy{}, QualityDistribution: map[string]int{}, Warnings: []DraftWarning{}}
	participants := map[ids.XID]data.ProgramMembership{}
	for _, p := range s.Participants {
		participants[p.StudentID] = p
	}
	grades := map[ids.XID]int{}
	for _, g := range s.Grades {
		grades[g.ID] = g.Ordinal
	}
	offerings := map[ids.XID]data.Offering{}
	occupancy := map[ids.XID]int{}
	for _, o := range s.Offerings {
		offerings[o.ID] = o
	}
	assigned := map[ids.XID]data.Assignment{}
	for _, a := range s.Assignments {
		assigned[a.StudentID] = a
		occupancy[a.OfferingID]++
		r.QualityDistribution[a.RealizedQuality]++
	}
	answer := map[ids.XID]map[ids.XID]data.RankedChoiceAnswer{}
	for _, a := range s.Answers {
		if answer[a.StudentID] == nil {
			answer[a.StudentID] = map[ids.XID]data.RankedChoiceAnswer{}
		}
		answer[a.StudentID][a.OfferingID] = a.Answer
	}
	excluded := map[ids.XID]map[ids.XID]bool{}
	for _, e := range s.Exclusions {
		if excluded[e.StudentID] == nil {
			excluded[e.StudentID] = map[ids.XID]bool{}
		}
		excluded[e.StudentID][e.OfferingID] = true
	}
	overrides := map[ids.XID]bool{}
	for _, o := range s.Overrides {
		overrides[o.AssignmentID] = true
	}

	for studentID, p := range participants {
		a, ok := assigned[studentID]
		if !ok {
			r.Unplaced = append(r.Unplaced, DraftPlacement{StudentName: displayName(p), Assignment: data.Assignment{StudentID: studentID}})
			continue
		}
		placement := DraftPlacement{Assignment: a, StudentName: displayName(p), CurrentPreference: answer[studentID][a.OfferingID], Warnings: []DraftWarning{}}
		o, exists := offerings[a.OfferingID]
		if exists {
			if p.GradeLevelID != nil && (grades[*p.GradeLevelID] < grades[o.MinGradeLevelID] || grades[*p.GradeLevelID] > grades[o.MaxGradeLevelID]) {
				placement.Warnings = append(placement.Warnings, assignmentWarning("grade-out-of-range", a, "warning"))
			}
		}
		if excluded[studentID][a.OfferingID] {
			placement.Warnings = append(placement.Warnings, assignmentWarning("exclusion-overridden", a, "warning"))
		}
		if placement.CurrentPreference == data.RankedChoiceNotInterested {
			placement.Warnings = append(placement.Warnings, assignmentWarning("non-preferred-placement", a, "warning"))
			r.Unwanted = append(r.Unwanted, placement)
		}
		profile := s.Profiles[studentID]
		hasApplicableProfile := exists && o.InterestAreaID != nil && profile[*o.InterestAreaID] != "" && profile[*o.InterestAreaID] != data.InterestProfileUnrated
		if len(answer[studentID]) == 0 && !hasApplicableProfile {
			placement.Warnings = append(placement.Warnings, assignmentWarning("no-preference-signal", a, "info"))
			r.NoSignal = append(r.NoSignal, placement)
		}
		if overrides[a.ID] {
			r.Overridden = append(r.Overridden, placement)
		}
		r.Placements = append(r.Placements, placement)
		r.Warnings = append(r.Warnings, placement.Warnings...)
	}
	for _, o := range s.Offerings {
		item := OfferingOccupancy{OfferingID: o.ID, Enrolled: occupancy[o.ID], Capacity: o.Capacity, MinimumViableEnrollment: o.MinimumViableEnrollment, Warnings: []DraftWarning{}}
		if item.Enrolled > o.Capacity {
			item.Warnings = append(item.Warnings, offeringWarning("capacity-exceeded", o.ID, "warning"))
			for _, p := range r.Placements {
				if p.Assignment.OfferingID == o.ID {
					w := assignmentWarning("capacity-exceeded", p.Assignment, "warning")
					r.Warnings = append(r.Warnings, w)
				}
			}
		}
		if o.MinimumViableEnrollment != nil && item.Enrolled < *o.MinimumViableEnrollment {
			item.Warnings = append(item.Warnings, offeringWarning("below-minimum-enrollment", o.ID, "warning"))
		}
		r.Warnings = append(r.Warnings, item.Warnings...)
		r.Offerings = append(r.Offerings, item)
	}
	totalCapacity := 0
	for _, offering := range s.Offerings {
		totalCapacity += offering.Capacity
	}
	if totalCapacity < len(s.Participants) {
		r.Warnings = append(r.Warnings, DraftWarning{ID: "catalog-capacity-short", Severity: "warning", HostType: "session", HostID: s.Session.ID})
	}
	for _, participant := range s.Participants {
		if participant.GradeLevelID == nil {
			continue
		}
		eligible := false
		for _, offering := range s.Offerings {
			if grades[*participant.GradeLevelID] >= grades[offering.MinGradeLevelID] && grades[*participant.GradeLevelID] <= grades[offering.MaxGradeLevelID] {
				eligible = true
				break
			}
		}
		if !eligible {
			r.Warnings = append(r.Warnings, DraftWarning{ID: "catalog-grade-gap", Severity: "warning", HostType: "session", HostID: s.Session.ID})
			break
		}
	}
	for _, area := range s.InterestAreas {
		demanded := false
		for _, profile := range s.Profiles {
			if profile[area.ID] == data.InterestProfileVeryInterested {
				demanded = true
				break
			}
		}
		if !demanded {
			continue
		}
		offered := false
		for _, offering := range s.Offerings {
			if offering.InterestAreaID != nil && *offering.InterestAreaID == area.ID {
				offered = true
				break
			}
		}
		if !offered {
			r.Warnings = append(r.Warnings, DraftWarning{ID: "catalog-area-gap", Severity: "info", HostType: "session", HostID: s.Session.ID})
		}
	}
	if s.Session.DraftAssignmentsStale {
		r.Warnings = append(r.Warnings, DraftWarning{ID: "stale-draft", Severity: "warning", HostType: "session", HostID: s.Session.ID})
	}
	sort.Slice(r.Unplaced, func(i, j int) bool { return r.Unplaced[i].StudentName < r.Unplaced[j].StudentName })
	sort.Slice(r.Unwanted, func(i, j int) bool { return r.Unwanted[i].StudentName < r.Unwanted[j].StudentName })
	sort.Slice(r.NoSignal, func(i, j int) bool { return r.NoSignal[i].StudentName < r.NoSignal[j].StudentName })
	return r
}

func assignmentWarning(id string, a data.Assignment, severity string) DraftWarning {
	idv, sv, ov := a.ID, a.StudentID, a.OfferingID
	return DraftWarning{ID: id, Severity: severity, HostType: "assignment", HostID: a.ID, AssignmentID: &idv, StudentID: &sv, OfferingID: &ov}
}
func offeringWarning(id string, offeringID ids.XID, severity string) DraftWarning {
	value := offeringID
	return DraftWarning{ID: id, Severity: severity, HostType: "offering", HostID: offeringID, OfferingID: &value}
}
func displayName(p data.ProgramMembership) string { return p.LegalGivenName + " " + p.LegalFamilyName }

// GetAssignmentQuality reads all inputs in one tenant transaction.
func (s *Service) GetAssignmentQuality(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (AssignmentQuality, error) {
	var result AssignmentQuality
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		non, err := tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		nonParticipants := map[ids.XID]bool{}
		for _, row := range non {
			nonParticipants[row.StudentID] = true
		}
		participants := []data.ProgramMembership{}
		for _, row := range memberships {
			if !nonParticipants[row.StudentID] {
				participants = append(participants, row)
			}
		}
		grades, err := tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		offerings, err := tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		assignments, err := tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		exclusions, err := tx.ListAssignmentExclusions(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		overrides, err := tx.ListAssignmentOverrides(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		areas, err := tx.ListInterestAreas(ctx, schoolYearID, programID, false)
		if err != nil {
			return err
		}
		profiles := make(map[ids.XID]map[ids.XID]data.InterestProfileRating, len(participants))
		for _, participant := range participants {
			values, err := tx.EffectiveInterestProfile(ctx, schoolYearID, programID, participant.StudentID)
			if err != nil {
				return err
			}
			profile := make(map[ids.XID]data.InterestProfileRating, len(values))
			for _, value := range values {
				profile[value.InterestAreaID] = value.Rating
			}
			profiles[participant.StudentID] = profile
		}
		result = EvaluateAssignmentQuality(AssignmentQualitySnapshot{Session: session, Participants: participants, Grades: grades, Offerings: offerings, Assignments: assignments, Exclusions: exclusions, Overrides: overrides, Answers: answers, Profiles: profiles, InterestAreas: areas})
		return nil
	})
	if err != nil {
		return AssignmentQuality{}, fmt.Errorf("get assignment quality: %w", err)
	}
	return result, nil
}
