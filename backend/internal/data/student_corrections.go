package data

import (
	"context"
	"fmt"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

// StudentCorrectionDependencies describes records that make a placeholder
// reconciliation unsafe until an administrator reviews them explicitly.
type StudentCorrectionDependencies struct {
	GuardianRelationships int64
	PreferenceRecords     int64
}

// StudentCorrectionMove is the set of dependent records moved during a
// reviewed placeholder reconciliation.
type StudentCorrectionMove struct {
	ProgramMemberships       int64
	SessionNonParticipations int64
}

// StudentCorrectionReviewCount is a compact signal for repeated administrative
// activity on one student in the recent review window.
type StudentCorrectionReviewCount struct {
	StudentID       ids.XID
	CorrectionCount int64
}

func (tx *Tx) StudentCorrectionDependencies(ctx context.Context, schoolYearID, studentID ids.XID) (StudentCorrectionDependencies, error) {
	guardianRelationships, err := tx.queries.CountStudentGuardianRelationships(ctx, db.CountStudentGuardianRelationshipsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, StudentID: studentID,
	})
	if err != nil {
		return StudentCorrectionDependencies{}, fmt.Errorf("count student guardian relationships: %w", err)
	}
	preferenceRecords, err := tx.queries.CountStudentPreferenceDependencies(ctx, db.CountStudentPreferenceDependenciesParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, StudentID: studentID,
	})
	if err != nil {
		return StudentCorrectionDependencies{}, fmt.Errorf("count student preference dependencies: %w", err)
	}
	return StudentCorrectionDependencies{GuardianRelationships: guardianRelationships, PreferenceRecords: preferenceRecords}, nil
}

func (tx *Tx) MoveStudentDependentRecords(ctx context.Context, schoolYearID, sourceStudentID, targetStudentID ids.XID) (StudentCorrectionMove, error) {
	programMembershipConflicts, err := tx.queries.CountStudentProgramMembershipConflicts(ctx, db.CountStudentProgramMembershipConflictsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, StudentID: sourceStudentID, StudentID_2: targetStudentID,
	})
	if err != nil {
		return StudentCorrectionMove{}, fmt.Errorf("count student programme membership conflicts: %w", err)
	}
	sessionNonParticipationConflicts, err := tx.queries.CountStudentSessionNonParticipationConflicts(ctx, db.CountStudentSessionNonParticipationConflictsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, StudentID: sourceStudentID, StudentID_2: targetStudentID,
	})
	if err != nil {
		return StudentCorrectionMove{}, fmt.Errorf("count student session non-participation conflicts: %w", err)
	}
	if programMembershipConflicts > 0 || sessionNonParticipationConflicts > 0 {
		return StudentCorrectionMove{}, fmt.Errorf("student dependent records conflict: programme_memberships=%d session_non_participations=%d", programMembershipConflicts, sessionNonParticipationConflicts)
	}
	programMemberships, err := tx.queries.MoveStudentProgramMemberships(ctx, db.MoveStudentProgramMembershipsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, StudentID: sourceStudentID, StudentID_2: targetStudentID,
	})
	if err != nil {
		return StudentCorrectionMove{}, fmt.Errorf("move student programme memberships: %w", err)
	}
	sessionNonParticipations, err := tx.queries.MoveStudentSessionNonParticipations(ctx, db.MoveStudentSessionNonParticipationsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, StudentID: sourceStudentID, StudentID_2: targetStudentID,
	})
	if err != nil {
		return StudentCorrectionMove{}, fmt.Errorf("move student session non-participations: %w", err)
	}
	return StudentCorrectionMove{ProgramMemberships: programMemberships, SessionNonParticipations: sessionNonParticipations}, nil
}

func (tx *Tx) ListRecentStudentCorrectionCounts(ctx context.Context, schoolYearID ids.XID) ([]StudentCorrectionReviewCount, error) {
	rows, err := tx.queries.ListRecentStudentCorrectionCounts(ctx, db.ListRecentStudentCorrectionCountsParams{
		OrganizationID: tx.organizationID, SchoolYearID: &schoolYearID,
	})
	if err != nil {
		return nil, fmt.Errorf("list recent student correction counts: %w", err)
	}
	result := make([]StudentCorrectionReviewCount, 0, len(rows))
	for _, row := range rows {
		if row.ObjectID == nil {
			continue
		}
		result = append(result, StudentCorrectionReviewCount{StudentID: *row.ObjectID, CorrectionCount: row.CorrectionCount})
	}
	return result, nil
}
