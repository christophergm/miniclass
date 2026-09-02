package preference

import (
	"context"
	"sort"
	"strings"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
)

type ResponseTrackingInstrumentType string

const (
	ResponseTrackingInterestProfile ResponseTrackingInstrumentType = "interest_profile_survey"
	ResponseTrackingRankedChoice    ResponseTrackingInstrumentType = "ranked_choice_session"
)

const (
	ResponseTrackingUnreachable            = "unreachable"
	ResponseTrackingGuardianFollowUpStatus = "guardian_follow_up"
	ResponseTrackingGuardianNoEmail        = "no_email"
	ResponseTrackingGuardianPending        = "not_responded"
)

// ResponseTracking is a read-only projection for one survey or session. The
// student list is the source of every aggregate; guardian follow-up rows are
// intentionally separate because one student may have several guardians.
type ResponseTrackingSummary struct {
	InstrumentType       ResponseTrackingInstrumentType
	InstrumentID         ids.XID
	InstrumentName       string
	State                string
	SchoolYearID         ids.XID
	ProgramID            ids.XID
	TotalStudents        int
	RespondedStudents    int
	CompletionPercentage float64
}

type ResponseTracking struct {
	InstrumentType       ResponseTrackingInstrumentType
	InstrumentID         ids.XID
	InstrumentName       string
	SchoolYearID         ids.XID
	ProgramID            ids.XID
	TotalStudents        int
	RespondedStudents    int
	CompletionPercentage float64
	GradeBreakdown       []ResponseTrackingBreakdown
	HomeroomBreakdown    []ResponseTrackingBreakdown
	NonResponders        []ResponseTrackingNonResponder
	GuardianFollowUp     []ResponseTrackingGuardianFollowUp
}

type ResponseTrackingBreakdown struct {
	ID                   string
	Label                string
	TotalStudents        int
	RespondedStudents    int
	CompletionPercentage float64
}

type ResponseTrackingNonResponder struct {
	StudentID     ids.XID
	DisplayName   string
	GradeLevelID  *ids.XID
	GradeLabel    string
	HomeroomID    ids.XID
	HomeroomName  string
	ContactStatus string
}

type ResponseTrackingGuardianFollowUp struct {
	AdultID       ids.XID
	AdultName     string
	Email         *string
	StudentID     ids.XID
	StudentName   string
	ContactStatus string
}

func (s *Service) ListResponseTrackingSummaries(ctx context.Context, organizationID string, schoolYearID, programID ids.XID) ([]ResponseTrackingSummary, error) {
	if s == nil || s.database == nil {
		return nil, ErrPreferenceServiceNil
	}
	var result []ResponseTrackingSummary
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		rows, err := tx.ListResponseTrackingSummaries(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result = make([]ResponseTrackingSummary, 0, len(rows))
		for _, row := range rows {
			result = append(result, ResponseTrackingSummary{
				InstrumentType: ResponseTrackingInstrumentType(row.InstrumentType), InstrumentID: row.InstrumentID,
				InstrumentName: row.InstrumentName, State: row.State, SchoolYearID: row.SchoolYearID,
				ProgramID: row.ProgramID, TotalStudents: row.TotalStudents, RespondedStudents: row.RespondedStudents,
				CompletionPercentage: completionPercentage(row.RespondedStudents, row.TotalStudents),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) GetInterestProfileResponseTracking(ctx context.Context, organizationID string, schoolYearID, programID, surveyID ids.XID) (ResponseTracking, error) {
	if s == nil || s.database == nil {
		return ResponseTracking{}, ErrPreferenceServiceNil
	}
	var result ResponseTracking
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		students, err := tx.ListInterestProfileResponseTrackingStudents(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		relationships, adults, err := responseTrackingContacts(ctx, tx, schoolYearID)
		if err != nil {
			return err
		}
		result = buildResponseTracking(ResponseTrackingInterestProfile, survey.ID, survey.Name, schoolYearID, programID, students, relationships, adults)
		return err
	})
	if err != nil {
		return ResponseTracking{}, err
	}
	return result, nil
}

func (s *Service) GetRankedChoiceResponseTracking(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID) (ResponseTracking, error) {
	if s == nil || s.database == nil {
		return ResponseTracking{}, ErrPreferenceServiceNil
	}
	var result ResponseTracking
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		if session.RankedChoice == nil {
			return ErrRankedChoiceNotConfigured
		}
		students, err := tx.ListRankedChoiceResponseTrackingStudents(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		relationships, adults, err := responseTrackingContacts(ctx, tx, schoolYearID)
		if err != nil {
			return err
		}
		result = buildResponseTracking(ResponseTrackingRankedChoice, session.ID, session.Name, schoolYearID, programID, students, relationships, adults)
		return nil
	})
	if err != nil {
		return ResponseTracking{}, err
	}
	return result, nil
}

// buildResponseTracking reads the relationship side only after the student
// denominator has been fixed by the instrument-specific query. This keeps
// multiple guardian edges out of totals and lets the same student appear in
// several follow-up rows.
func responseTrackingContacts(ctx context.Context, tx *data.Tx, schoolYearID ids.XID) ([]data.GuardianRelationship, []data.Adult, error) {
	relationships, err := tx.ListGuardianRelationships(ctx, schoolYearID, data.GuardianRelationshipFilter{})
	if err != nil {
		return nil, nil, err
	}
	adults, err := tx.ListAdults(ctx, schoolYearID, false)
	if err != nil {
		return nil, nil, err
	}
	return relationships, adults, nil
}

func buildResponseTracking(instrumentType ResponseTrackingInstrumentType, instrumentID ids.XID, instrumentName string, schoolYearID, programID ids.XID, students []data.ResponseTrackingStudentRow, relationships []data.GuardianRelationship, adults []data.Adult) ResponseTracking {
	result := ResponseTracking{
		InstrumentType: instrumentType, InstrumentID: instrumentID, InstrumentName: instrumentName,
		SchoolYearID: schoolYearID, ProgramID: programID,
		GradeBreakdown: []ResponseTrackingBreakdown{}, HomeroomBreakdown: []ResponseTrackingBreakdown{},
		NonResponders: []ResponseTrackingNonResponder{}, GuardianFollowUp: []ResponseTrackingGuardianFollowUp{},
	}
	result.TotalStudents = len(students)
	guardiansByStudent := make(map[ids.XID][]data.GuardianRelationship)
	adultsByID := make(map[ids.XID]data.Adult)
	for _, relationship := range relationships {
		guardiansByStudent[relationship.StudentID] = append(guardiansByStudent[relationship.StudentID], relationship)
	}
	for _, adult := range adults {
		adultsByID[adult.ID] = adult
	}

	grade := make(map[string]*ResponseTrackingBreakdown)
	homeroom := make(map[string]*ResponseTrackingBreakdown)
	for _, student := range students {
		if student.Responded {
			result.RespondedStudents++
		}
		gradeKey := stringValue(student.GradeLevelID)
		gradeLabel := strings.TrimSpace(student.GradeLevelLabel)
		if gradeLabel == "" {
			gradeLabel = "Unassigned"
		}
		gradeEntry := grade[gradeKey]
		if gradeEntry == nil {
			gradeEntry = &ResponseTrackingBreakdown{ID: gradeKey, Label: gradeLabel}
			grade[gradeKey] = gradeEntry
		}
		gradeEntry.TotalStudents++
		if student.Responded {
			gradeEntry.RespondedStudents++
		}
		homeroomKey := string(student.HomeroomID)
		homeroomEntry := homeroom[homeroomKey]
		if homeroomEntry == nil {
			homeroomEntry = &ResponseTrackingBreakdown{ID: homeroomKey, Label: student.HomeroomName}
			homeroom[homeroomKey] = homeroomEntry
		}
		homeroomEntry.TotalStudents++
		if student.Responded {
			homeroomEntry.RespondedStudents++
		}
		if student.Responded {
			continue
		}

		legalGiven, legalFamily := student.LegalGivenName, student.LegalFamilyName
		studentName := people.DisplayName(student.PreferredGivenName, &legalGiven, &legalFamily)
		relationships := guardiansByStudent[student.ID]
		contactStatus := ResponseTrackingGuardianFollowUpStatus
		if len(relationships) == 0 {
			contactStatus = ResponseTrackingUnreachable
		}
		result.NonResponders = append(result.NonResponders, ResponseTrackingNonResponder{
			StudentID: student.ID, DisplayName: studentName, GradeLevelID: student.GradeLevelID,
			GradeLabel: gradeLabel, HomeroomID: student.HomeroomID,
			HomeroomName: student.HomeroomName, ContactStatus: contactStatus,
		})
		for _, relationship := range relationships {
			adult, ok := adultsByID[relationship.AdultID]
			if !ok {
				continue
			}
			adultGiven, adultFamily := adult.LegalGivenName, adult.LegalFamilyName
			adultName := people.DisplayName(adult.PreferredGivenName, &adultGiven, &adultFamily)
			contactStatus := ResponseTrackingGuardianPending
			if adult.Email == nil || strings.TrimSpace(*adult.Email) == "" {
				contactStatus = ResponseTrackingGuardianNoEmail
			}
			result.GuardianFollowUp = append(result.GuardianFollowUp, ResponseTrackingGuardianFollowUp{
				AdultID: relationship.AdultID, AdultName: adultName, Email: adult.Email,
				StudentID: student.ID, StudentName: studentName, ContactStatus: contactStatus,
			})
		}
	}
	result.CompletionPercentage = completionPercentage(result.RespondedStudents, result.TotalStudents)
	result.GradeBreakdown = breakdownValues(grade)
	result.HomeroomBreakdown = breakdownValues(homeroom)
	sort.SliceStable(result.GuardianFollowUp, func(i, j int) bool {
		if !strings.EqualFold(result.GuardianFollowUp[i].StudentName, result.GuardianFollowUp[j].StudentName) {
			return strings.ToLower(result.GuardianFollowUp[i].StudentName) < strings.ToLower(result.GuardianFollowUp[j].StudentName)
		}
		if !strings.EqualFold(result.GuardianFollowUp[i].AdultName, result.GuardianFollowUp[j].AdultName) {
			return strings.ToLower(result.GuardianFollowUp[i].AdultName) < strings.ToLower(result.GuardianFollowUp[j].AdultName)
		}
		return result.GuardianFollowUp[i].AdultID < result.GuardianFollowUp[j].AdultID
	})
	return result
}

func stringValue(value *ids.XID) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func breakdownValues(values map[string]*ResponseTrackingBreakdown) []ResponseTrackingBreakdown {
	result := make([]ResponseTrackingBreakdown, 0, len(values))
	for _, value := range values {
		value.CompletionPercentage = completionPercentage(value.RespondedStudents, value.TotalStudents)
		result = append(result, *value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := strings.ToLower(result[i].Label), strings.ToLower(result[j].Label)
		if left != right {
			return left < right
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func completionPercentage(responded, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(responded) * 100 / float64(total)
}
