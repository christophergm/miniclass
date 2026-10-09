package program

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
)

type ArtifactKind string

const (
	ArtifactClassList         ArtifactKind = "class_list"
	ArtifactHomeroomDismissal ArtifactKind = "homeroom_dismissal"
)

var ErrArtifactKind = errors.New("unsupported artifact kind")

type ArtifactDocument struct {
	Kind        ArtifactKind
	SessionName string
	GeneratedAt time.Time
	Sections    []ArtifactSection
	Warnings    []ArtifactWarning
}
type ArtifactSection struct {
	ID, Title string
	Blocks    []ArtifactBlock
}
type ArtifactBlock struct{ Kind, Label, Text string }
type ArtifactWarning struct {
	Code, Message string
	StudentNames  []string
}

type artifactRoster struct {
	session           data.Session
	students          []data.Student
	memberships       []data.ProgramMembership
	nonParticipations []data.SessionNonParticipation
	offerings         []data.Offering
	assignments       []data.Assignment
	grades            []data.GradeLevel
	homerooms         []data.Homeroom
}

// GenerateArtifact implements SPEC §§18.3–18.5 within the agreed admin-only,
// current-assignment scope: no snapshots, publication, links, staffing or tags.
// It deliberately does not read the workspace, comments, preferences or adults.
func (s *Service) GenerateArtifact(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID, kind ArtifactKind) (ArtifactDocument, error) {
	if kind != ArtifactClassList && kind != ArtifactHomeroomDismissal {
		return ArtifactDocument{}, ErrArtifactKind
	}
	if s == nil || s.database == nil {
		return ArtifactDocument{}, errors.New("generate artifact: data service is nil")
	}
	var roster artifactRoster
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		roster.session, err = tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		dates, err := tx.ListMeetingDates(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		roster.session.MeetingDates = meetingDateTimes(dates)
		roster.memberships, err = tx.ListProgramMemberships(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		roster.nonParticipations, err = tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		roster.students, err = tx.ListStudents(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		roster.grades, err = tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		roster.homerooms, err = tx.ListHomerooms(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		roster.offerings, err = tx.ListOfferings(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		roster.assignments, err = tx.ListAssignments(ctx, schoolYearID, programID, sessionID)
		return err
	})
	if err != nil {
		return ArtifactDocument{}, fmt.Errorf("generate artifact: %w", err)
	}
	return renderArtifact(roster, kind, time.Now().UTC()), nil
}

type artifactStudent struct {
	id                    ids.XID
	name, grade, homeroom string
	homeroomID            ids.XID
	ordinal               *int
	missingHomeroom       bool
}

func artifactDisplayName(student data.Student) string {
	// Enforce privacy at rendering even when the source still retains old names.
	if student.DeletedAt != nil {
		return "Deleted student"
	}
	if student.IsPlaceholder {
		return people.DisplayName(nil, &student.LegalGivenName, &student.LegalFamilyName)
	}
	return people.DisplayName(student.PreferredGivenName, &student.LegalGivenName, &student.LegalFamilyName)
}
func artifactStudentNames(students []artifactStudent) []string {
	ordered := append([]artifactStudent{}, students...)
	sort.Slice(ordered, func(i, j int) bool {
		return artifactLess(ordered[i].name, ordered[j].name, string(ordered[i].id), string(ordered[j].id))
	})
	names := make([]string, 0, len(ordered))
	for _, student := range ordered {
		names = append(names, student.name)
	}
	return names
}

func artifactText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
func artifactLess(a, b, aID, bID string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if a == b {
		return aID < bID
	}
	return a < b
}
func renderArtifact(r artifactRoster, kind ArtifactKind, now time.Time) ArtifactDocument {
	doc := ArtifactDocument{Kind: kind, SessionName: r.session.Name, GeneratedAt: now, Sections: []ArtifactSection{}, Warnings: []ArtifactWarning{}}
	grades := map[ids.XID]data.GradeLevel{}
	for _, g := range r.grades {
		grades[g.ID] = g
	}
	rooms := map[ids.XID]string{}
	for _, h := range r.homerooms {
		rooms[h.ID] = h.Name
	}
	students := map[ids.XID]data.Student{}
	for _, s := range r.students {
		students[s.ID] = s
	}
	active := map[ids.XID]bool{}
	for _, m := range r.memberships {
		if s, ok := students[m.StudentID]; ok && s.DeletedAt == nil {
			active[m.StudentID] = true
		}
	}
	for _, n := range r.nonParticipations {
		delete(active, n.StudentID)
	}
	offerings := append([]data.Offering{}, r.offerings...)
	sort.Slice(offerings, func(i, j int) bool {
		return artifactLess(offerings[i].Name, offerings[j].Name, string(offerings[i].ID), string(offerings[j].ID))
	})
	known := map[ids.XID]bool{}
	for _, o := range offerings {
		known[o.ID] = true
	}
	placed := map[ids.XID]bool{}
	byOffering := map[ids.XID][]artifactStudent{}
	missingGrades := map[ids.XID]artifactStudent{}
	missingHomerooms := map[ids.XID]artifactStudent{}
	for _, a := range r.assignments {
		s, ok := students[a.StudentID]
		if !ok || !known[a.OfferingID] || (!active[s.ID] && s.DeletedAt == nil) {
			continue
		}
		p := artifactStudent{id: s.ID, name: artifactDisplayName(s), grade: "Grade not set", homeroom: "Homeroom not set", missingHomeroom: true}
		if s.GradeLevelID != nil {
			if g, ok := grades[*s.GradeLevelID]; ok && strings.TrimSpace(g.Label) != "" {
				p.grade = "Grade " + g.Label
				p.ordinal = &g.Ordinal
			}
		}
		if name, ok := rooms[s.HomeroomID]; ok {
			p.homeroom = artifactText(name, "Homeroom not set")
			p.homeroomID = s.HomeroomID
			p.missingHomeroom = strings.TrimSpace(name) == ""
		}
		if p.ordinal == nil {
			missingGrades[p.id] = p
		}
		if p.missingHomeroom {
			missingHomerooms[p.id] = p
		}
		byOffering[a.OfferingID] = append(byOffering[a.OfferingID], p)
		placed[s.ID] = true
	}
	unplaced := []string{}
	for id := range active {
		if !placed[id] {
			unplaced = append(unplaced, artifactDisplayName(students[id]))
		}
	}
	sort.Slice(unplaced, func(i, j int) bool { return artifactLess(unplaced[i], unplaced[j], unplaced[i], unplaced[j]) })
	if len(unplaced) > 0 {
		doc.Warnings = append(doc.Warnings, ArtifactWarning{Code: "unplaced_students", Message: "Participating students without a current placement are not included in the document.", StudentNames: unplaced})
	}
	if r.session.DraftAssignmentsStale {
		doc.Warnings = append(doc.Warnings, ArtifactWarning{Code: "stale_assignments", Message: "Current assignments may be stale after roster or catalog changes.", StudentNames: []string{}})
	}
	for _, missing := range []struct {
		code, message string
		students      map[ids.XID]artifactStudent
	}{
		{"missing_grade", "Placed students have no grade set.", missingGrades},
		{"missing_homeroom", "Placed students have no homeroom set.", missingHomerooms},
	} {
		if len(missing.students) > 0 {
			list := make([]artifactStudent, 0, len(missing.students))
			for _, student := range missing.students {
				list = append(list, student)
			}
			doc.Warnings = append(doc.Warnings, ArtifactWarning{Code: missing.code, Message: missing.message, StudentNames: artifactStudentNames(list)})
		}
	}
	dates := []string{}
	for _, d := range r.session.MeetingDates {
		dates = append(dates, d.Format("2006-01-02"))
	}
	sort.Strings(dates)
	roomSections := map[ids.XID]*ArtifactSection{}
	for _, o := range offerings {
		list := byOffering[o.ID]
		// Empty offerings are rendered only by the class list. Warn without
		// adding operational guidance to the printable document itself.
		if (kind == ArtifactClassList || len(list) > 0) && strings.TrimSpace(o.MeetingPoint) == "" && strings.TrimSpace(o.Location) == "" {
			doc.Warnings = append(doc.Warnings, ArtifactWarning{Code: "missing_destination", Message: fmt.Sprintf("Offering %q has no meeting destination set.", o.Name), StudentNames: artifactStudentNames(list)})
		}
		sort.Slice(list, func(i, j int) bool {
			if kind == ArtifactClassList {
				if list[i].ordinal == nil && list[j].ordinal != nil {
					return false
				}
				if list[i].ordinal != nil && list[j].ordinal == nil {
					return true
				}
				if list[i].ordinal != nil && list[j].ordinal != nil && *list[i].ordinal != *list[j].ordinal {
					return *list[i].ordinal < *list[j].ordinal
				}
			}
			return artifactLess(list[i].name, list[j].name, string(list[i].id), string(list[j].id))
		})
		if kind == ArtifactClassList {
			section := ArtifactSection{ID: string(o.ID), Title: o.Name, Blocks: []ArtifactBlock{}}
			paragraph := func(label, text string) {
				section.Blocks = append(section.Blocks, ArtifactBlock{Kind: "paragraph", Label: label, Text: text})
			}
			if strings.TrimSpace(o.Description) != "" {
				paragraph("Description", o.Description)
			}
			if strings.TrimSpace(o.MeetingPoint) != "" && strings.TrimSpace(o.MeetingPoint) != strings.TrimSpace(o.Location) {
				paragraph("Meeting point", o.MeetingPoint)
			}
			paragraph("Location", artifactText(o.Location, "Meeting location not set"))
			if strings.TrimSpace(o.MeetingInstructions) != "" {
				paragraph("Meeting instructions", o.MeetingInstructions)
			}
			paragraph("Meeting dates", strings.Join(dates, ", "))
			min, max := grades[o.MinGradeLevelID].Label, grades[o.MaxGradeLevelID].Label
			gradeRange := "Grade not set"
			if strings.TrimSpace(min) != "" && strings.TrimSpace(max) != "" {
				gradeRange = min
				if min != max {
					gradeRange += "–" + max
				}
			}
			paragraph("Grades", gradeRange)
			paragraph("Enrolled count", strconv.Itoa(len(list)))
			section.Blocks = append(section.Blocks, ArtifactBlock{Kind: "heading", Text: "Students"})
			for _, p := range list {
				section.Blocks = append(section.Blocks, ArtifactBlock{Kind: "numbered", Label: p.name, Text: p.grade + ", " + p.homeroom})
			}
			doc.Sections = append(doc.Sections, section)
		} else {
			groups := map[ids.XID][]artifactStudent{}
			for _, p := range list {
				groups[p.homeroomID] = append(groups[p.homeroomID], p)
			}
			for roomID, group := range groups {
				section := roomSections[roomID]
				if section == nil {
					id := string(roomID)
					if id == "" {
						id = "homeroom-not-set"
					}
					section = &ArtifactSection{ID: id, Title: group[0].homeroom, Blocks: []ArtifactBlock{}}
					roomSections[roomID] = section
				}
				destination := artifactText(o.MeetingPoint, artifactText(o.Location, "Meeting location not set"))
				section.Blocks = append(section.Blocks, ArtifactBlock{Kind: "paragraph", Label: o.Name, Text: "meet at " + destination})
				for _, p := range group {
					section.Blocks = append(section.Blocks, ArtifactBlock{Kind: "bullet", Text: p.name + " (" + p.grade + ")"})
				}
			}
		}
	}
	if kind == ArtifactHomeroomDismissal {
		for _, section := range roomSections {
			doc.Sections = append(doc.Sections, *section)
		}
		sort.Slice(doc.Sections, func(i, j int) bool {
			return artifactLess(doc.Sections[i].Title, doc.Sections[j].Title, doc.Sections[i].ID, doc.Sections[j].ID)
		})
	}
	return doc
}
