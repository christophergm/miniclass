package data

import (
	"context"
	"fmt"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

// ResponseTrackingStudentRow is the read snapshot used to build the
// student-centric response report. Responded is computed in SQL with an
// exists predicate, so repeated submissions and repeated guardian links
// cannot duplicate the student denominator.
type ResponseTrackingSummary struct {
	InstrumentType    string
	InstrumentID      ids.XID
	InstrumentName    string
	State             string
	SchoolYearID      ids.XID
	ProgramID         ids.XID
	TotalStudents     int
	RespondedStudents int
}

type ResponseTrackingStudentRow struct {
	ID                 ids.XID
	OrganizationID     ids.XID
	SchoolYearID       ids.XID
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	GradeLevelID       *ids.XID
	GradeLevelLabel    string
	HomeroomID         ids.XID
	HomeroomName       string
	Responded          bool
}

func (tx *Tx) ListResponseTrackingSummaries(ctx context.Context, schoolYearID, programID ids.XID) ([]ResponseTrackingSummary, error) {
	rows, err := tx.queries.ListResponseTrackingSummaries(ctx, db.ListResponseTrackingSummariesParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		ProgramID:      programID,
	})
	if err != nil {
		return nil, fmt.Errorf("list response tracking summaries: %w", err)
	}
	result := make([]ResponseTrackingSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, ResponseTrackingSummary{
			InstrumentType: row.InstrumentType, InstrumentID: row.InstrumentID,
			InstrumentName: row.InstrumentName, State: string(row.State),
			SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID,
			TotalStudents: int(row.TotalStudents), RespondedStudents: int(row.RespondedStudents),
		})
	}
	return result, nil
}

func (tx *Tx) ListInterestProfileResponseTrackingStudents(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]ResponseTrackingStudentRow, error) {
	rows, err := tx.queries.ListInterestProfileResponseTrackingStudents(ctx, db.ListInterestProfileResponseTrackingStudentsParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		ProgramID:      programID,
		SurveyID:       &surveyID,
	})
	if err != nil {
		return nil, fmt.Errorf("list interest profile response tracking students: %w", err)
	}
	result := make([]ResponseTrackingStudentRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, ResponseTrackingStudentRow{
			ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
			LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName,
			PreferredGivenName: nullableStudentString(row.PreferredGivenName), GradeLevelID: row.GradeLevelID,
			GradeLevelLabel: row.GradeLevelLabel, HomeroomID: row.HomeroomID,
			HomeroomName: row.HomeroomName, Responded: row.Responded,
		})
	}
	return result, nil
}

func (tx *Tx) ListRankedChoiceResponseTrackingStudents(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]ResponseTrackingStudentRow, error) {
	rows, err := tx.queries.ListRankedChoiceResponseTrackingStudents(ctx, db.ListRankedChoiceResponseTrackingStudentsParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		ProgramID:      programID,
		SessionID:      sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("list ranked choice response tracking students: %w", err)
	}
	result := make([]ResponseTrackingStudentRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, ResponseTrackingStudentRow{
			ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
			LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName,
			PreferredGivenName: nullableStudentString(row.PreferredGivenName), GradeLevelID: row.GradeLevelID,
			GradeLevelLabel: row.GradeLevelLabel, HomeroomID: row.HomeroomID,
			HomeroomName: row.HomeroomName, Responded: row.Responded,
		})
	}
	return result, nil
}
