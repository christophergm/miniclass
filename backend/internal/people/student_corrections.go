package people

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var (
	ErrCorrectionReasonRequired         = errors.New("student correction reason is required")
	ErrPlaceholderGradeRequired         = errors.New("placeholder student grade is required")
	ErrPlaceholderLabelRequired         = errors.New("placeholder student abbreviated label is required")
	ErrStudentIsNotPlaceholder          = errors.New("source student is not a placeholder")
	ErrReconciliationTargetNotConsented = errors.New("reconciliation target has no consented guardian relationship")
	ErrStudentHasGuardianDependencies   = errors.New("placeholder reconciliation cannot move guardian relationships")
	ErrStudentHasPreferenceDependencies = errors.New("placeholder reconciliation cannot move preference records")
	ErrStudentDependentConflict         = errors.New("placeholder reconciliation has conflicting dependent records")
)

// PlaceholderStudentInput is intentionally narrower than a normal roster
// import. Placeholders have a short administrator-supplied label, a required
// grade and homeroom, and no external identity fields.
type PlaceholderStudentInput struct {
	LegalGivenName  string
	LegalFamilyName string
	GradeLevelID    ids.XID
	HomeroomID      ids.XID
	Reason          string
}

type StudentReconciliationResult struct {
	SourceStudentID               ids.XID
	TargetStudentID               ids.XID
	ProgramMembershipsMoved       int64
	SessionNonParticipationsMoved int64
	ArtifactsRegenerated          int
}

// ArtifactRegenerator is called in the reconciliation transaction after
// dependent records move. The current repository has not introduced the
// Phase 6 published-artifact tables yet; the default implementation therefore
// records the requested regeneration in the reconciliation audit entry while
// allowing the publication subsystem to supply a real regenerator.
type ArtifactRegenerator func(context.Context, *data.Tx, ids.XID, ids.XID) (int, error)

// StudentReviewSignal is the administrator review surface's stable read
// model. It deliberately contains opaque IDs and no guardian contact data.
type StudentReviewSignal struct {
	Code      string
	Severity  string
	StudentID ids.XID
	Detail    string
	Count     int64
}

func (s *Service) CreateStudentCorrection(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, input StudentCreateInput, reason string) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("create student correction: data service is nil")
	}
	reason, err := correctionReason(reason)
	if err != nil {
		return data.Student{}, err
	}
	var result data.Student
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		created, err := tx.CreateStudentWithMetadata(ctx, schoolYearID, input.GradeLevelID, input.HomeroomID, input.LegalGivenName, input.LegalFamilyName, input.PreferredGivenName, input.ExternalIdentifier, false, "administrator_correction")
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionStudentAdminCorrection, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: studentSummary(nil, &created)})
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("create student correction: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateStudentCorrection(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input StudentUpdateInput, reason string) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("update student correction: data service is nil")
	}
	reason, err := correctionReason(reason)
	if err != nil {
		return data.Student{}, err
	}
	var result data.Student
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		updatedInput, changed := applyStudentUpdate(current, input)
		if !changed {
			return ErrStudentNoChanges
		}
		updated, err := tx.UpdateStudent(ctx, schoolYearID, id, updatedInput.LegalGivenName, updatedInput.LegalFamilyName, updatedInput.PreferredGivenName, updatedInput.GradeLevelID, updatedInput.HomeroomID, updatedInput.ExternalIdentifier)
		if err != nil {
			return err
		}
		result = updated
		year := updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionStudentAdminCorrection, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: studentSummary(&current, &updated)})
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("update student correction: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteStudentCorrection(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, reason string) error {
	if s == nil || s.database == nil {
		return errors.New("delete student correction: data service is nil")
	}
	reason, err := correctionReason(reason)
	if err != nil {
		return err
	}
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		deleted, err := tx.SoftDeleteStudent(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		year := current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionStudentAdminCorrection, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: studentSummary(&current, nil)})
	})
	if err != nil {
		return fmt.Errorf("delete student correction: %w", err)
	}
	return nil
}

func (s *Service) CreatePlaceholderStudent(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, input PlaceholderStudentInput) (data.Student, error) {
	if s == nil || s.database == nil {
		return data.Student{}, errors.New("create placeholder student: data service is nil")
	}
	reason, err := correctionReason(input.Reason)
	if err != nil {
		return data.Student{}, err
	}
	if strings.TrimSpace(input.LegalGivenName) == "" || strings.TrimSpace(input.LegalFamilyName) == "" {
		return data.Student{}, ErrPlaceholderLabelRequired
	}
	if len([]rune(strings.TrimSpace(input.LegalGivenName))) > 80 || len([]rune(strings.TrimSpace(input.LegalFamilyName))) > 80 {
		return data.Student{}, fmt.Errorf("%w: labels must be at most 80 characters", ErrPlaceholderLabelRequired)
	}
	if strings.TrimSpace(string(input.GradeLevelID)) == "" {
		return data.Student{}, ErrPlaceholderGradeRequired
	}
	var result data.Student
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		if _, err := tx.GetGradeLevelByID(ctx, schoolYearID, input.GradeLevelID); err != nil {
			return err
		}
		if _, err := tx.GetHomeroomByID(ctx, schoolYearID, input.HomeroomID); err != nil {
			return err
		}
		created, err := tx.CreateStudentWithMetadata(ctx, schoolYearID, &input.GradeLevelID, input.HomeroomID, input.LegalGivenName, input.LegalFamilyName, nil, nil, true, "placeholder")
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionPlaceholderStudentCreate, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: studentSummary(nil, &created)})
	})
	if err != nil {
		return data.Student{}, fmt.Errorf("create placeholder student: %w", err)
	}
	return result, nil
}

func (s *Service) ReconcilePlaceholderStudent(ctx context.Context, organizationID string, schoolYearID, placeholderID, targetID ids.XID, actor audit.Actor, reason string) (StudentReconciliationResult, error) {
	if s == nil || s.database == nil {
		return StudentReconciliationResult{}, errors.New("reconcile placeholder student: data service is nil")
	}
	reason, err := correctionReason(reason)
	if err != nil {
		return StudentReconciliationResult{}, err
	}
	if placeholderID == targetID {
		return StudentReconciliationResult{}, errors.New("reconcile placeholder student: source and target must differ")
	}
	var result StudentReconciliationResult
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		placeholder, err := tx.GetStudentByID(ctx, schoolYearID, placeholderID)
		if err != nil {
			return err
		}
		target, err := tx.GetStudentByID(ctx, schoolYearID, targetID)
		if err != nil {
			return err
		}
		if !placeholder.IsPlaceholder {
			return ErrStudentIsNotPlaceholder
		}
		if target.IsPlaceholder {
			return ErrStudentIsNotPlaceholder
		}
		dependencies, err := tx.StudentCorrectionDependencies(ctx, schoolYearID, placeholderID)
		if err != nil {
			return err
		}
		if dependencies.GuardianRelationships > 0 {
			return ErrStudentHasGuardianDependencies
		}
		if dependencies.PreferenceRecords > 0 {
			return ErrStudentHasPreferenceDependencies
		}
		targetDependencies, err := tx.StudentCorrectionDependencies(ctx, schoolYearID, targetID)
		if err != nil {
			return err
		}
		if targetDependencies.GuardianRelationships == 0 {
			return ErrReconciliationTargetNotConsented
		}
		move, err := tx.MoveStudentDependentRecords(ctx, schoolYearID, placeholderID, targetID)
		if err != nil {
			if strings.Contains(err.Error(), "conflicting dependent records") {
				return ErrStudentDependentConflict
			}
			return err
		}
		deleted, err := tx.SoftDeleteStudent(ctx, schoolYearID, placeholderID)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		artifactsRegenerated := 0
		if s.artifactRegenerator != nil {
			artifactsRegenerated, err = s.artifactRegenerator(ctx, tx, placeholderID, targetID)
			if err != nil {
				return fmt.Errorf("regenerate affected published artifacts: %w", err)
			}
		}
		result = StudentReconciliationResult{SourceStudentID: placeholderID, TargetStudentID: targetID, ProgramMembershipsMoved: move.ProgramMemberships, SessionNonParticipationsMoved: move.SessionNonParticipations, ArtifactsRegenerated: artifactsRegenerated}
		id, year := target.ID, target.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionStudentReconciliation, ObjectType: "student", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: reconciliationSummary(result)})
	})
	if err != nil {
		return StudentReconciliationResult{}, fmt.Errorf("reconcile placeholder student: %w", err)
	}
	return result, nil
}

func (s *Service) ListStudentReviewSignals(ctx context.Context, organizationID string, schoolYearID ids.XID) ([]StudentReviewSignal, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list student review signals: data service is nil")
	}
	var result []StudentReviewSignal
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		students, err := tx.ListStudents(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		relationships, err := tx.ListGuardianRelationships(ctx, schoolYearID, data.GuardianRelationshipFilter{})
		if err != nil {
			return err
		}
		adults, err := tx.ListAdults(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		recentCorrections, err := tx.ListRecentStudentCorrectionCounts(ctx, schoolYearID)
		if err != nil {
			return err
		}
		result = studentReviewSignals(students, relationships, adults, recentCorrections)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list student review signals: %w", err)
	}
	return result, nil
}

func (s *Service) WithArtifactRegenerator(regenerator ArtifactRegenerator) *Service {
	if s != nil {
		s.artifactRegenerator = regenerator
	}
	return s
}

func correctionReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", ErrCorrectionReasonRequired
	}
	return reason, nil
}

func reconciliationSummary(result StudentReconciliationResult) json.RawMessage {
	encoded, err := json.Marshal(map[string]any{
		"source_student_id": result.SourceStudentID, "target_student_id": result.TargetStudentID,
		"program_memberships_moved":        result.ProgramMembershipsMoved,
		"session_non_participations_moved": result.SessionNonParticipationsMoved,
		"artifacts_regenerated":            result.ArtifactsRegenerated,
		"artifact_regeneration_requested":  true,
	})
	if err != nil {
		return json.RawMessage(`{"error":"could not encode reconciliation summary"}`)
	}
	return encoded
}

func studentReviewSignals(students []data.Student, relationships []data.GuardianRelationship, adults []data.Adult, recent []data.StudentCorrectionReviewCount) []StudentReviewSignal {
	studentByID := make(map[ids.XID]data.Student, len(students))
	for _, student := range students {
		studentByID[student.ID] = student
	}
	relationshipCount := make(map[ids.XID]int, len(students))
	adultByID := make(map[ids.XID]data.Adult, len(adults))
	for _, adult := range adults {
		adultByID[adult.ID] = adult
	}
	signals := make([]StudentReviewSignal, 0)
	for _, relationship := range relationships {
		relationshipCount[relationship.StudentID]++
		adult := adultByID[relationship.AdultID]
		if adult.Email == nil || strings.TrimSpace(*adult.Email) == "" {
			signals = append(signals, StudentReviewSignal{Code: "registration-no-email", Severity: "warning", StudentID: relationship.StudentID, Detail: "A guardian relationship has no verified email address."})
		}
	}
	duplicateGroups := make(map[string][]data.Student)
	for _, student := range students {
		if student.IsPlaceholder {
			signals = append(signals, StudentReviewSignal{Code: "placeholder-student", Severity: "info", StudentID: student.ID, Detail: "Placeholder student requires administrator review before reconciliation."})
			continue
		}
		if relationshipCount[student.ID] == 0 {
			signals = append(signals, StudentReviewSignal{Code: "registration-unlinked", Severity: "warning", StudentID: student.ID, Detail: "No guardian relationship is registered for this student."})
		}
		duplicateGroups[normalizeReviewName(student.LegalGivenName)+"\x00"+normalizeReviewName(student.LegalFamilyName)] = append(duplicateGroups[normalizeReviewName(student.LegalGivenName)+"\x00"+normalizeReviewName(student.LegalFamilyName)], student)
	}
	for _, group := range duplicateGroups {
		if len(group) < 2 {
			continue
		}
		for _, student := range group {
			signals = append(signals, StudentReviewSignal{Code: "matching-duplicate", Severity: "warning", StudentID: student.ID, Detail: "Multiple active students share the same normalized name; matching requires human review."})
		}
	}
	for _, correction := range recent {
		if _, ok := studentByID[correction.StudentID]; !ok {
			continue
		}
		signals = append(signals, StudentReviewSignal{Code: "unusual-activity", Severity: "warning", StudentID: correction.StudentID, Detail: "This student has received repeated administrative correction activity in the last 30 days.", Count: correction.CorrectionCount})
	}
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].Code != signals[j].Code {
			return signals[i].Code < signals[j].Code
		}
		return signals[i].StudentID < signals[j].StudentID
	})
	return signals
}

func normalizeReviewName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
