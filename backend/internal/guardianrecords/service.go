package guardianrecords

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
)

var (
	ErrOutOfScope           = errors.New("student is outside the guardian scope")
	ErrGradeRequired        = errors.New("grade is required for guardian-managed students")
	ErrNoChanges            = errors.New("guardian student update has no changes")
	ErrCandidateInvalid     = errors.New("guardian candidate does not exist")
	ErrConfirmationRequired = errors.New("confirmation is required")
)

// GuardianSessionRevoker is implemented by the identity service. Identity
// tokens remain behind internal/identity; this narrow callback keeps that
// accessor out of the tenant data package.
type GuardianSessionRevoker interface {
	RevokeGuardianSessionsAndOTPs(context.Context, ids.XID, ids.XID, ids.XID) error
}

type ReviewWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Student struct {
	ID                 ids.XID
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	GradeLevelID       *ids.XID
	GradeLabel         string
	HomeroomID         ids.XID
	HomeroomLabel      string
	Warnings           []ReviewWarning
}

type CandidateInput struct {
	GivenName  string
	FamilyName string
}

type CreateInput struct {
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	GradeLevelID       ids.XID
	HomeroomID         ids.XID
	RelationshipType   data.GuardianRelationshipType
}

type UpdateInput struct {
	LegalGivenName     *string
	LegalFamilyName    *string
	PreferredGivenName *string
	GradeLevelID       *ids.XID
	HomeroomID         *ids.XID
}

type ProfileInput struct {
	LegalGivenName     *string
	LegalFamilyName    *string
	PreferredGivenName *string
	Email              *string
	Phone              *string
}

type Service struct {
	database *data.DB
	sessions GuardianSessionRevoker
}

func New(database *data.DB, sessions ...GuardianSessionRevoker) *Service {
	var revoker GuardianSessionRevoker
	if len(sessions) > 0 {
		revoker = sessions[0]
	}
	return &Service{database: database, sessions: revoker}
}

// Detach removes only this guardian's relationship. If it was the last
// active guardian, the student is hard-deleted when no dependent history
// exists, otherwise it is de-identified and retained for history.
func (s *Service) Detach(ctx context.Context, principal auth.GuardianPrincipal, studentID ids.XID, confirmed bool, actor audit.Actor) error {
	if !confirmed {
		return ErrConfirmationRequired
	}
	if s == nil || s.database == nil {
		return errors.New("detach guardian student: data service is nil")
	}
	return s.database.InTenant(ctx, string(principal.OrganizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if err := ensureYearScope(ctx, tx, principal, studentID); err != nil {
			return err
		}
		removed, err := tx.DeleteGuardianRelationshipForStudent(ctx, principal.SchoolYearID, principal.AdultID, studentID)
		if err != nil {
			return err
		}
		if !removed {
			return ErrOutOfScope
		}
		other, err := tx.CountOtherActiveGuardians(ctx, principal.SchoolYearID, studentID, principal.AdultID)
		if err != nil {
			return err
		}
		outcome := "detached"
		if other == 0 {
			associated, err := tx.CountStudentAssociatedData(ctx, principal.SchoolYearID, studentID)
			if err != nil {
				return err
			}
			if associated == 0 {
				if err := tx.HardDeleteStudent(ctx, principal.SchoolYearID, studentID); err != nil {
					return err
				}
				outcome = "hard_deleted"
			} else {
				if _, err := tx.DeidentifyStudent(ctx, principal.SchoolYearID, studentID); err != nil {
					return err
				}
				outcome = "deidentified"
			}
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionGuardianDetach, ObjectType: "student", ObjectID: &studentID, SchoolYearID: &principal.SchoolYearID, ChangeSummary: json.RawMessage(fmt.Sprintf(`{"outcome":%q}`, outcome))})
	})
}

// DeleteSelf removes the guardian adult profile and applies the same
// relationship outcome to each student. A linked administrative account is
// deliberately untouched.
func (s *Service) DeleteSelf(ctx context.Context, principal auth.GuardianPrincipal, confirmed bool, actor audit.Actor) error {
	if !confirmed {
		return ErrConfirmationRequired
	}
	if s == nil || s.database == nil {
		return errors.New("delete guardian profile: data service is nil")
	}
	err := s.database.InTenant(ctx, string(principal.OrganizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		students, err := tx.DeleteGuardianRelationshipsForAdult(ctx, principal.SchoolYearID, principal.AdultID)
		if err != nil {
			return err
		}
		for _, studentID := range students {
			other, err := tx.CountOtherActiveGuardians(ctx, principal.SchoolYearID, studentID, principal.AdultID)
			if err != nil {
				return err
			}
			if other != 0 {
				continue
			}
			associated, err := tx.CountStudentAssociatedData(ctx, principal.SchoolYearID, studentID)
			if err != nil {
				return err
			}
			if associated == 0 {
				if err := tx.HardDeleteStudent(ctx, principal.SchoolYearID, studentID); err != nil {
					return err
				}
			} else if _, err := tx.DeidentifyStudent(ctx, principal.SchoolYearID, studentID); err != nil {
				return err
			}
		}
		if _, err := tx.SoftDeleteAdult(ctx, principal.SchoolYearID, principal.AdultID); err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionPersonalDataDelete, ObjectType: "adult", ObjectID: &principal.AdultID, SchoolYearID: &principal.SchoolYearID, ChangeSummary: json.RawMessage(`{"surface":"guardian_self_delete"}`)})
	})
	if err != nil {
		return err
	}
	if s.sessions != nil {
		return s.sessions.RevokeGuardianSessionsAndOTPs(ctx, principal.OrganizationID, principal.SchoolYearID, principal.AdultID)
	}
	return nil
}

func (s *Service) List(ctx context.Context, principal auth.GuardianPrincipal) ([]Student, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list guardian students: data service is nil")
	}
	var result []Student
	err := s.database.InTenantRead(ctx, string(principal.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		scope, err := tx.ResolveGuardianScope(ctx, principal.SchoolYearID, principal.AdultID)
		if err != nil {
			return err
		}
		result = make([]Student, 0, len(scope.StudentIDs))
		for _, studentID := range scope.StudentIDs {
			student, err := tx.GetStudentByID(ctx, principal.SchoolYearID, studentID)
			if err != nil {
				return err
			}
			view, err := studentView(ctx, tx, student)
			if err != nil {
				return err
			}
			result = append(result, view)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list guardian students: %w", err)
	}
	return result, nil
}

func (s *Service) FindCandidates(ctx context.Context, principal auth.GuardianPrincipal, input CandidateInput) ([]Student, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("find guardian student candidates: data service is nil")
	}
	given, family := normalize(input.GivenName), normalize(input.FamilyName)
	if given == "" || family == "" {
		return []Student{}, nil
	}
	var result []Student
	err := s.database.InTenantRead(ctx, string(principal.OrganizationID), func(ctx context.Context, tx *data.Tx) error {
		students, err := tx.ListStudents(ctx, principal.SchoolYearID, false)
		if err != nil {
			return err
		}
		for _, student := range students {
			givenMatches := normalize(student.LegalGivenName) == given
			if student.PreferredGivenName != nil {
				givenMatches = givenMatches || normalize(*student.PreferredGivenName) == given
			}
			if isPlaceholder(student) || !givenMatches || normalize(student.LegalFamilyName) != family {
				continue
			}
			view, err := studentView(ctx, tx, student)
			if err != nil {
				return err
			}
			result = append(result, view)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find guardian student candidates: %w", err)
	}
	return result, nil
}

func (s *Service) Select(ctx context.Context, principal auth.GuardianPrincipal, studentID ids.XID, relationshipType data.GuardianRelationshipType, actor audit.Actor) (Student, error) {
	if s == nil || s.database == nil {
		return Student{}, errors.New("select guardian student: data service is nil")
	}
	var result Student
	err := s.database.InTenant(ctx, string(principal.OrganizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.ResolveGuardianScope(ctx, principal.SchoolYearID, principal.AdultID); err != nil {
			return err
		}
		student, err := tx.GetStudentByID(ctx, principal.SchoolYearID, studentID)
		if err != nil || isPlaceholder(student) {
			if err == nil {
				err = ErrCandidateInvalid
			}
			return err
		}
		relationship, err := tx.CreateGuardianRelationship(ctx, principal.SchoolYearID, principal.AdultID, studentID, relationshipType)
		if err != nil {
			return err
		}
		id, year := relationship.ID, relationship.SchoolYearID
		if err := tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "guardian_relationship", ObjectID: &id, SchoolYearID: &year, ChangeSummary: guardianSelectionSummary(studentID, true)}); err != nil {
			return err
		}
		result, err = studentView(ctx, tx, student)
		return err
	})
	if err != nil {
		return Student{}, fmt.Errorf("select guardian student: %w", err)
	}
	return result, nil
}

func (s *Service) Create(ctx context.Context, principal auth.GuardianPrincipal, input CreateInput, actor audit.Actor) (Student, error) {
	if s == nil || s.database == nil {
		return Student{}, errors.New("create guardian student: data service is nil")
	}
	if input.GradeLevelID == "" {
		return Student{}, ErrGradeRequired
	}
	var result Student
	err := s.database.InTenant(ctx, string(principal.OrganizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.ResolveGuardianScope(ctx, principal.SchoolYearID, principal.AdultID); err != nil {
			return err
		}
		if _, err := tx.GetGradeLevelByID(ctx, principal.SchoolYearID, input.GradeLevelID); err != nil {
			return err
		}
		if _, err := tx.GetHomeroomByID(ctx, principal.SchoolYearID, input.HomeroomID); err != nil {
			return err
		}
		student, err := tx.CreateStudentWithMetadata(ctx, principal.SchoolYearID, &input.GradeLevelID, input.HomeroomID, input.LegalGivenName, input.LegalFamilyName, input.PreferredGivenName, nil, false, "guardian")
		if err != nil {
			return err
		}
		relationship, err := tx.CreateGuardianRelationship(ctx, principal.SchoolYearID, principal.AdultID, student.ID, input.RelationshipType)
		if err != nil {
			return err
		}
		studentID, year := student.ID, student.SchoolYearID
		if err := tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "student", ObjectID: &studentID, SchoolYearID: &year, ChangeSummary: guardianStudentSummary(&student)}); err != nil {
			return err
		}
		relationshipID := relationship.ID
		if err := tx.Record(ctx, audit.Entry{Action: audit.ActionMembershipChange, ObjectType: "guardian_relationship", ObjectID: &relationshipID, SchoolYearID: &year, ChangeSummary: guardianSelectionSummary(student.ID, false)}); err != nil {
			return err
		}
		result, err = studentView(ctx, tx, student)
		return err
	})
	if err != nil {
		return Student{}, fmt.Errorf("create guardian student: %w", err)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, principal auth.GuardianPrincipal, studentID ids.XID, input UpdateInput, actor audit.Actor) (Student, error) {
	if s == nil || s.database == nil {
		return Student{}, errors.New("update guardian student: data service is nil")
	}
	var result Student
	err := s.database.InTenant(ctx, string(principal.OrganizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if err := ensureYearScope(ctx, tx, principal, studentID); err != nil {
			return err
		}
		current, err := tx.GetStudentByID(ctx, principal.SchoolYearID, studentID)
		if err != nil {
			return err
		}
		given, family, preferred := current.LegalGivenName, current.LegalFamilyName, current.PreferredGivenName
		grade, homeroom := current.GradeLevelID, current.HomeroomID
		changed := false
		if input.LegalGivenName != nil && normalize(*input.LegalGivenName) != normalize(given) {
			given, changed = strings.TrimSpace(*input.LegalGivenName), true
		}
		if input.LegalFamilyName != nil && normalize(*input.LegalFamilyName) != normalize(family) {
			family, changed = strings.TrimSpace(*input.LegalFamilyName), true
		}
		if input.PreferredGivenName != nil && !sameText(preferred, input.PreferredGivenName) {
			preferred, changed = input.PreferredGivenName, true
		}
		if input.GradeLevelID != nil && !sameID(grade, input.GradeLevelID) {
			grade, changed = input.GradeLevelID, true
		}
		if input.HomeroomID != nil && !sameID(&homeroom, input.HomeroomID) {
			homeroom, changed = *input.HomeroomID, true
		}
		if !changed {
			return ErrNoChanges
		}
		if grade == nil {
			return ErrGradeRequired
		}
		if _, err := tx.GetGradeLevelByID(ctx, principal.SchoolYearID, *grade); err != nil {
			return err
		}
		if _, err := tx.GetHomeroomByID(ctx, principal.SchoolYearID, homeroom); err != nil {
			return err
		}
		updated, err := tx.UpdateStudent(ctx, principal.SchoolYearID, studentID, given, family, preferred, grade, homeroom, current.ExternalIdentifier)
		if err != nil {
			return err
		}
		result, err = studentView(ctx, tx, updated)
		if err != nil {
			return err
		}
		if current.GradeLevelID != updated.GradeLevelID || current.HomeroomID != updated.HomeroomID {
			result.Warnings = []ReviewWarning{{Code: "student-attributes-review", Message: "Grade or homeroom changed. Review affected preferences, programme membership, assignments, and published information; historical submissions and completed work were not changed."}}
		}
		id, year := updated.ID, updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, ChangeSummary: guardianStudentChangeSummary(&current, &updated, result.Warnings)})
	})
	if err != nil {
		return Student{}, fmt.Errorf("update guardian student: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateProfile(ctx context.Context, principal auth.GuardianPrincipal, input ProfileInput, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("update guardian profile: data service is nil")
	}
	return s.database.InTenant(ctx, string(principal.OrganizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetAdultByID(ctx, principal.SchoolYearID, principal.AdultID)
		if err != nil {
			return err
		}
		given, family, preferred, email, phone := current.LegalGivenName, current.LegalFamilyName, current.PreferredGivenName, current.Email, current.Phone
		changed := false
		if input.LegalGivenName != nil {
			given, changed = strings.TrimSpace(*input.LegalGivenName), true
		}
		if input.LegalFamilyName != nil {
			family, changed = strings.TrimSpace(*input.LegalFamilyName), true
		}
		if input.PreferredGivenName != nil {
			preferred, changed = input.PreferredGivenName, true
		}
		if input.Email != nil {
			email, changed = input.Email, true
		}
		if input.Phone != nil {
			phone, changed = input.Phone, true
		}
		if !changed {
			return people.ErrNoChanges
		}
		updated, err := tx.UpdateAdult(ctx, principal.SchoolYearID, principal.AdultID, given, family, preferred, email, phone, current.ExternalIdentifier, current.ParticipationIntent)
		if err != nil {
			return err
		}
		id, year := updated.ID, updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "adult", ObjectID: &id, SchoolYearID: &year, ChangeSummary: json.RawMessage(`{"surface":"guardian_profile","fields":["permitted_profile_fields"]}`)})
	})
}

func ensureYearScope(ctx context.Context, tx *data.Tx, principal auth.GuardianPrincipal, studentID ids.XID) error {
	scope, err := tx.ResolveGuardianScope(ctx, principal.SchoolYearID, principal.AdultID)
	if err != nil {
		return err
	}
	for _, id := range scope.StudentIDs {
		if id == studentID {
			return nil
		}
	}
	return ErrOutOfScope
}

func studentView(ctx context.Context, tx *data.Tx, student data.Student) (Student, error) {
	view := Student{ID: student.ID, LegalGivenName: student.LegalGivenName, LegalFamilyName: student.LegalFamilyName, PreferredGivenName: student.PreferredGivenName, GradeLevelID: student.GradeLevelID, HomeroomID: student.HomeroomID}
	if student.GradeLevelID != nil {
		grade, err := tx.GetGradeLevelByID(ctx, student.SchoolYearID, *student.GradeLevelID)
		if err != nil {
			return Student{}, err
		}
		view.GradeLabel = grade.Label
	}
	homeroom, err := tx.GetHomeroomByID(ctx, student.SchoolYearID, student.HomeroomID)
	if err != nil {
		return Student{}, err
	}
	view.HomeroomLabel = homeroom.Name
	return view, nil
}

func normalize(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func isPlaceholder(student data.Student) bool {
	if student.IsPlaceholder {
		return true
	}
	value := normalize(student.LegalGivenName + " " + student.LegalFamilyName)
	for _, marker := range []string{"placeholder", "unknown", "not known", "n/a", "tbd"} {
		if value == marker || strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func sameText(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return strings.TrimSpace(*a) == strings.TrimSpace(*b)
}
func sameID(a, b *ids.XID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func guardianStudentSummary(student *data.Student) json.RawMessage {
	return guardianStudentChangeSummary(nil, student, nil)
}
func guardianStudentChangeSummary(before, after *data.Student, warnings []ReviewWarning) json.RawMessage {
	value := map[string]any{"warnings": warnings}
	if before != nil {
		value["before"] = map[string]any{"legal_given_name": before.LegalGivenName, "legal_family_name": before.LegalFamilyName, "grade_level_id": before.GradeLevelID, "homeroom_id": before.HomeroomID}
	}
	if after != nil {
		value["after"] = map[string]any{"legal_given_name": after.LegalGivenName, "legal_family_name": after.LegalFamilyName, "grade_level_id": after.GradeLevelID, "homeroom_id": after.HomeroomID}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode guardian student audit summary"}`)
	}
	return encoded
}
func guardianSelectionSummary(studentID ids.XID, selected bool) json.RawMessage {
	value := map[string]any{"student_id": studentID, "selected_existing": selected}
	encoded, _ := json.Marshal(value)
	return encoded
}
