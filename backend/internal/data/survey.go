package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type InterestProfileSurveyState string

const (
	InterestProfileSurveyDraft  InterestProfileSurveyState = "draft"
	InterestProfileSurveyOpen   InterestProfileSurveyState = "open"
	InterestProfileSurveyClosed InterestProfileSurveyState = "closed"
)

type InterestProfileSurveyAudienceType string

const (
	SurveyAudienceAllMembers       InterestProfileSurveyAudienceType = "all_members"
	SurveyAudienceExplicitStudents InterestProfileSurveyAudienceType = "explicit_students"
	SurveyAudienceGradeLevel       InterestProfileSurveyAudienceType = "grade_level"
	SurveyAudienceResponseState    InterestProfileSurveyAudienceType = "response_state"
)

type InterestProfileSurveyResponseState string

const (
	SurveyResponded    InterestProfileSurveyResponseState = "responded"
	SurveyNotResponded InterestProfileSurveyResponseState = "not_responded"
)

type InterestProfileSurvey struct {
	ID                    ids.XID
	OrganizationID        ids.XID
	SchoolYearID          ids.XID
	ProgramID             ids.XID
	Name                  string
	Introduction          string
	State                 InterestProfileSurveyState
	OpensAt               *time.Time
	ClosesAt              *time.Time
	AudienceType          InterestProfileSurveyAudienceType
	AudienceGradeLevelID  *ids.XID
	AudiencePriorSurveyID *ids.XID
	AudienceResponseState *InterestProfileSurveyResponseState
	ScaleVersion          string
	OpenedAt              *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type InterestProfileSurveyDefinitionStudent struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SurveyID       ids.XID
	StudentID      ids.XID
	CreatedAt      time.Time
}

type InterestProfileSurveyQuestion struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SurveyID       ids.XID
	InterestAreaID ids.XID
	Ordinal        int
	Label          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type InterestProfileSurveyScaleOption struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SurveyID       ids.XID
	Value          string
	Label          string
	Ordinal        int
	CreatedAt      time.Time
}

type InterestProfileSurveyAudienceSnapshot struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SurveyID       ids.XID
	StudentID      ids.XID
	CreatedAt      time.Time
}

type InterestProfileSurveyAccessCode struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SurveyID       ids.XID
	StudentID      ids.XID
	IssuedAt       time.Time
	RevokedAt      *time.Time
}

type InterestProfileSurveyAudienceStudent struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SurveyID       ids.XID
	StudentID      ids.XID
	CreatedAt      time.Time
}

type InterestProfileSurveyAudienceCandidate struct {
	StudentID    ids.XID
	GradeLevelID *ids.XID
}

type InterestProfileSurveyQuestionInput struct {
	InterestAreaID ids.XID
	Ordinal        int
	Label          string
}

type InterestProfileSurveyScaleOptionInput struct {
	Value   string
	Label   string
	Ordinal int
}

func (tx *Tx) CreateInterestProfileSurvey(ctx context.Context, schoolYearID, programID ids.XID, name, introduction string, opensAt, closesAt *time.Time, audienceType InterestProfileSurveyAudienceType, audienceGradeLevelID, audiencePriorSurveyID *ids.XID, audienceResponseState *InterestProfileSurveyResponseState, scaleVersion string) (InterestProfileSurvey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return InterestProfileSurvey{}, errors.New("create interest profile survey: name is required")
	}
	if strings.TrimSpace(scaleVersion) == "" {
		scaleVersion = defaultSurveyScaleVersion
	}
	row, err := tx.queries.CreateInterestProfileSurvey(ctx, db.CreateInterestProfileSurveyParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
		Name: name, Introduction: strings.TrimSpace(introduction), OpensAt: nullableSurveyTime(opensAt), ClosesAt: nullableSurveyTime(closesAt),
		AudienceType: db.InterestProfileSurveyAudienceType(audienceType), AudienceGradeLevelID: audienceGradeLevelID,
		AudiencePriorSurveyID: audiencePriorSurveyID, AudienceResponseState: nullableSurveyResponseState(audienceResponseState), ScaleVersion: strings.TrimSpace(scaleVersion),
	})
	if err != nil {
		return InterestProfileSurvey{}, fmt.Errorf("create interest profile survey: %w", err)
	}
	return interestProfileSurveyValues(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Introduction, row.State, row.OpensAt, row.ClosesAt, row.AudienceType, row.AudienceGradeLevelID, row.AudiencePriorSurveyID, row.AudienceResponseState, row.ScaleVersion, row.OpenedAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) ListInterestProfileSurveys(ctx context.Context, schoolYearID, programID ids.XID) ([]InterestProfileSurvey, error) {
	rows, err := tx.queries.ListInterestProfileSurveys(ctx, db.ListInterestProfileSurveysParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile surveys: %w", err)
	}
	result := make([]InterestProfileSurvey, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyValues(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Introduction, row.State, row.OpensAt, row.ClosesAt, row.AudienceType, row.AudienceGradeLevelID, row.AudiencePriorSurveyID, row.AudienceResponseState, row.ScaleVersion, row.OpenedAt, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetInterestProfileSurvey(ctx context.Context, schoolYearID, programID, surveyID ids.XID) (InterestProfileSurvey, error) {
	row, err := tx.queries.GetInterestProfileSurvey(ctx, db.GetInterestProfileSurveyParams{ID: surveyID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return InterestProfileSurvey{}, fmt.Errorf("get interest profile survey: %w", err)
	}
	return interestProfileSurveyValues(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Introduction, row.State, row.OpensAt, row.ClosesAt, row.AudienceType, row.AudienceGradeLevelID, row.AudiencePriorSurveyID, row.AudienceResponseState, row.ScaleVersion, row.OpenedAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateInterestProfileSurvey(ctx context.Context, schoolYearID, programID, surveyID ids.XID, name, introduction string, opensAt, closesAt *time.Time, audienceType InterestProfileSurveyAudienceType, audienceGradeLevelID, audiencePriorSurveyID *ids.XID, audienceResponseState *InterestProfileSurveyResponseState, scaleVersion string) (InterestProfileSurvey, error) {
	row, err := tx.queries.UpdateInterestProfileSurvey(ctx, db.UpdateInterestProfileSurveyParams{
		ID: surveyID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
		Name: strings.TrimSpace(name), Introduction: strings.TrimSpace(introduction), OpensAt: nullableSurveyTime(opensAt), ClosesAt: nullableSurveyTime(closesAt),
		AudienceType: db.InterestProfileSurveyAudienceType(audienceType), AudienceGradeLevelID: audienceGradeLevelID,
		AudiencePriorSurveyID: audiencePriorSurveyID, AudienceResponseState: nullableSurveyResponseState(audienceResponseState), ScaleVersion: strings.TrimSpace(scaleVersion),
	})
	if err != nil {
		return InterestProfileSurvey{}, fmt.Errorf("update interest profile survey: %w", err)
	}
	return interestProfileSurvey(row)
}

func (tx *Tx) SetInterestProfileSurveyState(ctx context.Context, schoolYearID, programID, surveyID ids.XID, state InterestProfileSurveyState, opensAt, closesAt, openedAt *time.Time) (InterestProfileSurvey, error) {
	row, err := tx.queries.SetInterestProfileSurveyState(ctx, db.SetInterestProfileSurveyStateParams{
		ID: surveyID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
		State: db.InterestProfileSurveyState(state), OpensAt: nullableSurveyTime(opensAt), ClosesAt: nullableSurveyTime(closesAt), OpenedAt: nullableSurveyTime(openedAt),
	})
	if err != nil {
		return InterestProfileSurvey{}, fmt.Errorf("set interest profile survey state: %w", err)
	}
	return interestProfileSurvey(row)
}

func (tx *Tx) DeleteInterestProfileSurvey(ctx context.Context, schoolYearID, programID, surveyID ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteInterestProfileSurvey(ctx, db.DeleteInterestProfileSurveyParams{ID: surveyID, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return false, fmt.Errorf("delete interest profile survey: %w", err)
	}
	return rows == 1, nil
}

func (tx *Tx) CountInterestProfileSurveySubmissions(ctx context.Context, schoolYearID, programID, surveyID ids.XID) (int64, error) {
	submissions, err := tx.queries.CountInterestProfileSurveySubmissions(ctx, db.CountInterestProfileSurveySubmissionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: &surveyID})
	if err != nil {
		return 0, fmt.Errorf("count interest profile survey submissions: %w", err)
	}
	return submissions, nil
}

func (tx *Tx) ListInterestProfileSurveyMembers(ctx context.Context, schoolYearID, programID ids.XID) ([]InterestProfileSurveyAudienceCandidate, error) {
	rows, err := tx.queries.ListInterestProfileSurveyMembers(ctx, db.ListInterestProfileSurveyMembersParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile survey members: %w", err)
	}
	result := make([]InterestProfileSurveyAudienceCandidate, 0, len(rows))
	for _, row := range rows {
		result = append(result, InterestProfileSurveyAudienceCandidate{StudentID: row.StudentID, GradeLevelID: row.GradeLevelID})
	}
	return result, nil
}

func (tx *Tx) ListInterestProfileSurveyDefinitionStudents(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]InterestProfileSurveyAudienceStudent, error) {
	rows, err := tx.queries.ListInterestProfileSurveyDefinitionStudents(ctx, db.ListInterestProfileSurveyDefinitionStudentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile survey audience students: %w", err)
	}
	result := make([]InterestProfileSurveyAudienceStudent, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyDefinitionStudent(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) CreateInterestProfileSurveyDefinitionStudent(ctx context.Context, schoolYearID, programID, surveyID, studentID ids.XID) (InterestProfileSurveyAudienceStudent, error) {
	row, err := tx.queries.CreateInterestProfileSurveyDefinitionStudent(ctx, db.CreateInterestProfileSurveyDefinitionStudentParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID, StudentID: studentID})
	if err != nil {
		return InterestProfileSurveyAudienceStudent{}, fmt.Errorf("create interest profile survey audience student: %w", err)
	}
	return interestProfileSurveyDefinitionStudent(row)
}

func (tx *Tx) DeleteInterestProfileSurveyDefinitionStudents(ctx context.Context, schoolYearID, programID, surveyID ids.XID) error {
	if err := tx.queries.DeleteInterestProfileSurveyDefinitionStudents(ctx, db.DeleteInterestProfileSurveyDefinitionStudentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID}); err != nil {
		return fmt.Errorf("delete interest profile survey audience students: %w", err)
	}
	return nil
}

func (tx *Tx) ListInterestProfileSurveyQuestions(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]InterestProfileSurveyQuestion, error) {
	rows, err := tx.queries.ListInterestProfileSurveyQuestions(ctx, db.ListInterestProfileSurveyQuestionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile survey questions: %w", err)
	}
	result := make([]InterestProfileSurveyQuestion, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyQuestion(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) CreateInterestProfileSurveyQuestion(ctx context.Context, schoolYearID, programID, surveyID ids.XID, input InterestProfileSurveyQuestionInput) (InterestProfileSurveyQuestion, error) {
	row, err := tx.queries.CreateInterestProfileSurveyQuestion(ctx, db.CreateInterestProfileSurveyQuestionParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID, InterestAreaID: input.InterestAreaID, Ordinal: int32(input.Ordinal), Label: strings.TrimSpace(input.Label)})
	if err != nil {
		return InterestProfileSurveyQuestion{}, fmt.Errorf("create interest profile survey question: %w", err)
	}
	return interestProfileSurveyQuestion(row)
}

func (tx *Tx) DeleteInterestProfileSurveyQuestions(ctx context.Context, schoolYearID, programID, surveyID ids.XID) error {
	if err := tx.queries.DeleteInterestProfileSurveyQuestions(ctx, db.DeleteInterestProfileSurveyQuestionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID}); err != nil {
		return fmt.Errorf("delete interest profile survey questions: %w", err)
	}
	return nil
}

func (tx *Tx) ListInterestProfileSurveyScaleOptions(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]InterestProfileSurveyScaleOption, error) {
	rows, err := tx.queries.ListInterestProfileSurveyScaleOptions(ctx, db.ListInterestProfileSurveyScaleOptionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile survey scale options: %w", err)
	}
	result := make([]InterestProfileSurveyScaleOption, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyScaleOption(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) CreateInterestProfileSurveyScaleOption(ctx context.Context, schoolYearID, programID, surveyID ids.XID, input InterestProfileSurveyScaleOptionInput) (InterestProfileSurveyScaleOption, error) {
	row, err := tx.queries.CreateInterestProfileSurveyScaleOption(ctx, db.CreateInterestProfileSurveyScaleOptionParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID, Value: strings.TrimSpace(input.Value), Label: strings.TrimSpace(input.Label), Ordinal: int32(input.Ordinal)})
	if err != nil {
		return InterestProfileSurveyScaleOption{}, fmt.Errorf("create interest profile survey scale option: %w", err)
	}
	return interestProfileSurveyScaleOption(row)
}

func (tx *Tx) DeleteInterestProfileSurveyScaleOptions(ctx context.Context, schoolYearID, programID, surveyID ids.XID) error {
	if err := tx.queries.DeleteInterestProfileSurveyScaleOptions(ctx, db.DeleteInterestProfileSurveyScaleOptionsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID}); err != nil {
		return fmt.Errorf("delete interest profile survey scale options: %w", err)
	}
	return nil
}

func (tx *Tx) CreateInterestProfileSurveyAudienceSnapshot(ctx context.Context, schoolYearID, programID, surveyID, studentID ids.XID) (InterestProfileSurveyAudienceSnapshot, error) {
	row, err := tx.queries.CreateInterestProfileSurveyAudienceSnapshot(ctx, db.CreateInterestProfileSurveyAudienceSnapshotParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID, StudentID: studentID})
	if err != nil {
		return InterestProfileSurveyAudienceSnapshot{}, fmt.Errorf("create interest profile survey audience snapshot: %w", err)
	}
	return interestProfileSurveyAudienceSnapshot(row)
}

func (tx *Tx) ListInterestProfileSurveyAudienceSnapshot(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]InterestProfileSurveyAudienceSnapshot, error) {
	rows, err := tx.queries.ListInterestProfileSurveyAudienceSnapshot(ctx, db.ListInterestProfileSurveyAudienceSnapshotParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile survey audience snapshot: %w", err)
	}
	result := make([]InterestProfileSurveyAudienceSnapshot, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyAudienceSnapshot(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) CreateInterestProfileSurveyAccessCode(ctx context.Context, schoolYearID, programID, surveyID, studentID ids.XID, codeHash string) (InterestProfileSurveyAccessCode, error) {
	row, err := tx.queries.CreateInterestProfileSurveyAccessCode(ctx, db.CreateInterestProfileSurveyAccessCodeParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID, StudentID: studentID, CodeHash: codeHash})
	if err != nil {
		return InterestProfileSurveyAccessCode{}, fmt.Errorf("create interest profile survey access code: %w", err)
	}
	return interestProfileSurveyAccessCode(row)
}

func (tx *Tx) ListActiveInterestProfileSurveyAccessCodes(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]InterestProfileSurveyAccessCode, error) {
	rows, err := tx.queries.ListActiveInterestProfileSurveyAccessCodes(ctx, db.ListActiveInterestProfileSurveyAccessCodesParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID})
	if err != nil {
		return nil, fmt.Errorf("list active interest profile survey access codes: %w", err)
	}
	result := make([]InterestProfileSurveyAccessCode, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyAccessCode(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) RevokeInterestProfileSurveyAccessCodes(ctx context.Context, schoolYearID, programID, surveyID ids.XID) (int64, error) {
	count, err := tx.queries.RevokeInterestProfileSurveyAccessCodes(ctx, db.RevokeInterestProfileSurveyAccessCodesParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID})
	if err != nil {
		return 0, fmt.Errorf("revoke interest profile survey access codes: %w", err)
	}
	return count, nil
}

func (tx *Tx) FindActiveInterestProfileSurveyAccessCode(ctx context.Context, schoolYearID, programID, surveyID ids.XID, codeHash string) (ids.XID, error) {
	row, err := tx.queries.FindActiveInterestProfileSurveyAccessCode(ctx, db.FindActiveInterestProfileSurveyAccessCodeParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: surveyID, CodeHash: codeHash})
	if err != nil {
		return "", fmt.Errorf("find active interest profile survey access code: %w", err)
	}
	return row.StudentID, nil
}

func (tx *Tx) ListInterestProfileSurveyPriorResponders(ctx context.Context, schoolYearID, programID, surveyID ids.XID) ([]ids.XID, error) {
	rows, err := tx.queries.ListInterestProfileSurveyPriorResponders(ctx, db.ListInterestProfileSurveyPriorRespondersParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SurveyID: &surveyID})
	if err != nil {
		return nil, fmt.Errorf("list interest profile survey prior responders: %w", err)
	}
	return rows, nil
}

func (tx *Tx) ListAllInterestProfileSurveysForRegistry(ctx context.Context) ([]InterestProfileSurvey, error) {
	rows, err := tx.queries.ListAllInterestProfileSurveysForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list interest profile surveys for registry: %w", err)
	}
	result := make([]InterestProfileSurvey, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurvey(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSurveyForRegistry(ctx context.Context, id ids.XID) (InterestProfileSurvey, error) {
	row, err := tx.queries.FindInterestProfileSurveyForRegistry(ctx, db.FindInterestProfileSurveyForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSurvey{}, nil
	}
	if err != nil {
		return InterestProfileSurvey{}, fmt.Errorf("find interest profile survey for registry: %w", err)
	}
	return interestProfileSurvey(row)
}

func (tx *Tx) UpdateInterestProfileSurveyForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	_, err := tx.queries.UpdateInterestProfileSurveyForRegistry(ctx, db.UpdateInterestProfileSurveyForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (tx *Tx) DeleteInterestProfileSurveyForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteInterestProfileSurveyForRegistry(ctx, db.DeleteInterestProfileSurveyForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) ListAllInterestProfileSurveyDefinitionStudentsForRegistry(ctx context.Context) ([]InterestProfileSurveyAudienceStudent, error) {
	rows, err := tx.queries.ListAllInterestProfileSurveyDefinitionStudentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]InterestProfileSurveyAudienceStudent, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyDefinitionStudent(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSurveyDefinitionStudentForRegistry(ctx context.Context, id ids.XID) (InterestProfileSurveyAudienceStudent, error) {
	row, err := tx.queries.FindInterestProfileSurveyDefinitionStudentForRegistry(ctx, db.FindInterestProfileSurveyDefinitionStudentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSurveyAudienceStudent{}, nil
	}
	if err != nil {
		return InterestProfileSurveyAudienceStudent{}, err
	}
	return interestProfileSurveyDefinitionStudent(row)
}

func (tx *Tx) UpdateInterestProfileSurveyDefinitionStudentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.UpdateInterestProfileSurveyDefinitionStudentForRegistry(ctx, db.UpdateInterestProfileSurveyDefinitionStudentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) DeleteInterestProfileSurveyDefinitionStudentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteInterestProfileSurveyDefinitionStudentForRegistry(ctx, db.DeleteInterestProfileSurveyDefinitionStudentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) ListAllInterestProfileSurveyQuestionsForRegistry(ctx context.Context) ([]InterestProfileSurveyQuestion, error) {
	rows, err := tx.queries.ListAllInterestProfileSurveyQuestionsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]InterestProfileSurveyQuestion, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyQuestion(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSurveyQuestionForRegistry(ctx context.Context, id ids.XID) (InterestProfileSurveyQuestion, error) {
	row, err := tx.queries.FindInterestProfileSurveyQuestionForRegistry(ctx, db.FindInterestProfileSurveyQuestionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSurveyQuestion{}, nil
	}
	if err != nil {
		return InterestProfileSurveyQuestion{}, err
	}
	return interestProfileSurveyQuestion(row)
}

func (tx *Tx) UpdateInterestProfileSurveyQuestionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.UpdateInterestProfileSurveyQuestionForRegistry(ctx, db.UpdateInterestProfileSurveyQuestionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) DeleteInterestProfileSurveyQuestionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteInterestProfileSurveyQuestionForRegistry(ctx, db.DeleteInterestProfileSurveyQuestionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) ListAllInterestProfileSurveyScaleOptionsForRegistry(ctx context.Context) ([]InterestProfileSurveyScaleOption, error) {
	rows, err := tx.queries.ListAllInterestProfileSurveyScaleOptionsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]InterestProfileSurveyScaleOption, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyScaleOption(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSurveyScaleOptionForRegistry(ctx context.Context, id ids.XID) (InterestProfileSurveyScaleOption, error) {
	row, err := tx.queries.FindInterestProfileSurveyScaleOptionForRegistry(ctx, db.FindInterestProfileSurveyScaleOptionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSurveyScaleOption{}, nil
	}
	if err != nil {
		return InterestProfileSurveyScaleOption{}, err
	}
	return interestProfileSurveyScaleOption(row)
}

func (tx *Tx) UpdateInterestProfileSurveyScaleOptionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.UpdateInterestProfileSurveyScaleOptionForRegistry(ctx, db.UpdateInterestProfileSurveyScaleOptionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) DeleteInterestProfileSurveyScaleOptionForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.DeleteInterestProfileSurveyScaleOptionForRegistry(ctx, db.DeleteInterestProfileSurveyScaleOptionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) ListAllInterestProfileSurveyAudienceSnapshotsForRegistry(ctx context.Context) ([]InterestProfileSurveyAudienceSnapshot, error) {
	rows, err := tx.queries.ListAllInterestProfileSurveyAudienceSnapshotsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]InterestProfileSurveyAudienceSnapshot, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyAudienceSnapshot(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSurveyAudienceSnapshotForRegistry(ctx context.Context, id ids.XID) (InterestProfileSurveyAudienceSnapshot, error) {
	row, err := tx.queries.FindInterestProfileSurveyAudienceSnapshotForRegistry(ctx, db.FindInterestProfileSurveyAudienceSnapshotForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSurveyAudienceSnapshot{}, nil
	}
	if err != nil {
		return InterestProfileSurveyAudienceSnapshot{}, err
	}
	return interestProfileSurveyAudienceSnapshot(row)
}

func (tx *Tx) ListAllInterestProfileSurveyAccessCodesForRegistry(ctx context.Context) ([]InterestProfileSurveyAccessCode, error) {
	rows, err := tx.queries.ListAllInterestProfileSurveyAccessCodesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]InterestProfileSurveyAccessCode, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSurveyAccessCode(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSurveyAccessCodeForRegistry(ctx context.Context, id ids.XID) (InterestProfileSurveyAccessCode, error) {
	row, err := tx.queries.FindInterestProfileSurveyAccessCodeForRegistry(ctx, db.FindInterestProfileSurveyAccessCodeForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSurveyAccessCode{}, nil
	}
	if err != nil {
		return InterestProfileSurveyAccessCode{}, err
	}
	return interestProfileSurveyAccessCode(row)
}

func (tx *Tx) RevokeInterestProfileSurveyAccessCodeForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.RevokeInterestProfileSurveyAccessCodeForRegistry(ctx, db.RevokeInterestProfileSurveyAccessCodeForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

const defaultSurveyScaleVersion = "interest-profile-3-point-v1"

func nullableSurveyTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func nullableSurveyResponseState(value *InterestProfileSurveyResponseState) db.NullInterestProfileSurveyResponseState {
	if value == nil {
		return db.NullInterestProfileSurveyResponseState{}
	}
	return db.NullInterestProfileSurveyResponseState{InterestProfileSurveyResponseState: db.InterestProfileSurveyResponseState(*value), Valid: true}
}

func interestProfileSurvey(row db.InterestProfileSurvey) (InterestProfileSurvey, error) {
	return interestProfileSurveyValues(row.ID, row.OrganizationID, row.SchoolYearID, row.ProgramID, row.Name, row.Introduction, row.State, row.OpensAt, row.ClosesAt, row.AudienceType, row.AudienceGradeLevelID, row.AudiencePriorSurveyID, row.AudienceResponseState, row.ScaleVersion, row.OpenedAt, row.CreatedAt, row.UpdatedAt)
}

func interestProfileSurveyValues(id, organizationID, schoolYearID, programID ids.XID, name, introduction string, state interface{}, opensAt, closesAt pgtype.Timestamptz, audienceType db.InterestProfileSurveyAudienceType, audienceGradeLevelID, audiencePriorSurveyID *ids.XID, audienceResponseState db.NullInterestProfileSurveyResponseState, scaleVersion string, openedAt, createdAt, updatedAt pgtype.Timestamptz) (InterestProfileSurvey, error) {
	var stateValue db.InterestProfileSurveyState
	switch value := state.(type) {
	case db.InterestProfileSurveyState:
		stateValue = value
	case string:
		stateValue = db.InterestProfileSurveyState(value)
	default:
		return InterestProfileSurvey{}, fmt.Errorf("interest profile survey: invalid state %T", state)
	}
	created, err := programTime(createdAt, "created_at")
	if err != nil {
		return InterestProfileSurvey{}, err
	}
	updated, err := programTime(updatedAt, "updated_at")
	if err != nil {
		return InterestProfileSurvey{}, err
	}
	var responseState *InterestProfileSurveyResponseState
	if audienceResponseState.Valid {
		value := InterestProfileSurveyResponseState(audienceResponseState.InterestProfileSurveyResponseState)
		responseState = &value
	}
	return InterestProfileSurvey{ID: id, OrganizationID: organizationID, SchoolYearID: schoolYearID, ProgramID: programID, Name: name, Introduction: introduction, State: InterestProfileSurveyState(stateValue), OpensAt: nullableTime(opensAt), ClosesAt: nullableTime(closesAt), AudienceType: InterestProfileSurveyAudienceType(audienceType), AudienceGradeLevelID: audienceGradeLevelID, AudiencePriorSurveyID: audiencePriorSurveyID, AudienceResponseState: responseState, ScaleVersion: scaleVersion, OpenedAt: nullableTime(openedAt), CreatedAt: created, UpdatedAt: updated}, nil
}

func interestProfileSurveyDefinitionStudent(row db.InterestProfileSurveyAudienceStudent) (InterestProfileSurveyAudienceStudent, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestProfileSurveyAudienceStudent{}, err
	}
	return InterestProfileSurveyAudienceStudent{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SurveyID: row.SurveyID, StudentID: row.StudentID, CreatedAt: created}, nil
}

func interestProfileSurveyQuestion(row db.InterestProfileSurveyQuestion) (InterestProfileSurveyQuestion, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestProfileSurveyQuestion{}, err
	}
	updated, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return InterestProfileSurveyQuestion{}, err
	}
	return InterestProfileSurveyQuestion{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SurveyID: row.SurveyID, InterestAreaID: row.InterestAreaID, Ordinal: int(row.Ordinal), Label: row.Label, CreatedAt: created, UpdatedAt: updated}, nil
}

func interestProfileSurveyScaleOption(row db.InterestProfileSurveyScaleOption) (InterestProfileSurveyScaleOption, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestProfileSurveyScaleOption{}, err
	}
	return InterestProfileSurveyScaleOption{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SurveyID: row.SurveyID, Value: row.Value, Label: row.Label, Ordinal: int(row.Ordinal), CreatedAt: created}, nil
}

func interestProfileSurveyAudienceSnapshot(row db.InterestProfileSurveyAudienceSnapshot) (InterestProfileSurveyAudienceSnapshot, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestProfileSurveyAudienceSnapshot{}, err
	}
	return InterestProfileSurveyAudienceSnapshot{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SurveyID: row.SurveyID, StudentID: row.StudentID, CreatedAt: created}, nil
}

func interestProfileSurveyAccessCode(row db.InterestProfileSurveyAccessCode) (InterestProfileSurveyAccessCode, error) {
	issued, err := programTime(row.IssuedAt, "issued_at")
	if err != nil {
		return InterestProfileSurveyAccessCode{}, err
	}
	return InterestProfileSurveyAccessCode{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SurveyID: row.SurveyID, StudentID: row.StudentID, IssuedAt: issued, RevokedAt: nullableTime(row.RevokedAt)}, nil
}
